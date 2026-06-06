package cmd

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tommi2day/gomodules/common"
	"golang.org/x/crypto/pkcs12"
)

const (
	tlsCmdName  = "tls"
	pemCertType = "CERTIFICATE"
)

const jksMagic uint32 = 0xFEEDFEED
const jksTrustedCert = 2

var (
	tlsPort      string
	tlsRootCA    string
	tlsStartTLS  string
	tlsCertFile  string
	tlsTimeout   int
	tlsShowChain bool
)

// TLSResult holds the result of a TLS validation
type TLSResult struct {
	Address   string
	Host      string
	PeerCerts []*x509.Certificate
	Valid     bool
	Err       error
}

var tlsCmd = &cobra.Command{
	Use:          tlsCmdName,
	Short:        "Validate a TLS connection or local certificate",
	Long:         "Connect to a server and validate its TLS certificate against the system trust store or a custom CA. Use --certfile to check a local certificate file instead.",
	RunE:         runTLSValidate,
	SilenceUsage: true,
}

var tlsShowCmd = &cobra.Command{
	Use:          "show",
	Short:        "Show TLS certificate details and chain",
	Long:         "Connect to a server and display the certificate's subject, issuer, SANs, validity dates and optionally the full chain.",
	RunE:         runTLSShow,
	SilenceUsage: true,
}

func init() {
	tlsCmd.PersistentFlags().StringVarP(&queryAddress, "address", "a", "", "host[:port] to connect to")
	tlsCmd.PersistentFlags().StringVarP(&tlsPort, "port", "p", "443", "TCP port")
	tlsCmd.PersistentFlags().StringVarP(&tlsRootCA, "rootca", "r", "", "root CA: PEM file, directory, Java trust store (.jks) or PKCS12 (.p12/.pfx)")
	tlsCmd.PersistentFlags().StringVar(&tlsStartTLS, "starttls", "", "upgrade via STARTTLS: smtp, imap, pop3, ftp")
	tlsCmd.PersistentFlags().IntVarP(&tlsTimeout, "timeout", "t", 5, "connection timeout in seconds")

	tlsCmd.Flags().StringVarP(&tlsCertFile, "certfile", "f", "", "validate a local certificate file instead of connecting")

	tlsShowCmd.Flags().BoolVar(&tlsShowChain, "chain", false, "show full certificate chain")

	tlsCmd.AddCommand(tlsShowCmd)
	RootCmd.AddCommand(tlsCmd)
}

