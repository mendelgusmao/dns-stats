package main

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"dns-stats/stats"
	"fmt"
	"github.com/ziutek/syslog"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
)

var (
	dbname     = "/home/mendel/dns.sqlite3"
	db         *sqlite3.Conn
	message    = regexp.MustCompile(`(UD|TC)P (.*),.* --> .*,53 ALLOW: Outbound access request \[DNS query for (.*)\]`)
	syslogPort = ":1514"
)

type handler struct {
	// To simplify implementation of our handler we embed helper
	// syslog.BaseHandler struct.
	*syslog.BaseHandler
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

		args := sqlite3.NamedArgs{
			"$date":        m.Time,
			"$origin":      matches[2],
			"$destination": matches[3],
		}

		if err := db.Exec("INSERT INTO queries VALUES($date, $origin, $destination)", args); err != nil {
			fmt.Println("Error inserting:", err)
		}
	}
	fmt.Println("Exit handler")
	h.End()
}

func main() {
	var err error
	if db, err = sqlite3.Open(dbname); err != nil {
		fmt.Println("Error opening database:", err)
		return
	}

	if err := db.Exec("CREATE TABLE IF NOT EXISTS queries (date DATE, origin TEXT, destination TEXT)"); err != nil {
		fmt.Println("Error creating table:", err)
		return
	}

	fmt.Println("Initializing DNS listener")

	// Create a server with one handler and run one listen gorutine
	s := syslog.NewServer()
	s.AddHandler(newHandler())
	s.Listen(syslogPort)

	go stats.Stats(dbname)

	// Wait for terminating signal
	sc := make(chan os.Signal, 2)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	<-sc

	// Shutdown the server
	fmt.Println("Shutdown the server...")
	s.Shutdown()
	fmt.Println("Server is down")
}
