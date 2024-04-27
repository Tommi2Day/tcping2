package cmd

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/tommi2day/gomodules/common"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"os"
	"regexp"

	"time"
)

type ICMPing struct {
	Address  string
	IP       *net.IPAddr
	Duration time.Duration
	IPType   IPType
}

type TCPing struct {
	Address string
	Msg     string
	Code    int
}

// IPType is a struct that contains the type of IP address to use
type IPType struct {
	Type               string
	ListenAddr         string
	Network            string
	ICMPNetwork        string
	ProtocolNumber     int
	RequestMessageType icmp.Type
	ReplyMessageType   icmp.Type
}

var (
	// IPType4 is the type of IP address to use for IPv4
	IPType4 = IPType{
		Type:               "4",
		ListenAddr:         "0.0.0.0",
		Network:            "ip4",
		ICMPNetwork:        "ip4:icmp",
		ProtocolNumber:     1,
		RequestMessageType: ipv4.ICMPTypeEcho,
		ReplyMessageType:   ipv4.ICMPTypeEchoReply,
	}
	// IPType6 is the type of IP address to use for IPv6
	IPType6 = IPType{
		Type:               "6",
		ListenAddr:         "::",
		Network:            "ip6",
		ICMPNetwork:        "ip6:ipv6-icmp",
		ProtocolNumber:     58,
		RequestMessageType: ipv6.ICMPTypeEchoRequest,
		ReplyMessageType:   ipv6.ICMPTypeEchoReply,
	}
)

var (
	icmpCmd = &cobra.Command{
		Use:          "icmp",
		Short:        "Ping using ICMP protocol",
		Long:         ``,
		RunE:         runICMPPing,
		SilenceUsage: true,
	}
	tcpCmd = &cobra.Command{
		Use:          "tcp",
		Short:        "Ping using TCP protocol",
		Long:         ``,
		RunE:         runTCPPing,
		SilenceUsage: true,
	}
	pingTimeout = 3
)

func init() {
	icmpCmd.Flags().StringVarP(&queryAddress, "address", "a", "", "ip/host to query")
	tcpCmd.Flags().StringVarP(&queryAddress, "address", "a", "", "ip/host to ping")
	tcpCmd.Flags().StringVarP(&queryPort, "port", "p", "", "tcp port to ping")
	tcpCmd.Flags().IntVarP(&pingTimeout, "timeout", "t", pingTimeout, "Ping Timeout in sec")

	RootCmd.AddCommand(tcpCmd)
	RootCmd.AddCommand(icmpCmd)
}

func runICMPPing(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		queryAddress = args[0]
	}
	log.Debugf("ICMPing called with %s:%s", queryAddress, queryPort)
	if queryAddress == "" {
		return fmt.Errorf("please specify an address to query")
	}
	host, _, err := common.GetHostPort(queryAddress)
	if err != nil {
		log.Debugf("ICMPing GetHostPort failed: %v", err)
		return err
	}
	ips, err := dnsConfig.LookupIP(host)
	if err != nil {
		log.Debugf("DNS lookup failed: %v", err)
		return err
	}
	i := new(ICMPing)
	for _, ip := range ips {
		err = i.Run(ip.String())
		i.Log(err)
	}
	log.Debugf("ICMPing done")
	return nil
}

func runTCPPing(_ *cobra.Command, args []string) error {
	// get arguments
	if len(args) > 0 {
		queryAddress = args[0]
	}
	if len(args) > 1 {
		queryPort = args[1]
	}
	log.Debugf("TCPing called with %s:%s", queryAddress, queryPort)

	// check if address is set
	ips, err := normalizeAddress()
	if err != nil {
		log.Debugf("TCPing normalizeAddress failed: %v", err)
		return err
	}

	// iterate over returned IPs
	t := new(TCPing)
	for _, ip := range ips {
		dst := net.JoinHostPort(ip.String(), queryPort)
		_ = t.Run(dst)
		t.Log()
	}
	log.Debugf("TCPing done")
	return nil
}

func normalizeAddress() (ips []net.IP, err error) {
	ips = []net.IP{}
	var host string
	var port int
	if queryAddress == "" {
		err = fmt.Errorf("please specify an address to ping")
		return
	}
	// join address and port if port set
	if queryPort != "" {
		queryAddress = fmt.Sprintf("%s:%s", queryAddress, queryPort)
		log.Debugf("TCPing address set: %s", queryAddress)
	}
	// normalize host and port
	host, port, err = common.GetHostPort(queryAddress)
	if err != nil {
		log.Debugf("Parsing host and port failed: %v", err)
		return
	}
	// if port is set and queryPort is not set, use port
	if port > 0 && queryPort == "" {
		queryPort = fmt.Sprintf("%d", port)
	}
	// finally check if port is really set
	if queryPort == "" {
		err = fmt.Errorf("please specify a port to ping")
		return
	}
	// query DNS
	ips, err = dnsConfig.LookupIP(host)
	return
}

