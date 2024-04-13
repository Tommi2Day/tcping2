package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/tommi2day/gomodules/common"

	"github.com/spf13/cobra"
)

// MTR is a struct that contains the MTR report
type MTR struct {
	Report ReportMTR `json:"report"`
}

// ReportMTR is a struct that contains the MTR report parts
type ReportMTR struct {
	Desc DescMTR   `json:"mtr"`
	Hops []HopsMTR `json:"hubs"`
}

// DescMTR is a struct that contains the MTR call information
type DescMTR struct {
	Src        string `json:"src"`
	Dst        string `json:"dst"`
	Tos        int    `json:"tos"`
	Tests      int    `json:"tests"`
	Psize      string `json:"psize"`
	Bitpattern string `json:"bitpattern"`
}

// HopsMTR is a struct that contains the MTR hop information
type HopsMTR struct {
	Count int     `json:"count"`
	Host  string  `json:"host"`
	Loss  float64 `json:"Loss%"`
	Snt   int     `json:"Snt"`
	Last  float64 `json:"Last"`
	Avg   float64 `json:"Avg"`
	Best  float64 `json:"Best"`
	Wrst  float64 `json:"Wrst"`
	StDev float64 `json:"StDev"`
}

var (
	mtrCmd = &cobra.Command{
		Use:          "mtr",
		Short:        "Traceroute using MTR",
		Long:         ``,
		RunE:         runMTR,
		SilenceUsage: true,
	}
	tcpFlag = false
	mtrBin  = "mtr"
)

const osWin = "windows"

func init() {
	mtrCmd.Flags().StringVarP(&queryAddress, "address", "a", "", "ip/host to ping")
	mtrCmd.Flags().StringVarP(&queryPort, "port", "p", "", "tcp port to ping")
	mtrCmd.Flags().StringVarP(&mtrBin, "mtr", "m", mtrBin, "mtr binary path")
	mtrCmd.Flags().BoolVarP(&tcpFlag, "tcp", "t", false, "use TCP instead of ICMP")
	os := runtime.GOOS
	if os != osWin {
		// mtr cli is not available on windows
		RootCmd.AddCommand(mtrCmd)
	}
}

func runMTR(_ *cobra.Command, args []string) error {
	if !common.CommandExists(mtrBin) {
		return fmt.Errorf("mtr command '%s' not found", mtrBin)
	}
	if len(args) > 0 {
		queryAddress = args[0]
	}
	if queryAddress == "" {
		return fmt.Errorf("please specify an address to query")
	}
	if len(args) > 1 {
		queryPort = args[1]
		queryAddress = queryAddress + ":" + queryPort
	}
	host, port, err := common.GetHostPort(queryAddress)
	if err != nil {
		return err
	}
	if port > 0 && queryPort == "" {
		queryPort = fmt.Sprintf("%d", port)
	}
	if queryPort == "" && tcpFlag {
		return fmt.Errorf("please specify a port to ping")
	}

	ips, err := dnsConfig.LookupIP(host)
	if err != nil {
		return err
	}
	var mtr = new(MTR)
	for _, ip := range ips {
		a := ip.String()
		err = mtr.Run(a, queryPort, tcpFlag)
		if err != nil {
			fmt.Printf("%s%s\n", cyan("%-7s", "MTR"), red("%-10s", err))
			continue
		}
		mtr.Log()
	}
	return nil
}

// Log logs the mtr results
func (mtr *MTR) Log() {
	for _, h := range mtr.Report.Hops {
		fmt.Printf("%s %3d %-60s Loss: %6.2f%% Avg:%6.2fms\n", cyan("%-4s", "Hop"), h.Count, h.Host, h.Loss, h.Avg)
	}
}

// Run runs system mtr command and returns the IP addresses of the hops
func (mtr *MTR) Run(ip string, port string, t bool) (err error) {
	var cmd *exec.Cmd
	txt := ip
	if t {
		cmd = exec.Command("mtr", "-j", ip, "-T", "-P", port)
		txt = ip + ":" + port
	} else {
		cmd = exec.Command("mtr", "-j", ip)
	}
	log.Debugf("command %s", strings.Join(cmd.Args, " "))
	fmt.Printf("Waiting for MTR results to %s ...\n", txt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Warnf("error running mtr: %v:%s", err, string(out))
		err = fmt.Errorf("%s:%s", err, string(out))
		return
	}
	if len(out) == 0 {
		log.Warnf("no output")
		err = fmt.Errorf("no output")
		return
	}
	if out[0] != '{' {
		log.Warnf("no json output: %s", string(out))
		err = fmt.Errorf("no json output: %s", string(out))
		return
	}
	_, err = mtr.Parse(out)
	return
}

func (mtr *MTR) Parse(b []byte) (hops []HopsMTR, err error) {
	hops = []HopsMTR{}

	// sanity
	b = sanityJSON(b)
	err = json.Unmarshal(b, &mtr)
	if err != nil {
		log.Warnf("error parsing json: %v:%s", err, string(b))
		err = fmt.Errorf("error parsing json: %v:%s", err, string(b))
		return
	}
	hops = mtr.Report.Hops
	return
}

func sanityJSON(b []byte) []byte {
	var v int64
	var e error
	s := string(b)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		f := strings.Split(l, ":")
		if len(f) > 1 {
			z := f[1]
			switch {
			case strings.Contains(f[0], "tos"), strings.Contains(f[0], "tests"), strings.Contains(f[0], "count"):

				v, e = getHexValInt64(z)
				if e != nil {
					log.Warnf("error parsing intval in json: %v:%s", e, string(b))
					v = 0
				}
				// take care to repeat last comma in json
				if strings.LastIndex(z, ",") > 0 {
					lines[i] = fmt.Sprintf(`%s: %d,`, f[0], int(v))
				} else {
					lines[i] = fmt.Sprintf(`%s: %d`, f[0], int(v))
				}
			}
		}
	}
	s = strings.Join(lines, "\n")
	b = []byte(s)
	return b
}

func getHexValInt64(value string) (int64, error) {
	var v int64
	var err error
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, ",")
	value = strings.ReplaceAll(value, `"`, "")
	if strings.Contains(value, "0x") {
		value = strings.TrimPrefix(value, "0x")
		v, err = strconv.ParseInt(value, 16, 64)
	} else {
		v, err = strconv.ParseInt(value, 10, 64)
	}
	return v, err
}
