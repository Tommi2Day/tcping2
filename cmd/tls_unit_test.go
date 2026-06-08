package cmd

// Unit-level tests that call tls.go functions directly (no cobra / network required).

import (
	"bufio"
	"bytes"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

// ── buildCertPool ────────────────────────────────────────────────────────────

func TestBuildCertPoolNoRootCA(t *testing.T) {
	pool, err := buildCertPool("")
	assert.NoError(t, err)
	assert.NotNil(t, pool)
}

func TestBuildCertPoolPEMFile(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() { _ = os.Remove(certFile) })

	pool, err := buildCertPool(certFile)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
}

func TestBuildCertPoolDirectory(t *testing.T) {
	dir := t.TempDir()
	certFile := writeTempCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() { _ = os.Remove(certFile) })

	data, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	// .pem, .crt extensions should be loaded; .txt skipped; subdirs silently ignored.
	_ = os.WriteFile(filepath.Join(dir, "ca.pem"), data, 0o600) //nolint:gosec
	_ = os.WriteFile(filepath.Join(dir, "ca.crt"), data, 0o600) //nolint:gosec
	_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignored"), 0o600)
	_ = os.Mkdir(filepath.Join(dir, "subdir"), 0o700)

	pool, err := buildCertPool(dir)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
}

func TestBuildCertPoolNonExistentFile(t *testing.T) {
	_, err := buildCertPool("/no/such/file.pem")
	assert.Error(t, err)
}

func TestBuildCertPoolInvalidP12(t *testing.T) {
	f, err := os.CreateTemp("", "bad-*.p12")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write([]byte("not a p12 file"))
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	_, err = buildCertPool(f.Name())
	assert.Error(t, err)
}

// ── addJKSTrustStore / jksReadTrustedCert ────────────────────────────────────

func TestAddJKSTrustStoreValid(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() { _ = os.Remove(certFile) })

	der := readCertDERBytes(t, certFile)
	jksFile := makeMinimalJKS(t, der)
	t.Cleanup(func() { _ = os.Remove(jksFile) })

	pool := x509.NewCertPool()
	err := addJKSTrustStore(pool, jksFile)
	assert.NoError(t, err)
}

func TestAddJKSTrustStoreTooSmall(t *testing.T) {
	f, err := os.CreateTemp("", "bad-*.jks")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write([]byte{0x01, 0x02})
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	err = addJKSTrustStore(x509.NewCertPool(), f.Name())
	assert.Error(t, err)
}

func TestAddJKSTrustStoreBadMagic(t *testing.T) {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, uint32(0xDEADBEEF))
	_ = binary.Write(&buf, binary.BigEndian, uint32(2))
	_ = binary.Write(&buf, binary.BigEndian, uint32(0))

	f, err := os.CreateTemp("", "bad-*.jks")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write(buf.Bytes())
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	err = addJKSTrustStore(x509.NewCertPool(), f.Name())
	assert.Error(t, err)
}

func TestAddJKSTrustStoreNonCertEntry(t *testing.T) {
	// A JKS with a private-key entry (tag=1) — we stop and report no trusted certs.
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, jksMagic)
	_ = binary.Write(&buf, binary.BigEndian, uint32(2))
	_ = binary.Write(&buf, binary.BigEndian, uint32(1))
	_ = binary.Write(&buf, binary.BigEndian, uint32(1)) // tag 1 = private key
	_ = binary.Write(&buf, binary.BigEndian, uint16(1))
	_, _ = buf.Write([]byte{'k'})
	_ = binary.Write(&buf, binary.BigEndian, int64(0))

	f, err := os.CreateTemp("", "pk-*.jks")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write(buf.Bytes())
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	err = addJKSTrustStore(x509.NewCertPool(), f.Name())
	assert.Error(t, err, "no trusted certs expected")
}

func TestJksReadTrustedCertTruncated(t *testing.T) {
	tag, certDER, _, ok := jksReadTrustedCert([]byte{}, 0)
	assert.False(t, ok)
	assert.Nil(t, certDER)
	assert.Zero(t, tag)
}

