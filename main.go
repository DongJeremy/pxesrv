package main

import (
	"flag"

	"github.com/DongJeremy/pxesrv/pxecore"
)

func main() {
	var log = pxecore.GetLogger("pxesrv")
	var configFileName = flag.String("c", "pxe.yml", "config file path (default config.ini)")
	flag.Parse()
	log.Info("starting pxe server...")
	serve := pxecore.Server{Config: pxecore.GetConf(*configFileName)}
	serve.Serve()
}
