package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var tlsInfoProbe bool

// TLSConnInfo holds the result of a TLS connection info query.
type TLSConnInfo struct {
	Address         string
	Host            string
	Err             error
	Version         uint16
	CipherSuite     uint16
	NegotiatedProto string
	PeerCerts       []*x509.Certificate
	HasOCSP         bool
	HasSCT          bool
	// populated when --probe is set
	SupportedVersions []uint16
	SupportedCiphers  []uint16
}

var tlsInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show TLS connection parameters (version, cipher, ALPN, ...)",
	Long: "Connect to a server and display the negotiated TLS parameters: " +
		"version, cipher suite, ALPN protocol and certificate key info.\n" +
		"Use --probe to also discover other supported TLS versions and cipher suites.",
	RunE:         runTLSInfo,
	SilenceUsage: true,
}

func init() {
	tlsInfoCmd.Flags().BoolVar(&tlsInfoProbe, "probe", false, "probe for other supported TLS versions and cipher suites")
	tlsCmd.AddCommand(tlsInfoCmd)
}

func runTLSInfo(_ *cobra.Command, args []string) error {
	if len(args) > 0 && queryAddress == "" {
		queryAddress = args[0]
	}
	if queryAddress == "" {
		return fmt.Errorf("please specify an address to connect to")
	}

	host, port, err := parseTLSAddress()
	if err != nil {
		return err
	}

	pool, err := buildCertPool(tlsRootCA)
	if err != nil {
		return err
	}

	info := &TLSConnInfo{Address: net.JoinHostPort(host, port), Host: host}
	info.Err = tlsDialInfo(info, host, port, pool)

	if tlsInfoProbe && info.Err == nil {
		probeVersions(info, host, port, pool)
		probeCiphers(info, host, port, pool)
	}

	info.Log()
	log.Debugf("TLS info done")
	return nil
}

// tlsDialInfo dials TLS and populates TLSConnInfo from the connection state.
func tlsDialInfo(info *TLSConnInfo, host, port string, pool *x509.CertPool) error {
	timeout := time.Duration(tlsTimeout) * time.Second
	addr := net.JoinHostPort(host, port)
	cfg := &tls.Config{
		ServerName: host,
		RootCAs:    pool,
	}

	var tlsConn *tls.Conn
	var err error

	if tlsStartTLS != "" {
		tlsConn, err = startTLS(addr, tlsStartTLS, cfg, timeout)
	} else {
		dialer := &tls.Dialer{
			NetDialer: &net.Dialer{Timeout: timeout},
			Config:    cfg,
		}
		conn, e := dialer.Dial("tcp", addr)
		if e != nil {
			return e
		}
		tlsConn = conn.(*tls.Conn)
	}
	if err != nil {
		return err
	}
	defer func() { _ = tlsConn.Close() }()

	state := tlsConn.ConnectionState()
	info.Version = state.Version
	info.CipherSuite = state.CipherSuite
	info.NegotiatedProto = state.NegotiatedProtocol
	info.PeerCerts = state.PeerCertificates
	info.HasOCSP = len(state.OCSPResponse) > 0
	info.HasSCT = len(state.SignedCertificateTimestamps) > 0
	return nil
}

// tlsVersionName returns a human-readable TLS version string.
func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", v)
	}
}

// weakTLSVersions lists version constants considered cryptographically weak.
var weakTLSVersions = map[uint16]bool{
	tls.VersionTLS10: true,
	tls.VersionTLS11: true,
}

// versionLabel returns the version name coloured green (strong) or yellow (weak).
func versionLabel(v uint16) string {
	name := tlsVersionName(v)
	if weakTLSVersions[v] {
		return yellow(name)
	}
	return green(name)
}

// probeVersions attempts a connection at each TLS version and records which the server accepts.
func probeVersions(info *TLSConnInfo, host, port string, pool *x509.CertPool) {
	probeList := []uint16{tls.VersionTLS13, tls.VersionTLS12, tls.VersionTLS11, tls.VersionTLS10}
	timeout := time.Duration(tlsTimeout) * time.Second
	addr := net.JoinHostPort(host, port)

	for _, v := range probeList {
		cfg := &tls.Config{
			ServerName: host,
			RootCAs:    pool,
			MinVersion: v,
			MaxVersion: v,
		}
		dialer := &tls.Dialer{
			NetDialer: &net.Dialer{Timeout: timeout},
			Config:    cfg,
		}
		conn, err := dialer.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			info.SupportedVersions = append(info.SupportedVersions, v)
		}
		log.Debugf("TLS probe version %s: %v", tlsVersionName(v), err == nil)
	}
}

