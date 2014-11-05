package fetchers

import (
	"fmt"
	"log"
	"reflect"

	"github.com/jinzhu/gorm"
)

var fetchers = make(map[string]Fetcher)

type Fetcher interface {
	sql() string
	Fetch(*gorm.DB, string, int64, int) ([]string, int)
}

func register(fetcher Fetcher) {
	obj := reflect.TypeOf(fetcher)

	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}

	name := obj.Name()

	if _, ok := fetchers[name]; ok {
		panic(fmt.Sprintf("fetchers.Register: %s is already registered", name))
	}

	fetchers[name] = fetcher
}

func Find(name string) Fetcher {
	fetcher, ok := fetchers[name]

	if !ok {
		log.Printf("fetcher.Enable: %s is not registered")
		return nil
	}

	return fetcher
}
