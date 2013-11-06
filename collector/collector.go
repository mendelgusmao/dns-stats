package collector

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"dns-stats/collector/routers"
	"fmt"
	"github.com/ziutek/syslog"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	sqlInsertHost = `INSERT INTO hosts (address)
					 VALUES ($address)`
	sqlInsertQuery = `INSERT INTO queries
					  VALUES ($at, (SELECT id FROM hosts WHERE address = $origin), (SELECT id FROM hosts WHERE address = $destination))`
	notUnique = "column address is not unique"
)

var (
	cache         = make([]Query, 0)
	mtx           sync.RWMutex
	DBName        string
	CollectorPort string
	StoreInterval string
	RouterName    string
	expression    *regexp.Regexp
)

type handler struct {
	*syslog.BaseHandler
}

type Query struct {
	at          time.Time
	origin      string
	destination string
}

func filter(m *syslog.Message) bool {
	return strings.Contains(m.Content, "DNS query")
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

		origin, destination := routers.Extract(expression, expression.FindStringSubmatch(m.Content))

		query := Query{
			at:          m.Time,
			origin:      origin,
			destination: destination,
		}

		fmt.Println("Received syslog: @", m.Content, "@")
		fmt.Println("Generated query: @", query, "@")

		mtx.Lock()
		cache = append(cache, query)
		mtx.Unlock()
	}
	fmt.Println("Exit handler")
	h.End()
}

func cacheStore() {
	interval, _ := time.ParseDuration(StoreInterval)

	c := time.Tick(interval)
	for now := range c {
		fmt.Println("tick:", now)
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

func Run() *syslog.Server {
	fmt.Println("Initializing syslog collector")
	expression = routers.Find(RouterName)

	if expression == nil {
		fmt.Println("Registered routers:", routers.Registered())
		return nil
	}

	go cacheStore()

	s := syslog.NewServer()
	s.AddHandler(newHandler())
	s.Listen(CollectorPort)

	return s
}
