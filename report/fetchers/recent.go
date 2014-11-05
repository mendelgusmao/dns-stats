package fetchers

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

const (
	format = "02/01/06 15:04:05"
)

type Recent struct{}

func (_ Recent) sql() string {
	return `SELECT at, address
			FROM hosts, queries
			WHERE at >= $from
			AND origin IN (SELECT id FROM hosts WHERE address LIKE $origin)
			AND id = destination
			ORDER BY at DESC
			LIMIT $limit`
}

func (r Recent) Fetch(db *gorm.DB, origin string, from int64, lines int) ([]string, int) {
	queries := make([]string, lines)
	max := 0
	index := 0

	for stmt, err := db.Query(r.sql(), from, origin, lines); err == nil; err = stmt.Next() {
		row := make(map[string]interface{})
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