func TestJksReadTrustedCertTruncationCases(t *testing.T) {
	// Case 1: only 4 bytes (tag) — truncated before alias length can be read.
	d1 := make([]byte, 4)
	binary.BigEndian.PutUint32(d1, uint32(jksTrustedCert))
	_, _, _, ok := jksReadTrustedCert(d1, 0)
	assert.False(t, ok, "case 1: truncated at alias")

	// Case 2: tag + zero-length alias + short creation date — truncated before cert type.
	// tag(4) + aliasLen(2) + alias(0) + date(8) = needs 14 bytes; give only 13.
	d2 := make([]byte, 13)
	binary.BigEndian.PutUint32(d2, uint32(jksTrustedCert))
	_, _, _, ok = jksReadTrustedCert(d2, 0)
	assert.False(t, ok, "case 2: truncated at cert type")

	// Case 3: full header through cert type string — truncated before cert DER length.
	// Needs tag(4)+aliasLen(2)+alias(0)+date(8)+certTypeLen(2)+certType(5)+certDERlen(4)=25; give 24.
	d3 := make([]byte, 24)
	binary.BigEndian.PutUint32(d3, uint32(jksTrustedCert))
	binary.BigEndian.PutUint16(d3[14:], 5) // certTypeLen = 5
	copy(d3[16:], "X.509")
	_, _, _, ok = jksReadTrustedCert(d3, 0)
	assert.False(t, ok, "case 3: truncated at cert DER length")

	// Case 4: certLen field present but cert DER data is shorter than advertised.
	d4 := make([]byte, 50)
	binary.BigEndian.PutUint32(d4, uint32(jksTrustedCert))
	binary.BigEndian.PutUint16(d4[14:], 5) // certTypeLen = 5
	copy(d4[16:], "X.509")
	binary.BigEndian.PutUint32(d4[21:], 100) // certLen = 100, but only 25 bytes remain
	_, _, _, ok = jksReadTrustedCert(d4, 0)
	assert.False(t, ok, "case 4: truncated cert DER data")
}

// ── validateCertFile ─────────────────────────────────────────────────────────

func TestValidateCertFileDER(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() { _ = os.Remove(certFile) })

	der := readCertDERBytes(t, certFile)

	f, err := os.CreateTemp("", "der-*.cer")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write(der)
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	err = validateCertFile(f.Name())
	assert.NoError(t, err)
}

func TestValidateCertFileNotYetValid(t *testing.T) {
	// NotBefore is in the future — exercises the "notYet" branch in logCertValidation.
	certFile := writeTempCert(t, time.Now().Add(24*time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() {
		_ = os.Remove(certFile)
		resetCertfileFlag()
	})
	err := validateCertFile(certFile)
	assert.NoError(t, err)
}

func TestValidateCertFileInvalidData(t *testing.T) {
	f, err := os.CreateTemp("", "bad-*.pem")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write([]byte("not a certificate"))
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	err = validateCertFile(f.Name())
	assert.Error(t, err)
}

func TestValidateCertFilePEMNoCertBlocks(t *testing.T) {
	// A PEM file whose only block is a PRIVATE KEY — no CERTIFICATE entries → count==0 error.
	f, err := os.CreateTemp("", "nocer-*.pem")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_ = pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("fake")})
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	err = validateCertFile(f.Name())
	assert.Error(t, err)
}

func TestValidateCertFileNotFound(t *testing.T) {
	err := validateCertFile("/no/such/cert.pem")
	assert.Error(t, err)
}

// ── TLSResult methods ────────────────────────────────────────────────────────

const tlsTestAddr = "test.example.com:443"

func TestLogValidateInvalidResult(t *testing.T) {
	r := &TLSResult{
		Address: tlsTestAddr,
		Valid:   false,
		Err:     errors.New("x509: certificate has expired"),
	}
	r.LogValidate() // exercises the INVALID branch
	t.Log("LogValidate INVALID path covered")
}

func TestLogValidateWeakAlgorithm(t *testing.T) {
	cert := &x509.Certificate{
		NotAfter:           time.Now().Add(90 * 24 * time.Hour),
		SignatureAlgorithm: x509.SHA1WithRSA,
	}
	r := &TLSResult{
		Address:   tlsTestAddr,
		Valid:     true,
		PeerCerts: []*x509.Certificate{cert},
	}
	r.LogValidate() // exercises VALID + WEAK warning branch
	t.Log("LogValidate WEAK path covered")
}

