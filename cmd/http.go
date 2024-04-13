package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tommi2day/gomodules/common"
	"golang.org/x/net/http/httpproxy"
)

// HTTPing is a struct that contains the statistics of the httping
type HTTPing struct {
	URL      string
	Proxy    bool
	Scheme   string
	DNS      int64
	TCP      int64
	TLS      int64
	Process  int64
	Transfer int64
	Total    int64
}

var (
	httpCmd = &cobra.Command{
		Use:          "http",
		Short:        "Run httptrace to the target",
		Long:         ``,
		RunE:         runHTTPPing,
		SilenceUsage: true,
	}
)

const schemeHTTP = "http"
const schemeHTTPS = "https"

func init() {
	httpCmd.Flags().StringVarP(&queryAddress, "address", "a", "", "URL to query")
	RootCmd.AddCommand(httpCmd)
}

func runHTTPPing(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		queryAddress = args[0]
	}
	if queryAddress == "" {
		return fmt.Errorf("please specify an URL to query")
	}
	h := new(HTTPing)
	err := h.Run(queryAddress)
	if err != nil {
		log.Debugf("HTTPing failed: %v", err)
		return err
	}
	h.Log()
	log.Debugf("HTTPing done")
	return nil
}

// Run New sends an HTTP request to a given address and returns the time it took to get a reply
func (h *HTTPing) Run(address string) (err error) {
	var t0, t1, t2, t3, t4, t5, t6, t7 int64
	log.Debugf("HTTPing started for %s", address)
	// check if is address really an URL, if not add https://
	if !strings.Contains(address, "://") {
		log.Debugf("Adding scheme to address %s", address)
		address = schemeHTTPS + "://" + address
	}
	h.URL = address
	switch {
	case strings.HasPrefix(address, schemeHTTP+"://"):
		h.Scheme = schemeHTTP
	case strings.HasPrefix(address, schemeHTTPS+"://"):
		h.Scheme = schemeHTTPS
	default:
		return fmt.Errorf("invalid scheme in URL %s, only http and https allowed", address)
	}

	// create a new HTTP request
	req, _ := http.NewRequest("GET", address, nil)
	// create a new HTTP trace definition
	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			t0 = time.Now().UnixNano()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			t1 = time.Now().UnixNano()
			if info.Err != nil {
				err = info.Err
				log.Warn(info.Err)
			}
		},
		ConnectStart: func(_, _ string) {
		},
		ConnectDone: func(_, addr string, err error) {
			if err != nil {
				log.Warnf("unable to connect to host %v: %v", addr, err)
			}
			t2 = time.Now().UnixNano()
		},
		GotConn: func(_ httptrace.GotConnInfo) {
			t3 = time.Now().UnixNano()
		},
		GotFirstResponseByte: func() {
			t4 = time.Now().UnixNano()
		},
		TLSHandshakeStart: func() {
			t5 = time.Now().UnixNano()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			t6 = time.Now().UnixNano()
		},
	}
	// add the trace and run the request
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	c := &http.Client{
		Timeout: 5 * time.Second,
	}
	log.Debugf("HTTPing do request")
	_, err = c.Do(req)
	if err != nil {
		match, _ := regexp.MatchString("Client.Timeout exceeded", err.Error())
		if match {
			err = fmt.Errorf("HTTP connection timeout")
		}
		err = fmt.Errorf("HTTP Client returned '%s'", err)
		log.Debugf("HTTPing failed: %v", err)
		return
	}
	log.Debugf("HTTPing create Statistics")
	// create statistics
	t7 = time.Now().UnixNano()

	if t0 == 0 {
		t0 = t2
		t1 = t2
	}

	h.DNS = t1 - t0
	h.TCP = t2 - t1
	h.Process = t4 - t3
	h.Transfer = t7 - t4
	h.TLS = t6 - t5
	h.Total = t7 - t0

	// Detect system proxies
	pc := httpproxy.FromEnvironment()
	if pc.HTTPProxy != "" {
		log.Debugf("HTTPing detected proxy %s", pc.HTTPProxy)
		h.Proxy = true
	}
	return
}

// Log logs the httping results
func (h *HTTPing) Log() {
	log.Debugf("enter log HTTPing results for  %s", h.URL)
	host, port, err := common.GetHostPort(h.URL)
	if err == nil {
		fmt.Printf("%s:    %s\n", cyan("%-10s", "URL"), h.URL)
		fmt.Printf("%s:    %s\n", cyan("%-10s", "Proxy"), strconv.FormatBool(h.Proxy))
		fmt.Printf("%s:    %s\n", cyan("%-10s", "Scheme"), h.Scheme)
		fmt.Printf("%s:    %s\n", cyan("%-10s", "Host"), host)
		fmt.Printf("%s:    %d\n", cyan("%-10s", "Port"), port)
		fmt.Printf("%s:    %.2f ms\n", cyan("%-10s", "DNS Lookup"), float64(h.DNS)/1e6)
		fmt.Printf("%s:    %.2f ms\n", cyan("%-10s", "TCP"), float64(h.TCP)/1e6)
		if h.Scheme == "https" {
			fmt.Printf("%s:    %.2f ms\n", cyan("%-10s", "TLS"), float64(h.TLS)/1e6)
		}
		fmt.Printf("%s:    %.2f ms\n", cyan("%-10s", "Process"), float64(h.Process)/1e6)
		fmt.Printf("%s:    %.2f ms\n", cyan("%-10s", "Transfer"), float64(h.Transfer)/1e6)
		fmt.Printf("%s:    %.2f ms\n", cyan("%-10s", "Total"), float64(h.Total)/1e6)
		log.Debugf("result HTTPing for %s: OK", h.URL)
		return
	}
	fmt.Printf("%s%s%s\n", cyan("%-7s", "HTTP"), red(" %s: ", "ERROR"), err)
	log.Debugf("result HTTPing for %s: ERROR (%v)", h.URL, err)
}
