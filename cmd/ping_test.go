package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
	"golang.org/x/net/icmp"
)

func TestTCPPing(t *testing.T) {
	// TestTCPPing tests the TCPPing function
	var err error
	var out = ""
	t.Run("CMD TCP", func(t *testing.T) {
		args := []string{
			"tcp",
			flagAddress, testURL,
			"-p", "",
			flagUnitTest,
			flagDebug,
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "TCP command should not return an error:%s", err)
		assert.NotEmpty(t, out, "TCP command should not return an empty string")
		assert.Contains(t, out, " OPEN", "TCP command should contain OPEN")
		assert.Containsf(t, out, "TCPing done", "Query command should contain TCPing done")
		t.Log(out)
	})
}

func TestICMPPing(t *testing.T) {
	var err error
	var out = ""
	if os.Getenv("SKIP_ICMP") != "" {
		t.Skip("Skipping ICMP tests")
	}
	// Auto-skip if the process lacks raw-socket permission (needs CAP_NET_RAW or root).
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		t.Skipf("skipping ICMP: no raw socket permission (needs CAP_NET_RAW or sudo): %v", err)
	}
	_ = c.Close()
	t.Run("CMD ICMP", func(t *testing.T) {
		args := []string{
			"icmp",
			flagAddress, testURL,
			flagUnitTest,
			flagDebug,
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "ICMP command should not return an error:%s", err)
		assert.NotEmpty(t, out, "ICMP command should not return an empty string")
		assert.Contains(t, out, " OPEN", "ICMP command should contain OPEN")
		assert.Containsf(t, out, "ICMPing done", "ICMP command should contain ICMPPing done")
		t.Log(out)
	})
}
