package report

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/MendelGusmao/dns-stats/report/fetchers"
	"github.com/jinzhu/gorm"
)

var (
	cachedHosts = make(map[string]string)
)

const (
	network = "192.168.0.%"
	sql     = `SELECT DISTINCT address
			   FROM hosts, queries
			   WHERE at >= $from
			   AND id = origin`
	format = "02/01/06 15:04:05"
)

type report struct {
	db       *gorm.DB
	iface    string
	lines    int
	fetchers []fetchers.Fetcher
}

func New(db *gorm.DB, iface string, lines int, fetcherNames []string) *report {
	enabledFetchers := make([]fetchers.Fetcher, len(fetcherNames))

	for _, fetcherName := range fetcherNames {
		if fetcher := fetchers.Find(fetcherName); fetcher != nil {
			enabledFetchers = append(enabledFetchers, fetcher)
		}
	}

	return &report{
		db:       db,
		iface:    iface,
		lines:    lines,
		fetchers: enabledFetchers,
	}
}

func (r *report) Run() {
	log.Println("report.Run: initializing HTTP daemon")

	http.HandleFunc("/dns", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain")

		if len(req.URL.RawQuery) == 0 {
			w.Header().Add("Location", "/dns?day")
			w.WriteHeader(http.StatusMovedPermanently)
		} else {
			fmt.Fprintln(w, r.Render(req.URL.RawQuery))
		}
	})

	log.Println(http.ListenAndServe(r.iface, nil))
}

func (r *report) Render(period string) string {
	from := defineFrom(period)

	buffersLength := r.lines*len(r.fetchers) + 2*len(r.fetchers) + 1
	start := time.Now()
	buffer := make([]string, buffersLength)
	origins := r.fetchOrigins(from.Unix())

	for _, origin := range origins {
		prebuffer := make([]string, buffersLength)
		prebuffer[0] = strings.Replace(origin, "%", "0", -1)
		originAddr := prebuffer[0]

		var hostName string
		var ok bool
		if hostName, ok = cachedHosts[originAddr]; !ok {
			if hosts, err := net.LookupAddr(originAddr); err == nil {
				hostName = hosts[0]
			}

			cachedHosts[originAddr] = hostName
		}

		if len(hostName) > 0 {
			prebuffer[0] = fmt.Sprintf("%s (%s)", originAddr, hostName)
		}

		max := len(prebuffer[0])
		i := 2

		for _, fetcher := range r.fetchers {
			queries, newMax := fetcher.Fetch(r.db, origin, from.Unix(), r.lines)

			if newMax > max {
				max = newMax
			}

			for _, query := range queries {
				prebuffer[i] = query

				i++
			}

			prebuffer[i] = ""
			i++
		}

		format := fmt.Sprintf("%%s%%-%ds", max+5)

		for index, line := range prebuffer {
			buffer[index] = fmt.Sprintf(format, buffer[index], line)
		}
	}

	buffer[len(buffer)-1] = fmt.Sprintf(
		"%s ~ %s // took %f seconds to generate",
		from.Format(format),
		time.Now().Format(format),
		time.Now().Sub(start).Seconds())

	return strings.Join(buffer, "\n")
}

func (r *report) fetchOrigins(from int64) []string {
	origins := make([]string, 0)

	rows, err := r.db.Raw(sql, from).Rows()

	if err != nil {
		fmt.Println("report.fetchOrigins (querying):", err)
		return origins
	}

	for rows.Next() {
		row := make(map[string]interface{})
		errs := rows.Scan(row)

		if errs != nil {
			log.Println("report.fetchOrigins (scanning):", errs)
			return nil
		}

		origins = append(origins, row["address"].(string))
	}

	sort.Sort(vector(origins))
	newOrigins := []string{network}
	newOrigins = append(newOrigins, origins...)

	return newOrigins
}

func defineFrom(period string) (from time.Time) {
	now := time.Now()

	switch period {
	case "day":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return
	case "week":
		from = time.Date(now.Year(), now.Month(), now.Day()-int(now.Weekday()), 0, 0, 0, 0, now.Location())
		return
	case "month":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return
	case "year":
		from = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
		return
	}

	duration, err := time.ParseDuration(period)
	if err != nil {
		log.Printf("report.defineFrom: period '%s' is invalid. Using default: 24h\n", period)
		duration, _ = time.ParseDuration("24h")
	}

	from = time.Now().Add(-duration)
	return
}
