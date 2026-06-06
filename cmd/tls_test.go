package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

const (
	flagPort     = "-p"
	flagStartTLS = "--starttls"
	flagCertFile = "-f"
	flagChain    = "--chain"
	tlsTestHost  = "www.google.com"
	tlsTestPort  = "443"
	tlsSMTPHost  = "smtp.gmail.com"
	tlsSMTPPort  = "587"
)

func TestTLSValidate(t *testing.T) {
	t.Run("validate TLS connection", func(t *testing.T) {
		args := []string{
			tlsCmdName,
			flagAddress, tlsTestHost,
			flagPort, tlsTestPort,
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "tls command should not return an error: %s", err)
		assert.Contains(t, out, "TLS VALID", "tls command should show TLS VALID")
		t.Log(out)
	})
}

func TestTLSValidateCertFile(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-24*time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() {
		_ = os.Remove(certFile)
		resetCertfileFlag()
	})

	t.Run("validate valid cert file", func(t *testing.T) {
		args := []string{
			tlsCmdName,
			flagCertFile, certFile,
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "tls --certfile should not return an error: %s", err)
		assert.Contains(t, out, "TLS VALID", "certfile check should show TLS VALID")
		t.Log(out)
	})
}

func TestTLSValidateExpiredCertFile(t *testing.T) {
	certFile := writeTempCert(t, time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
	t.Cleanup(func() {
		_ = os.Remove(certFile)
		resetCertfileFlag()
	})

	t.Run("validate expired cert file", func(t *testing.T) {
		args := []string{
			tlsCmdName,
			flagCertFile, certFile,
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "tls --certfile on expired cert should not return command error: %s", err)
		assert.Contains(t, out, "TLS INVALID", "expired certfile check should show TLS INVALID")
		t.Log(out)
	})
}

// resetCertfileFlag clears the certfile flag state so subsequent tests are not affected.
func resetCertfileFlag() {
	tlsCertFile = ""
	if f := tlsCmd.Flags().Lookup("certfile"); f != nil {
		f.Changed = false
	}
}

// resetStartTLSFlag clears the starttls persistent flag state between tests.
func resetStartTLSFlag() {
	tlsStartTLS = ""
	if f := tlsCmd.PersistentFlags().Lookup("starttls"); f != nil {
		f.Changed = false
	}
}

func TestTLSShow(t *testing.T) {
	t.Run("show certificate details", func(t *testing.T) {
		args := []string{
			tlsCmdName, "show",
			flagAddress, tlsTestHost,
			flagPort, tlsTestPort,
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "tls show should not return an error: %s", err)
		assert.Contains(t, out, "Subject:", "tls show should log Subject")
		assert.Contains(t, out, "Not After:", "tls show should log Not After")
		t.Log(out)
	})
}

func TestTLSShowChain(t *testing.T) {
	t.Run("show certificate chain", func(t *testing.T) {
		args := []string{
			tlsCmdName, "show",
			flagAddress, tlsTestHost,
			flagPort, tlsTestPort,
			flagChain,
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "tls show --chain should not return an error: %s", err)
		assert.Contains(t, out, "TLS CERT", "tls show chain should log TLS CERT")
		t.Log(out)
	})
}

func TestTLSStartTLSSMTP(t *testing.T) {
	if os.Getenv("SKIP_STARTTLS") != "" {
		t.Skip("Skipping STARTTLS tests")
	}
	t.Cleanup(resetStartTLSFlag)
	t.Run("SMTP STARTTLS", func(t *testing.T) {
		args := []string{
			tlsCmdName,
			flagAddress, tlsSMTPHost,
			flagPort, tlsSMTPPort,
			flagStartTLS, "smtp",
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "tls SMTP STARTTLS should not return an error: %s", err)
		assert.Contains(t, out, "TLS VALID", "SMTP STARTTLS should show TLS VALID")
		t.Log(out)
	})
}

func TestTLSWeakHashCertFile(t *testing.T) {
	certFile := writeTempCertSHA1(t, time.Now().Add(-24*time.Hour), time.Now().Add(90*24*time.Hour))
	t.Cleanup(func() {
		_ = os.Remove(certFile)
		resetCertfileFlag()
	})

	t.Run("detect weak SHA-1 signature", func(t *testing.T) {
		args := []string{
			tlsCmdName,
			flagCertFile, certFile,
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "tls --certfile on SHA-1 cert should not return command error: %s", err)
		assert.Contains(t, out, "TLS VALID", "SHA-1 cert should still be VALID if not expired")
		assert.Contains(t, out, "TLS WEAK", "SHA-1 cert should trigger WEAK warning")
		t.Log(out)
	})
}

// writeTempCert creates a self-signed PEM certificate file for testing (ECDSA/SHA-256).
func writeTempCert(t *testing.T, notBefore, notAfter time.Time) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test.example.com"},
		DNSNames:     []string{"test.example.com"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	f, err := os.CreateTemp("", "tls-test-*.pem")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	_ = pem.Encode(f, &pem.Block{Type: pemCertType, Bytes: der})
	_ = f.Close()
	return f.Name()
}

// writeTempCertSHA1 creates a self-signed PEM certificate signed with the weak SHA-1WithRSA algorithm.
func writeTempCertSHA1(t *testing.T, notBefore, notAfter time.Time) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:       big.NewInt(2),
		Subject:            pkix.Name{CommonName: "sha1-test.example.com"},
		DNSNames:           []string{"sha1-test.example.com"},
		NotBefore:          notBefore,
		NotAfter:           notAfter,
		SignatureAlgorithm: x509.SHA1WithRSA,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create SHA-1 cert: %v", err)
	}
	f, err := os.CreateTemp("", "tls-sha1-*.pem")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	_ = pem.Encode(f, &pem.Block{Type: pemCertType, Bytes: der})
	_ = f.Close()
	return f.Name()
}
