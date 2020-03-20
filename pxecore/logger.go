package pxecore

import (
	"encoding/base64"
	"fmt"
	"net"
	"time"
)

const savedEventsPerMachine = 10

type machineState int

func (m machineState) String() string {
	switch m {
	case machineStateProxyDHCP:
		return "Made boot offer (ProxyDHCP)"
	case machineStatePXE:
		return "Made boot offer (PXE)"
	case machineStateTFTP:
		return "Sent iPXE binary (TFTP)"
	case machineStateProxyDHCPIpxe:
		return "Made iPXE boot offer (ProxyDHCP)"
	case machineStateIpxeScript:
		return "Sent iPXE script (HTTP)"
	case machineStateKernel:
		return "Sent kernel (HTTP)"
	case machineStateInitrd:
		return "Sent initrd(s) (HTTP)"
	case machineStateBooted:
		return "Booted machine"
	default:
		return "Unknown"
	}
}

func (m machineState) Progress() string {
	return fmt.Sprintf("%.0f%%", float32(m)/float32(machineStateBooted)*100)
}

const (
	machineStateProxyDHCP = iota
	machineStatePXE
	machineStateTFTP
	machineStateProxyDHCPIpxe
	machineStateIpxeScript
	machineStateKernel
	machineStateInitrd
	machineStateBooted

	machineStateIgnored
)

type machineEvent struct {
	Timestamp time.Time
	State     machineState
	Message   string
}

func (s *Server) machineEvent(mac net.HardwareAddr, state machineState, format string, args ...interface{}) {
	evt := machineEvent{
		Timestamp: time.Now(),
		State:     state,
		Message:   fmt.Sprintf(format, args...),
	}
	k := mac.String()

	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	s.events[k] = append(s.events[k], evt)
	if len(s.events[k]) > savedEventsPerMachine {
		s.events[k] = s.events[k][len(s.events[k])-savedEventsPerMachine:]
	}
}

func (s *Server) log(subsystem, format string, args ...interface{}) {
	if s.Log == nil {
		return
	}
	s.Log(subsystem, fmt.Sprintf(format, args...))
}

func (s *Server) error(subsystem, format string, args ...interface{}) {
	if s.Error == nil {
		return
	}
	s.Log(subsystem, fmt.Errorf(format, args...).Error())
}

func (s *Server) debug(subsystem, format string, args ...interface{}) {
	if s.Debug == nil {
		return
	}
	s.Debug(subsystem, fmt.Sprintf(format, args...))
}

func (s *Server) debugPacket(subsystem string, layer int, packet []byte) {
	if s.Debug == nil {
		return
	}
	s.Debug(subsystem, fmt.Sprintf("PKT %d %s END", layer, base64.StdEncoding.EncodeToString(packet)))
}
