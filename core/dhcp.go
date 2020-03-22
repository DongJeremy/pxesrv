package core

import (
	"fmt"
	"net"
	"sync"
	"time"

	dhcp "github.com/krolaw/dhcp4"
)

func (s *Service) serveDHCP(conn dhcp.ServeConn) error {
	ipxeBootScript := fmt.Sprintf("http://%s:%s/%s", s.ServiceIP, s.HTTPPort, s.IPXEBootScript)
	dhcpService := &DHCPService{
		ServiceIP:          net.ParseIP(s.ServiceIP),
		IPRangeStart:       net.ParseIP(s.IPRangeStart),
		IPRangeEnd:         net.ParseIP(s.IPRangeEnd),
		leasesByMACAddress: make(map[string]*RecordLease),
		LeaseDuration:      24 * time.Hour,
		EnableIPXE:         s.EnableIPXE,
		stateLock:          &sync.Mutex{},
		PXEBootImage:       s.PXEBootImage,
		IPXEBootScript:     ipxeBootScript,
		dhcpOptions: dhcp.Options{
			dhcp.OptionSubnetMask:       net.ParseIP(s.NetMask).To4(),
			dhcp.OptionRouter:           []byte(s.Router),
			dhcp.OptionDomainNameServer: []byte(s.DNSServer),      // Presuming Server is also your DNS server
			dhcp.OptionTFTPServerName:   []byte(s.TFTPServerName), // tftp_files server address
		},
	}
	log.Infof("starting dhcp server and linstening on %s:%s", s.ServiceIP, s.DHCPPort)

	if err := dhcp.Serve(conn, dhcpService); err != nil {
		log.Errorf("DHCP server shut down: %s", err)
		return err
	}
	return nil
}

// A DHCPService represents the state for the All service.
type DHCPService struct {
	//Config Config
	ServiceIP          net.IP
	IPRangeStart       net.IP // dhcp ip range start
	IPRangeEnd         net.IP // dhcp ip range end
	LeaseDuration      time.Duration
	TFTPServerName     string
	PXEBootImage       string // PXE boot file (TFTP)
	IPXEBootScript     string // iPXE boot script (HTTP)
	EnableIPXE         bool
	dhcpOptions        dhcp.Options
	leasesByMACAddress map[string]*RecordLease
	stateLock          *sync.Mutex
}

// NewDHCPService creates new Service state.
func NewDHCPService() *DHCPService {
	service := &DHCPService{
		leasesByMACAddress: make(map[string]*RecordLease),
		LeaseDuration:      24 * time.Hour,
		dhcpOptions: dhcp.Options{
			dhcp.OptionDomainNameServer: []byte{8, 8, 8, 8},
		},
		EnableIPXE: true,
		stateLock:  &sync.Mutex{},
	}
	return service
}

// ServeDHCP handles an incoming DHCP request.
func (s *DHCPService) ServeDHCP(request dhcp.Packet, msgType dhcp.MessageType, requestOptions dhcp.Options) (response dhcp.Packet) {
	switch msgType {
	case dhcp.Discover:
		response = s.handleDiscover(request, requestOptions)

	case dhcp.Request:
		response = s.handleRequest(request, requestOptions)

	case dhcp.Release:
		response = s.handleRelease(request, requestOptions)
	default:
		log.Infof("[TXN: %s] Ignoring unhandled DHCP message type (%s).",
			getTransactionID(request),
			msgType.String(),
		)

		response = s.replyNAK(request)
	}

	if response != nil {
		response.PadToMinSize() // Must add padding AFTER all other options.
	}

	return
}

// handleDiscover Handle a DHCP Discover packet.
func (s *DHCPService) handleDiscover(request dhcp.Packet, requestOptions dhcp.Options) (response dhcp.Packet) {
	transactionID := getTransactionID(request)
	clientMACAddress := request.CHAddr().String()

	log.Infof("[TXN: %s] Discover message from client with MAC address %s (IP '%s').",
		transactionID,
		clientMACAddress,
		request.CIAddr().String(),
	)

	var targetIP net.IP

	existingLease, ok := s.leasesByMACAddress[clientMACAddress]
	if ok {
		targetIP = existingLease.IPAddress
	} else {
		newRecordLease, err := s.createIP(clientMACAddress, s.IPRangeStart, s.IPRangeEnd)
		if err != nil {
			log.Infof("[TXN: %s] MAC address %s could not get a new available IP address (no reply will be sent).",
				transactionID,
				clientMACAddress,
			)
			return s.noReply()
		}
		targetIP = newRecordLease.IPAddress
	}

	return s.replyOffer(request, targetIP, requestOptions)
}

// Create an Offer reply packet (in response to Discover packet).
func (s *DHCPService) replyOffer(request dhcp.Packet, targetIP net.IP, requestOptions dhcp.Options) (response dhcp.Packet) {
	transactionID := getTransactionID(request)
	clientMACAddress := request.CHAddr().String()

	reply := newReply(request, dhcp.Offer, s.ServiceIP,
		targetIP,
		s.LeaseDuration,
		s.dhcpOptions.SelectOrderOrAll(requestOptions[dhcp.OptionParameterRequestList]),
	)

	log.Infof("[TXN: %s] Offer message from server with MAC address %s (IP '%s').",
		transactionID,
		clientMACAddress,
		targetIP.String(),
	)

	// Configure host name from server name.
	reply.AddOption(dhcp.OptionHostName,
		[]byte(""),
	)

	// Add DHCP options for PXE / iPXE, if required.
	if s.EnableIPXE && isPXEClient(requestOptions) {
		s.addIPXEOptions(request, requestOptions, reply)
	}

	// Set the DHCP server identity (i.e. DHCP server address).
	reply.SetSIAddr(s.ServiceIP)

	return reply
}

