package collector

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm"

	"github.com/MendelGusmao/dns-stats/collector/arp"
	"github.com/MendelGusmao/dns-stats/collector/routers"
	"github.com/MendelGusmao/dns-stats/model"

	"code.google.com/p/go-sqlite/go1/sqlite3"
	"github.com/ziutek/syslog"
)

const (
	sqlInsertHost = `INSERT INTO hosts (address)
					 VALUES ($address)`
	sqlInsertMachine = `INSERT INTO machines (address, mac)
					    VALUES ($address, $mac)`
	sqlInsertQuery = `INSERT INTO queries
					  VALUES (
					  	$at, 
					  	(SELECT id FROM machines WHERE address = $origin AND mac = $origin_mac), 
					  	(SELECT id FROM hosts WHERE address = $destination)
				  	  )`
	notUnique = "column address is not unique"
)

type collector struct {
	db            *gorm.DB
	port          int
	storeInterval int
	expressions   map[string]*regexp.Regexp
	cache         []model.Query
	cacheMtx      sync.RWMutex
}

type handler struct {
	*syslog.BaseHandler
	expressions map[string]*regexp.Regexp
}

func New(db *gorm.DB, port, storeInterval int, sources map[string]string) *collector {
	if len(sources) == 0 {
		log.Println("Not enough sources configured")
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
		cache:         make([]model.Query, 0),
	}
}

func (c *collector) Run() {
	log.Println("Initializing syslog collector")

	go cacheStore()

	s := syslog.NewServer()
	s.AddHandler(c.handler())
	s.Listen(c.port)
}

func (c *collector) handler() *handler {
	h := handler{syslog.NewBaseHandler(5, func(m *syslog.Message) bool {
		return true
	}, false), c.expressions}
	go h.mainLoop()
	return &h
}

func (c *collector) cacheStore() {
	interval, _ := time.ParseDuration(c.storeInterval)

	ch := time.Tick(interval)
	for now := range ch {
		log.Println("cacheStore ticking", now)
		Store()
	}
}

func (c *collector) Store() {
	if len(cache) == 0 {
		return
	}

	cacheMtx.Lock()
	defer cacheMtx.Unlock()

	var db *sqlite3.Conn
	var err error
	if db, err = sqlite3.Open(DBName); err != nil {
		log.Println("Error opening database:", err)
		return
	}

	defer db.Close()

	if err := db.Begin(); err != nil {
		log.Println("Error opening transaction:\n", err)
		log.Printf("There are %d items waiting\n", len(cache))
	} else {
		errors := false
		for _, query := range cache {
			errors = insertHost(db, query.destination)
			errors = insertHost(db, query.origin)

			args := sqlite3.NamedArgs{
				"$at":          query.at,
				"$origin":      query.origin,
				"$mac":         query.mac,
				"$destination": query.destination,
			}

			if err := db.Exec(sqlInsertQuery, args); err != nil {
				log.Println("Error inserting query:", err, "[", args, "]")
				errors = true
			}
		}

		if errors {
			if err := db.Rollback(); err != nil {
				log.Println("Error rolling back transaction:", err)
			}
			return
		}

		if err := db.Commit(); err != nil {
			log.Println("Error committing transaction:", err)
			log.Printf("There are %d items waiting\n", len(cache))
		} else {
			log.Printf("Transaction is successful, %d items inserted\n", len(cache))
			cache = make([]Query, 0)
		}
	}

}

func insertHost(db *sqlite3.Conn, address string) (errors bool) {
	args := sqlite3.NamedArgs{
		"$address": address,
	}

	if err := db.Exec(sqlInsertHost, args); err != nil && !strings.Contains(err.Error(), notUnique) {
		log.Println("Error inserting host:", err, "[", args, "]")
		errors = true
	}

	return
}

func insertMachine(db *sqlite3.Conn, address, mac string) (errors bool) {
	args := sqlite3.NamedArgs{
		"$address": address,
		"$mac":     mac,
	}

	if err := db.Exec(sqlInsertMachine, args); err != nil && !strings.Contains(err.Error(), notUnique) {
		log.Println("Error inserting host:", err, "[", args, "]")
		errors = true
	}

	return
}

func (h *handler) mainLoop() {
	for {
		m := h.Get()
		if m == nil {
			break
		}

		expression, ok := expressions[m.Hostname]

		if !ok {
			if Verbose {
				log.Printf("Source %s is unknown\n", m.Hostname)
			}

			continue
		}

		origin, destination, err := routers.Extract(expression, expression.FindStringSubmatch(m.Content))

		if err != nil {
			if Verbose {
				log.Println(err)
				log.Println("Received syslog: @", m.Content, "@")
			}
			continue
		}

		hwAddr, err := arp.FindByIP(origin)

		if err != nil {
			log.Println("arp.FindByIP: ", err)
		}

		query := Query{
			source:      m.Source,
			at:          m.Time,
			origin:      origin,
			destination: destination,
		}

		if Verbose {
			log.Println("Received syslog: @", m.Content, "@")
			log.Println("Generated query: @", query, "@")
		}

		cacheMtx.Lock()
		cache = append(cache, query)
		cacheMtx.Unlock()
	}

	h.End()
}
