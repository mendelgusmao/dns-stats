package report

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type fetcher func(*sqlite3.Conn, string) ([]string, int)

var (
	DBName     string
	ReportPort string
	Lines      int

	fetchers    = []fetcher{fetchTopQueries, fetchRecentQueries}
	cachedHosts = make(map[string]string)
)

const (
	network = "192.168.0.%"
	format  = "02/01/06 15:04:05"

	sql = `SELECT DISTINCT address
		   FROM hosts, queries
		   WHERE id = origin`
	sqlTop = `SELECT address, COUNT(*) AS c 
			  FROM hosts, queries
			  WHERE origin IN (SELECT id FROM hosts WHERE address LIKE $origin)
			  AND id = destination
			  GROUP BY address
			  ORDER BY c DESC
			  LIMIT $limit`
	sqlRecent = `SELECT at, address
				 FROM hosts, queries
				 WHERE origin IN (SELECT id FROM hosts WHERE address LIKE $origin)
				 AND id = destination
				 ORDER BY at DESC
				 LIMIT $limit`
)

func Run() {
	fmt.Println("Initializing HTTP stats")

	http.HandleFunc("/dns", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		fmt.Fprintln(w, Render())
	})

	fmt.Println(http.ListenAndServe(ReportPort, nil))
}

func Render() string {
	var err error
	var db *sqlite3.Conn
	if db, err = sqlite3.Open(DBName); err != nil {
		fmt.Println("Error opening database:", err)
		return ""
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Println("Error closing database:", err)
		}
	}()

	buffersLength := Lines*len(fetchers) + 2*len(fetchers) + 1
	start := time.Now()
	buffer := make([]string, buffersLength)
	origins := fetchOrigins(db)

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

		for _, fetcher := range fetchers {
			queries, newMax := fetcher(db, origin)

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

	buffer[len(buffer)-1] = fmt.Sprintf("took %f seconds to generate", time.Now().Sub(start).Seconds())

	return strings.Join(buffer, "\n")
}

func fetchOrigins(db *sqlite3.Conn) []string {
	origins := make([]string, 0)

	for stmt, err := db.Query(sql); err == nil; err = stmt.Next() {
		row := make(sqlite3.RowMap)
		errs := stmt.Scan(row)

		if errs != nil {
			fmt.Println("Error scanning:", errs)
			return nil
		}

		origins = append(origins, row["address"].(string))
	}

	sort.Sort(vector(origins))
	newOrigins := []string{network}
	newOrigins = append(newOrigins, origins...)

	return newOrigins
}

func fetchTopQueries(db *sqlite3.Conn, origin string) ([]string, int) {
	queries := make([]string, Lines)
	max := 0
	pairs := make([][]interface{}, 0)
	maxCount := 0

	for stmt, err := db.Query(sqlTop, origin, Lines); err == nil; err = stmt.Next() {
		row := make(sqlite3.RowMap)
		errs := stmt.Scan(row)

		if errs != nil {
			fmt.Println("Error scanning:", errs)
			return queries, 0
		}

		count := row["c"].(int64)
		lenCount := len(strconv.FormatInt(count, 10))

		if lenCount > maxCount {
			maxCount = lenCount
		}

		pairs = append(pairs, []interface{}{count, row["address"]})
	}

	format := fmt.Sprintf("%%-%dd %%s", maxCount+1)

	for index, pair := range pairs {
		line := fmt.Sprintf(format, pair[0], pair[1])

		if len(line) > max {
			max = len(line)
		}

		queries[index] = line
	}

	return queries, max
}

func fetchRecentQueries(db *sqlite3.Conn, origin string) ([]string, int) {
	queries := make([]string, Lines)
	max := 0
	index := 0

	for stmt, err := db.Query(sqlRecent, origin, Lines); err == nil; err = stmt.Next() {
		row := make(sqlite3.RowMap)
		errs := stmt.Scan(row)

		if errs != nil {
			fmt.Println("Error scanning:", errs)
			return queries, 0
		}

		line := fmt.Sprintf("%s %s", row["at"].(time.Time).Format(format), row["address"])

		if len(line) > max {
			max = len(line)
		}

		queries[index] = line
		index++
	}

	return queries, max
}