// Create an empty reply packet (i.e. no reply should be sent)
func (s *DHCPService) noReply() dhcp.Packet {
	return dhcp.Packet{}
}

// Create a NAK reply packet (in response to Discover or Request packet)
func (s *DHCPService) replyNAK(request dhcp.Packet) (response dhcp.Packet) {
	reply := newReply(request, dhcp.NAK, s.ServiceIP,
		nil,
		0,
		nil,
	)

	reply.SetSIAddr(s.ServiceIP)

	return reply
}

// Handle a DHCP Request packet.
func (s *DHCPService) handleRequest(request dhcp.Packet, requestOptions dhcp.Options) (response dhcp.Packet) {
	transactionID := getTransactionID(request)
	clientMACAddress := request.CHAddr().String()

	log.Infof("[TXN: %s] Request message from client with MAC address %s (IP '%s').",
		transactionID,
		clientMACAddress,
		request.CIAddr().String(),
	)

	// Is this a renewal?
	existingLease, ok := s.leasesByMACAddress[clientMACAddress]
	if ok {
		if !existingLease.IsExpired() {
			log.Infof("[TXN: %s] Renew lease on IPv4 address %s for server %s and send ACK reply.",
				transactionID,
				existingLease.IPAddress.String(),
				clientMACAddress,
			)

			s.renewLease(existingLease)

			return s.replyACK(request, existingLease.IPAddress, requestOptions)
		}
		// New lease
		targetIP := existingLease.IPAddress
		log.Infof("[TXN: %s] Create lease on IPv4 address %s for server (MAC address %s) and send ACK reply.",
			transactionID,
			targetIP.String(),
			clientMACAddress,
		)
		newLease := s.createLease(clientMACAddress, targetIP)

		return s.replyACK(request, newLease.IPAddress, requestOptions)

	}
	return s.replyNAK(request)
}

// Create an ACK reply packet (in response to Request packet).
func (s *DHCPService) replyACK(request dhcp.Packet, targetIP net.IP, requestOptions dhcp.Options) (response dhcp.Packet) {
	reply := newReply(request, dhcp.ACK, s.ServiceIP,
		targetIP,
		s.LeaseDuration,
		s.dhcpOptions.SelectOrderOrAll(requestOptions[dhcp.OptionParameterRequestList]),
	)

	// Configure host name from server name.
	reply.AddOption(dhcp.OptionHostName,
		[]byte(""),
	)

	// Add DHCP options for PXE / iPXE, if required.
	if s.EnableIPXE && isPXEClient(requestOptions) {
		s.addIPXEOptions(request, requestOptions, reply)
	}

	// Set the DHCP server identity (i.e. DHCP server address).
	reply.SetSIAddr(s.ServiceIP)

	return reply
}

// Handle a DHCP Release packet.
func (s *DHCPService) handleRelease(request dhcp.Packet, requestOptions dhcp.Options) (response dhcp.Packet) {
	transactionID := getTransactionID(request)
	clientMACAddress := request.CHAddr().String()

	log.Infof("[TXN: %s] Release message from client with MAC address %s (IP '%s').",
		transactionID,
		clientMACAddress,
		request.CIAddr().String(),
	)

	existingLease, ok := s.leasesByMACAddress[clientMACAddress]
	if ok && !existingLease.IsExpired() {
		log.Infof("[TXN: %s] Server '%s' requested termination of lease on IPv4 address %s.",
			transactionID,
			clientMACAddress,
			existingLease.IPAddress.String(),
		)

		s.expireLease(existingLease)
	} else {
		log.Infof("[TXN: %s] Server '%s' requested requested termination of expired or non-existent lease; request ignored.",
			transactionID,
			clientMACAddress,
		)
	}

	return s.noReply() // No reply is necessary for Release.
}

// Add options for PXE / iPXE to a DHCP response.
func (s *DHCPService) addIPXEOptions(request dhcp.Packet, requestOptions dhcp.Options, reply dhcp.Packet) {
	transactionID := getTransactionID(request)

	if isIPXEClient(requestOptions) {
		// This is an iPXE client; direct them to load the iPXE boot script.
		log.Infof("[TXN: %s] Client with MAC address %s is an iPXE client; directing them to boot script '%s'.",
			transactionID,
			request.CHAddr().String(),
			s.IPXEBootScript,
		)

		s.addIPXEBootScript(reply)
	} else {
		// This is a PXE client; direct them to load the standard PXE boot image.
		log.Infof("[TXN: %s] Client with MAC address %s is a regular PXE (or non-PXE) client; directing them to iPXE boot image 'tftp://%s/%s'.",
			transactionID,
			request.CHAddr().String(),
			s.ServiceIP,
			s.PXEBootImage,
		)

		s.addPXEBootImage(reply)
	}
}

// Add an IPXE boot script URL to a DHCP response.
func (s *DHCPService) addIPXEBootScript(response dhcp.Packet) {
	ipxeBootScript := s.IPXEBootScript

	addBootFile(response, ipxeBootScript)
	addBootFileOption(response, ipxeBootScript)
}

// Add a PXE boot image (and TFTP server) to a DHCP response.
func (s *DHCPService) addPXEBootImage(response dhcp.Packet) {
	pxeBootImage := s.PXEBootImage

	addBootFile(response, pxeBootImage)
	addTFTPBootFile(response, s.TFTPServerName, pxeBootImage)
}
