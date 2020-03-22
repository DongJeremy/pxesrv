package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	dhcp "github.com/krolaw/dhcp4"
)

// RecordLease represents a DHCP address and lease.
type RecordLease struct {
	// The MAC address of the machine to which the lease belongs.
	MACAddress string
	// The leased IPv4 address.
	IPAddress net.IP
	// The date and time when the lease expires.
	Expires time.Time
}

func init() {
	// set rand seed
	rand.Seed(time.Now().Unix())
}

// IsExpired determines whether the lease has expired.
func (r *RecordLease) IsExpired() bool {
	return time.Now().Sub(r.Expires) >= 0
}

// createIP allocates a new lease in the provided range.
func (s *DHCPService) createIP(clientMACAddress string, rangeStart net.IP, rangeEnd net.IP) (*RecordLease, error) {
	s.acquireStateLock("createAddress")
	defer s.releaseStateLock("createAddress")
	ip := make([]byte, 4)
	rangeStartInt := binary.BigEndian.Uint32(rangeStart.To4())
	rangeEndInt := binary.BigEndian.Uint32(rangeEnd.To4())
	binary.BigEndian.PutUint32(ip, random(rangeStartInt, rangeEndInt))
	taken := s.checkIfTaken(ip)
	for taken {
		ipInt := binary.BigEndian.Uint32(ip)
		ipInt++
		binary.BigEndian.PutUint32(ip, ipInt)
		if ipInt > rangeEndInt {
			break
		}
		taken = s.checkIfTaken(ip)
	}
	for taken {
		ipInt := binary.BigEndian.Uint32(ip)
		ipInt--
		binary.BigEndian.PutUint32(ip, ipInt)
		if ipInt < rangeStartInt {
			return &RecordLease{}, errors.New("no new IP addresses available")
		}
		taken = s.checkIfTaken(ip)
	}
	newLease := &RecordLease{
		MACAddress: clientMACAddress,
		IPAddress:  ip,
		Expires:    time.Now(),
	}
	s.leasesByMACAddress[clientMACAddress] = newLease

	return newLease, nil
}

// check if an IP address is already leased. DHCPv4 only.
func (s *DHCPService) checkIfTaken(ip net.IP) bool {
	taken := false
	for _, v := range s.leasesByMACAddress {
		if v.IPAddress.String() == ip.String() && (v.Expires.After(time.Now())) {
			taken = true
			break
		}
	}
	return taken
}

func random(min uint32, max uint32) uint32 {
	return uint32(rand.Intn(int(max-min))) + min
}

func (s *DHCPService) createLease(clientMACAddress string, ipAddress net.IP) RecordLease {
	newLease := &RecordLease{
		MACAddress: clientMACAddress,
		IPAddress:  ipAddress,
		Expires:    time.Now().Add(s.LeaseDuration),
	}
	s.leasesByMACAddress[clientMACAddress] = newLease

	return *newLease
}

// Renew lease.
func (s *DHCPService) renewLease(lease *RecordLease) {
	s.acquireStateLock("renewLease")
	defer s.releaseStateLock("renewLease")

	lease.Expires = time.Now().Add(s.LeaseDuration)
}

// Remove a lease.
func (s *DHCPService) expireLease(lease *RecordLease) {
	s.acquireStateLock("expireLease")
	defer s.releaseStateLock("expireLease")

	lease.Expires = time.Now()

	delete(s.leasesByMACAddress, lease.MACAddress)
}

// Remove expired leases.
func (s *DHCPService) pruneLeases() {
	now := time.Now()

	var expired []string
	for macAddress := range s.leasesByMACAddress {
		leaseExpires := s.leasesByMACAddress[macAddress].Expires

		if now.Sub(leaseExpires) >= 0 {
			expired = append(expired, macAddress)
		}
	}

	for _, macAddress := range expired {
		delete(s.leasesByMACAddress, macAddress)
	}
}

func (s *DHCPService) acquireStateLock(reason string) {
	s.stateLock.Lock()
}

func (s *DHCPService) releaseStateLock(reason string) {
	s.stateLock.Unlock()
}

// Get the DHCP transaction Id as a string.
func getTransactionID(request dhcp.Packet) string {
	xid := request.XId()
	return fmt.Sprintf("0x%02X%02X%02X%02X", xid[0], xid[1], xid[2], xid[3])
}

// Get the DHCP user class from the request options.
func getUserClass(requestOptions dhcp.Options) string {
	if userClass, ok := requestOptions[dhcp.OptionUserClass]; ok {
		return string(userClass)
	}
	return ""
}

// Get the DHCP vendor class identifier from the request options.
func getVendorClassIdentifier(requestOptions dhcp.Options) string {
	vendorClassIdentifier, ok := requestOptions[dhcp.OptionVendorClassIdentifier]
	if ok {
		return string(vendorClassIdentifier)
	}

	return ""
}

// Create a reply packet.
func newReply(request dhcp.Packet, messageType dhcp.MessageType, serverIP, clientIP net.IP,
	leaseDuration time.Duration, options []dhcp.Option) (reply dhcp.Packet) {
	reply = dhcp.NewPacket(dhcp.BootReply)
	reply.SetXId(request.XId())
	reply.SetFlags(request.Flags())
	reply.SetYIAddr(clientIP)
	reply.SetGIAddr(request.GIAddr())
	reply.SetCHAddr(request.CHAddr())
	reply.AddOption(dhcp.OptionDHCPMessageType, []byte{byte(messageType)})
	reply.AddOption(dhcp.OptionServerIdentifier, []byte(serverIP))
	if leaseDuration > 0 {
		reply.AddOption(dhcp.OptionIPAddressLeaseTime, dhcp.OptionsLeaseTime(leaseDuration))
	}
	for _, option := range options {
		reply.AddOption(option.Code, option.Value)
	}

	// We don't add padding until ALL options have been added (DHCP packet implementation is a bit buggy).

	return reply
}

// Add a BOOTP-style boot file path to a DHCP response.
func addBootFile(response dhcp.Packet, bootFile string) {
	response.SetFile(
		[]byte(bootFile),
	)
}

// Add a DHCP-style boot file path option to a DHCP response.
func addBootFileOption(response dhcp.Packet, bootFile string) {
	response.AddOption(dhcp.OptionBootFileName,
		[]byte(bootFile),
	)
}

// Add DHCP TFTPServerName and BootFileName options (i.e. option 66, option 67) to a DHCP response.
func addTFTPBootFile(response dhcp.Packet, tftpServerName string, bootFile string) {
	addBootFileOption(response, bootFile)

	response.AddOption(dhcp.OptionTFTPServerName,
		[]byte(tftpServerName),
	)
}

// Determine if the DHCP request comes from a PXE-capable client seeking a boot server.
func isPXEClient(requestOptions dhcp.Options) bool {
	return strings.HasPrefix(
		getVendorClassIdentifier(requestOptions),
		"PXEClient:",
	)
}

// Determine if the DHCP request comes from an iPXE client.
func isIPXEClient(requestOptions dhcp.Options) bool {
	return getUserClass(requestOptions) == "iPXE"
}