// runTLSValidate validates a TLS connection or a local certificate file.
func runTLSValidate(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && queryAddress == "" {
		queryAddress = args[0]
	}

	if common.CmdFlagChanged(cmd, "certfile") && tlsCertFile != "" {
		return validateCertFile(tlsCertFile)
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

	result := &TLSResult{Address: net.JoinHostPort(host, port), Host: host}
	result.Err = tlsDial(result, host, port, pool)
	result.Valid = result.Err == nil
	result.LogValidate()
	log.Debugf("TLS validate done")
	return nil
}

// runTLSShow connects and prints certificate details.
func runTLSShow(_ *cobra.Command, args []string) error {
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

	result := &TLSResult{Address: net.JoinHostPort(host, port), Host: host}
	result.Err = tlsDial(result, host, port, pool)
	result.Valid = result.Err == nil
	result.LogShow(tlsShowChain)
	log.Debugf("TLS show done")
	return nil
}

// parseTLSAddress resolves the host and port from flags and args.
// GetHostPort errors are intentionally discarded: they mean no port was embedded in the address,
// so we fall back to the port flag value.
func parseTLSAddress() (string, string, error) {
	addr := queryAddress
	if tlsPort != "443" {
		addr = fmt.Sprintf("%s:%s", queryAddress, tlsPort)
	}
	h, p, _ := common.GetHostPort(addr)
	if p == 0 {
		return queryAddress, tlsPort, nil
	}
	return h, fmt.Sprintf("%d", p), nil
}

// tlsDial establishes a TLS connection, optionally via STARTTLS.
func tlsDial(result *TLSResult, host, port string, pool *x509.CertPool) error {
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

	result.PeerCerts = tlsConn.ConnectionState().PeerCertificates
	return nil
}

// startTLS performs a plaintext connection then upgrades via STARTTLS.
func startTLS(addr, proto string, cfg *tls.Config, timeout time.Duration) (*tls.Conn, error) {
	proto = strings.ToLower(proto)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	switch proto {
	case "smtp":
		if err = smtpStartTLS(rw); err != nil {
			_ = conn.Close()
			return nil, err
		}
	case "imap":
		if err = imapStartTLS(rw); err != nil {
			_ = conn.Close()
			return nil, err
		}
	case "pop3":
		if err = pop3StartTLS(rw); err != nil {
			_ = conn.Close()
			return nil, err
		}
	case "ftp":
		if err = ftpStartTLS(rw); err != nil {
			_ = conn.Close()
			return nil, err
		}
	default:
		_ = conn.Close()
		return nil, fmt.Errorf("unsupported STARTTLS protocol: %s (use smtp, imap, pop3, ftp)", proto)
	}

	tlsConn := tls.Client(conn, cfg)
	if err = tlsConn.Handshake(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}
	return tlsConn, nil
}

func smtpStartTLS(rw *bufio.ReadWriter) error {
	if _, err := readResponse(rw.Reader, "220"); err != nil {
		return fmt.Errorf("SMTP banner: %w", err)
	}
	if _, err := sendRecv(rw, "EHLO tcping2\r\n", "250"); err != nil {
		return fmt.Errorf("SMTP EHLO: %w", err)
	}
	if _, err := sendRecv(rw, "STARTTLS\r\n", "220"); err != nil {
		return fmt.Errorf("SMTP STARTTLS: %w", err)
	}
	return nil
}

func imapStartTLS(rw *bufio.ReadWriter) error {
	if _, err := readResponse(rw.Reader, "* OK"); err != nil {
		return fmt.Errorf("IMAP banner: %w", err)
	}
	if _, err := sendRecv(rw, "a001 CAPABILITY\r\n", "a001 OK"); err != nil {
		return fmt.Errorf("IMAP CAPABILITY: %w", err)
	}
	if _, err := sendRecv(rw, "a002 STARTTLS\r\n", "a002 OK"); err != nil {
		return fmt.Errorf("IMAP STARTTLS: %w", err)
	}
	return nil
}

func pop3StartTLS(rw *bufio.ReadWriter) error {
	if _, err := readResponse(rw.Reader, "+OK"); err != nil {
		return fmt.Errorf("POP3 banner: %w", err)
	}
	if _, err := sendRecv(rw, "STLS\r\n", "+OK"); err != nil {
		return fmt.Errorf("POP3 STLS: %w", err)
	}
	return nil
}

func ftpStartTLS(rw *bufio.ReadWriter) error {
	if _, err := readResponse(rw.Reader, "220"); err != nil {
		return fmt.Errorf("FTP banner: %w", err)
	}
	if _, err := sendRecv(rw, "AUTH TLS\r\n", "234"); err != nil {
		return fmt.Errorf("FTP AUTH TLS: %w", err)
	}
	return nil
}

// sendRecv writes a command and reads until a line containing the expected prefix.
func sendRecv(rw *bufio.ReadWriter, cmd, expect string) (string, error) {
	if _, err := fmt.Fprint(rw.Writer, cmd); err != nil {
		return "", err
	}
	if err := rw.Flush(); err != nil {
		return "", err
	}
	return readResponse(rw.Reader, expect)
}

// readResponse reads lines until one contains the expected prefix (or error).
func readResponse(r *bufio.Reader, expect string) (string, error) {
	var last string
	for {
		line, err := r.ReadString('\n')
		last = strings.TrimSpace(line)
		if err != nil {
			return last, fmt.Errorf("read error waiting for %q: %w", expect, err)
		}
		log.Debugf("< %s", last)
		if strings.Contains(last, expect) {
			return last, nil
		}
	}
}

// buildCertPool creates an x509.CertPool from the system store plus any custom CA.
func buildCertPool(rootCA string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		log.Debugf("system cert pool unavailable: %v – using empty pool", err)
		pool = x509.NewCertPool()
	}
	if rootCA == "" {
		return pool, nil
	}

	info, err := os.Stat(rootCA)
	if err != nil {
		return nil, fmt.Errorf("rootca %q: %w", rootCA, err)
	}

	if info.IsDir() {
		return pool, addCertsFromDir(pool, rootCA)
	}

	ext := strings.ToLower(filepath.Ext(rootCA))
	switch ext {
	case ".jks":
		return pool, addJKSTrustStore(pool, rootCA)
	case ".p12", ".pfx":
		return pool, addPKCS12TrustStore(pool, rootCA)
	default:
		return pool, addPEMFile(pool, rootCA)
	}
}

