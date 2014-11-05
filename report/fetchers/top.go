package fetchers

import (
	"fmt"
	"strconv"

	"github.com/jinzhu/gorm"
)

type top struct{}

func (_ top) sql() string {
	return `SELECT address, COUNT(*) AS c 
			FROM hosts, queries
			WHERE at >= $from
			AND origin IN (SELECT id FROM hosts WHERE address LIKE $origin)
			AND id = destination
			GROUP BY address
			ORDER BY c DESC
			LIMIT $limit`
}

func (f top) Fetch(db *gorm.DB, origin string, from int64, lines int) ([]string, int) {
	queries := make([]string, lines)
	max := 0
	pairs := make([][]interface{}, 0)
	maxCount := 0

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

func init() {
	register(top{})
}
