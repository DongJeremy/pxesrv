package core

import (
	"fmt"
	"net"

	"github.com/spf13/viper"
)

// A Service represents the state for the All service.
type Service struct {
	//Config Config
	ServiceIP      string
	DocRoot        string
	ListenIP       string
	HTTPPort       string // http listen port default 80
	HTTPRoot       string // http document root default netboot
	TFTPPort       string // tftp listen port default 69
	TFTPRoot       string // tftp document root default netboot
	DHCPPort       string // dhcp listen port default 67
	IPRangeStart   string // dhcp ip range start
	IPRangeEnd     string // dhcp ip range end
	NetMask        string // dhcp netmask default 255.255.255.0
	Router         string
	DNSServer      string
	TFTPServerName string
	PXEBootImage   string // PXE boot file (TFTP)
	IPXEBootScript string // iPXE boot script (HTTP)
	EnableIPXE     bool
	errs           chan error
}

// NewService creates new Service state.
func NewService() *Service {
	return &Service{
		EnableIPXE: true,
		errs:       make(chan error, 5),
	}
}

// Initialize the service configuration.
func (s *Service) Initialize(path string) error {
	viper.SetConfigType("yaml")
	viper.SetConfigFile(path)
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}
	s.ServiceIP = viper.GetString("global.ip_address")
	s.DocRoot = viper.GetString("global.doc_root")
	s.ListenIP = viper.GetString("pxe.listen_ip")
	s.HTTPPort = viper.GetString("pxe.http_port")
	s.HTTPRoot = viper.GetString("pxe.http_root")
	s.TFTPPort = viper.GetString("pxe.tftp_port")
	s.TFTPRoot = viper.GetString("pxe.tftp_root")
	s.DHCPPort = viper.GetString("pxe.dhcp_port")
	s.IPRangeStart = viper.GetString("pxe.start_ip")
	s.IPRangeEnd = viper.GetString("pxe.end_ip")
	s.NetMask = viper.GetString("pxe.netmask")
	s.Router = viper.GetString("pxe.router")
	s.DNSServer = viper.GetString("pxe.dns_server")
	s.TFTPServerName = viper.GetString("global.ip_address")
	s.PXEBootImage = viper.GetString("pxe.pxe_file")
	s.IPXEBootScript = viper.GetString("pxe.ipxe_file")
	s.EnableIPXE = viper.GetBool("pxe.enable_ipxe")
	err = s.Prepare()
	if err != nil {
		return err
	}
	return nil
}

// Prepare env
func (s *Service) Prepare() error {
	if err := s.LoadAndRenderTemplates(); err != nil {
		log.Errorf("error during template rendering, error: %s", err)
		return err
	}
	return nil
}

// Start the service.
func (s *Service) Start() error {

	dhcp, err := net.ListenPacket("udp4", fmt.Sprintf("%s:%s", s.ListenIP, s.DHCPPort))
	if err != nil {
		log.Errorf("start DHCP failed, %s", err)
		return err
	}

	a, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", s.ListenIP, s.TFTPPort))
	if err != nil {
		log.Errorf("resolveUDP failed, %s", err)
		return err
	}
	tftp, err := net.ListenUDP("udp4", a)
	if err != nil {
		log.Errorf("start TFTP failed, %s", err)
		dhcp.Close()
		return err
	}

	http, err := net.Listen("tcp4", fmt.Sprintf("%s:%s", s.ListenIP, s.HTTPPort))
	if err != nil {
		log.Errorf("start HTTP failed, %s", err)
		dhcp.Close()
		tftp.Close()
		return err
	}
	// 4 buffer slots, one for each goroutine, plus one for
	// Shutdown(). We only ever pull the first error out, but shutdown
	// will likely generate some spurious errors from the other
	// goroutines, and we want them to be able to dump them without
	// blocking.
	//s.errs = make(chan error, 5)

	//log.debug("Init", "Starting Pixiecore goroutines")

	go func() { s.errs <- s.serveDHCP(dhcp) }()
	go func() { s.errs <- s.serveTFTP(tftp) }()
	go func() { s.errs <- s.serveHTTP(http) }()

	// Wait for either a fatal error, or Shutdown().
	err = <-s.errs
	dhcp.Close()
	tftp.Close()
	http.Close()
	return err
}

// Shutdown causes Serve() to exit, cleaning up behind itself.
func (s *Service) Shutdown() {
	select {
	case s.errs <- nil:
	default:
	}
}
