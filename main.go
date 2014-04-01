package main

import (
	"dns-stats/collector"
	"dns-stats/database"
	"dns-stats/report"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	dbDriver      string
	dbURI         string
	collectorPort string
	reportPort    string
	storeInterval string
	period        string
	stdOutReport  bool
	reportLines   int
	verbose       bool
	sources       collector.SourceParameters
)

func init() {
	flag.StringVar(&dbDriver, "db-driver", "sqlite3", "Database driver (mysql, postgres or sqlite3)")
	flag.StringVar(&dbURI, "db-uri", os.Getenv("HOME")+"/dns.sqlite3", "Database URI")
	flag.StringVar(&collectorPort, "collector-port", ":1514", "Address for syslog collector to listen to")
	flag.StringVar(&reportPort, "report-port", ":8514", "Address for report server to listen to")
	flag.StringVar(&storeInterval, "store-interval", "1m", "Defines the interval for cached queries storage")
	flag.StringVar(&period, "period", "720h", "Defines the report period")
	flag.BoolVar(&stdOutReport, "stdout", false, "Print report to stdout")
	flag.BoolVar(&verbose, "verbose", false, "Display received syslog messages")
	flag.IntVar(&reportLines, "lines", 25, "Number of records in report (per category)")
	flag.Var(&sources, "source", "The source and router from which the collector will receive messages (can be set multiple times)")
}

func main() {
	flag.Parse()

	if err := database.Init(dbDriver, dbURI); err != nil {
		fmt.Println("Error initializing database:", err)
		os.Exit(1)
	}

	report.ReportPort = reportPort
	report.Lines = reportLines

	if stdOutReport {
		fmt.Println(report.Render(period))
	} else {
		fmt.Printf("Configuration parameters: \n")
		fmt.Printf("  db -> (%s) %s\n", dbDriver, dbURI)
		fmt.Printf("  sources -> %s\n", sources)
		fmt.Printf("  collector-port -> %s\n", collectorPort)
		fmt.Printf("  report-port -> %s\n", reportPort)
		fmt.Printf("  store-interval -> %s\n", storeInterval)
		fmt.Printf("  report-lines -> %d\n", reportLines)

		collector.CollectorPort = collectorPort
		collector.StoreInterval = storeInterval
		collector.Sources = sources
		collector.Verbose = verbose

		go report.Run()
		s := collector.Run()

		if s == nil {
			os.Exit(1)
		}

		sc := make(chan os.Signal, 2)
		signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
		<-sc

		fmt.Println("Storing cached queries")
		collector.Store()
		fmt.Println("Shutting down the server")
		s.Shutdown()
		fmt.Println("Server is down!")
	}
}
