package model

import (
	"fmt"
	"log"
	"reflect"

	"github.com/jinzhu/gorm"
)

var models = make(map[string]interface{})

func register(fetcher interface{}) {
	obj := reflect.TypeOf(fetcher)

	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}

	name := obj.Name()

	if _, ok := models[name]; ok {
		panic(fmt.Sprintf("models.Register: %s is already registered", name))
	}

	models[name] = fetcher
}

func BuildDatabase(db gorm.DB) bool {
	ok := true

	for name, model := range models {
		db.AutoMigrate(model)

		if db.Error != nil {
			log.Printf("Error creating table for %s: %v\n", name, db.Error)
			ok = false
		}
	}

	return ok
}
