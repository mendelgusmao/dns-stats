package collector

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm"

	"github.com/MendelGusmao/dns-stats/arp"
	"github.com/MendelGusmao/dns-stats/collector/routers"

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

var (
	Sources = make(SourceParameters, 0)
	Verbose bool
)

type Collector struct {
	DB            *gorm.DB
	Port          int
	StoreInterval int
	expressions   map[string]*regexp.Regexp
	cache         []Query
	mtx           sync.RWMutex
}

type handler struct {
	*syslog.BaseHandler
}

func filter(m *syslog.Message) bool {
	return true
}

func newHandler() *handler {
	h := handler{syslog.NewBaseHandler(5, filter, false)}
	go h.mainLoop()
	return &h
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
				fmt.Printf("Source %s is unknown\n", m.Hostname)
			}

			continue
		}

		origin, destination, err := routers.Extract(expression, expression.FindStringSubmatch(m.Content))

		if err != nil {
			if Verbose {
				fmt.Println(err)
				fmt.Println("Received syslog: @", m.Content, "@")
			}
			continue
		}

		hwAddr, err := arp.FindByIP(origin)

		if err != nil {
			fmt.Println("arp.FindByIP: ", err)
		}

		query := Query{
			source:      m.Source,
			at:          m.Time,
			origin:      origin,
			destination: destination,
		}

		if Verbose {
			fmt.Println("Received syslog: @", m.Content, "@")
			fmt.Println("Generated query: @", query, "@")
		}

		mtx.Lock()
		cache = append(cache, query)
		mtx.Unlock()
	}

	h.End()
}

func cacheStore() {
	interval, _ := time.ParseDuration(StoreInterval)

	c := time.Tick(interval)
	for now := range c {
		if Verbose {
			fmt.Println("tick:", now)
		}
		Store()
	}
}

func Store() {
	if len(cache) == 0 {
		return
	}

	mtx.Lock()
	defer mtx.Unlock()

	var db *sqlite3.Conn
	var err error
	if db, err = sqlite3.Open(DBName); err != nil {
		fmt.Println("Error opening database:", err)
		return
	}

	defer db.Close()

	if err := db.Begin(); err != nil {
		fmt.Println("Error opening transaction:\n", err)
		fmt.Printf("There are %d items waiting\n", len(cache))
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
				fmt.Println("Error inserting query:", err, "[", args, "]")
				errors = true
			}
		}

		if errors {
			if err := db.Rollback(); err != nil {
				fmt.Println("Error rolling back transaction:", err)
			}
			return
		}

		if err := db.Commit(); err != nil {
			fmt.Println("Error committing transaction:", err)
			fmt.Printf("There are %d items waiting\n", len(cache))
		} else {
			fmt.Printf("Transaction is successful, %d items inserted\n", len(cache))
			cache = make([]Query, 0)
		}
	}

}

func insertHost(db *sqlite3.Conn, address string) (errors bool) {
	args := sqlite3.NamedArgs{
		"$address": address,
	}

	if err := db.Exec(sqlInsertHost, args); err != nil && !strings.Contains(err.Error(), notUnique) {
		fmt.Println("Error inserting host:", err, "[", args, "]")
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
		fmt.Println("Error inserting host:", err, "[", args, "]")
		errors = true
	}

	return
}

func Run() *syslog.Server {
	fmt.Println("Initializing syslog collector")

	if len(Sources) == 0 {
		fmt.Println("Not enough sources configured")
		return nil
	}

	for _, router := range Sources {
		expressions[router.Host] = routers.Find(router.Router)
	}

	go cacheStore()

	s := syslog.NewServer()
	s.AddHandler(newHandler())
	s.Listen(CollectorPort)

	return s
}
