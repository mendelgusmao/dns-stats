package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/MendelGusmao/dns-stats/collector/routers"
)

var DNSStats DNSStatsConfig

type DNSStatsConfig struct {
	DB          DBConfig
	Report      ReportConfig
	Collector   CollectorConfig
	ARP         ARPConfig
	RoutersFile string
	Routers     map[string]string
}

type DBConfig struct {
	Driver string
	URL    string
}

type ReportConfig struct {
	Port     int
	Lines    int
	Fetchers []string
}

type CollectorConfig struct {
	Port            int
	StorageInterval string `envconfig:storage_interval`
	Sources         string
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

func (c *CollectorConfig) ParsedSources() map[string]string {
	sources := make(map[string]string)

	for _, source := range strings.Split(c.Sources, " ") {
		source := strings.Split(source, ":")
		sources[source[0]] = source[1]
	}

	return sources
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

	if c.Report.Port == 0 {
		c.Report.Port = 8514
	}

	if c.Collector.Port == 0 {
		c.Collector.Port = 1514
	}

	if c.DB.Driver == "" {
		c.DB.Driver = "sqlite"
	}

	if c.DB.URL == "" {
		c.DB.URL = pwd + "/dns-stats.db"
	}
}
