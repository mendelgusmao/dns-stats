package fetchers

import (
	"code.google.com/p/go-sqlite/go1/sqlite3"
)

type Fetcher interface {
	sql() string
	Fetch(*sqlite3.Conn, string, int) ([]string, int)
}
