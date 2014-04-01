package database

import (
	"database/sql"
	"errors"
	"github.com/coopernurse/gorp"
	"net"
	"time"
)

var (
	DB *gorp.DbMap
)

type Query struct {
	Id          int64
	Source      Machine
	Origin      Machine
	Destination Host
	Created     time.Time
}

type Host struct {
	Id      int64
	Address string
	Created time.Time
}

type Machine struct {
	Id      int64
	MAC     string
	IP      net.IP
	Created time.Time
}

func Init(driver, uri string) error {
	db, err := sql.Open(driver, uri)

	if err != nil {
		return err
	}

	var dbmap *gorp.DbMap

	switch driver {
	case "mysql":
		dbmap = &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{}}
	case "postgres":
		dbmap = &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	case "sqlite3":
		dbmap = &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	default:
		return errors.New("Invalid database driver")
	}

	dbmap.AddTableWithName(Query{}, "queries").SetKeys(true, "Id")
	dbmap.AddTableWithName(Host{}, "hosts").SetKeys(true, "Id")
	dbmap.AddTableWithName(Machine{}, "machines").SetKeys(true, "Id")

	if err = dbmap.CreateTablesIfNotExists(); err != nil {
		return err
	}

	DB = dbmap
	return nil
}
