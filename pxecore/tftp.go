package pxecore

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"strings"

	"github.com/DongJeremy/pxesrv/tftp"
)

func (s *Server) serveTFTP(l net.PacketConn) error {
	ts := tftp.Server{
		Handler:     s.handleTFTP,
		InfoLog:     func(msg string) { s.debug("TFTP", msg) },
		TransferLog: s.logTFTPTransfer,
	}
	err := ts.Serve(l)
	if err != nil {
		return PXEErrorFromString("TFTP server shut down: %s", err)
	}
	return nil
}

func extractInfo(path string) (net.HardwareAddr, int, error) {
	pathElements := strings.Split(path, "/")
	if len(pathElements) != 2 {
		return nil, 0, PXEErrorFromString("not found")
	}

	mac, err := net.ParseMAC(pathElements[0])
	if err != nil {
		return nil, 0, PXEErrorFromString("invalid MAC address %q", pathElements[0])
	}

	i, err := strconv.Atoi(pathElements[1])
	if err != nil {
		return nil, 0, PXEErrorFromString("not found")
	}

	return mac, i, nil
}

func (s *Server) logTFTPTransfer(clientAddr net.Addr, path string, err error) {
	mac, _, pathErr := extractInfo(path)
	if pathErr != nil {
		s.log("TFTP", "unable to extract mac from request:%v", pathErr)
		return
	}
	if err != nil {
		s.log("TFTP", "Send of %q to %s failed: %s", path, clientAddr, err)
	} else {
		s.log("TFTP", "Sent %q to %s", path, clientAddr)
		s.machineEvent(mac, machineStateTFTP, "Sent iPXE to %s", clientAddr)
	}
}

func (s *Server) handleTFTP(path string, clientAddr net.Addr) (io.ReadCloser, int64, error) {
	_, i, err := extractInfo(path)
	if err != nil {
		return nil, 0, PXEErrorFromString("unknown path %q", path)
	}

	bs, ok := s.Ipxe[Firmware(i)]
	if !ok {
		return nil, 0, PXEErrorFromString("unknown firmware type %d", i)
	}

	return ioutil.NopCloser(bytes.NewBuffer(bs)), int64(len(bs)), nil
}
