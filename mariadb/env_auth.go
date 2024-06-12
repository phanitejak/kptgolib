package mariadb

import (
	"github.com/go-sql-driver/mysql"
	"github.com/kelseyhightower/envconfig"
	"github.com/phanitejak/gopkg/credentialreader"
)

// Config for SQL connection.
type Config struct {
	Username               string `envconfig:"DATABASE_USERNAME" required:"true"`
	Password               string `envconfig:"DATABASE_PASSWORD" required:"false"`
	Address                string `envconfig:"DATABASE_ADDRESS" required:"true"`
	DatabaseName           string `envconfig:"DATABASE_NAME" required:"true"`
	MaxOpenConns           int    `envconfig:"DATABASE_MAX_OPEN_CONNECTIONS" required:"false" default:"10"`
	MaxIdleConns           int    `envconfig:"DATABASE_MAX_IDLE_CONNECTIONS" required:"false" default:"5"`
	ConnMaxLifetimeSeconds int    `envconfig:"DATABASE_CONN_MAX_LIFETIME_SECONDS" required:"false" default:"30"`
	IsTLSEnabled           bool   `envconfig:"DATABASE_TLS_ENABLED" default:"false"`
}

// ParseConfigFromEnv parses Config struct from environment variables and
// if password is not provided it tries to fetch it from credentials provider.
func ParseConfigFromEnv() (*Config, error) {
	conf := Config{}
	if err := envconfig.Process("", &conf); err != nil {
		return nil, err
	}
	if conf.Password != "" {
		return &conf, nil
	}

	cred, err := credentialreader.FetchCredentialsForUser(conf.Username)
	if err != nil {
		return nil, err
	}

	conf.Username = cred.Username
	conf.Password = cred.Password
	return &conf, nil
}

// DataSourceName returns DSN (data source name) with database name suffix, e.g.: root:arthur!@tcp(localhost:3306)/scheduler.
func DataSourceName(username string, password string, address string, dbName string, isTLSEnabled bool) string {
	if isTLSEnabled {
		return username + ":" + password + "@tcp(" + address + ")/" + dbName + "?tls=skip-verify&parseTime=true"
	}
	return username + ":" + password + "@tcp(" + address + ")/" + dbName + "?parseTime=true"
}

// EnvAuth implements neosql.AuthProvider and loads DB credentials based on env variables.
type EnvAuth struct{}

// DSN will trigger reparsing of configuration and return DSN or an error.
func (EnvAuth) DSN() (string, error) {
	c, err := ParseConfigFromEnv()
	if err != nil {
		return "", err
	}
	return DataSourceName(c.Username, c.Password, c.Address, c.DatabaseName, c.IsTLSEnabled), nil
}

// IsAuthErr checks if given error is know auth error.
func (EnvAuth) IsAuthErr(err error) bool {
	return IsAuthenticationError(err)
}

// IsAuthenticationError checks if given error is know auth error.
func IsAuthenticationError(err error) bool {
	if mysqlError, ok := err.(*mysql.MySQLError); ok {
		if mysqlError.Number == 1045 || mysqlError.Number == 2003 {
			return true
		}
	}
	return false
}
