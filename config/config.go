package config

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
	Port int
}

type CollectorConfig struct {
	Port            int
	StorageInterval int `envconfig:storage_interval`
}

type ARPConfig struct {
	ScanInterval int `envconfig:scan_interval`
}

func (c *DNSStatsConfig) LoadRouters() {

}
