package pxecore

import (
	"net"
	"strconv"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
)

func getListenAddress(ipStr, portStr string) (*net.UDPAddr, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, PXEErrorFromString("dhcpv4: invalid `listen` port: %s", portStr)
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, PXEErrorFromString("dhcpv4: invalid IP address in `listen` directive: %s", ipStr)
	}
	listener := &net.UDPAddr{
		IP:   ip,
		Port: port,
	}
	return listener, nil
}

// Start will start the server asynchronously. See `Wait` to wait until
// the execution ends.
func (s *Server) serveDHCP() error {
	err := s.LoadConfig()
	if err != nil {
		return err
	}
	// listen
	log.Printf("Starting DHCPv4 listener on %v", s.Config.DHCP.ServerID)
	listener, err := getListenAddress(s.Config.DHCP.ServerID, s.Config.DHCP.Port)
	if err != nil {
		return err
	}
	interfaces := s.Config.DHCP.Interface
	s.Server4, err = server4.NewServer(interfaces, listener, s.MainHandler4)
	if err != nil {
		return err
	}
	go func() {
		s.errs <- s.Server4.Serve()
	}()
	return nil
}

// Wait waits until the end of the execution of the server.
func (s *Server) Wait() error {
	log.Print("Waiting")
	err := <-s.errs
	if s.Server4 != nil {
		s.Server4.Close()
	}
	return err
}

// NewServer creates a Server instance with the provided configuration.
// func NewServer(config *config.Config) *Server {
// 	return &Server{Config: config, errors: make(chan error, 1)}
// }

// MainHandler4 runs for every received DHCPv4 packet. It will run every
// registered handler in sequence, and reply with the resulting response.
// It will not reply if the resulting response is `nil`.
func (s *Server) MainHandler4(conn net.PacketConn, _peer net.Addr, req *dhcpv4.DHCPv4) {
	var (
		resp, tmp *dhcpv4.DHCPv4
		err       error
		stop      bool
	)
	if req.OpCode != dhcpv4.OpcodeBootRequest {
		log.Printf("MainHandler4: unsupported opcode %d. Only BootRequest (%d) is supported", req.OpCode, dhcpv4.OpcodeBootRequest)
		return
	}
	tmp, err = dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		log.Printf("MainHandler4: failed to build reply: %v", err)
		return
	}
	switch mt := req.MessageType(); mt {
	case dhcpv4.MessageTypeDiscover:
		tmp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))
	case dhcpv4.MessageTypeRequest:
		tmp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))
	default:
		log.Printf("plugins/server: Unhandled message type: %v", mt)
		return
	}

	resp = tmp
	for _, handler := range s.Handlers4 {
		resp, stop = handler(req, resp)
		if stop {
			break
		}
	}

	if resp != nil {
		var peer net.Addr
		if !req.GatewayIPAddr.IsUnspecified() {
			// TODO: make RFC8357 compliant
			peer = &net.UDPAddr{IP: req.GatewayIPAddr, Port: dhcpv4.ServerPort}
		} else if resp.MessageType() == dhcpv4.MessageTypeNak {
			peer = &net.UDPAddr{IP: net.IPv4bcast, Port: dhcpv4.ClientPort}
		} else if !req.ClientIPAddr.IsUnspecified() {
			peer = &net.UDPAddr{IP: req.ClientIPAddr, Port: dhcpv4.ClientPort}
		} else if req.IsBroadcast() {
			peer = &net.UDPAddr{IP: net.IPv4bcast, Port: dhcpv4.ClientPort}
		} else {
			// FIXME: we're supposed to unicast to a specific *L2* address, and an L3
			// address that's not yet assigned.
			// I don't know how to do that with this API...
			//peer = &net.UDPAddr{IP: resp.YourIPAddr, Port: dhcpv4.ClientPort}
			log.Warn("Cannot handle non-broadcast-capable unspecified peers in an RFC-compliant way. " +
				"Response will be broadcast")

			peer = &net.UDPAddr{IP: net.IPv4bcast, Port: dhcpv4.ClientPort}
		}

		if _, err := conn.WriteTo(resp.ToBytes(), peer); err != nil {
			log.Printf("MainHandler4: conn.Write to %v failed: %v", peer, err)
		}

	} else {
		log.Print("MainHandler4: dropping request because response is nil")
	}
}

func (s *Server) LoadConfig() error {
	// add serverid
	h, err := handleServerID(s.Config.DHCP.ServerID)
	if err != nil {
		return err
	}
	s.Handlers4 = append(s.Handlers4, h)

	// add dns
	h, err = handleDNS(s.Config.DHCP.DNSServer)
	if err != nil {
		return err
	}
	s.Handlers4 = append(s.Handlers4, h)

	// add router
	h, err = handleRouter(s.Config.DHCP.Router)
	if err != nil {
		return err
	}
	s.Handlers4 = append(s.Handlers4, h)

	// add router
	h, err = handleNetMask(s.Config.DHCP.NetMask)
	if err != nil {
		return err
	}
	s.Handlers4 = append(s.Handlers4, h)

	// add range
	h, err = handleRange(s.Config.DHCP.LeasesFile, s.Config.DHCP.StartIP, s.Config.DHCP.EndIP, s.Config.DHCP.Leases)
	if err != nil {
		return err
	}
	s.Handlers4 = append(s.Handlers4, h)

	// add netboot
	h, err = handleNetBoot(s.Config.DHCP.TFTPServer, s.Config.DHCP.PxeFile)
	if err != nil {
		return err
	}
	s.Handlers4 = append(s.Handlers4, h)

	return nil
}
