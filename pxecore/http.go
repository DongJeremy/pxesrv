package pxecore

import (
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/mash/go-accesslog"
)

func (s *Server) serveHTTP(l net.Listener) error {
	listen := net.JoinHostPort(s.Config.HTTP.IP, s.Config.HTTP.Port)
	rootPath := filepath.Join(s.Config.Common.RootPath, s.Config.HTTP.Root)

	accessLogger := logger{}
	http.Handle("/", accesslog.NewLoggingHandler(http.FileServer(http.Dir(rootPath)), accessLogger))
	log.Printf("starting http server %s and handle on path: %s", listen, rootPath)

	httpServer := &http.Server{
		Addr:           s.Config.HTTP.Port, // 监听的地址和端口
		Handler:        nil,                // 所有请求需要调用的Handler（实际上这里说是ServeMux更确切）如果为空则设置为DefaultServeMux
		ReadTimeout:    0 * time.Second,    // 读的最大Timeout时间
		WriteTimeout:   0 * time.Second,    // 写的最大Timeout时间
		MaxHeaderBytes: 256,                // 请求头的最大长度
		TLSConfig:      nil,                // 配置TLS
	}
	if err := httpServer.Serve(l); err != nil {
		log.Errorf("HTTP server shut down: %s", err)
		return err
	}
	return nil
}
