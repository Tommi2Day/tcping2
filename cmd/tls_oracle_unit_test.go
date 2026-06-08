package cmd

// Unit tests for Oracle Wallet support (addOracleSSO, scanAndAddDERCerts, parseDERLength).
// No cobra invocation or network required.

import (
	"bytes"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ── parseDERLength ───────────────────────────────────────────────────────────

func TestParseDERLengthShortForm(t *testing.T) {
	contentLen, hdrLen := parseDERLength([]byte{0x30, 0x05})
	assert.Equal(t, 5, contentLen)
	assert.Equal(t, 2, hdrLen)
}

func TestParseDERLengthLongForm(t *testing.T) {
	// 0x82 0x01 0x00 → two-byte length = 256
	contentLen, hdrLen := parseDERLength([]byte{0x30, 0x82, 0x01, 0x00})
	assert.Equal(t, 256, contentLen)
	assert.Equal(t, 4, hdrLen)
}

func TestParseDERLengthTruncated(t *testing.T) {
	contentLen, hdrLen := parseDERLength([]byte{0x30})
	assert.Equal(t, 0, contentLen)
	assert.Equal(t, 0, hdrLen)
}

func TestParseDERLengthIndefinite(t *testing.T) {
	// 0x80 = indefinite form — not supported
	contentLen, hdrLen := parseDERLength([]byte{0x30, 0x80})
	assert.Equal(t, 0, contentLen)
	assert.Equal(t, 0, hdrLen)
}

func TestParseDERLengthTooLarge(t *testing.T) {
	// length > 1 MiB → rejected by sanity cap
	// 0x83 0x10 0x00 0x01 = 1 048 577
	contentLen, hdrLen := parseDERLength([]byte{0x30, 0x83, 0x10, 0x00, 0x01})
	assert.Equal(t, 0, contentLen)
	assert.Equal(t, 0, hdrLen)
}

func TestParseDERLengthNumBytesTooMany(t *testing.T) {
	// 0x85 = long form with 5 length bytes — rejected (numBytes > 4)
	contentLen, hdrLen := parseDERLength([]byte{0x30, 0x85, 0x00, 0x00, 0x00, 0x00, 0x01})
	assert.Equal(t, 0, contentLen)
	assert.Equal(t, 0, hdrLen)
}

// ── scanAndAddDERCerts ───────────────────────────────────────────────────────

func TestScanAndAddDERCertsEmpty(t *testing.T) {
	n := scanAndAddDERCerts(x509.NewCertPool(), []byte{})
	assert.Equal(t, 0, n)
}

func TestScanAndAddDERCertsNoCert(t *testing.T) {
	// Random bytes with no valid DER certificate.
	n := scanAndAddDERCerts(x509.NewCertPool(), []byte{0x00, 0x01, 0x02, 0x30, 0x00})
	assert.Equal(t, 0, n)
}

func TestScanAndAddDERCertsWithCert(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() { _ = os.Remove(certFile) })
	der := readCertDERBytes(t, certFile)

	// Embed DER cert in surrounding noise.
	var buf bytes.Buffer
	_, _ = buf.Write([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	_, _ = buf.Write(der)
	_, _ = buf.Write([]byte{0xFF, 0xFE})

	pool := x509.NewCertPool()
	n := scanAndAddDERCerts(pool, buf.Bytes())
	assert.Equal(t, 1, n)
}

// ── addOracleSSO ─────────────────────────────────────────────────────────────

func TestAddOracleSSONonExistent(t *testing.T) {
	err := addOracleSSO(x509.NewCertPool(), "/no/such/file.sso")
	assert.Error(t, err)
}

func TestAddOracleSSONoCerts(t *testing.T) {
	f, err := os.CreateTemp("", "empty-*.sso")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write([]byte{0x00, 0xBA, 0xD0, 0xBA, 0xD1}) // fake header, no certs
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	err = addOracleSSO(x509.NewCertPool(), f.Name())
	assert.Error(t, err)
}

func TestAddOracleSSOWithEmbeddedCert(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() { _ = os.Remove(certFile) })
	der := readCertDERBytes(t, certFile)

	// Simulate cwallet.sso: proprietary header + embedded DER cert + trailing bytes.
	var buf bytes.Buffer
	_, _ = buf.Write([]byte{0x0B, 0xAD, 0x0B, 0xAD}) // fake magic
	_, _ = buf.Write(der)
	_, _ = buf.Write([]byte{0xFF, 0xFE, 0xFD})

	f, err := os.CreateTemp("", "fake-*.sso")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write(buf.Bytes())
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	pool := x509.NewCertPool()
	err = addOracleSSO(pool, f.Name())
	assert.NoError(t, err)
}

// ── buildCertPool — .sso extension ──────────────────────────────────────────

func TestBuildCertPoolSSOFile(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() { _ = os.Remove(certFile) })
	der := readCertDERBytes(t, certFile)

	f, err := os.CreateTemp("", "test-*.sso")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	_, _ = f.Write(der)
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	pool, err := buildCertPool(f.Name())
	assert.NoError(t, err)
	assert.NotNil(t, pool)
}

// ── addCertsFromDir — Oracle Wallet directory ────────────────────────────────

func TestAddCertsDirWithSSOFile(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() { _ = os.Remove(certFile) })
	der := readCertDERBytes(t, certFile)

	dir := t.TempDir()
	ssoData := append([]byte{0x0B, 0xAD, 0x0B, 0xAD}, der...)
	_ = os.WriteFile(filepath.Join(dir, "cwallet.sso"), ssoData, 0o600) //nolint:gosec

	pool := x509.NewCertPool()
	err := addCertsFromDir(pool, dir)
	assert.NoError(t, err)
}

func TestAddCertsDirWithP12File(t *testing.T) {
	// A directory containing an invalid .p12 should log and continue, not error.
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "ewallet.p12"), []byte("not a p12"), 0o600) //nolint:gosec

	pool := x509.NewCertPool()
	err := addCertsFromDir(pool, dir)
	assert.NoError(t, err, "invalid p12 in directory should be skipped, not fail")
}