// addPEMFile appends all certificates in a PEM file to the pool.
func addPEMFile(pool *x509.CertPool, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !pool.AppendCertsFromPEM(data) {
		return fmt.Errorf("no valid PEM certificates found in %s", path)
	}
	return nil
}

// addCertsFromDir appends certificates from all .pem/.crt files in a directory.
func addCertsFromDir(pool *x509.CertPool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".pem" || ext == ".crt" || ext == ".cer" {
			path := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				log.Debugf("skipping %s: %v", path, err)
				continue
			}
			pool.AppendCertsFromPEM(data)
		}
	}
	return nil
}

// jksReadTrustedCert parses one trusted-certificate entry from JKS binary data at off.
// Returns the tag, DER bytes, new offset, and whether parsing can continue.
// ok=false means the data was truncated; tag != jksTrustedCert means a non-cert entry.
func jksReadTrustedCert(data []byte, off int) (tag int, certDER []byte, next int, ok bool) {
	if off+4 > len(data) {
		return 0, nil, off, false
	}
	tag = int(binary.BigEndian.Uint32(data[off : off+4]))
	off += 4

	// alias: 2-byte length + UTF-8 bytes; creation date: 8 bytes
	if off+2 > len(data) {
		return tag, nil, off, false
	}
	off += 2 + int(binary.BigEndian.Uint16(data[off:off+2])) + 8

	if tag != jksTrustedCert {
		return tag, nil, off, true
	}

	// cert type string: 2-byte length + UTF-8
	if off+2 > len(data) {
		return tag, nil, off, false
	}
	off += 2 + int(binary.BigEndian.Uint16(data[off:off+2]))

	// cert DER: 4-byte length + bytes
	if off+4 > len(data) {
		return tag, nil, off, false
	}
	certLen := int(binary.BigEndian.Uint32(data[off : off+4]))
	off += 4
	if off+certLen > len(data) {
		return tag, nil, off, false
	}
	return tag, data[off : off+certLen], off + certLen, true
}

// addJKSTrustStore reads trusted certificate entries from a Java KeyStore file.
func addJKSTrustStore(pool *x509.CertPool, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 12 {
		return fmt.Errorf("%s is too small to be a JKS file", path)
	}
	if binary.BigEndian.Uint32(data[0:4]) != jksMagic {
		return fmt.Errorf("%s is not a JKS file (bad magic)", path)
	}

	count := int(binary.BigEndian.Uint32(data[8:12]))
	off := 12
	added := 0

	for i := 0; i < count; i++ {
		tag, certDER, next, ok := jksReadTrustedCert(data, off)
		if !ok {
			break
		}
		off = next
		if tag != jksTrustedCert {
			break // private key entries have a complex structure; stop here
		}
		cert, err := x509.ParseCertificate(certDER)
		if err != nil {
			log.Debugf("JKS: skipping unparseable cert at entry %d: %v", i, err)
			continue
		}
		pool.AddCert(cert)
		added++
	}

	if added == 0 {
		return fmt.Errorf("no trusted certificates found in JKS file %s", path)
	}
	log.Debugf("JKS: loaded %d trusted certificate(s) from %s", added, path)
	return nil
}

// addPKCS12TrustStore reads trusted certificates from a PKCS12 / P12 file.
// Password-less and empty-password P12 files (typical Java cacerts exports) are tried first.
func addPKCS12TrustStore(pool *x509.CertPool, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	certs, err := decodePKCS12TrustStore(data, "")
	if err != nil {
		// Try with "changeit" – the Java default trust store password
		certs, err = decodePKCS12TrustStore(data, "changeit")
		if err != nil {
			return fmt.Errorf("cannot decode PKCS12 trust store %s: %w", path, err)
		}
	}

	for _, c := range certs {
		pool.AddCert(c)
	}
	log.Debugf("PKCS12: loaded %d trusted certificate(s) from %s", len(certs), path)
	return nil
}

