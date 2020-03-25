package core

import (
	"os"
	"path"

	"github.com/mash/go-accesslog"
	"github.com/op/go-logging"
)

var format = logging.MustStringFormatter(
	`%{color}%{time:2006-01-02T15:04:05Z07:00} %{level:.5s} %{module:.10s}%{color:reset} %{message}`,
)

func initLogger(logFilePath, logFileName string) *logging.Logger {
	fileName := path.Join(logFilePath, logFileName)
	err := os.MkdirAll(logFilePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	_, err = os.Stat(fileName)
	if os.IsNotExist(err) {
		// if file not exist, create it
		os.Create(fileName)
	}
	logFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	backend1 := logging.NewLogBackend(logFile, "", 0)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend1Formatter := logging.NewBackendFormatter(backend1, format)
	backend2Leveled := logging.AddModuleLevel(backend2)
	backend2Leveled.SetLevel(logging.INFO, "")
	// Set the backends to be used.
	logging.SetBackend(backend2Leveled, backend1Formatter)
	return logging.MustGetLogger("pxesrv")
}

type logger struct {
	Logger *logging.Logger
}

func (l logger) Log(record accesslog.LogRecord) {
	l.Logger.Info("[HTTP] " + record.Method + " " + record.Uri)
}
