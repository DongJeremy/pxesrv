package pxecore

import (
	"bufio"
	"encoding/binary"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

// Handler4 behaves like Handler6, but for DHCPv4 packets.
type Handler4 func(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool)

type Record struct {
	IP      net.IP
	expires time.Time
}

var (
	ipRangeStart net.IP
	ipRangeEnd   net.IP
	LeaseTime    time.Duration
	err          error
	Recordsv4    map[string]*Record
	dnsServers4  []net.IP
	routers      []net.IP
	filename     string
	v4ServerID   net.IP
	netmask      net.IPMask
	opt66, opt67 *dhcpv4.Option
)

func handleServerID(args ...string) (Handler4, error) {
	serverID := net.ParseIP(args[0])
	if serverID == nil {
		return nil, PXEErrorFromString("invalid or empty IP address")
	}
	if serverID.To4() == nil {
		return nil, PXEErrorFromString("not a valid IPv4 address")
	}
	v4ServerID = serverID.To4()
	return func(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
		if v4ServerID == nil {
			log.Fatal("BUG: Plugin is running uninitialized!")
			return nil, true
		}
		if req.OpCode != dhcpv4.OpcodeBootRequest {
			log.Warningf("not a BootRequest, ignoring")
			return resp, false
		}
		if req.ServerIPAddr != nil &&
			!req.ServerIPAddr.Equal(net.IPv4zero) &&
			!req.ServerIPAddr.Equal(v4ServerID) {
			// This request is not for us, drop it.
			log.Infof("requested server ID does not match this server's ID. Got %v, want %v", req.ServerIPAddr, v4ServerID)
			return nil, true
		}
		resp.ServerIPAddr = make(net.IP, net.IPv4len)
		copy(resp.ServerIPAddr[:], v4ServerID)
		resp.UpdateOption(dhcpv4.OptServerIdentifier(v4ServerID))
		return resp, false
	}, nil
}

func handleDNS(args ...string) (Handler4, error) {
	log.Printf("loaded plugin for DHCPv4.")
	if len(args) < 1 {
		return nil, PXEErrorFromString("need at least one DNS server")
	}
	for _, arg := range args {
		DNSServer := net.ParseIP(arg)
		if DNSServer.To4() == nil {
			return nil, PXEErrorFromString("expected an DNS server address, got: " + arg)
		}
		dnsServers4 = append(dnsServers4, DNSServer)
	}
	log.Infof("loaded %d DNS servers.", len(dnsServers4))
	return func(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
		if req.IsOptionRequested(dhcpv4.OptionDomainNameServer) {
			resp.Options.Update(dhcpv4.OptDNS(dnsServers4...))
		}
		return resp, false
	}, nil
}

func handleRouter(args ...string) (Handler4, error) {
	log.Printf("Loaded plugin for DHCPv4.")
	if len(args) < 1 {
		return nil, PXEErrorFromString("need at least one router IP address")
	}
	for _, arg := range args {
		router := net.ParseIP(arg)
		if router.To4() == nil {
			return nil, PXEErrorFromString("expected an router IP address, got: " + arg)
		}
		routers = append(routers, router)
	}
	log.Infof("loaded %d router IP addresses.", len(routers))
	return func(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
		resp.Options.Update(dhcpv4.OptRouter(routers...))
		return resp, false
	}, nil
}

func handleNetMask(args ...string) (Handler4, error) {
	log.Printf("loaded plugin for DHCPv4.")
	if len(args) != 1 {
		return nil, PXEErrorFromString("need at least one netmask IP address")
	}
	netmaskIP := net.ParseIP(args[0])
	if netmaskIP.IsUnspecified() {
		return nil, PXEErrorFromString("netmask is not valid, got: " + args[1])
	}
	netmaskIP = netmaskIP.To4()
	if netmaskIP == nil {
		return nil, PXEErrorFromString("expected an netmask address, got: " + args[1])
	}
	netmask = net.IPv4Mask(netmaskIP[0], netmaskIP[1], netmaskIP[2], netmaskIP[3])
	if !checkValidNetmask(netmask) {
		return nil, PXEErrorFromString("netmask is not valid, got: " + args[1])
	}
	log.Printf("loaded client netmask")
	return func(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
		resp.Options.Update(dhcpv4.OptSubnetMask(netmask))
		return resp, false
	}, nil
}

func checkValidNetmask(netmask net.IPMask) bool {
	netmaskInt := binary.BigEndian.Uint32(netmask)
	x := ^netmaskInt
	y := x + 1
	return (y & x) == 0
}

func handleNetBoot(args ...string) (Handler4, error) {
	if len(args) != 2 {
		return nil, PXEErrorFromString("Exactly one argument must be passed to NBP plugin, got %d", len(args))
	}
	otsn := dhcpv4.OptTFTPServerName(args[0])
	opt66 = &otsn
	obfn := dhcpv4.OptBootFileName(args[1])
	opt67 = &obfn
	log.Printf("loaded NBP plugin for DHCPv4.")
	return func(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
		if opt66 == nil || opt67 == nil {
			// nothing to do
			return resp, true
		}
		if req.IsOptionRequested(dhcpv4.OptionTFTPServerName) {
			resp.Options.Update(*opt66)
		}
		if req.IsOptionRequested(dhcpv4.OptionBootfileName) {
			resp.Options.Update(*opt67)
		}
		log.Debugf("Added NBP %s / %s to request", opt66, opt67)
		return resp, true
	}, nil
}

func handleRange(args ...string) (Handler4, error) {
	if len(args) < 4 {
		return nil, PXEErrorFromString("invalid number of arguments, want: 4 (file name, start IP, end IP, lease time), got: %d", len(args))
	}
	filename = args[0]
	if filename == "" {
		return nil, PXEErrorFromString("file name cannot be empty")
	}
	ipRangeStart = net.ParseIP(args[1])
	if ipRangeStart.To4() == nil {
		return nil, PXEErrorFromString("invalid IPv4 address: %v", args[1])
	}
	ipRangeEnd = net.ParseIP(args[2])
	if ipRangeEnd.To4() == nil {
		return nil, PXEErrorFromString("invalid IPv4 address: %v", args[2])
	}
	if binary.BigEndian.Uint32(ipRangeStart.To4()) >= binary.BigEndian.Uint32(ipRangeEnd.To4()) {
		return nil, PXEErrorFromString("start of IP range has to be lower than the end of an IP range")
	}
	LeaseTime, err = time.ParseDuration(args[3])
	if err != nil {
		return nil, PXEErrorFromString("invalid duration: %v", args[3])
	}
	r, err := os.Open(filename)
	defer func() {
		if err := r.Close(); err != nil {
			log.Warningf("Failed to close file %s: %v", filename, err)
		}
	}()
	if err != nil {
		return nil, PXEErrorFromString("cannot open lease file %s: %v", filename, err)
	}
	Recordsv4, err = loadRecords(r)
	if err != nil {
		return nil, PXEErrorFromString("failed to load records: %v", err)
	}
	rand.Seed(time.Now().Unix())
	log.Printf("Loaded %d DHCPv4 leases from %s", len(Recordsv4), filename)
	return func(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
		record, ok := Recordsv4[req.ClientHWAddr.String()]
		if !ok {
			log.Printf("MAC address %s is new, leasing new IPv4 address", req.ClientHWAddr.String())
			rec, err := createIP(ipRangeStart, ipRangeEnd)
			if err != nil {
				log.Error(err)
				return nil, true
			}
			err = saveIPAddress(req.ClientHWAddr, rec)
			if err != nil {
				log.Printf("SaveIPAddress for MAC %s failed: %v", req.ClientHWAddr.String(), err)
			}
			Recordsv4[req.ClientHWAddr.String()] = rec
			record = rec
		}
		resp.YourIPAddr = record.IP
		resp.Options.Update(dhcpv4.OptIPAddressLeaseTime(LeaseTime))
		log.Printf("found IP address %s for MAC %s", record.IP, req.ClientHWAddr.String())
		return resp, false
	}, nil
}

func loadRecords(r io.Reader) (map[string]*Record, error) {
	sc := bufio.NewScanner(r)
	records := make(map[string]*Record)
	for sc.Scan() {
		line := sc.Text()
		if len(line) == 0 {
			continue
		}
		tokens := strings.Fields(line)
		if len(tokens) != 3 {
			return nil, PXEErrorFromString("malformed line, want 3 fields, got %d: %s", len(tokens), line)
		}
		hwaddr, err := net.ParseMAC(tokens[0])
		if err != nil {
			return nil, PXEErrorFromString("malformed hardware address: %s", tokens[0])
		}
		ipaddr := net.ParseIP(tokens[1])
		if ipaddr.To4() == nil {
			return nil, PXEErrorFromString("expected an IPv4 address, got: %v", ipaddr)
		}
		expires, err := time.Parse(time.RFC3339, tokens[2])
		if err != nil {
			return nil, PXEErrorFromString("expected time of exipry in RFC3339 format, got: %v", tokens[2])
		}
		records[hwaddr.String()] = &Record{IP: ipaddr, expires: expires}
	}
	return records, nil
}

// createIP allocates a new lease in the provided range.
// TODO this is not concurrency-safe
func createIP(rangeStart net.IP, rangeEnd net.IP) (*Record, error) {
	ip := make([]byte, 4)
	rangeStartInt := binary.BigEndian.Uint32(rangeStart.To4())
	rangeEndInt := binary.BigEndian.Uint32(rangeEnd.To4())
	binary.BigEndian.PutUint32(ip, random(rangeStartInt, rangeEndInt))
	taken := checkIfTaken(ip)
	for taken {
		ipInt := binary.BigEndian.Uint32(ip)
		ipInt++
		binary.BigEndian.PutUint32(ip, ipInt)
		if ipInt > rangeEndInt {
			break
		}
		taken = checkIfTaken(ip)
	}
	for taken {
		ipInt := binary.BigEndian.Uint32(ip)
		ipInt--
		binary.BigEndian.PutUint32(ip, ipInt)
		if ipInt < rangeStartInt {
			return &Record{}, PXEErrorFromString("no new IP addresses available")
		}
		taken = checkIfTaken(ip)
	}
	return &Record{IP: ip, expires: time.Now().Add(LeaseTime)}, nil
}

func random(min uint32, max uint32) uint32 {
	return uint32(rand.Intn(int(max-min))) + min
}

// check if an IP address is already leased. DHCPv4 only.
func checkIfTaken(ip net.IP) bool {
	taken := false
	for _, v := range Recordsv4 {
		if v.IP.String() == ip.String() && (v.expires.After(time.Now())) {
			taken = true
			break
		}
	}
	return taken
}

func saveIPAddress(mac net.HardwareAddr, record *Record) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(mac.String() + " " + record.IP.String() + " " + record.expires.Format(time.RFC3339) + "\n")
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	return nil
}
