package fetchers

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
	"fmt"
	"strconv"
)

type Top struct{}

func (_ Top) sql() string {
	return `SELECT address, COUNT(*) AS c 
			FROM hosts, queries
			WHERE origin IN (SELECT id FROM hosts WHERE address LIKE $origin)
			AND id = destination
			GROUP BY address
			ORDER BY c DESC
			LIMIT $limit`
}

func (t Top) Fetch(db *sqlite3.Conn, origin string, lines int) ([]string, int) {
	queries := make([]string, lines)
	max := 0
	pairs := make([][]interface{}, 0)
	maxCount := 0

	for stmt, err := db.Query(t.sql(), origin, lines); err == nil; err = stmt.Next() {
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
