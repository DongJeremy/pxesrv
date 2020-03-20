package pxecore

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pin/tftp"
)

// readHandler is called when client starts file download from server
func (s *Server) tftpReadHandler(filename string, rf io.ReaderFrom) error {
	rootPath := filepath.Join(s.Config.Common.RootPath, s.Config.PXE.TFTPRoot, filename)
	file, err := os.Open(rootPath)
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
	rootPath := filepath.Join(s.Config.Common.RootPath, s.Config.PXE.TFTPRoot, filename)
	file, err := os.OpenFile(rootPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
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
	log.Printf("TFTP: tftp_files recieved and stored file to %s", rootPath)
	return nil
}

func (s *Server) serveTFTP(l *net.UDPConn) error {
	rootPath := filepath.Join(s.Config.Common.RootPath, s.Config.PXE.TFTPRoot)
	tftpServer := tftp.NewServer(s.tftpReadHandler, s.tftWriteHandler)
	tftpServer.SetTimeout(5 * time.Second) // optional
	log.Printf("starting tftp server and listening on port %s handle on path: %s", s.Config.PXE.TFTPPort, rootPath)
	tftpServer.Serve(l) // blocks until s.Shutdown() is called
	return nil
}
