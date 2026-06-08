package cmd

// Unit tests for tls_info.go — no cobra invocation or network required.

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTLSVersionName(t *testing.T) {
	cases := []struct {
		v    uint16
		want string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
		{0xFFFF, "unknown (0xffff)"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tlsVersionName(tc.v))
	}
}

func TestVersionLabel(_ *testing.T) {
	// Covers both branches — green for strong, yellow for weak.
	versionLabel(tls.VersionTLS13)
	versionLabel(tls.VersionTLS10)
}

func TestMakeSuiteMap(t *testing.T) {
	m := makeSuiteMap()
	assert.NotEmpty(t, m)
	for _, s := range tls.CipherSuites() {
		_, ok := m[s.ID]
		assert.True(t, ok, "suite %s should be in map", s.Name)
	}
}

func TestTLSConnInfoLogFailed(_ *testing.T) {
	info := &TLSConnInfo{
		Address: tlsTestAddr,
		Err:     errors.New("connection refused"),
	}
	info.Log()
}

func TestTLSConnInfoLogSuccess(_ *testing.T) {
	cert := &x509.Certificate{
		SerialNumber:       big.NewInt(1),
		Subject:            pkix.Name{CommonName: "test.example.com"},
		NotBefore:          time.Now().Add(-time.Hour),
		NotAfter:           time.Now().Add(90 * 24 * time.Hour),
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}
	info := &TLSConnInfo{
		Address:         tlsTestAddr,
		Version:         tls.VersionTLS13,
		CipherSuite:     tls.TLS_AES_256_GCM_SHA384,
		NegotiatedProto: "h2",
		PeerCerts:       []*x509.Certificate{cert},
		HasOCSP:         true,
		HasSCT:          true,
	}
	info.Log()
}

func TestTLSConnInfoLogWeakVersion(_ *testing.T) {
	info := &TLSConnInfo{
		Address:     tlsTestAddr,
		Version:     tls.VersionTLS10,
		CipherSuite: tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	}
	info.Log()
}

func TestTLSConnInfoLogWeakCertSig(_ *testing.T) {
	cert := &x509.Certificate{
		SerialNumber:       big.NewInt(2),
		Subject:            pkix.Name{CommonName: "sha1.example.com"},
		NotBefore:          time.Now().Add(-time.Hour),
		NotAfter:           time.Now().Add(90 * 24 * time.Hour),
		SignatureAlgorithm: x509.SHA1WithRSA,
	}
	info := &TLSConnInfo{
		Address:     tlsTestAddr,
		Version:     tls.VersionTLS12,
		CipherSuite: tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		PeerCerts:   []*x509.Certificate{cert},
	}
	info.Log()
}

func TestTLSConnInfoLogWithProbeResults(_ *testing.T) {
	// Covers version and cipher probe output paths including the negotiated marker.
	info := &TLSConnInfo{
		Address:           tlsTestAddr,
		Version:           tls.VersionTLS12,
		CipherSuite:       tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		SupportedVersions: []uint16{tls.VersionTLS13, tls.VersionTLS12, tls.VersionTLS10},
		SupportedCiphers: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, // negotiated — marked with *
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_RC4_128_SHA, // insecure — should be yellow
		},
	}
	info.Log()
}

func TestTLSConnInfoLogCiphersUnknownID(_ *testing.T) {
	// An ID not in the suite map should be silently skipped.
	info := &TLSConnInfo{
		Address:          tlsTestAddr,
		Version:          tls.VersionTLS12,
		CipherSuite:      tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		SupportedCiphers: []uint16{0xDEAD},
	}
	info.Log()
}
