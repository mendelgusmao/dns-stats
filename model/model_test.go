package model

import (
	"fmt"
	"log"
	"testing"

	"github.com/MendelGusmao/go-testdb"
	"github.com/jinzhu/gorm"
)

type dummy struct {
	Dummy string
}

type anotherDummy dummy

type buffer struct {
	content [][]byte
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.content = append(b.content, (p))
	return len(b.content), nil
}

func TestRegister(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Fatal()
		}
	}()

	models = make(map[string]interface{})

	register(&dummy{})
	register(dummy{})
}

func TestBuildDatabase(t *testing.T) {
	models = make(map[string]interface{})
	b := &buffer{}
	log.SetOutput(b)

	register(dummy{})

	db, _ := gorm.Open("testdb", "")

	testdb.StubExec(
		"CREATE TABLE `dummies` (`dummy` VARCHAR(255))",
		testdb.NewResult(1, nil, 1, nil))

	if !BuildDatabase(db) {
		t.Fatal()
	}

	models = make(map[string]interface{})

	register(anotherDummy{})
	testdb.StubExecError(
		"CREATE TABLE `another_dummies` (`dummy` VARCHAR(255))",
		fmt.Errorf("Forged Error"))

	if BuildDatabase(db) {
		t.Fatal()
	}
}