// probeCiphers tests each TLS 1.2 cipher suite individually and records which the server accepts.
func probeCiphers(info *TLSConnInfo, host, port string, pool *x509.CertPool) {
	var suites []*tls.CipherSuite
	suites = append(suites, tls.CipherSuites()...)
	suites = append(suites, tls.InsecureCipherSuites()...)

	timeout := time.Duration(tlsTimeout) * time.Second
	addr := net.JoinHostPort(host, port)

	for _, suite := range suites {
		cfg := &tls.Config{
			ServerName:   host,
			RootCAs:      pool,
			MinVersion:   tls.VersionTLS12,
			MaxVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{suite.ID}, //nolint:gosec // intentional: probing for server-supported suites including potentially insecure ones
		}
		dialer := &tls.Dialer{
			NetDialer: &net.Dialer{Timeout: timeout},
			Config:    cfg,
		}
		conn, err := dialer.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			info.SupportedCiphers = append(info.SupportedCiphers, suite.ID)
		}
		log.Debugf("TLS probe cipher %s: %v", suite.Name, err == nil)
	}
}

// Log prints TLS connection parameters to stdout.
func (info *TLSConnInfo) Log() {
	label := cyan("%-7s", "TLS")
	if info.Err != nil {
		log.Debugf("TLS INFO FAILED %s: %s", info.Address, info.Err)
		fmt.Printf("%s%s%s\n      %s%s\n",
			label, red("%-10s", "FAILED"), info.Address,
			cyan("%-10s", "REASON"), info.Err)
		return
	}
	info.logParams()
	info.logProbeResults()
}

func (info *TLSConnInfo) logParams() {
	versionStr := tlsVersionName(info.Version)
	cipherStr := tls.CipherSuiteName(info.CipherSuite)
	versionDisplay := versionStr
	versionFn := green
	if weakTLSVersions[info.Version] {
		versionFn = yellow
		versionDisplay += "  [WEAK]"
	}

	log.Debugf("TLS INFO %s version %s cipher %s", info.Address, versionStr, cipherStr)
	fmt.Printf("%s%s%s\n", cyan("%-7s", "TLS"), cyan("%-10s", "INFO"), info.Address)
	fmt.Printf("  %-16s %s\n", "Version:", versionFn(versionDisplay))
	fmt.Printf("  %-16s %s\n", "Cipher suite:", cipherStr)

	if info.NegotiatedProto != "" {
		log.Debugf("TLS INFO ALPN %s", info.NegotiatedProto)
		fmt.Printf("  %-16s %s\n", "ALPN:", info.NegotiatedProto)
	}

	if len(info.PeerCerts) > 0 {
		leaf := info.PeerCerts[0]
		sigAlg := leaf.SignatureAlgorithm.String()
		if isWeakSigAlg(leaf.SignatureAlgorithm) {
			sigAlg = yellow("%s  [WEAK]", sigAlg)
		}
		log.Debugf("TLS INFO cert %s sig %s", leaf.Subject.CommonName, leaf.SignatureAlgorithm.String())
		fmt.Printf("  %-16s %s\n", "Cert subject:", leaf.Subject.CommonName)
		fmt.Printf("  %-16s %s\n", "Cert signature:", sigAlg)
	}

	ocspStr := "no"
	if info.HasOCSP {
		ocspStr = green("yes")
		log.Debugf("TLS INFO OCSP stapling yes")
	}
	fmt.Printf("  %-16s %s\n", "OCSP stapling:", ocspStr)

	if info.HasSCT {
		log.Debugf("TLS INFO SCT yes")
		fmt.Printf("  %-16s %s\n", "SCT (CT logs):", green("yes"))
	}
}

func (info *TLSConnInfo) logProbeResults() {
	if len(info.SupportedVersions) > 0 {
		log.Debugf("TLS INFO versions probed %d", len(info.SupportedVersions))
		fmt.Printf("  %-16s", "TLS versions:")
		for _, v := range info.SupportedVersions {
			fmt.Printf(" %s", versionLabel(v))
		}
		fmt.Println()
	}
	if len(info.SupportedCiphers) > 0 {
		log.Debugf("TLS INFO ciphers probed %d", len(info.SupportedCiphers))
		info.logCiphers(makeSuiteMap())
	}
}

func (info *TLSConnInfo) logCiphers(suiteMap map[uint16]*tls.CipherSuite) {
	fmt.Printf("  %-16s (TLS 1.2, * = negotiated)\n", "Cipher suites:")
	for _, id := range info.SupportedCiphers {
		s, ok := suiteMap[id]
		if !ok {
			continue
		}
		marker := "    "
		if id == info.CipherSuite {
			marker = "  * "
		}
		name := s.Name
		if s.Insecure {
			name = yellow(s.Name)
		}
		fmt.Printf("%s%s\n", marker, name)
		log.Debugf("TLS INFO cipher %s insecure=%v", s.Name, s.Insecure)
	}
}

// makeSuiteMap builds an ID→*CipherSuite lookup for all known cipher suites.
func makeSuiteMap() map[uint16]*tls.CipherSuite {
	m := make(map[uint16]*tls.CipherSuite)
	for _, s := range tls.CipherSuites() {
		m[s.ID] = s
	}
	for _, s := range tls.InsecureCipherSuites() {
		m[s.ID] = s
	}
	return m
}
