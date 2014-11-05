package fetchers

import "github.com/jinzhu/gorm"

var Fetchers = make([]Fetchers, 0)

type Fetcher interface {
	sql() string
	Fetch(*gorm.DB, string, int64, int) ([]string, int)
}

func register(fetcher Fetcher) {
	fetchers = append(fetchers, fetcher)
}
