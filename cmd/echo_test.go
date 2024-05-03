package cmd

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"tcping2/test"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

var echoPort = common.GetEnv("ECHO_PORT", "39207")
var echoHost = common.GetEnv("ECHO_HOST", "127.0.0.1")
var echoIP = common.GetEnv("ECHO_SERVER", "0.0.0.0")

func TestEchoClient(t *testing.T) {
	var err error
	var out string
	test.InitTestDirs()

	if err != nil {
		log.Fatalf("prepareEchoContainer failed: %s", err)
	}
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

	t.Run("create Echo Server", func(t *testing.T) {
		if os.Getenv("SKIP_ECHO_SERVER") != "" {
			t.Skip("Skipping Echo Server tests")
		}

		// start server
		testCh := make(chan echoResult)
		// defer close(testCh)
		go runEchoServer(echoIP, echoPort, testCh)
		time.Sleep(1 * time.Second)

		// test server
		c, e := net.Dial("tcp", echoIP+":"+echoPort)
		_ = c.SetDeadline(time.Now().Add(3 * time.Second))
		assert.NoErrorf(t, e, "Echo server should not return an error:%s", e)
		testEcho := []byte("TEST TCPING\n")
		_, _ = c.Write(testEcho)
		r := bufio.NewReader(c)
		answer, _ := r.ReadBytes('\n')
		assert.Equal(t, testEcho, answer, "Echo server should return the same message")

		// run client
		if os.Getenv("SKIP_ECHO_CONTAINER") != "" {
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
			"--timeout", "10",
		}

		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "Echo command should not return an error:%s", err)
		assert.NotEmpty(t, out, "Echo command should not return an empty string")
		exp := fmt.Sprintf("is %s", echoPrefix)
		assert.Containsf(t, out, exp, "Echo command should contain '%s'", exp)
		t.Logf(out)
		_, _ = c.Write([]byte("QUIT\n"))
		_ = c.Close()
		select {
		case r := <-testCh:
			assert.NoErrorf(t, r.err, "Echo server should not return an error:%s", r.err)
		case <-time.After(time.Duration(10) * time.Second):
			err = fmt.Errorf("timeout waiting for server shutdown")
			t.Log(err)
		}
	})
}
