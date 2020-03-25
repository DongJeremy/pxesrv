package main

import (
	"flag"

	"github.com/DongJeremy/pxesrv/core"
)

func main() {
	var configFileName = flag.String("c", "pxe.yml", "config file path (default config.ini)")
	flag.Parse()
	service := core.NewService()
	err := service.Initialize(*configFileName)
	if err != nil {
		service.Logger.Panic(err)
	}
	service.Start()
}
