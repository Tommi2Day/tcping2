// Package cmd commands
package cmd

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "version print version string",
		Long:  ``,
		Run: func(_ *cobra.Command, _ []string) {
			v := GetVersion(true)
			log.Debugf("Version: %s", v)
		},
	}
)

// Version, Build Commit and Date are filled in during build by the Makefile
// noinspection GoUnusedGlobalVariable
var (
	Name    = configName
	Version = "- not set -"
	Commit  = "snapshot"
	Date    = ""
)

func init() {
	RootCmd.AddCommand(versionCmd)
}

// GetVersion extract compiled version info
func GetVersion(doPrint bool) (txt string) {
	name := Name
	commit := Commit
	version := Version
	date := Date
	if date == "" {
		currentTime := time.Now()
		date = currentTime.Format("2006-01-02 15:04:05")
	}
	txt = fmt.Sprintf("%s version %s (%s - %s)", name, version, commit, date)
	if doPrint {
		fmt.Println(txt)
	}
	return
}
