package cmd

import (
	"net"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

func TestTCPPing(t *testing.T) {
	// TestTCPPing tests the TCPPing function
	t.Run("CMD TCP", func(t *testing.T) {
		args := []string{
			"tcp",
			flagAddress, testURL,
			flagPort, "",
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "TCP command should not return an error:%s", err)
		assert.NotEmpty(t, out, "TCP command should not return an empty string")
		assert.Contains(t, out, " OPEN", "TCP command should contain OPEN")
		assert.Containsf(t, out, "TCPing done", "Query command should contain TCPing done")
		t.Log(out)
	})
}

func TestTCPPingReturnsErrorForClosedPort(t *testing.T) {
	t.Cleanup(func() {
		queryAddress = ""
		queryPort = ""
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host/port: %v", err)
	}

	args := []string{
		"tcp",
		flagAddress, host,
		flagPort, port,
		flagUnitTest,
		flagDebug,
	}
	out, runErr := common.CmdRun(RootCmd, args)
	assert.Error(t, runErr, "TCP command should return a non-zero exit status for a closed port")
	assert.Contains(t, out, "REFUSED/CLOSED")
}

func TestICMPPing(t *testing.T) {
	if os.Getenv("SKIP_ICMP") != "" {
		t.Skip("SKIP_ICMP set")
	}
	skipIfNoRawICMP(t)

	t.Run("CMD ICMP", func(t *testing.T) {
		args := []string{
			"icmp",
			flagAddress, testURL,
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.Contains(t, out, "ICMPing done", "ICMP command should contain ICMPing done")
		if strings.Contains(out, "ERROR") {
			t.Skipf("ICMP to %s failed — ICMP may be blocked at the network level", testURL)
		}
		assert.NoErrorf(t, err, "ICMP command should not return an error:%s", err)
		assert.NotEmpty(t, out, "ICMP command should not return an empty string")
		assert.Contains(t, out, " OPEN", "ICMP command should contain OPEN")
		t.Log(out)
	})
}

func TestICMPPingReturnsErrorWithoutPermission(t *testing.T) {
	if os.Getenv("SKIP_ICMP") != "" {
		t.Skip("SKIP_ICMP set")
	}
	if hasRawICMPPermission() {
		t.Skip("running with raw ICMP socket permission; cannot exercise the no-permission error path")
	}
	t.Cleanup(func() { queryAddress = "" })

	args := []string{
		"icmp",
		flagAddress, testURL,
		flagUnitTest,
		flagDebug,
	}
	_, err := common.CmdRun(RootCmd, args)
	assert.Error(t, err, "ICMP command should return a non-zero exit status without raw socket permission")
}

// skipIfNoRawICMP calls t.Skip when the process cannot open a raw ICMP socket.
// It uses net.ListenPacket from the standard library — a strict SOCK_RAW call
// with no fallback — rather than icmp.ListenPacket, which may silently succeed
// via an unprivileged SOCK_DGRAM socket on kernels with a permissive
// net.ipv4.ping_group_range, masking the actual lack of raw-socket permission.
func skipIfNoRawICMP(t *testing.T) {
	t.Helper()
	if !hasRawICMPPermission() {
		t.Skipf("skipping ICMP: no raw socket permission (needs CAP_NET_RAW or root)")
	}
}

func hasRawICMPPermission() bool {
	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}
