package main

import (
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

	_ "code.google.com/p/go-sqlite/go1/sqlite3"
)

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
		config.DNSStats.Collector.Interface,
		config.DNSStats.Collector.StorageInterval,
		config.DNSStats.Collector.Sources,
	)

	r := report.New(
		db,
		config.DNSStats.Report.Interface,
		config.DNSStats.Report.Lines,
		config.DNSStats.Report.Fetchers,
	)

	if c == nil || r == nil {
		os.Exit(1)
	}

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
	db, err := gorm.Open(config.DNSStats.DB.Driver, config.DNSStats.DB.URL)
	defer db.Close()

	if err != nil {
		panic(err)
	}

	for _, err := range model.BuildDatabase(db) {
		log.Println("Error building database:", err)
	}

	return &db
}
