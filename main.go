package main

import (
	"flag"
	"fmt"

	"github.com/DongJeremy/pxesrv/pxecore"
)

func main() {
	var configFileName = flag.String("c", "pxe.yml", "config file path (default config.ini)")
	flag.Parse()
	fmt.Println("starting pxe server...")
	serve := pxecore.Server{
		Address:    "0.0.0.0",
		Config:     pxecore.GetConf(*configFileName),
		DHCPNoBind: true}
	serve.Prepare()
	fmt.Println(serve.Serve())
}
