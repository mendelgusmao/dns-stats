package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/MendelGusmao/dns-stats/collector/routers"
)

var DNSStats DNSStatsConfig

type DNSStatsConfig struct {
	DB          DBConfig
	Report      ReportConfig
	Collector   CollectorConfig
	ARP         ARPConfig
	RoutersFile string `envconfig:routers`
	Routers     map[string]string
}

type DBConfig struct {
	Driver string
	URL    string
}

type ReportConfig struct {
	Interface string
	Lines     int
	Fetchers  []string
}

type CollectorConfig struct {
	Interface       string
	StorageInterval string `envconfig:storage_interval`
	Sources         map[string]string
	ARPScanInterval string `envconfig:arpscan_interval`
}

type ARPConfig struct {
	ScanInterval string `envconfig:scan_interval`
}

func (c *DNSStatsConfig) LoadRouters() error {
	if err := jsonLoad(c.RoutersFile, &c.Routers); err != nil {
		return err
	}

	for routerName, expression := range c.Routers {
		routers.Register(routerName, expression)
	}

	return nil
}

func jsonLoad(filename string, object interface{}) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(content, object)
	if err != nil {
		return err
	}

	return nil
}

func (c *DNSStatsConfig) Defaults() {
	pwd := os.Getenv("PWD")

	if c.RoutersFile == "" {
		c.RoutersFile = pwd + "/routers.json"
	}

	if c.Report.Interface == "" {
		c.Report.Interface = ":8514"
	}

	if c.Report.Lines == 0 {
		c.Report.Lines = 25
	}

	if c.Collector.Interface == "" {
		c.Collector.Interface = ":1514"
	}

	if c.Collector.StorageInterval == "" {
		c.Collector.StorageInterval = "1m"
	}

	if c.Collector.ARPScanInterval == "" {
		c.Collector.ARPScanInterval = "5m"
	}

	if c.DB.Driver == "" {
		c.DB.Driver = "sqlite3"
	}

	if c.DB.URL == "" {
		c.DB.URL = pwd + "/dns-stats.db"
	}
}
