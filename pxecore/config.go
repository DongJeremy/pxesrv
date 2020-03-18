package pxecore

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config config for pxe
type Config struct {
	HTTP HTTP `yaml:"http"`
	TFTP TFTP `yaml:"tftp"`
	DHCP DHCP `yaml:"dhcp"`
}

// HTTP config
type HTTP struct {
	// which ip address that http server listening
	IP       string `yaml:"listen_ip,omitempty"`
	Port     string `yaml:"listen_port,omitempty"` // listening port of http server
	RootPath string `yaml:"rootpath,omitempty"`    // http file server path
}

// TFTP config
type TFTP struct {
	Root string `yaml:"tftp_root,omitempty"`   // tftp_files server path
	IP   string `yaml:"listen_ip,omitempty"`   // ip address that tftp_files server listening on
	Port string `yaml:"listen_port,omitempty"` // listening port of tfpt server
}

// DHCP config
type DHCP struct {
	IP         string `yaml:"listen_ip,omitempty"` // which ip address that dhcp server was listening on
	Port       string `yaml:"listen_port,omitempty"`
	TftpServer string `yaml:"tftp_server,omitempty"`
	StartIP    string `yaml:"start_ip"`
	Range      int    `yaml:"lease_range"`       // lease ip address count
	NetMask    string `yaml:"netmask,omitempty"` // default /24
	Router     string `yaml:"router,omitempty"`
	DNSServer  string `yaml:"dns_server,omitempty"`
	PxeFile    string `yaml:"pxe_file"` // pxe file name
}

// GetConf return runtime configurations
func GetConf(filename string) Config {
	c := new(Config)
	// set default options
	c.HTTP.IP = "0.0.0.0"
	c.HTTP.Port = "80"
	c.HTTP.RootPath = "/mnt/dhtp/http"
	c.TFTP.IP = "0.0.0.0"
	c.TFTP.Root = "/mnt/dhtp/tftp"
	c.TFTP.Port = "69"
	c.DHCP.IP = "0.0.0.0"
	c.DHCP.Port = "67"
	c.DHCP.StartIP = "192.168.1.201"
	c.DHCP.Range = 10
	c.DHCP.PxeFile = "pxelinux.0"
	c.DHCP.NetMask = "255.255.255.0"
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Errorf("read config file from %s failed, %s", filename, err)
	}
	err = yaml.Unmarshal(f, c)
	if err != nil {
		log.Errorf("parse config file failed, %s", err)
	}
	return *c
}
