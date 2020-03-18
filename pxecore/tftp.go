package pxecore

import (
	"io"
	"net"
	"os"
	"time"

	"github.com/pin/tftp"
)

// readHandler is called when client starts file download from server
func (s *Server) tftpReadHandler(filename string, rf io.ReaderFrom) error {
	file, err := os.Open(s.Config.TFTP.Root + "/" + filename)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	n, err := rf.ReadFrom(file)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	log.Printf("TFTP: tftp_files %d bytes sent", n)
	return nil
}

// writeHandler is called when client starts file upload to server
func (s *Server) tftWriteHandler(filename string, wt io.WriterTo) error {
	file, err := os.OpenFile(s.Config.TFTP.Root+"/"+filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	n, err := wt.WriteTo(file)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	log.Printf("TFTP: tftp_files %d bytes received", n)
	log.Printf("TFTP: tftp_files recieved and stored file to %s", s.Config.TFTP.Root+"/"+filename)
	return nil
}

func (s *Server) serveTFTP(l *net.UDPConn) error {
	tftpServer := tftp.NewServer(s.tftpReadHandler, s.tftWriteHandler)
	tftpServer.SetTimeout(5 * time.Second) // optional
	log.Printf("starting tftp server and listening on port %s handle on path: %s", s.Config.TFTP.Port, s.Config.TFTP.Root)
	tftpServer.Serve(l) // blocks until s.Shutdown() is called
	return nil
}
