// Package mariadb gorp client library
package mariadb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp/v3"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"gopkg/neosql"
)

// NewDBMap wraps *sql.DB into *gorp.DbMap.
func NewDBMap(db *sql.DB) *gorp.DbMap {
	return &gorp.DbMap{
		Db: db,
		Dialect: gorp.MySQLDialect{
			Engine:   "InnoDB",
			Encoding: "UTF8",
		},
	}
}

// NewDBMapFromEnv tries to resolve db configurations and then wrap *sql.DB into *gorp.DbMap.
func NewDBMapFromEnv() (*gorp.DbMap, error) {
	db, err := NewDBFromEnv()
	if err != nil {
		return nil, err
	}
	return NewDBMap(db), nil
}

// NewDB tries to create *sql.DB with given configuration.
func NewDB(conf Config) (*sql.DB, error) {
	dsn := DataSourceName(conf.Username, conf.Password, conf.Address, conf.DatabaseName, conf.IsTLSEnabled)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifetimeSeconds))
	db.SetMaxOpenConns(conf.MaxOpenConns)
	db.SetMaxIdleConns(conf.MaxIdleConns)

	return db, nil
}

// NewDBFromEnv tries to resolve db configurations and create *sql.DB.
func NewDBFromEnv() (*sql.DB, error) {
	conf, err := ParseConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewDB(*conf)
}

// NewNeoDB creates sql.DB with NEO specific setup.
// Configuration is parsed from environment and
// credentials are automatically updated on auth failure.
func NewNeoDB() (*sql.DB, error) {
	conf, err := ParseConfigFromEnv()
	if err != nil {
		return nil, err
	}

	driverName := fmt.Sprintf("mysql-dyn-auth-%d", time.Now().UnixNano())
	driver := neosql.NewAuthRefreshDriver(&mysql.MySQLDriver{}, &EnvAuth{})
	sql.Register(driverName, driver)

	db, err := sql.Open(driverName, "")
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifetimeSeconds))
	db.SetMaxOpenConns(conf.MaxOpenConns)
	db.SetMaxIdleConns(conf.MaxIdleConns)

	return db, nil
}

// NewNeoDBMap creates gorp.DbMap with NEO specific setup.
// Configuration is parsed from environment and
// credentials are automatically updated on auth failure.
func NewNeoDBMap() (*gorp.DbMap, error) {
	db, err := NewNeoDB()
	if err != nil {
		return nil, err
	}
	return NewDBMap(db), nil
}
