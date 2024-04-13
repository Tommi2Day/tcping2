package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

var testURL = common.GetEnv("TEST_URL", "https://www.google.com")

func TestHTTPing(t *testing.T) {
	var err error
	var out = ""
	t.Run("CMD HTTP", func(t *testing.T) {
		args := []string{
			"http",
			"-a", testURL,
			"--unit-test",
			"--debug",
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "HTTP command should not return an error:%s", err)
		assert.NotEmpty(t, out, "HTTP command should not return an empty string")
		assert.Containsf(t, out, "HTTPing done", "HTTP command should contain HTTPing done")
		assert.Contains(t, out, " OK", "HTTP command should contain OK")
		t.Logf(out)
	})
}
