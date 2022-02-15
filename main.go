package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/segfaultax/go-nagios"
	"github.com/spf13/pflag"
)

var (
	showHelp    bool
	warning     string
	critical    string
	host        string
	metricName  string
	query       string
	emptyResult string
	timeout     int
)

const usage string = `usage: go-check-prometheus [options]
The purpose of this tool is to check that the value given by a prometheus
query falls within certain warning and critical thresholds. Warning and
critical ranges can be provided in Nagios threshold format.
Example:
go-check-prometheus -H 'my.host' -q 'query' -w 10 -c 100
Meaning: The sum of all non-null values returned by the Prometheus query
'query' is OK if less than or equal to 10, warning if greater than
10 but less than or equal to 100, critical if greater than 100. If it's
less than zero, it's critical.
ullcnt / total points)
`

func init() {
	pflag.BoolVarP(&showHelp, "help", "h", false, "show help")
	pflag.StringVarP(&host, "host", "H", "", "prometheus host")

	pflag.StringVarP(&warning, "warning", "w", "", "warning range")
	pflag.StringVarP(&critical, "critical", "c", "", "critical range")
	pflag.StringVarP(&emptyResult, "empty", "e", "unknown", "exit status if query returns empty result. Can be one of ok, crit, warn or unknown")

	pflag.StringVarP(&metricName, "name", "n", "metric", "Short, descriptive name for metric")
	pflag.StringVarP(&query, "query", "q", "", "prometheus query")

	pflag.IntVarP(&timeout, "timeout", "t", 30, "Execution timeout")
}

func main() {
	pflag.Parse()

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	err := checkRequiredOptions()
	if err != nil {
		printUsageErrorAndExit(3, err)
	}

	if !(strings.HasPrefix(host, "https://") || strings.HasPrefix(host, "http://")) {
		host = "http://" + host
	}

	validStatus := map[string]bool{
		"ok":      true,
		"crit":    true,
		"unknown": true,
		"warn":    true,
	}

	emptyResult = strings.ToLower(emptyResult)

	if !validStatus[emptyResult] {
		invalidStatus := fmt.Errorf("empty needs to be one of ok, crit, warn or unknown")
		printUsageErrorAndExit(3, invalidStatus)
	}

	timeout_duration := time.Duration(timeout)

	client, err := api.NewClient(api.Config{
		Address: host,
		RoundTripper: (&http.Transport{
			DialContext: (&net.Dialer{
				Timeout: timeout_duration*time.Second,
				KeepAlive: timeout_duration*time.Second,
			}).DialContext,
			TLSHandshakeTimeout: timeout_duration*time.Second,
		}),
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	c, err := nagios.NewRangeCheckParse(warning, critical)

	if err != nil {
		printUsageErrorAndExit(3, err)
	}

	defer c.Done()

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), timeout_duration*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		c.Unknown("Error querying Prometheus: %v", err)
		return
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}

	if len(result.String()) == 0 {
		switch emptyResult {
		case "ok":
			c.Status = nagios.StatusOK
		case "crit":
			c.Status = nagios.StatusCrit
		case "warn":
			c.Status = nagios.StatusWarn
		default:
			c.Status = nagios.StatusUnknown
		}
		c.SetMessage("The query did not return any result")
		return
	}

	runCheck(c, result)
}

func checkRequiredOptions() error {
	switch {
	case host == "":
		return fmt.Errorf("host is required")
	case query == "":
		return fmt.Errorf("query is required")
	case warning == "":
		return fmt.Errorf("warning is required")
	case critical == "":
		return fmt.Errorf("critical is required")
	}
	return nil
}

func printUsageErrorAndExit(code int, err error) {
	fmt.Printf("execution failed: %s\n", err)
	printUsage()
	os.Exit(code)
}

func printUsage() {
	fmt.Println(usage)
	pflag.PrintDefaults()
}
