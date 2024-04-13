package cmd

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tommi2day/gomodules/common"
)

const mtr092 = `{
  "report": {
    "mtr": {
      "src": "s",
      "dst": "google.com",
      "tos": "0x0",
      "psize": "64",
      "bitpattern": "0x00",
      "tests": "10"
    },
    "hubs": [{
      "count": "1",
      "host": "a",
      "Loss%": 0.00,
      "Snt": 10,
      "Last": 0.46,
      "Avg": 0.51,
      "Best": 0.43,
      "Wrst": 0.73,
      "StDev": 0.09
    },
    {
      "count": "2",
      "host": "b",
      "Loss%": 0.00,
      "Snt": 10,
      "Last": 0.81,
      "Avg": 0.66,
      "Best": 0.38,
      "Wrst": 0.98,
      "StDev": 0.22
        }]
  }
}
`
const mtr095 = `{
    "report": {
        "mtr": {
            "src": "s",
            "dst": "google.com",
            "tos": 0,
            "tests": 10,
            "psize": "64",
            "bitpattern": "0x00"
        },
        "hubs": [
            {
                "count": 1,
                "host": "a",
                "Loss%": 0.0,
                "Snt": 10,
                "Last": 0.54,
                "Avg": 0.649,
                "Best": 0.322,
                "Wrst": 0.971,
                "StDev": 0.226
            },
            {
                "count": 2,
                "host": "b",
                "Loss%": 0.0,
                "Snt": 10,
                "Last": 0.753,
                "Avg": 1.056,
                "Best": 0.727,
                "Wrst": 1.454,
                "StDev": 0.283
            }]
  }
}
`

func TestMTR092(t *testing.T) {
	var mtr MTR
	b := sanityJSON([]byte(mtr092))
	err := json.Unmarshal(b, &mtr)
	assert.NoError(t, err, "json.Unmarshal failed: %v", err)
	assert.Equal(t, "s", mtr.Report.Desc.Src, "SRC not expected")
	assert.Equal(t, 1, mtr.Report.Hops[0].Count, "count not expected: %d")
}
func TestMTR095(t *testing.T) {
	var mtr MTR
	b := sanityJSON([]byte(mtr095))
	err := json.Unmarshal(b, &mtr)
	assert.NoError(t, err, "json.Unmarshal failed: %v", err)
	assert.Equal(t, "s", mtr.Report.Desc.Src, "SRC not expected")
	assert.Equal(t, 1, mtr.Report.Hops[0].Count, "count not expected: %d")
}

func TestMTR(t *testing.T) {
	var out string
	var err error
	if runtime.GOOS == osWin {
		t.Skip("Skipping MTR test on Windows")
	}
	if !common.CommandExists("mtr") {
		t.Skip("Skipping MTR test, not in path")
	}

	t.Run("CMD MTR", func(t *testing.T) {
		args := []string{
			"mtr",
			"-a", testURL,
			"--unit-test",
			"--debug",
		}
		out, err = common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "mtr command should not return an error:%s", err)
		assert.NotEmpty(t, out, "mtr command should not return an empty string")
		assert.Containsf(t, out, "HTTPing done", "HTTP command should contain HTTPing done")
		t.Logf(out)
	})
}
