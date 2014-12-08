package collector

import (
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/MendelGusmao/dns-stats/collector/arp"
	"github.com/MendelGusmao/dns-stats/collector/routers"
	"github.com/MendelGusmao/dns-stats/model"
	"github.com/davecgh/go-spew/spew"
	"github.com/jinzhu/gorm"

	"github.com/ziutek/syslog"
)

type collector struct {
	dbDriver        string
	dbURL           string
	iface           string
	storeInterval   time.Duration
	arpScanInterval time.Duration
	expressions     map[string]*regexp.Regexp
	buffer          []model.Query
	bufferMtx       sync.RWMutex
	syslogServer    *syslog.Server
}

type handler struct {
	*syslog.BaseHandler
	expressions map[string]*regexp.Regexp
}

func New(dbDriver, dbURL, iface, store, arpScan string, sources map[string]string) *collector {
	if len(sources) == 0 {
		log.Println("collector.New: not enough sources configured")
		return nil
	}

	expressions := make(map[string]*regexp.Regexp)

	for address, router := range sources {
		expressions[address] = routers.Find(router)
	}

	storeInterval, err := time.ParseDuration(store)

	if err != nil {
		log.Println("collector.New: invalid value for storeInterval")
		return nil
	}

	arpScanInterval, err := time.ParseDuration(arpScan)

	if err != nil {
		log.Println("collector.New: invalid value for arpScanInterval")
		return nil
	}

	return &collector{
		dbDriver:        dbDriver,
		dbURL:           dbURL,
		iface:           iface,
		storeInterval:   storeInterval,
		arpScanInterval: arpScanInterval,
		expressions:     expressions,
		buffer:          make([]model.Query, 0),
		syslogServer:    syslog.NewServer(),
	}
}

func (c *collector) Run() {
	log.Println("collector.Run: initializing syslog collector")

	go c.storeBuffer()
	go c.arpScan()

	arp.Scan()

	c.syslogServer.AddHandler(c.handler())
	c.syslogServer.Listen(c.iface)
}

func (c *collector) SyslogServer() *syslog.Server {
	return c.syslogServer
}

func (c *collector) handler() *handler {
	h := handler{syslog.NewBaseHandler(5, func(m *syslog.Message) bool {
		return true
	}, false), c.expressions}

	go h.mainLoop(c)

	return &h
}

func (c *collector) storeBuffer() {
	for now := range time.Tick(c.storeInterval) {
		log.Println("collector.storeBuffer: ticking", now)
		c.StoreBuffer()
	}
}

func (c *collector) arpScan() {
	for now := range time.Tick(c.arpScanInterval) {
		log.Println("collector.arpScanInterval: ticking", now)
		arp.Scan()
	}
}

func (c *collector) StoreBuffer() {
	if len(c.buffer) == 0 {
		return
	}

	db, err := gorm.Open(c.dbDriver, c.dbURL)

	if err != nil {
		log.Printf("collector.StoreBuffer (opening database connection):", err)
		log.Printf("collector.StoreBuffer: %d items waiting\n", len(c.buffer))
		return
	}

	log.Printf("collector.StoreBuffer: got to store %d buffered queries\n", len(c.buffer))
	spew.Dump(c.buffer)

	c.bufferMtx.Lock()
	defer c.bufferMtx.Unlock()

	tx := db.Begin()

	if err := tx.Error; err != nil {
		log.Println("collector.StoreBuffer (opening transaction):", err)
		log.Printf("collector.StoreBuffer: %d items waiting\n", len(c.buffer))
		return
	}

	errors := false
	for _, query := range c.buffer {
		err := tx.FirstOrCreate(&query.Origin, query.Origin).Error

		if err != nil {
			log.Println("collector.StoreBuffer (origin):", err)
			return
		}

		err = tx.FirstOrCreate(&query.Destination, query.Destination).Error

		if err != nil {
			log.Println("collector.StoreBuffer (destination):", err)
			return
		}

		err = tx.Save(&query).Error

		if err != nil {
			log.Println("collector.StoreBuffer (query):", err)
			return
		}
	}

	if errors {
		tx.Rollback()

		if err := tx.Error; err != nil {
			log.Println("collector.StoreBuffer (rolling back transaction):", err)
		}

		return
	}

	tx.Commit()

	if err := tx.Error; err != nil {
		if err := tx.Error; err != nil {
			log.Println("collector.StoreBuffer (committing transaction):", err)
		}

		log.Printf("collector.StoreBuffer: %d items waiting\n", len(c.buffer))

		return
	}

	log.Printf("collector.StoreBuffer: transaction is successful, %d items inserted\n", len(c.buffer))
	c.buffer = c.buffer[:cap(c.buffer)]
}

func (h *handler) mainLoop(c *collector) {
	for {
		m := h.Get()
		if m == nil {
			break
		}

		expression, ok := c.expressions[m.Hostname]

		if !ok {
			log.Printf("collector.mainLoop: source %s is unknown\n", m.Hostname)
			continue
		}

		query, err := routers.Extract(expression, m.Content)

		if err != nil {
			log.Println("collector.mainLoop:", err)
			continue
		}

		hwAddr, err := arp.FindByIP(query.Origin.StringIP)

		if err != nil {
			log.Println("arp.FindByIP: ", err)
		}

		query.SetSource(m.Source)
		query.At = m.Time
		query.Origin.MAC = hwAddr

		c.bufferMtx.Lock()
		c.buffer = append(c.buffer, *query)
		c.bufferMtx.Unlock()
	}

	h.End()
}
