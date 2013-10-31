package report

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"fmt"
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

	fetchers = []fetcher{fetchTopQueries, fetchRecentQueries}
)

const (
	net    = "192.168.0.%"
	format = "02/01/06 15:04:05"

	sql = `SELECT DISTINCT fqdn
		   FROM hosts, queries
		   WHERE id = origin`
	sqlTop = `SELECT fqdn, COUNT(*) AS c 
			  FROM hosts, queries
			  WHERE origin IN (SELECT id FROM hosts WHERE fqdn LIKE $origin)
			  AND id = destination
			  GROUP BY fqdn
			  ORDER BY c DESC
			  LIMIT $limit`
	sqlRecent = `SELECT date, fqdn
				 FROM hosts, queries
				 WHERE origin IN (SELECT id FROM hosts WHERE fqdn LIKE $origin)
				 AND id = destination
				 ORDER BY date DESC
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

	buffersLength := (Lines * len(fetchers)) + 5
	start := time.Now()
	buffer := make([]string, buffersLength)
	origins := fetchOrigins(db)

	for _, origin := range origins {
		prebuffer := make([]string, buffersLength)
		prebuffer[0] = strings.Replace(origin, "%", "0", -1)
		max := len(prebuffer[0])
		i := 2

		for _, fetcher := range fetchers {
			queries, newMax := fetcher(db, origin)

			for _, query := range queries {
				prebuffer[i] = query

				if newMax > max {
					max = newMax
				}

				i++
			}

			prebuffer[i] = ""
			i++
		}

		for index, line := range prebuffer {
			buffer[index] = fmt.Sprintf(fmt.Sprintf("%%s%%-%ds", max+5), buffer[index], line)
		}
	}

	buffer[len(buffer)-1] = fmt.Sprintf("took %f seconds to generate", time.Now().Sub(start).Seconds())

	return strings.Join(buffer, "\n")
}

func fetchOrigins(db *sqlite3.Conn) []string {
	origins := make([]string, 1)

	for stmt, err := db.Query(sql); err == nil; err = stmt.Next() {
		row := make(sqlite3.RowMap)
		errs := stmt.Scan(row)

		if errs != nil {
			fmt.Println("Error scanning:", errs)
			return nil
		}

		origins = append(origins, row["fqdn"].(string))
	}

	sort.Sort(vector(origins))
	newOrigins := []string{net}
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

		pairs = append(pairs, []interface{}{count, row["fqdn"]})
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

		line := fmt.Sprintf("%s %s", row["date"].(time.Time).Format(format), row["fqdn"])

		if len(line) > max {
			max = len(line)
		}

		queries[index] = line
		index++
	}

	return queries, max
}