// decodePKCS12TrustStore tries to extract all certificates from a PKCS12 bundle.
func decodePKCS12TrustStore(data []byte, password string) ([]*x509.Certificate, error) {
	blocks, err := pkcs12.ToPEM(data, password)
	if err != nil {
		return nil, err
	}
	var certs []*x509.Certificate
	for _, b := range blocks {
		if b.Type != pemCertType {
			continue
		}
		cert, err := x509.ParseCertificate(b.Bytes)
		if err != nil {
			log.Debugf("PKCS12: skipping unparseable cert: %v", err)
			continue
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in PKCS12 data")
	}
	return certs, nil
}

// validateCertFile reads a PEM certificate file and reports its validity.
func validateCertFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}

	now := time.Now()
	block, rest := pem.Decode(data)
	if block == nil {
		// try DER
		cert, err := x509.ParseCertificate(data)
		if err != nil {
			return fmt.Errorf("no valid certificate found in %s", path)
		}
		logCertValidation(path, cert, now)
		return nil
	}

	count := 0
	for block != nil {
		if block.Type == pemCertType {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				log.Debugf("skipping unparseable cert: %v", err)
			} else {
				logCertValidation(path, cert, now)
				count++
			}
		}
		block, rest = pem.Decode(rest)
	}
	if count == 0 {
		return fmt.Errorf("no valid certificates found in %s", path)
	}
	return nil
}

// weakSigAlgorithms lists signature algorithms that are considered cryptographically weak.
var weakSigAlgorithms = map[x509.SignatureAlgorithm]bool{
	x509.MD2WithRSA:    true,
	x509.MD5WithRSA:    true,
	x509.SHA1WithRSA:   true,
	x509.DSAWithSHA1:   true,
	x509.ECDSAWithSHA1: true,
}

// isWeakSigAlg reports whether the algorithm is considered outdated.
func isWeakSigAlg(alg x509.SignatureAlgorithm) bool {
	return weakSigAlgorithms[alg]
}

// logCertValidation prints validation status for a single certificate.
func logCertValidation(source string, cert *x509.Certificate, now time.Time) {
	expired := now.After(cert.NotAfter)
	notYet := now.Before(cert.NotBefore)
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
	weak := isWeakSigAlg(cert.SignatureAlgorithm)

	label := cyan("%-7s", "TLS")
	switch {
	case expired:
		log.Debugf("TLS INVALID %s: expired on %s", source, cert.NotAfter.UTC().Format("2006-01-02"))
		fmt.Printf("%s%s%s  REASON: expired on %s\n",
			label, red("%-10s", "INVALID"), source,
			cert.NotAfter.UTC().Format("2006-01-02"))
	case notYet:
		log.Debugf("TLS INVALID %s: not valid before %s", source, cert.NotBefore.UTC().Format("2006-01-02"))
		fmt.Printf("%s%s%s  REASON: not valid before %s\n",
			label, red("%-10s", "INVALID"), source,
			cert.NotBefore.UTC().Format("2006-01-02"))
	default:
		log.Debugf("TLS VALID %s expires %s %d days", source, cert.NotAfter.UTC().Format("2006-01-02"), daysLeft)
		fmt.Printf("%s%s%s  (expires %s, %d days)\n",
			label, green("%-10s", "VALID"), source,
			cert.NotAfter.UTC().Format("2006-01-02"), daysLeft)
	}
	if weak {
		log.Debugf("TLS WEAK signature algorithm: %s", cert.SignatureAlgorithm)
		fmt.Printf("       %s%s uses a weak signature algorithm (%s)\n",
			yellow("%-10s", "WARN"), source, cert.SignatureAlgorithm)
	}
}

