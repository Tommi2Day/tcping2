package cmd

import (
	"fmt"
	"os"
	"tcping2/test"
	"testing"

	"github.com/ory/dockertest/v3"
	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

var echoPort = common.GetEnv("ECHO_PORT", "39207")
var echoHost = common.GetEnv("ECHO_HOST", "127.0.0.1")
var echoIP = common.GetEnv("ECHO_SERVER", "0.0.0.0")
var echoContainer *dockertest.Resource

func TestMain(m *testing.M) {
	var err error

	test.InitTestDirs()
	if os.Getenv("SKIP_ECHO_SERVER") != "" {
		return
	}
	echoContainer, err = prepareEchoContainer()
	if err != nil {
		log.Fatalf("prepareEchoContainer failed: %s", err)
	}
	code := m.Run()
	destroyEchoContainer(echoContainer)
	os.Exit(code)
}

func TestEchoClient(t *testing.T) {
	var err error
	var out string
	test.InitTestDirs()
	t.Run("Standard Server", func(t *testing.T) {
		unitTestFlag = true
		args := []string{
			"echo",
			"-a", testURL,
			"-p", "",
			"--server=false",
			"--unit-test",
			"--debug",
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "Echo command should not return an error:%s", err)
		assert.NotEmpty(t, out, "Echo command should not return an empty string")
		assert.Contains(t, out, "but connected", "Echo command should contain 'but connected'")
		t.Logf(out)
	})
	t.Run("TCPING Echo server", func(t *testing.T) {
		if os.Getenv("SKIP_ECHO_SERVER") != "" {
			t.Skip("Skipping Echo Server tests")
		}

		queryPort = echoPort
		unitTestFlag = true
		args := []string{
			"echo",
			"-a", echoHost,
			"-p", echoPort,
			"--dnsIPv4=true",
			"--server=false",
			"--unit-test",
			"--debug",
		}
		go func() {
			err = runEchoServer(echoIP, echoPort)
			if err != nil {
				t.Errorf("Echo server error: %s", err)
			}
		}()
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "Echo command should not return an error:%s", err)
		assert.NotEmpty(t, out, "Echo command should not return an empty string")
		exp := fmt.Sprintf("is %s", echoPrefix)
		assert.Containsf(t, out, exp, "Echo command should contain '%s'", exp)
		t.Logf(out)
	})
}
