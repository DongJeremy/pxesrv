package pxecore

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/DongJeremy/pxesrv/dhcp4"
)

const (
	portDHCP = 67
	portTFTP = 69
	portHTTP = 80
	portPXE  = 4011
)

// An ID is an identifier used by Booters to reference files.
type ID string

// Architecture describes a kind of CPU architecture.
type Architecture int

// Architecture types that Pxecore knows how to boot.
//
// These architectures are self-reported by the booting machine. The
// machine may support additional execution modes. For example, legacy
// PC BIOS reports itself as an ArchIA32, but may also support ArchX64
// execution.
const (
	// ArchIA32 is a 32-bit x86 machine. It _may_ also support X64
	// execution, but Pxecore has no way of knowing.
	ArchIA32 Architecture = iota
	// ArchX64 is a 64-bit x86 machine (aka amd64 aka X64).
	ArchX64
)

func (a Architecture) String() string {
	switch a {
	case ArchIA32:
		return "IA32"
	case ArchX64:
		return "X64"
	default:
		return "Unknown architecture"
	}
}

// A Machine describes a machine that is attempting to boot.
type Machine struct {
	MAC  net.HardwareAddr
	Arch Architecture
}

// A Spec describes a kernel and associated configuration.
type Spec struct {
	// The kernel to boot
	Kernel ID
	// Optional init ramdisks for linux kernels
	Initrd []ID
	// Optional kernel commandline. This string is evaluated as a
	// text/template template, in which "ID(x)" function is
	// available. Invoking ID(x) returns a URL that will call
	// Booter.ReadBootFile(x) when fetched.
	Cmdline string
	// Message to print on the client machine before booting.
	Message string

	// A raw iPXE script to run. Overrides all of the above.
	//
	// THIS IS NOT A STABLE INTERFACE. This will only work for
	// machines that get booted via iPXE. Currently, that is all of
	// them, but there is no guarantee that this will remain
	// true. When passing a custom iPXE script, it is your
	// responsibility to make the boot succeed, Pxecore's
	// involvement ends when it serves your script.
	IpxeScript string
}

// A Booter provides boot instructions and files for machines.
//
// Due to the stateless nature of various boot protocols, BootSpec()
// will be called multiple times in the course of a single boot
// attempt.
type Booter interface {
	// The given MAC address wants to know what it should
	// boot. What should Pxecore make it boot?
	//
	// Returning an error or a nil BootSpec will make Pxecore ignore
	// the client machine's request.
	BootSpec(m Machine) (*Spec, error)
	// Get the bytes corresponding to an ID given in Spec.
	//
	// Additionally returns the total number of bytes in the
	// ReadCloser, or -1 if the size is unknown. Be warned, returning
	// -1 will make the boot process orders of magnitude slower due to
	// poor ipxe behavior.
	ReadBootFile(id ID) (io.ReadCloser, int64, error)
	// Write the given Reader to an ID given in Spec.
	WriteBootFile(id ID, body io.Reader) error
}

// Firmware describes a kind of firmware attempting to boot.
//
// This should only be used for selecting the right bootloader within
// Pxecore, kernel selection should key off the more generic
// Architecture.
type Firmware int

// The bootloaders that Pxecore knows how to handle.
const (
	FirmwareX86PC       Firmware = iota // "Classic" x86 BIOS with PXE/UNDI support
	FirmwareEFI32                       // 32-bit x86 processor running EFI
	FirmwareEFI64                       // 64-bit x86 processor running EFI
	FirmwareEFIBC                       // 64-bit x86 processor running EFI
	FirmwareX86Ipxe                     // "Classic" x86 BIOS running iPXE (no UNDI support)
	FirmwarePxecoreIpxe                 // Pxecore's iPXE, which has replaced the underlying firmware
)

// A Server boots machines using a Booter.
type Server struct {
	Booter Booter

	// Address to listen on, or empty for all interfaces.
	Address string
	// HTTP port for boot services.
	HTTPPort int
	// HTTP port for human-readable information. Can be the same as
	// HTTPPort.
	HTTPStatusPort int

	// Ipxe lists the supported bootable Firmwares, and their
	// associated ipxe binary.
	Ipxe map[Firmware][]byte

	// Log receives logs on Pxecore's operation. If nil, logging
	// is suppressed.
	Log func(subsystem, msg string)
	// Debug receives extensive logging on Pxecore's internals. Very
	// useful for debugging, but very verbose.
	Debug func(subsystem, msg string)

	// Error receives error logs on Pxecore's operation. If nil, logging
	// is suppressed.
	Error func(subsystem, msg string)

	Config Config

	// These ports can technically be set for testing, but the
	// protocols burned in firmware on the client side hardcode these,
	// so if you change them in production, nothing will work.
	DHCPPort int
	TFTPPort int
	PXEPort  int

	// Listen for DHCP traffic without binding to the DHCP port. This
	// enables coexistence of Pxecore with another DHCP server.
	//
	// Currently only supported on Linux.
	DHCPNoBind bool

	// Read UI assets from this path, rather than use the builtin UI
	// assets. Used for development of Pxecore.
	UIAssetsDir string

	errs     chan error
	eventsMu sync.Mutex
	events   map[string][]machineEvent
}

func (s *Server) Prepare() error {
	if err := s.LoadTemplates(); err != nil {
		return err
	}
	if err := s.RenderFile(); err != nil {
		return err
	}
	return nil
}

func (s *Server) Serve() error {

	if s.DHCPPort == 0 {
		s.DHCPPort = portDHCP
	}
	if s.TFTPPort == 0 {
		s.TFTPPort = portTFTP
	}
	if s.PXEPort == 0 {
		s.PXEPort = portPXE
	}
	if s.HTTPPort == 0 {
		s.HTTPPort = portHTTP
	}
	newDHCP := dhcp4.NewConn
	if s.DHCPNoBind {
		newDHCP = dhcp4.NewSnooperConn
	}
	dhcp, err := newDHCP(fmt.Sprintf("%s:%d", s.Address, s.DHCPPort))
	if err != nil {
		fmt.Println(err)
		return err
	}
	tftp, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", s.Address, s.TFTPPort))
	if err != nil {
		dhcp.Close()
		return err
	}
	pxe, err := net.ListenPacket("udp4", fmt.Sprintf("%s:%d", s.Address, s.PXEPort))
	if err != nil {
		dhcp.Close()
		tftp.Close()
		return err
	}
	http, err := net.Listen("tcp4", fmt.Sprintf("%s:%s", s.Config.HTTP.IP, s.Config.HTTP.Port))
	if err != nil {
		dhcp.Close()
		tftp.Close()
		pxe.Close()
		return err
	}
	// 4 buffer slots, one for each goroutine, plus one for
	// Shutdown(). We only ever pull the first error out, but shutdown
	// will likely generate some spurious errors from the other
	// goroutines, and we want them to be able to dump them without
	// blocking.
	s.errs = make(chan error, 6)

	fmt.Println("Init", "Starting Pxecore goroutines")

	go func() { s.errs <- s.serveDHCP(dhcp) }()
	go func() { s.errs <- s.servePXE(pxe) }()
	go func() { s.errs <- s.serveTFTP(tftp) }()
	go func() { s.errs <- s.serveHTTP(http) }()

	// Wait for either a fatal error, or Shutdown().
	err = <-s.errs
	dhcp.Close()
	tftp.Close()
	pxe.Close()
	http.Close()
	return err
}

// Shutdown causes Serve() to exit, cleaning up behind itself.
func (s *Server) Shutdown() {
	select {
	case s.errs <- nil:
	default:
	}
}
