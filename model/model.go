package model

import (
	"github.com/jinzhu/gorm"
)

var models = make([]interface{}, 0)

func register(model interface{}) {
	models = append(models, model)
}

func BuildDatabase(db gorm.DB) []error {
	errors := make([]error, 0)

	for _, model := range models {
		db.AutoMigrate(model)

		if db.Error != nil {
			errors = append(errors, db.Error)
		}
	}

	return errors
}
