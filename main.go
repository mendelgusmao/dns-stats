package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MendelGusmao/dns-stats/collector"
	"github.com/MendelGusmao/dns-stats/config"
	"github.com/MendelGusmao/dns-stats/model"
	"github.com/MendelGusmao/dns-stats/report"
	"github.com/MendelGusmao/envconfig"
	"github.com/jinzhu/gorm"
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
	config.DNSStats.Defaults()
	db := initDatabase()

	if err := config.DNSStats.LoadRouters(); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	c := collector.New(
		db,
		config.DNSStats.Collector.Port,
		config.DNSStats.Collector.StorageInterval,
		config.DNSStats.Collector.ParsedSources(),
	)

	r := report.New(
		db,
		config.DNSStats.Collector.Port,
		config.DNSStats.Collector.Lines,
	)

	go r.Run()
	go c.Run()

	sc := make(chan os.Signal, 2)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	<-sc

	log.Println("Storing buffered queries")
	c.StoreBuffer()
	log.Println("Shutting down syslog server")
	c.SyslogServer().Shutdown()
	log.Println("Exiting")
}

func initDatabase() *gorm.DB {
	db, err := gorm.Open(config.Ephemeris.Database.Driver, config.Ephemeris.Database.URL)
	defer db.Close()

	if err != nil {
		panic(err)
	}

	for _, err := range model.BuildDatabase(db) {
		log.Println("Error building database:", err)
	}

	return db
}
