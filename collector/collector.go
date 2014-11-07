package collector

import (
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/MendelGusmao/dns-stats/collector/arp"
	"github.com/MendelGusmao/dns-stats/collector/routers"
	"github.com/MendelGusmao/dns-stats/model"
	"github.com/jinzhu/gorm"

	"github.com/ziutek/syslog"
)

type collector struct {
	db            *gorm.DB
	iface         string
	storeInterval string
	expressions   map[string]*regexp.Regexp
	buffer        []model.Query
	bufferMtx     sync.RWMutex
	syslogServer  *syslog.Server
}

type handler struct {
	*syslog.BaseHandler
	expressions map[string]*regexp.Regexp
}

func New(db *gorm.DB, iface, storeInterval string, sources map[string]string) *collector {
	if len(sources) == 0 {
		log.Println("collector.New: not enough sources configured")
		return nil
	}

	expressions := make(map[string]*regexp.Regexp)

	for address, router := range sources {
		expressions[address] = routers.Find(router)
	}

	return &collector{
		db:            db,
		iface:         iface,
		storeInterval: storeInterval,
		expressions:   expressions,
		buffer:        make([]model.Query, 0),
		syslogServer:  syslog.NewServer(),
	}
}

func (c *collector) Run() {
	log.Println("collector.Run: initializing syslog collector")

	go c.storeBuffer()

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
	interval, _ := time.ParseDuration(c.storeInterval)

	for now := range time.Tick(interval) {
		log.Println("collector.storeBuffer: ticking", now)
		c.StoreBuffer()
	}
}

func (c *collector) StoreBuffer() {
	if len(c.buffer) == 0 {
		return
	}

	c.bufferMtx.Lock()
	defer c.bufferMtx.Unlock()

	tx := c.db.Begin()

	if err := c.db.Error; err != nil {
		log.Println("collector.StoreBuffer:", err)
		log.Printf("collector.StoreBuffer: %d items waiting\n", len(c.buffer))
		return
	}

	errors := false
	for _, query := range c.buffer {
		tx.FirstOrCreate(&query.Origin, query.Origin)

		if err := tx.Error; err != nil {
			log.Println("collector.StoreBuffer (origin):", err)
			return
		}

		tx.FirstOrCreate(&query.Destination, query.Destination)

		if err := tx.Error; err != nil {
			log.Println("collector.StoreBuffer (destination):", err)
			return
		}

		tx.Save(&query)

		if err := tx.Error; err != nil {
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
