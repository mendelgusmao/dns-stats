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
	port          int
	storeInterval int
	expressions   map[string]*regexp.Regexp
	buffer        []model.Query
	bufferMtx     sync.RWMutex
	syslogServer  *syslog.Server
}

type handler struct {
	*syslog.BaseHandler
	expressions map[string]*regexp.Regexp
}

func New(db *gorm.DB, port, storeInterval int, sources map[string]string) *collector {
	if len(sources) == 0 {
		log.Println("collector.Run: not enough sources configured")
		return nil
	}

	expressions := make(map[string]*regexp.Regexp)

	for address, router := range sources {
		expressions[address] = routers.Find(router)
	}

	return &collector{
		db:            db,
		port:          port,
		storeInterval: storeInterval,
		expressions:   expressions,
		buffer:        make([]model.Query, 0),
		syslogServer:  syslog.NewServer(),
	}
}

func (c *collector) Run() {
	log.Println("collector.Run: initializing syslog collector")

	go storeBuffer()

	s.SyslogServer.AddHandler(c.handler())
	s.SyslogServer.Listen(c.port)
}

func (c *collector) SyslogServer() *syslog.Server {
	return c.syslogServer
}

func (c *collector) handler() *handler {
	h := handler{syslog.NewBaseHandler(5, func(m *syslog.Message) bool {
		return true
	}, false), c.expressions}

	go h.mainLoop()

	return &h
}

func (c *collector) storeBuffer() {
	interval, _ := time.ParseDuration(c.storeInterval)

	for now := range time.Tick(interval) {
		log.Println("collector.storeBuffer: ticking", now)
		Store()
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
		log.Printf("collector.StoreBuffer: %d items waiting\n", len(buffer))
		return
	}

	errors := false
	for _, query := range buffer {
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

		log.Printf("collector.StoreBuffer: %d items waiting\n", len(buffer))

		return
	}

	log.Printf("collector.StoreBuffer: transaction is successful, %d items inserted\n", len(buffer))
	buffer = make([]Query, 0)
}

func (h *handler) mainLoop() {
	for {
		m := h.Get()
		if m == nil {
			break
		}

		expression, ok := expressions[m.Hostname]

		if !ok {
			log.Printf("collector.mainLoop: source %s is unknown\n", m.Hostname)
			continue
		}

		origin, destination, err := routers.Extract(expression, expression.FindStringSubmatch(m.Content))

		if err != nil {
			log.Println("collector.mainLoop:", err)
			continue
		}

		hwAddr, err := arp.FindByIP(origin)

		if err != nil {
			log.Println("arp.FindByIP: ", err)
		}

		query := model.Query{
			Source:      m.Source,
			Origin:      model.Machine{MAC: hwAddr}.SetIP(origin),
			Destination: model.Host{Address: destination},
			At:          m.Time,
		}

		c.bufferMtx.Lock()
		buffer = append(buffer, query)
		c.bufferMtx.Unlock()
	}

	h.End()
}
