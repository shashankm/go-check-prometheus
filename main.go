package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/segfaultax/go-nagios"
	"github.com/spf13/pflag"
)

var (
	showHelp   bool
	warning    string
	critical   string
	host       string
	metricName string
	query      string
	timeout    int
)

const usage string = `usage: check-prometheus [options]
The purpose of this tool is to check that the value given by a prometheus
query falls within certain warning and critical thresholds. Warning and
critical ranges can be provided in Nagios threshold format.
Example:
check-prometheus -g localhost -H 'my.host' -q 'query' -w 10 -c 100
Meaning: The sum of all non-null values returned by the Prometheus query
'my.metric' is OK if less than or equal to 10, warning if greater than
10 but less than or equal to 100, critical if greater than 100. If it's
less than zero, it's critical.
ullcnt / total points)
`

func init() {
	pflag.BoolVarP(&showHelp, "help", "h", false, "show help")
	pflag.StringVarP(&host, "host", "H", "", "prometheus host")

	pflag.StringVarP(&warning, "warning", "w", "", "warning range")
	pflag.StringVarP(&critical, "critical", "c", "", "critical range")

	pflag.StringVarP(&metricName, "name", "n", "metric", "Short, descriptive name for metric")
	pflag.StringVarP(&query, "query", "q", "", "prometheus query")

	pflag.IntVarP(&timeout, "timeout", "t", 10, "Execution timeout")
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

	client, err := api.NewClient(api.Config{
		Address: host,
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}
	check, err := nagios.NewRangeCheckParse(warning, critical)
	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}
	vec := result.(model.Vector)
	if len(result.String()) == 0 {
		check.Unknown("OK: The query did not return any result")
		return
	}
	valStr := vec[0].Value.String()
	val, _ := strconv.ParseFloat(valStr, 64)
	if err != nil {
		printUsageErrorAndExit(3, err)
	}
	defer check.Done()

	check.CheckValue(val)
	check.AddPerfData(nagios.NewPerfData(metricName, val, ""))
	check.SetMessage("%s (%s is %s)", metricName, vec[0].Metric, valStr)

}

func checkRequiredOptions() error {
	switch {
	case host == "":
		return fmt.Errorf("host is required")
	case query == "":
		return fmt.Errorf("query is required")
	case warning == "" && critical == "":
		return fmt.Errorf("must supply at least one of -w or -c")
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
