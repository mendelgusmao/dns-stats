package main

import (
	"flag"
	"fmt"
	"gorm"
	"os"
	"os/signal"
	"syscall"

	"github.com/MendelGusmao/dns-stats/collector"
	"github.com/MendelGusmao/dns-stats/model"
	"github.com/MendelGusmao/dns-stats/report"

	"github.com/MendelGusmao/envconfig"
)

const (
	sql = `CREATE TABLE IF NOT EXISTS queries (at DATE, origin INTEGER, destination INTEGER);
		   CREATE TABLE IF NOT EXISTS machines (id INTEGER PRIMARY KEY, address TEXT, mac TEXT);
		   CREATE TABLE IF NOT EXISTS hosts (id INTEGER PRIMARY KEY, address TEXT UNIQUE);
		   CREATE INDEX IF NOT EXISTS address_idx ON hosts (address COLLATE NOCASE);
		   CREATE INDEX IF NOT EXISTS machines_address_idx ON hosts (address COLLATE NOCASE);
		   CREATE INDEX IF NOT EXISTS machines_mac_idx ON hosts (mac COLLATE NOCASE);`
)

var (
	// dbname        string
	// collectorPort string
	// reportPort    string
	// storeInterval string
	period       string
	stdOutReport bool
	reportLines  int
	verbose      bool
	sources      collector.SourceParameters
)

func init() {
	// flag.StringVar(&dbname, "db", os.Getenv("HOME")+"/dns.sqlite3", "Absolute path to SQLite3 database")
	// flag.StringVar(&collectorPort, "collector-port", ":1514", "Address for syslog collector to listen to")
	// flag.StringVar(&reportPort, "report-port", ":8514", "Address for report server to listen to")
	// flag.StringVar(&storeInterval, "store-interval", "1m", "Defines the interval for cached queries storage")
	// flag.StringVar(&period, "period", "720h", "Defines the report period")
	flag.BoolVar(&stdOutReport, "stdout", false, "Print report to stdout")
	flag.BoolVar(&verbose, "verbose", false, "Display received syslog messages")
	flag.IntVar(&reportLines, "lines", 25, "Number of records in report (per category)")
	flag.Var(&sources, "source", "The source and router from which the collector will receive messages (can be set multiple times)")
}

func main() {
	envconfig.Process("dns_stats", &config.DNSStats)
	buildDatabase()

	r := rep

	report.ReportPort = reportPort
	report.DBName = dbname
	report.Lines = reportLines

	if stdOutReport {
		fmt.Println(report.Render(period))
	} else {
		fmt.Printf("Configuration parameters: \n")
		fmt.Printf("  db -> %s\n", dbname)
		fmt.Printf("  sources -> %s\n", sources)
		fmt.Printf("  collector-port -> %s\n", collectorPort)
		fmt.Printf("  report-port -> %s\n", reportPort)
		fmt.Printf("  store-interval -> %s\n", storeInterval)
		fmt.Printf("  report-lines -> %d\n", reportLines)

		collector.CollectorPort = collectorPort
		collector.DBName = dbname
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

		fmt.Println("Storing...")
		collector.Store()
		fmt.Println("Shutdown the server...")
		s.Shutdown()
		fmt.Println("Server is down!")
	}
}

func buildDatabase() {
	db, err := gorm.Open(config.Ephemeris.Database.Driver, config.Ephemeris.Database.URL)
	defer db.Close()

	if err != nil {
		panic(err)
	}

	for _, err := range model.BuildDatabase(db) {
		fmt.Println("Error building database:", err)
	}
}
