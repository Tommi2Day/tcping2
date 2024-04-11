package cmd

import (
	"testing"

	"github.com/tommi2day/gomodules/common"

	"tcping2/test"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	var err error
	var out = ""
	test.InitTestDirs()
	t.Run("Version func", func(t *testing.T) {
		actual := GetVersion(false)
		assert.NotEmpty(t, actual, "GetVersion should not be empty")
		assert.Containsf(t, actual, configName, "GetVersion should contain %s", configName)
		t.Logf(actual)
	})
	t.Run("Version cmd", func(t *testing.T) {
		args := []string{
			"version",
			"--debug",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "version command should not return an error:%s", err)
		assert.NotEmpty(t, out, "version command should not return an empty string")
		assert.Containsf(t, out, configName, "version command should contain %s", configName)
		t.Logf(out)
	})
}
