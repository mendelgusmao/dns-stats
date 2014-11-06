package model

import (
	"log"

	"github.com/jinzhu/gorm"
)

var models = make([]interface{}, 0)

func register(model interface{}) {
	models = append(models, model)
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
