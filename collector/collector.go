package collector

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"fmt"
	"github.com/ziutek/syslog"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	sqlInsertHost = `INSERT INTO hosts (fqdn)
					  VALUES ($fqdn)`
	sqlInsertQuery = `INSERT INTO queries
					  VALUES ($date, (SELECT id FROM hosts WHERE fqdn = $origin), (SELECT id FROM hosts WHERE fqdn = $destination))`
	sqlUpdateHost = `UPDATE hosts SET fqdn = $fqdn WHERE fqdn = $fqdn2`
	regexMessage  = `(UD|TC)P (.*),.* --> .* ALLOW: Outbound access request \[DNS query for (.*)\]`
)

var (
	message       = regexp.MustCompile(regexMessage)
	cache         = make([]Query, 0)
	mtx           sync.RWMutex
	DBName        string
	CollectorPort string
	StoreInterval string
)

type handler struct {
	*syslog.BaseHandler
}

type Query struct {
	date        time.Time
	origin      string
	destination string
}

func filter(m *syslog.Message) bool {
	return strings.Contains(m.Content, "DNS")
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

		matches := message.FindStringSubmatch(m.Content)
		query := Query{
			date:        m.Time,
			origin:      matches[2],
			destination: matches[3],
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
				"$date":        query.date,
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

func insertHost(db *sqlite3.Conn, fqdn string) (errors bool) {
	args := sqlite3.NamedArgs{
		"$fqdn":  fqdn,
		"$fqdn2": fqdn,
	}

	if err := db.Exec(sqlUpdateHost, args); err != nil {
		fmt.Println("Error updating:", err, "[", args, "]")
		errors = true
	}

	if db.RowsAffected() == 0 {
		args := sqlite3.NamedArgs{
			"$fqdn": fqdn,
		}

		if err := db.Exec(sqlInsertHost, args); err != nil {
			fmt.Println("Error inserting host:", err, "[", args, "]")
			errors = true
		}
	}
	return
}

func Run() *syslog.Server {
	fmt.Println("Initializing syslog collector")

	go cacheStore()

	s := syslog.NewServer()
	s.AddHandler(newHandler())
	s.Listen(CollectorPort)

	return s
}
