// Package mariadb gorm client wrapper library, includes credential refresh from SCO
package mariadb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/phanitejak/kptgolib/credentialreader"
	"github.com/phanitejak/kptgolib/gerror"
	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/neosql"
	"github.com/phanitejak/kptgolib/tracing"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

// DB connection errors.
const (
	DBConnectError      gerror.ErrorCode = "Error connecting to the DB"
	GormGetDialectError gerror.ErrorCode = "Error connecting to the DB"
)

// NewDBClientFromENV returns new DB client.
func NewDBClientFromENV(log *tracing.Logger) (*gorm.DB, error) {
	conf, err := ParseConfigFromEnv()
	if err != nil {
		return nil, err
	}

	return NewDBClient(conf, log)
}

// NewDBClient with given config and logger.
func NewDBClient(conf *Config, log *tracing.Logger) (*gorm.DB, error) {
	var userCredentials credentialreader.AppCredentialResponse
	if conf.Password == "" {
		var err error
		userCredentials, err = credentialreader.FetchCredentialsForUser(conf.Username)
		if err != nil {
			return nil, err
		}
	} else {
		userCredentials = credentialreader.AppCredentialResponse{
			Username: conf.Username,
			Password: conf.Password,
		}
	}

	driverName := fmt.Sprintf("mysql-dyn-auth-%d", time.Now().UnixNano())
	driver := neosql.NewAuthRefreshDriver(&mysql.MySQLDriver{}, &EnvAuth{})
	sql.Register(driverName, driver)
	d, ok := gorm.GetDialect("mysql")
	if !ok {
		return nil, gerror.New(GormGetDialectError, "Failed to get Gorm dialect \"mysql\"")
	}

	gorm.RegisterDialect(driverName, d)
	session, err := gorm.Open(driverName, DataSourceName(userCredentials.Username, userCredentials.Password, conf.Address, conf.DatabaseName, conf.IsTLSEnabled))
	if err != nil {
		log.Infof("Failed to create mysql session: %v", err)
		return nil, gerror.NewFromError(DBConnectError, err)
	}

	session.DB().SetConnMaxLifetime(time.Duration(conf.ConnMaxLifetimeSeconds) * time.Second)
	session.DB().SetMaxIdleConns(conf.MaxIdleConns)
	session.DB().SetMaxOpenConns(conf.MaxOpenConns)
	session.SetLogger(log.Logger.(logging.StdLogger))
	return session, nil
}
