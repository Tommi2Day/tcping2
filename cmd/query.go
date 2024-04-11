package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"net/http"
	"regexp"
)

const (
	// API is the URL of ifconfig network info API
	API string = "https://ifconfig.is/json/"
)

var (
	QueryCmd = &cobra.Command{
		Use:          "query",
		Short:        "Query host ip information",
		Long:         ``,
		RunE:         runQuery,
		SilenceUsage: true,
	}
)

func init() {
	QueryCmd.Flags().StringVarP(&queryAddress, "address", "a", "", "ip/host to query")
	RootCmd.AddCommand(QueryCmd)
}

// IPInfo is the struct of IP information
type IPInfo struct {
	IP        string
	Continent string  `json:"Continent"`
	Country   string  `json:"Country"`
	City      string  `json:"City"`
	Latitude  float64 `json:"Latitude"`
	Longitude float64 `json:"Longitude"`
	TimeZone  string  `json:"TimeZone"`
	ASN       uint    `json:"ASN"`
	ORG       string  `json:"Organization"`
}

func runQuery(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		queryAddress = args[0]
	}
	if queryAddress == "" {
		return fmt.Errorf("please specify an address to query")
	}
	log.Debugf("query infos for %s", queryAddress)
	ips, err := dnsConfig.LookupIP(queryAddress)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		a := ip.String()
		log.Debugf("query https://ifconfig.is for %s", a)
		info, e := QueryInfo(a)
		if e != nil {
			log.Warnf("failed to query https://ifconfig.is for %s: %s", a, e)
			continue
		}
		info.IP = a
		logQuery(info)
	}
	return nil
}

// QueryInfo queries the IP information
func QueryInfo(address string) (info IPInfo, err error) {
	info = IPInfo{}
	var body []byte
	res, err := http.Get(API + address)
	if err != nil {
		match, _ := regexp.MatchString("connection reset by peer", err.Error())
		if match {
			err = fmt.Errorf("your connection was reset by magic power. You may need to set env http_proxy")
		}
		return
	}
	if res != nil {
		defer func(b io.ReadCloser) {
			_ = b.Close()
		}(res.Body)
		body, err = io.ReadAll(res.Body)
		if err != nil {
			return
		}
		err = json.Unmarshal(body, &info)
	}
	return
}

// logQuery logs the query results
func logQuery(info IPInfo) {
	v := reflect.ValueOf(info)
	names := make([]string, v.NumField())
	values := make([]string, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		names[i] = v.Type().Field(i).Name
		t := v.Field(i).Type().Name()
		switch t {
		case "string":
			values[i] = v.Field(i).Interface().(string)
		case "bool":
			values[i] = strconv.FormatBool(v.Field(i).Interface().(bool))
		case "float64":
			values[i] = fmt.Sprintf("%f", v.Field(i).Interface().(float64))
		case "uint":
			values[i] = fmt.Sprint(v.Field(i).Interface().(uint))
		}
	}
	l := getMaxNameLength(names)
	for i := 0; i < v.NumField(); i++ {
		if values[i] != "" {
			fmt.Printf("%s:    %s\n", cyan("%-*s", l, names[i]), values[i])
		}
	}
	// separate entries
	fmt.Println()
}

// getMaxNameLength returns the length of the longest field name
func getMaxNameLength(names []string) int {
	var length int
	for _, val := range names {
		if len(val) > length {
			length = len(val)
		}
	}
	return length
}
