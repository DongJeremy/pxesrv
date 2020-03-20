package pxecore

import (
	"fmt"
	"net"

	"github.com/DongJeremy/pxesrv/dhcp4"
	"golang.org/x/net/ipv4"
)

func (s *Server) servePXE(conn net.PacketConn) error {
	buf := make([]byte, 1024)
	l := ipv4.NewPacketConn(conn)
	if err := l.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		return PXEErrorFromString("Couldn't get interface metadata on PXE port: %s", err)
	}

	for {
		n, msg, addr, err := l.ReadFrom(buf)
		if err != nil {
			return PXEErrorFromString("Receiving packet: %s", err)
		}

		pkt, err := dhcp4.Unmarshal(buf[:n])
		if err != nil {
			s.debug("PXE", "Packet from %s is not a DHCP packet: %s", addr, err)
			continue
		}

		if err = s.isBootDHCP(pkt); err != nil {
			s.debug("PXE", "Ignoring packet from %s (%s): %s", pkt.HardwareAddr, addr, err)
		}
		fwtype, err := s.validatePXE(pkt)
		if err != nil {
			s.log("PXE", "Unusable packet from %s (%s): %s", pkt.HardwareAddr, addr, err)
			continue
		}

		intf, err := net.InterfaceByIndex(msg.IfIndex)
		if err != nil {
			s.log("PXE", "Couldn't get information about local network interface %d: %s", msg.IfIndex, err)
			continue
		}

		serverIP, err := interfaceIP(intf)
		if err != nil {
			s.log("PXE", "Want to boot %s (%s) on %s, but couldn't get a source address: %s", pkt.HardwareAddr, addr, intf.Name, err)
			continue
		}

		s.machineEvent(pkt.HardwareAddr, machineStatePXE, "Sent PXE configuration")

		resp, err := s.offerPXE(pkt, serverIP, fwtype)
		if err != nil {
			s.log("PXE", "Failed to construct PXE offer for %s (%s): %s", pkt.HardwareAddr, addr, err)
			continue
		}

		bs, err := resp.Marshal()
		if err != nil {
			s.log("PXE", "Failed to marshal PXE offer for %s (%s): %s", pkt.HardwareAddr, addr, err)
			continue
		}

		if _, err := l.WriteTo(bs, &ipv4.ControlMessage{
			IfIndex: msg.IfIndex,
		}, addr); err != nil {
			s.log("PXE", "Failed to send PXE response to %s (%s): %s", pkt.HardwareAddr, addr, err)
		}
	}
}

func (s *Server) validatePXE(pkt *dhcp4.Packet) (fwtype Firmware, err error) {
	fwt, err := pkt.Options.Uint16(93)
	if err != nil {
		return 0, PXEErrorFromString("malformed DHCP option 93 (required for PXE): %s", err)
	}
	switch fwt {
	case 6:
		fwtype = FirmwareEFI32
	case 7:
		fwtype = FirmwareEFI64
	case 9:
		fwtype = FirmwareEFIBC
	default:
		return 0, PXEErrorFromString("unsupported client firmware type '%d' (please file a bug!)", fwt)
	}
	if s.Ipxe[fwtype] == nil {
		return 0, PXEErrorFromString("unsupported client firmware type '%d' (please file a bug!)", fwtype)
	}

	guid := pkt.Options[97]
	switch len(guid) {
	case 0:
		// Accept missing GUIDs even though it's a spec violation,
		// same as in dhcp.go.
	case 17:
		if guid[0] != 0 {
			return 0, PXEErrorFromString("malformed client GUID (option 97), leading byte must be zero")
		}
	default:
		return 0, PXEErrorFromString("malformed client GUID (option 97), wrong size")
	}

	return fwtype, nil
}

func (s *Server) offerPXE(pkt *dhcp4.Packet, serverIP net.IP, fwtype Firmware) (resp *dhcp4.Packet, err error) {
	resp = &dhcp4.Packet{
		Type:           dhcp4.MsgAck,
		TransactionID:  pkt.TransactionID,
		HardwareAddr:   pkt.HardwareAddr,
		ClientAddr:     pkt.ClientAddr,
		RelayAddr:      pkt.RelayAddr,
		ServerAddr:     serverIP,
		BootServerName: serverIP.String(),
		BootFilename:   fmt.Sprintf("%s/%d", pkt.HardwareAddr, fwtype),
		Options: dhcp4.Options{
			dhcp4.OptServerIdentifier: serverIP,
			dhcp4.OptVendorIdentifier: []byte("PXEClient"),
		},
	}
	if pkt.Options[97] != nil {
		resp.Options[97] = pkt.Options[97]
	}

	return resp, nil
}
