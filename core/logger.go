package core

import (
	"os"

	"github.com/mash/go-accesslog"
	"github.com/op/go-logging"
)

var format = logging.MustStringFormatter(
	`%{color}%{time:2006-01-02T15:04:05Z07:00} %{level:.5s} %{module:.10s}%{color:reset} %{message}`,
)

func init() {
	//backend1 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	//backend1Leveled := logging.AddModuleLevel(backend1)
	//backend1Leveled.SetLevel(logging.ERROR, "")
	// Set the backends to be used.
	logging.SetBackend(backend2Formatter)
}

var log = logging.MustGetLogger("pxesrv")

type logger struct {
}

func (l logger) Log(record accesslog.LogRecord) {
	log.Info("[HTTP] " + record.Method + " " + record.Uri)
}
