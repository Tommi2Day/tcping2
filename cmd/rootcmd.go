// Package cmd commands
package cmd

import (
	"os"
	"time"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/tommi2day/gomodules/netlib"
)

var (
	// RootCmd function to execute in tests
	RootCmd = &cobra.Command{
		Use:   "tcping2",
		Short: "tcping2 – open port probe and ip info command line tool, supporting ICMP, TCP and HTTP protocols",
		Long: `Tcping2 is a ip probe command line tool, supporting ICMP and TCP protocols 
      It may also run an httptrace and ip traces (using system mtr installation, but not on windows).
      You can also use it to query IP network information from https://ifconfig.is.`,
	}

	debugFlag      = false
	infoFlag       = false
	noLogColorFlag = false
	unitTestFlag   = false
)

var (
	queryAddress string
	queryPort    string
	dnsServer    string
	dnsPort      int
	dnsTimeout   int
	dnsTCP       bool
	dnsIPv4Only  bool
	dnsConfig    *netlib.DNSconfig
)

const configName = "tcping2"

func init() {
	// parse commandline
	RootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "", false, "verbose debug output")
	RootCmd.PersistentFlags().BoolVarP(&infoFlag, "info", "", false, "reduced info output")
	RootCmd.PersistentFlags().BoolVar(&unitTestFlag, "unit-test", false, "redirect output for unit tests")
	RootCmd.PersistentFlags().BoolVar(&noLogColorFlag, "no-color", false, "disable colored log output")
	RootCmd.PersistentFlags().BoolVar(&dnsTCP, "dnsTCP", false, "Query DNS with TCP instead of UDP")
	RootCmd.PersistentFlags().BoolVar(&dnsIPv4Only, "dnsIPv4", false, "return only IPv4 Addresses from DNS Server")
	RootCmd.PersistentFlags().IntVar(&dnsTimeout, "dnsTimeout", 0, "DNS Timeout in sec")
	RootCmd.PersistentFlags().IntVar(&dnsPort, "dnsPort", 0, "DNS Server Port Address")
	RootCmd.PersistentFlags().StringVar(&dnsServer, "dnsServer", "", "DNS Server IP Address to query")
	cobra.OnInitialize(initConfig)
}

// Execute run application
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		// fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	// logger settings
	log.SetLevel(log.ErrorLevel)
	switch {
	case debugFlag:
		// report function name
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
	case infoFlag:
		log.SetLevel(log.InfoLevel)
	}
	logFormatter := &prefixed.TextFormatter{
		DisableColors:   noLogColorFlag,
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	}
	log.SetFormatter(logFormatter)
	if unitTestFlag {
		log.SetOutput(RootCmd.OutOrStdout())
		color.NoColor = true
	}
	if noLogColorFlag {
		color.NoColor = true
	}
	// DNS settings
	dnsConfig = netlib.NewResolver(dnsServer, dnsPort, dnsTCP)
	dnsConfig.IPv4Only = dnsIPv4Only
	if dnsTimeout > 0 {
		dnsConfig.Timeout = time.Duration(dnsTimeout) * time.Second
	}
}
