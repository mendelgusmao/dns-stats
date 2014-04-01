package collector

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"dns-stats/collector/routers"
	"dns-stats/database"
	"fmt"
	"github.com/ziutek/syslog"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	CollectorPort string
	StoreInterval string
	Sources       = make(SourceParameters, 0)
	Verbose       bool

	expressions = make(map[string]*regexp.Regexp)
	cache       = make([]database.Query, 0)
	mtx         sync.RWMutex
)

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

		sourceAddr, originAddr, destinationAddr, err := routers.Extract(expression, expression.FindStringSubmatch(m.Content))

		if err != nil {
			if Verbose {
				fmt.Println(err)
				fmt.Println("Received syslog: '", m.Content, "'")
			}
			continue
		}

		// query := database.Query{
		// 	Source:      m.Source,
		// 	At:          m.Time,
		// 	origin:      origin,
		// 	destination: destination,
		// }

		sourceIP := net.ParseIP(sourceAddr)

		if sourceIP == nil {
			fmt.Println("Error parsing source IP", sourceAddr)
			return
		}

		originIP := net.ParseIP(originAddr)

		if originIP == nil {
			fmt.Println("Error parsing origin IP", originAddr)
			return
		}

		mac := "00:00:00:00:00:00"
		source := database.DB.SelectOne("SELECT * FROM machines WHERE mac = ? AND ip = ?", mac, sourceIP)

		if err != nil {
			fmt.Println("Error selecting source machine:", err)
		}

		origin := database.DB.SelectOne("SELECT * FROM machines WHERE mac = ? and ip = ?", mac, originIP)

		if err != nil {
			fmt.Println("Error selecting origin machine:", err)
		}

		destination := database.DB.SelectOne("SELECT * FROM hosts WHERE address = ?", destinationAddr)

		if err != nil {
			fmt.Println("Error selecting host:", err)
		}

		query := database.Query{
			At:          m.Time,
			Source:      machine,
			SourceIP:    net.IP,
			Origin:      database.Host,
			Destination: database.Host,
		}

		if Verbose {
			fmt.Println("Received syslog: '", m.Content, "'")
			fmt.Println("Generated query: '", query, "'")
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
