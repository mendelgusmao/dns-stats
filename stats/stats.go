package stats

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	statsPort = ":8514"
	format    = "02/01/06 15:04:05"
	form      = `
	<form action="/dns" method="post">
		<input type="text" value="%s" name="sql" size="100" />
		<input type="submit" value="sql" />
	</form>`
)

func Stats(dbname string) {
	var err error
	var db *sqlite3.Conn
	if db, err = sqlite3.Open(dbname); err != nil {
		fmt.Println("Error opening database:", err)
		return
	}

	fmt.Println("Initializing HTTP stats")

	sql1 := "SELECT destination, COUNT(*) AS c FROM queries WHERE origin LIKE $origin GROUP BY destination ORDER BY c DESC LIMIT 25"
	sql2 := "SELECT date, destination FROM queries WHERE origin LIKE $origin ORDER BY date DESC LIMIT 25"

	http.HandleFunc("/dns", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sql := "SELECT DISTINCT origin FROM queries"

		if len(r.FormValue("sql")) > 0 {
			sql = r.FormValue("sql")
		}

		buffer := make([]string, 60)
		origins := make([]string, 0)

		for s, err := db.Query(sql); err == nil; err = s.Next() {
			row := make(sqlite3.RowMap)
			errs := s.Scan(row)

			if errs != nil {
				fmt.Println("Error scanning:", errs)
				return
			}

			origins = append(origins, row["origin"].(string))
		}

		neworigins := []string{"192.168.0.%"}
		sort.Sort(vector(origins))
		neworigins = append(neworigins, origins...)
		origins = neworigins

		for _, origin := range origins {
			i := 0
			prebuffer := make([]string, 60)
			prebuffer[i] = strings.Replace(origin, "%", "0", -1)
			i++
			prebuffer[i] = ""
			i++
			max := len(origin)

			for s2, err := db.Query(sql1, origin); err == nil; err = s2.Next() {
				row := make(sqlite3.RowMap)
				errs := s2.Scan(row)

				if errs != nil {
					fmt.Println("Error scanning:", errs)
					return
				}

				count := row["c"].(int64)
				line := fmt.Sprintf("%-6d %s", count, row["destination"])
				if len(line) > max {
					max = len(line)
				}
				prebuffer[i] = line
				i++
			}

			i = 26
			prebuffer[i] = ""
			i++

			for s1, err := db.Query(sql2, origin); err == nil; err = s1.Next() {
				row := make(sqlite3.RowMap)
				errs := s1.Scan(row)

				if errs != nil {
					fmt.Println("Error scanning:", errs)
					return
				}

				line := fmt.Sprintf("%s %s", row["date"].(time.Time).Format(format), row["destination"])
				if len(line) > max {
					max = len(line)
				}
				prebuffer[i] = line
				i++
			}

			for index, line := range prebuffer {
				buffer[index] = fmt.Sprintf(fmt.Sprintf("%%s%%-%ds", max+5), buffer[index], line)
			}
		}

		buffer[len(buffer)-7] = fmt.Sprintf("%d seconds to generate", time.Now().Second()-start.Second())

		w.Header().Add("Content-Type", "text/html")
		fmt.Fprintf(w, form, html.EscapeString(sql))
		fmt.Fprintf(w, "<pre>%s</pre>", strings.Join(buffer, "\n"))
	})

	fmt.Println(http.ListenAndServe(statsPort, nil))
}

type vector []string

func (v vector) Len() int {
	return len(v)
}

func (v vector) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// default comparison.
func (v vector) Less(i, j int) bool {
	return v.value(v[i]) > v.value(v[j])
}

func (v vector) value(in string) (out int) {
	blocks := strings.Split(in, ".")
	block := blocks[len(blocks)-1]
	out, _ = strconv.Atoi(block)
	return
}