// Run sends a TCP request to a given address and returns the status of the connection
func (t *TCPing) Run(address string) (msg string) {
	log.Debugf("TCPing started for %s", address)
	timeout := time.Duration(pingTimeout) * time.Second
	t.Address = address
	d := net.Dialer{Timeout: timeout}
	_, err := d.Dial("tcp", address)
	if err != nil {
		log.Debugf("TCPing dial message: %v", err)
		match, _ := regexp.MatchString("refused", err.Error())
		if match {
			// Closed
			msg = "REFUSED/CLOSED"
			t.Code = 1
			t.Msg = msg
			return
		}
		match, _ = regexp.MatchString("timeout", err.Error())
		if match {
			// Timeout
			t.Code = 2
			msg = "TIMEOUT/BLOCKED"
			t.Msg = msg
			return
		}
		// Default
		t.Code = 2
		msg = fmt.Sprintf("ERROR: %v", err)
		t.Msg = msg
		return
	}
	// Open
	t.Code = 0
	msg = "OPEN"
	t.Msg = msg
	return
}

// Log logs the tcping results
func (t *TCPing) Log() {
	log.Debugf("enter TCPing Log with %s code %d message: %v", t.Address, t.Code, t.Msg)
	switch t.Code {
	case 0:
		fmt.Printf("%s%s%s\n", cyan("%-7s", "TCP"), green("%-10s", t.Msg), t.Address)
	case 1:
		fmt.Printf("%s%s%s\n", cyan("%-7s", "TCP"), yellow("%-10s", t.Msg), t.Address)
	case 2:
		fmt.Printf("%s%s\n", cyan("%-7s", "TCP"), red("%-10s", t.Msg))
	}
}

// Log logs the ping results
func (i *ICMPing) Log(err error) {
	log.Debugf("enter ICMPing Log %s", i.IP.String())
	if err != nil {
		match, _ := regexp.MatchString("operation not permitted", err.Error())
		if match {
			fmt.Printf("%s%s%s\n",
				cyan("%-7s", "ICMP"),
				red("%-10s", "ERROR"),
				red("No privileges"))
		} else {
			fmt.Printf("%s%s%s\n",
				cyan("%-7s", "ICMP"),
				red("%-10s", "ERROR"), err)
		}
		log.Debugf("result ICMPing to %s: ERROR (%v)", i.IP.String(), err)
		return
	}
	log.Debugf("result ICMPing to %s: OPEN", i.IP.String())
	fmt.Printf("%s%s%-30s    %s ms\n",
		cyan("%-7s", "ICMP"),
		green("%-10s", "OPEN"), i.IP.String(),
		fmt.Sprintf("%.1f", float64(i.Duration.Microseconds())/1000))
}

// Run sends an ICMP echo request to a given address and returns the time it took to get a reply
func (i *ICMPing) Run(host string) (err error) {
	// Check ip type
	// Resolve address
	var dst *net.IPAddr
	var c *icmp.PacketConn
	log.Debugf("ICMPing Run to %s", host)
	dst, err = net.ResolveIPAddr("ip4", host)
	if err != nil {
		log.Debugf("ICMPing ResolveIPAddr ip4 failed: %v", err)
		dst, err = net.ResolveIPAddr("ip6", host)
		if err != nil {
			log.Debugf("ICMPing ResolveIPAddr ip6 failed: %v", err)
			return
		}
		i.IPType = IPType6
	} else {
		i.IPType = IPType4
	}
	i.IP = dst
	log.Debugf("prepare ICMPing to IP %s type %s", i.IP.String(), i.IPType.Type)
	// Start listening for icmp replies
	c, err = icmp.ListenPacket(i.IPType.ICMPNetwork, i.IPType.ListenAddr)
	if err != nil {
		log.Debugf("ICMPing ListenPacket failed: %v", err)
		return err
	}
	defer func(c *icmp.PacketConn) {
		_ = c.Close()
	}(c)

	// Make a new ICMP message
	m := icmp.Message{
		Type: i.IPType.RequestMessageType,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte(""),
		},
	}
	b, err := m.Marshal(nil)
	if err != nil {
		return err
	}

	// Send it
	start := time.Now()
	n, err := c.WriteTo(b, dst)
	if err != nil {
		log.Debugf("ICMPing WriteTo failed: %v", err)
		return err
	} else if n != len(b) {
		log.Debugf("ICMPing WriteTo failed: got %v; want %v", n, len(b))
		return fmt.Errorf("got %v; want %v", n, len(b))
	}

	// Wait for a reply
	log.Debugf("ICMPing waiting for reply")
	reply := make([]byte, 1500)
	err = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		log.Debugf("ICMPing SetReadDeadline failed: %v", err)
		return err
	}
	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		log.Debugf("ICMPing ReadFrom failed: %v", err)
		return err
	}
	duration := time.Since(start)

	// Pack it up boys, we're done here
	rm, err := icmp.ParseMessage(i.IPType.ProtocolNumber, reply[:n])
	if err != nil {
		log.Debugf("ICMPing ParseMessage failed: %v", err)
		return err
	}

	// return dst, duration, nil
	switch rm.Type {
	case i.IPType.ReplyMessageType:
		i.Duration = duration
		log.Debugf("ICMPing ReplyMessageType received, duration %v", i.Duration)
		return nil
	default:
		log.Debugf("ICMPing ReplyMessageType not received: got %+v from %v; want echo reply", rm, peer)
		return fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
	}
}
