package main

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"dns-stats/collector"
	"dns-stats/report"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const (
	sql = `CREATE TABLE IF NOT EXISTS queries (at DATE, origin INTEGER, destination INTEGER);
		   CREATE TABLE IF NOT EXISTS hosts (id INTEGER PRIMARY KEY, address TEXT UNIQUE);
		   CREATE INDEX IF NOT EXISTS address_idx ON hosts (address COLLATE NOCASE);`
)

var (
	dbname        string
	collectorPort string
	reportPort    string
	storeInterval string
	stdOutReport  bool
	reportLines   int
	routers       collector.Sources
)

func init() {
	flag.StringVar(&dbname, "db", os.Getenv("HOME")+"/dns.sqlite3", "Absolute path to SQLite3 database")
	flag.StringVar(&collectorPort, "collector-port", ":1514", "Address for syslog collector to listen to")
	flag.StringVar(&reportPort, "report-port", ":8514", "Address for report server to listen to")
	flag.StringVar(&storeInterval, "store-interval", "1m", "Defines the interval for cached queries storage")
	flag.BoolVar(&stdOutReport, "stdout", false, "Print report to stdout")
	flag.IntVar(&reportLines, "lines", 25, "Number of records in report (per category)")
	flag.Var(&routers, "source", "The source and router from which the collector will receive messages (can be set multiple times)")
}

func main() {
	flag.Parse()

	setupDB()

	report.ReportPort = reportPort
	report.DBName = dbname
	report.Lines = reportLines

	if stdOutReport {
		fmt.Println(report.Render())
	} else {
		fmt.Printf("Configuration parameters: \n")
		fmt.Printf("  db -> %s\n", dbname)
		fmt.Printf("  sources -> %s\n", routers)
		fmt.Printf("  collector-port -> %s\n", collectorPort)
		fmt.Printf("  report-port -> %s\n", reportPort)
		fmt.Printf("  store-interval -> %s\n", storeInterval)
		fmt.Printf("  report-lines -> %d\n", reportLines)

		collector.CollectorPort = collectorPort
		collector.DBName = dbname
		collector.StoreInterval = storeInterval
		collector.Routers = routers

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

func setupDB() {
	var db *sqlite3.Conn
	var err error
	if db, err = sqlite3.Open(dbname); err != nil {
		fmt.Println("Error opening database:", err)
		return
	}

	if err = db.Exec(sql); err != nil {
		fmt.Println("Error setting up database:", err)
		return
	}

	db.Close()
}