// LogValidate prints the TLS validation result.
func (r *TLSResult) LogValidate() {
	label := cyan("%-7s", "TLS")
	if r.Valid {
		leaf := r.leafCert()
		daysLeft := int(time.Until(leaf.NotAfter).Hours() / 24)
		log.Debugf("TLS VALID %s expires %s %d days", r.Address, leaf.NotAfter.UTC().Format("2006-01-02"), daysLeft)
		fmt.Printf("%s%s%s  (expires %s, %d days)\n",
			label, green("%-10s", "VALID"), r.Address,
			leaf.NotAfter.UTC().Format("2006-01-02"), daysLeft)
		if isWeakSigAlg(leaf.SignatureAlgorithm) {
			log.Debugf("TLS WEAK signature algorithm: %s", leaf.SignatureAlgorithm)
			fmt.Printf("       %s%s\n",
				yellow("%-10s", "WARN"),
				yellow("weak signature algorithm: %s", leaf.SignatureAlgorithm))
		}
	} else {
		reason := r.Err.Error()
		log.Debugf("TLS INVALID %s: %s", r.Address, reason)
		fmt.Printf("%s%s%s\n      %s%s\n",
			label, red("%-10s", "INVALID"), r.Address,
			cyan("%-10s", "REASON"), reason)
	}
}

// LogShow prints detailed certificate and chain information.
func (r *TLSResult) LogShow(showChain bool) {
	label := cyan("%-7s", "TLS")
	if !r.Valid {
		reason := r.Err.Error()
		log.Debugf("TLS INVALID %s: %s", r.Address, reason)
		fmt.Printf("%s%s%s\n      %s%s\n",
			label, red("%-10s", "INVALID"), r.Address,
			cyan("%-10s", "REASON"), reason)
		if len(r.PeerCerts) == 0 {
			return
		}
	}
	if len(r.PeerCerts) == 0 {
		fmt.Printf("%s%s%s  no certificates received\n",
			label, yellow("%-10s", "WARN"), r.Address)
		return
	}

	log.Debugf("TLS CERT %s", r.Address)
	fmt.Printf("%s%s%s\n", label, cyan("%-10s", "CERT"), r.Address)
	printCertDetails(r.PeerCerts[0], "  ")

	if showChain && len(r.PeerCerts) > 1 {
		for i, c := range r.PeerCerts[1:] {
			fmt.Printf("  %s\n", cyan("Chain[%d]:", i+1))
			printCertDetails(c, "    ")
		}
	}
}

// printCertDetails formats the fields of a single certificate.
func printCertDetails(cert *x509.Certificate, indent string) {
	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
	expiryColor := green
	if daysLeft < 0 {
		expiryColor = red
	} else if daysLeft < 30 {
		expiryColor = yellow
	}

	log.Debugf("Subject: %s", cert.Subject.String())
	log.Debugf("Not After: %s (%d days)", cert.NotAfter.UTC().Format("2006-01-02 15:04:05 UTC"), daysLeft)

	sigAlgStr := cert.SignatureAlgorithm.String()
	if isWeakSigAlg(cert.SignatureAlgorithm) {
		log.Debugf("TLS WEAK signature algorithm: %s", sigAlgStr)
		sigAlgStr = yellow("%s  [WEAK]", sigAlgStr)
	}

	fmt.Printf("%s%-14s %s\n", indent, "Subject:", cert.Subject.String())
	fmt.Printf("%s%-14s %s\n", indent, "Issuer:", cert.Issuer.String())
	fmt.Printf("%s%-14s %s\n", indent, "Signature:", sigAlgStr)
	fmt.Printf("%s%-14s %s\n", indent, "Not Before:", cert.NotBefore.UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Printf("%s%-14s %s\n", indent, "Not After:",
		expiryColor("%s  (%d days)", cert.NotAfter.UTC().Format("2006-01-02 15:04:05 UTC"), daysLeft))

	var sans []string
	sans = append(sans, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}
	sans = append(sans, cert.EmailAddresses...)
	if len(sans) > 0 {
		fmt.Printf("%s%-14s %s\n", indent, "SANs:", strings.Join(sans, ", "))
	}
	fmt.Printf("%s%-14s %s / SN %s\n", indent, "Serial:", cert.SerialNumber.Text(16), cert.SerialNumber.String())
}

// leafCert returns the first peer certificate or a zero-value placeholder.
func (r *TLSResult) leafCert() *x509.Certificate {
	if len(r.PeerCerts) > 0 {
		return r.PeerCerts[0]
	}
	return &x509.Certificate{}
}
