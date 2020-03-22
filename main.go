package main

import (
	"flag"

	"github.com/DongJeremy/pxesrv/core"
	"github.com/op/go-logging"
)

func main() {
	var log = logging.MustGetLogger("pxesrv")
	//var log = pxecore.GetLogger("pxesrv")
	var configFileName = flag.String("c", "pxe.yml", "config file path (default config.ini)")
	flag.Parse()
	log.Info("starting pxe server...")
	service := core.NewService()
	err := service.Initialize(*configFileName)
	if err != nil {
		log.Panic(err)
	}
	service.Start()
}
