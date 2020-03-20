package pxecore

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config config for pxe
type Config struct {
	PXE    PXE    `yaml:"pxe"`
	Common Common `yaml:"common"`
}

// Common config
type Common struct {
	ExportIP string `yaml:"export_ip,omitempty"`
	RootPath string `yaml:"root_path,omitempty"`
}

// PXE config
type PXE struct {
	// which ip address that http server listening
	ListenIP  string `yaml:"listen_ip,omitempty"`
	HTTPPort  string `yaml:"http_port,omitempty"` // listening port of http server
	HTTPRoot  string `yaml:"http_root,omitempty"` // http file server path
	TFTPPort  string `yaml:"tftp_port,omitempty"` // listening port of tfpt server
	TFTPRoot  string `yaml:"tftp_root,omitempty"` // listening port of tfpt server
	DHCPPort  string `yaml:"dhcp_port,omitempty"` // listening port of dhcp server
	StartIP   string `yaml:"start_ip"`
	Range     int    `yaml:"lease_range"`       // lease ip address count
	NetMask   string `yaml:"netmask,omitempty"` // default /24
	Router    string `yaml:"router,omitempty"`
	DNSServer string `yaml:"dns_server,omitempty"`
	PXEFile   string `yaml:"pxe_file"` // pxe file name
}

// GetConf return runtime configurations
func GetConf(filename string) Config {
	c := new(Config)
	// set default options
	c.PXE.ListenIP = "0.0.0.0"
	c.PXE.HTTPPort = "80"
	c.PXE.HTTPRoot = "netboot"
	c.PXE.TFTPPort = "69"
	c.PXE.TFTPRoot = "netboot"
	c.PXE.DHCPPort = "67"
	c.PXE.StartIP = "192.168.1.201"
	c.PXE.Range = 10
	c.PXE.PXEFile = "ipxelinux.0"
	c.PXE.NetMask = "255.255.255.0"
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
