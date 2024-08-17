package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

func TestQueryInfo(t *testing.T) {
	var err error
	var out = ""
	if os.Getenv("SKIP_QUERY") != "" {
		t.Skip("Skipping Query tests")
	}
	t.Run("CMD TCP", func(t *testing.T) {
		args := []string{
			"query",
			"-a", "www.google.com",
			"--unit-test",
			"--debug",
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "Query command should not return an error:%s", err)
		assert.NotEmpty(t, out, "Query command should not return an empty string")
		assert.Containsf(t, out, "Query done", "Query command should contain Query done")
		t.Log(out)
	})
}
