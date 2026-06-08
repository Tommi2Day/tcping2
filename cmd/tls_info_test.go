package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tommi2day/gomodules/common"
)

func TestTLSInfo(t *testing.T) {
	if os.Getenv("SKIP_TLS") != "" {
		t.Skip("Skipping TLS tests")
	}

	t.Run("show connection info", func(t *testing.T) {
		args := []string{
			tlsCmdName, "info",
			flagAddress, tlsTestHost,
			flagPort, tlsTestPort,
			flagUnitTest,
			flagDebug,
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "tls info should not return an error: %s", err)
		assert.Contains(t, out, "TLS INFO", "tls info should log TLS INFO")
		assert.Contains(t, out, "version", "tls info should log version")
		assert.Contains(t, out, "cipher", "tls info should log cipher")
		t.Log(out)
	})
}
