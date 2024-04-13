package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

func TestTCPPing(t *testing.T) {
	// TestTCPPing tests the TCPPing function
	var err error
	var out = ""
	t.Run("CMD TCP", func(t *testing.T) {
		args := []string{
			"tcp",
			"-a", testURL,
			"--unit-test",
			"--debug",
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "TCP command should not return an error:%s", err)
		assert.NotEmpty(t, out, "TCP command should not return an empty string")
		assert.Contains(t, out, " OPEN", "TCP command should contain OPEN")
		assert.Containsf(t, out, "TCPing done", "Query command should contain HTTPing done")
		t.Logf(out)
	})
}

func TestICMPPing(t *testing.T) {
	// TestTCPPing tests the TCPPing function
	var err error
	var out = ""
	if os.Getenv("SKIP_ICMP") != "" {
		t.Skip("Skipping ICMP tests")
	}
	t.Run("CMD ICMP", func(t *testing.T) {
		args := []string{
			"icmp",
			"-a", testURL,
			"--unit-test",
			"--debug",
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "ICMP command should not return an error:%s", err)
		assert.NotEmpty(t, out, "ICMP command should not return an empty string")
		assert.Contains(t, out, " OPEN", "ICMP command should contain OPEN")
		assert.Containsf(t, out, "ICMPing done", "ICMP command should contain ICMPPing done")
		t.Logf(out)
	})
}