func TestLogShowInvalidNoPeerCerts(t *testing.T) {
	r := &TLSResult{
		Address: tlsTestAddr,
		Valid:   false,
		Err:     errors.New("connection refused"),
	}
	r.LogShow(false) // invalid + no peer certs → early return
	r.LogShow(true)
	t.Log("LogShow INVALID path covered")
}

func TestLogShowValidNoCerts(t *testing.T) {
	r := &TLSResult{Address: tlsTestAddr, Valid: true}
	r.LogShow(false) // Valid=true but no PeerCerts → WARN "no certificates received"
	t.Log("LogShow WARN path covered")
}

func TestLeafCertNoPeerCerts(t *testing.T) {
	r := &TLSResult{}
	cert := r.leafCert()
	assert.NotNil(t, cert)
}

// ── startTLS error path ───────────────────────────────────────────────────────

func TestTLSStartTLSInvalidProtocol(t *testing.T) {
	t.Cleanup(resetStartTLSFlag)
	args := []string{
		tlsCmdName, tlsValidateCertCmdName,
		flagAddress, tlsTestHost,
		flagPort, tlsTestPort,
		flagStartTLS, "ldap",
		flagUnitTest,
		flagDebug,
	}
	out, err := common.CmdRun(RootCmd, args)
	assert.NoError(t, err, "tls command should not return a CLI error for unknown STARTTLS")
	assert.Contains(t, out, "TLS INVALID", "unsupported STARTTLS protocol should show TLS INVALID")
	t.Log(out)
}

// ── STARTTLS integration ──────────────────────────────────────────────────────

// TestStartTLSProtocols exercises the IMAP, POP3, and FTP STARTTLS handshake
// code paths using local mock servers. The TLS handshake is expected to fail
// (no valid cert on the mock), but the protocol-negotiation code is covered.
func TestStartTLSIMAPProtocol(t *testing.T) {
	tlsStartTLS = protoIMAP
	t.Cleanup(resetStartTLSFlag)
	addr := serveMockSTARTTLS(t, protoIMAP)
	host, port, _ := net.SplitHostPort(addr)
	pool, _ := buildCertPool("")
	result := &TLSResult{Address: addr, Host: host}
	_ = tlsDial(result, host, port, pool) // TLS handshake fails after STARTTLS — expected
}

func TestStartTLSPOP3Protocol(t *testing.T) {
	tlsStartTLS = protoPOP3
	t.Cleanup(resetStartTLSFlag)
	addr := serveMockSTARTTLS(t, protoPOP3)
	host, port, _ := net.SplitHostPort(addr)
	pool, _ := buildCertPool("")
	result := &TLSResult{Address: addr, Host: host}
	_ = tlsDial(result, host, port, pool)
}

func TestStartTLSFTPProtocol(t *testing.T) {
	tlsStartTLS = protoFTP
	t.Cleanup(resetStartTLSFlag)
	addr := serveMockSTARTTLS(t, protoFTP)
	host, port, _ := net.SplitHostPort(addr)
	pool, _ := buildCertPool("")
	result := &TLSResult{Address: addr, Host: host}
	_ = tlsDial(result, host, port, pool)
}

// TestStartTLSBannerErrors covers the error-return paths inside smtpStartTLS,
// imapStartTLS, pop3StartTLS, and ftpStartTLS by connecting to a server that
// immediately closes the connection (simulating a read error on the banner).
func TestStartTLSBannerErrors(t *testing.T) {
	for _, proto := range []string{protoSMTP, protoIMAP, protoPOP3, protoFTP} {
		proto := proto
		t.Run(proto, func(t *testing.T) {
			// Start a listener that immediately closes — banner read will get EOF.
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("listen: %v", err)
			}
			go func() {
				conn, _ := ln.Accept()
				if conn != nil {
					_ = conn.Close()
				}
				_ = ln.Close()
			}()

			addr := ln.Addr().String()
			tlsStartTLS = proto
			t.Cleanup(resetStartTLSFlag)
			_, err = startTLS(addr, proto, nil, 3*time.Second)
			assert.Error(t, err, "banner read should fail when server closes immediately")
		})
	}
}

// ── printCertDetails expiry color paths ──────────────────────────────────────

