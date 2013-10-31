package main

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"dns-stats/report"
	"flag"
	"fmt"
	"github.com/ziutek/syslog"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	dbname     string
	syslogPort string
	reportPort string

	message = regexp.MustCompile(`(UD|TC)P (.*),.* --> .* ALLOW: Outbound access request \[DNS query for (.*)\]`)
	cache   = make([]Query, 0)
	mtx     sync.RWMutex
)

func init() {
	flag.StringVar(&dbname, "db", os.Getenv("HOME")+"/dns.sqlite3", "Absolute path to SQLite3 database")
	flag.StringVar(&syslogPort, "syslog-port", ":1514", "Address for syslog collector to listen to")
	flag.StringVar(&reportPort, "report-port", ":8514", "Address for report server to listen to")
}

type handler struct {
	// To simplify implementation of our handler we embed helper
	// syslog.BaseHandler struct.
	*syslog.BaseHandler
}

type Query struct {
	Date        time.Time
	Origin      string
	Destination string
}

// Simple fiter for named/bind messages which can be used with BaseHandler
func filter(m *syslog.Message) bool {
	return strings.Contains(m.Content, "DNS")
}

func newHandler() *handler {
	h := handler{syslog.NewBaseHandler(5, filter, false)}
	go h.mainLoop() // BaseHandler needs some gorutine that reads from its queue
	return &h
}

// mainLoop reads from BaseHandler queue using h.Get and logs messages to stdout
func (h *handler) mainLoop() {
	for {
		m := h.Get()
		if m == nil {
			break
		}

		matches := message.FindStringSubmatch(m.Content)
		query := Query{
			Date:        m.Time,
			Origin:      matches[2],
			Destination: matches[3],
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
	interval, _ := time.ParseDuration("1m")

	c := time.Tick(interval)
	for now := range c {
		fmt.Println("tick:", now)
		store()
	}
}

func store() {
	if len(cache) == 0 {
		return
	}

	mtx.Lock()
	defer mtx.Unlock()

	var db *sqlite3.Conn
	var err error
	if db, err = sqlite3.Open(dbname); err != nil {
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
			errors = insertHost(db, query.Destination)
			errors = insertHost(db, query.Origin)

			args := sqlite3.NamedArgs{
				"$date":        query.Date,
				"$origin":      query.Origin,
				"$destination": query.Destination,
			}

			if err := db.Exec("INSERT INTO queries VALUES ($date, (SELECT id FROM hosts WHERE fqdn = $origin), (SELECT id FROM hosts WHERE fqdn = $destination))", args); err != nil {
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

	if err := db.Exec("UPDATE hosts SET fqdn = $fqdn WHERE fqdn = $fqdn2", args); err != nil {
		fmt.Println("Error updating:", err, "[", args, "]")
		errors = true
	}

	if db.RowsAffected() == 0 {
		args := sqlite3.NamedArgs{
			"$fqdn": fqdn,
		}

		if err := db.Exec("INSERT INTO hosts (fqdn) VALUES ($fqdn)", args); err != nil {
			fmt.Println("Error inserting host:", err, "[", args, "]")
			errors = true
		}
	}
	return
}

func main() {
	flag.Parse()

	var db *sqlite3.Conn
	var err error
	if db, err = sqlite3.Open(dbname); err != nil {
		fmt.Println("Error opening database:", err)
		return
	}

	if err = db.Exec("CREATE TABLE IF NOT EXISTS queries (date DATE, origin INTEGER, destination INTEGER)"); err != nil {
		fmt.Println("Error creating table:", err)
		return
	}

	if err = db.Exec("CREATE TABLE IF NOT EXISTS hosts (id INTEGER PRIMARY KEY, fqdn TEXT)"); err != nil {
		fmt.Println("Error creating table:", err)
		return
	}
	db.Close()

	fmt.Println("Initializing DNS listener")

	// Create a server with one handler and run one listen gorutine
	s := syslog.NewServer()
	s.AddHandler(newHandler())
	s.Listen(syslogPort)

	go report.Run(dbname, reportPort)
	go cacheStore()

	// Wait for terminating signal
	sc := make(chan os.Signal, 2)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	<-sc

	fmt.Println("Storing...")
	store()
	fmt.Println("Shutdown the server...")
	s.Shutdown()
	fmt.Println("Server is down")
}
