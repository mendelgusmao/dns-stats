package fetchers

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

const (
	format = "02/01/06 15:04:05"
)

type recent struct{}

func (_ recent) sql() string {
	return `SELECT at, address
			FROM hosts, queries
			WHERE at >= $from
			AND origin IN (SELECT id FROM hosts WHERE address LIKE $origin)
			AND id = destination
			ORDER BY at DESC
			LIMIT $limit`
}

func (f recent) Fetch(db *gorm.DB, origin string, from int64, lines int) ([]string, int) {
	queries := make([]string, lines)
	max := 0
	index := 0

	rows, err := db.Raw(f.sql(), from, origin, lines).Rows()

	if err != nil {
		fmt.Println("report.malware.Fetch (querying):", err)
		return queries, 0
	}

	for rows.Next() {
		row := make(map[string]interface{})
		errs := rows.Scan(row)

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

func init() {
	register(recent{})
}