func TestPrintCertDetailsExpiryColors(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name     string
		notAfter time.Time
	}{
		{"green-65days", now.Add(65 * 24 * time.Hour)},
		{"yellow-15days", now.Add(15 * 24 * time.Hour)},
		{"red-expired", now.Add(-24 * time.Hour)},
	} {
		tc := tc
		t.Run(tc.name, func(_ *testing.T) {
			cert := &x509.Certificate{
				NotBefore:          now.Add(-time.Hour),
				NotAfter:           tc.notAfter,
				SignatureAlgorithm: x509.ECDSAWithSHA256,
			}
			printCertDetails(cert, "  ")
		})
	}
}

func TestPrintCertDetailsWithIPAndEmailSAN(_ *testing.T) {
	cert := &x509.Certificate{
		NotBefore:          time.Now().Add(-time.Hour),
		NotAfter:           time.Now().Add(90 * 24 * time.Hour),
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		IPAddresses:        []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		EmailAddresses:     []string{"admin@example.com"},
	}
	printCertDetails(cert, "  ")
}

// ── logQuery / getMaxNameLength ───────────────────────────────────────────────

func TestLogQueryAndGetMaxNameLength(_ *testing.T) {
	info := IPInfo{
		IP:        "8.8.8.8",
		Continent: "North America",
		Country:   "United States",
		City:      "Mountain View",
		Latitude:  37.422,
		Longitude: -122.084,
		ASN:       15169,
		ORG:       "Google LLC",
	}
	// logQuery uses fmt.Printf — no panic is the assertion.
	logQuery(info)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// makeMinimalJKS creates a temporary JKS file with one trusted-certificate entry.
func makeMinimalJKS(t *testing.T, certDER []byte) string {
	t.Helper()
	alias := []byte("testcert")
	certType := []byte("X.509")

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, jksMagic)
	_ = binary.Write(&buf, binary.BigEndian, uint32(2))
	_ = binary.Write(&buf, binary.BigEndian, uint32(1))
	_ = binary.Write(&buf, binary.BigEndian, uint32(jksTrustedCert))
	_ = binary.Write(&buf, binary.BigEndian, uint16(len(alias))) //nolint:gosec
	_, _ = buf.Write(alias)
	_ = binary.Write(&buf, binary.BigEndian, time.Now().UnixMilli())
	_ = binary.Write(&buf, binary.BigEndian, uint16(len(certType))) //nolint:gosec
	_, _ = buf.Write(certType)
	_ = binary.Write(&buf, binary.BigEndian, uint32(len(certDER))) //nolint:gosec
	_, _ = buf.Write(certDER)

	f, err := os.CreateTemp("", "test-*.jks")
	if err != nil {
		t.Fatalf("create temp JKS: %v", err)
	}
	_, _ = f.Write(buf.Bytes())
	_ = f.Close()
	return f.Name()
}

// serveMockSTARTTLS starts a local TCP listener that speaks the minimal STARTTLS
// dialect for the given proto ("imap", "pop3", "ftp"). After sending the upgrade
// response it lets the connection close so the TLS handshake fails — that is
// intentional: the goal is coverage of the protocol-negotiation code, not a
// successful TLS session.
func serveMockSTARTTLS(t *testing.T, proto string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

		var lines []string
		switch proto {
		case protoIMAP:
			lines = []string{
				"* OK IMAP4rev1 ready\r\n",
				"* CAPABILITY IMAP4rev1 STARTTLS\r\na001 OK done\r\n",
				"a002 OK Begin TLS negotiation\r\n",
			}
		case protoPOP3:
			lines = []string{
				"+OK POP3 ready\r\n",
				"+OK Go ahead\r\n",
			}
		case protoFTP:
			lines = []string{
				"220 FTP ready\r\n",
				"234 AUTH TLS OK\r\n",
			}
		}

		for i, line := range lines {
			_, _ = fmt.Fprint(rw.Writer, line)
			_ = rw.Flush()
			// read the client command between responses (except after the last line)
			if i < len(lines)-1 {
				_, _ = rw.ReadString('\n')
			}
		}
		// Close: the TLS handshake will fail with EOF — that is expected.
	}()

	return ln.Addr().String()
}

// readCertDERBytes reads the DER bytes of the first certificate block in a PEM file.
func readCertDERBytes(t *testing.T, pemPath string) []byte {
	t.Helper()
	data, err := os.ReadFile(pemPath)
	if err != nil {
		t.Fatalf("read %s: %v", pemPath, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatalf("no PEM block in %s", pemPath)
	}
	return block.Bytes
}
