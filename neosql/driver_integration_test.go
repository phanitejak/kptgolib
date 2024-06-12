//go:build integration
// +build integration

package neosql_test

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/phanitejak/kptgolib/mariadb"
	"github.com/phanitejak/kptgolib/neosql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Make sure that AuthRefreshDriver implements the expected interfaces.
var (
	_ driver.Driver        = &neosql.AuthRefreshDriver{}
	_ driver.DriverContext = &neosql.AuthRefreshDriver{}
	_ driver.Connector     = &neosql.AuthRefreshDriver{}
)

//nolint:gosec
const (
	workingDSN      = "root:root@tcp(127.0.0.1:3306)/"
	wrongUserDSN    = "asd:asd@tcp(127.0.0.1:3306)/"
	wrongPassworDSN = "root:asd@tcp(127.0.0.1:3306)/"
	wrongPortDSN    = "root:root@tcp(127.0.0.1:1234)/"
)

type MockLoader interface {
	DSN() (string, error)
	IsAuthErr(error) bool
	Calls() int
}

type testParams struct {
	name          string
	authLoader    MockLoader
	sqlCalls      int
	maxOpenConns  int
	maxIdleConns  int
	expectedCalls int
	expectError   bool
}

func TestIntegrationAuthRefreshDriver(t *testing.T) {
	tests := []testParams{
		{
			name:          "load credentials once for single connection",
			authLoader:    &staticDSN{dsn: workingDSN},
			maxOpenConns:  1,
			maxIdleConns:  0,
			sqlCalls:      10,
			expectedCalls: 1,
			expectError:   false,
		},
		{
			name:          "load credentials once for 10 connections",
			authLoader:    &staticDSN{dsn: workingDSN},
			maxOpenConns:  10,
			maxIdleConns:  10,
			sqlCalls:      100,
			expectedCalls: 1,
			expectError:   false,
		},
		{
			name:          "load credentials twice for 10 connections",
			authLoader:    &dsnStack{dsns: []interface{}{wrongUserDSN, workingDSN}},
			maxOpenConns:  10,
			maxIdleConns:  10,
			sqlCalls:      100,
			expectedCalls: 2,
			expectError:   false,
		},
		{
			name:          "load credentials 11 times for 10 calls with 1 connection",
			authLoader:    &staticDSN{dsn: wrongPassworDSN},
			maxOpenConns:  1,
			maxIdleConns:  0,
			sqlCalls:      10,
			expectedCalls: 11,
			expectError:   true,
		},
		{
			name:          "load credentials 11 times for 10 calls with 10 connections",
			authLoader:    &staticDSN{dsn: wrongPassworDSN},
			maxOpenConns:  10,
			maxIdleConns:  10,
			sqlCalls:      10,
			expectedCalls: 11,
			expectError:   true,
		},
		{
			name:          "reload credentials after wrong user",
			authLoader:    &dsnStack{dsns: []interface{}{wrongUserDSN, workingDSN}},
			maxOpenConns:  1,
			maxIdleConns:  0,
			sqlCalls:      10,
			expectedCalls: 2,
			expectError:   false,
		},
		{
			name:          "reload credentials after wrong password",
			authLoader:    &dsnStack{dsns: []interface{}{wrongPassworDSN, workingDSN}},
			maxOpenConns:  1,
			maxIdleConns:  0,
			sqlCalls:      10,
			expectedCalls: 2,
			expectError:   false,
		},
		{
			name:          "no reload on wrong port",
			authLoader:    &dsnStack{dsns: []interface{}{wrongPortDSN, workingDSN}},
			maxOpenConns:  1,
			maxIdleConns:  0,
			sqlCalls:      10,
			expectedCalls: 1,
			expectError:   true,
		},
		{
			name:          "loading credentials fails on first try with one call",
			authLoader:    &dsnStack{dsns: []interface{}{errors.New("failed")}},
			maxOpenConns:  1,
			maxIdleConns:  0,
			sqlCalls:      1,
			expectedCalls: 1,
			expectError:   true,
		},
		{
			name:          "loading credentials fails on first try with five call",
			authLoader:    &dsnStack{dsns: []interface{}{errors.New("failed"), errors.New("failed"), errors.New("failed"), errors.New("failed"), errors.New("failed")}},
			maxOpenConns:  1,
			maxIdleConns:  0,
			sqlCalls:      5,
			expectedCalls: 5,
			expectError:   true,
		},
		{
			name:          "loading credentials fails on retry",
			authLoader:    &dsnStack{dsns: []interface{}{wrongPassworDSN, errors.New("failed")}},
			maxOpenConns:  1,
			maxIdleConns:  0,
			sqlCalls:      1,
			expectedCalls: 2,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		driverName := fmt.Sprintf("test-driver-%d", time.Now().UnixNano())

		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t, tt, driverName)
			defer func() { require.NoError(t, db.Close(), "failed to close db") }()

			wg := sync.WaitGroup{}
			wg.Add(tt.sqlCalls)

			for n := 0; n < tt.sqlCalls; n++ {
				go func() {
					defer wg.Done()
					if tt.expectError {
						assert.Error(t, db.Ping(), "db.Ping() should fail")
					} else {
						assert.NoError(t, db.Ping(), "db.Ping() shouldn't fail")
					}
				}()
			}
			wg.Wait()

			require.Equal(t, tt.expectedCalls, tt.authLoader.Calls(), "number of DSN call's doesn't match")
		})
	}
}

func TestIntegrationAuthRefreshDriver_resume_after_wrong_password_and_error(t *testing.T) {
	tt := testParams{
		authLoader:    &dsnStack{dsns: []interface{}{wrongPassworDSN, errors.New("failed"), workingDSN}},
		maxOpenConns:  1,
		maxIdleConns:  0,
		expectedCalls: 3,
	}

	db := setupDB(t, tt, fmt.Sprintf("test-driver-%d", time.Now().UnixNano()))
	defer func() { require.NoError(t, db.Close(), "failed to close db") }()

	wg := sync.WaitGroup{}
	wg.Add(tt.sqlCalls)

	assert.Error(t, db.Ping(), "db.Ping() should fail")
	assert.NoError(t, db.Ping(), "db.Ping() shouldn't fail")

	wg.Wait()

	require.Equal(t, tt.expectedCalls, tt.authLoader.Calls(), "number of DSN call's doesn't match")
}

func TestIntegrationAuthRefreshDriver_Driver(t *testing.T) {
	driverName := fmt.Sprintf("test-driver-%d", time.Now().UnixNano())
	expectedDriver := neosql.NewAuthRefreshDriver(&mysql.MySQLDriver{}, &staticDSN{})
	sql.Register(driverName, expectedDriver)

	db, err := sql.Open(driverName, "")
	require.NoError(t, err, "failed to open DB")

	gotDriver := db.Driver()
	require.Equal(t, expectedDriver, gotDriver, "drivers don't match")
}

func setupDB(t testing.TB, p testParams, driverName string) *sql.DB {
	sql.Register(driverName, neosql.NewAuthRefreshDriver(&mysql.MySQLDriver{}, p.authLoader))
	db, err := sql.Open(driverName, "")
	require.NoError(t, err, "failed to open db")
	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxOpenConns(p.maxOpenConns)
	db.SetMaxIdleConns(p.maxIdleConns)
	return db
}

type staticDSN struct {
	calls int
	dsn   string
}

func (s *staticDSN) Calls() int             { return s.calls }
func (s *staticDSN) IsAuthErr(e error) bool { return mariadb.IsAuthenticationError(e) }
func (s *staticDSN) DSN() (string, error) {
	s.calls++
	return s.dsn, nil
}

type dsnStack struct {
	calls int
	dsns  []interface{}
}

func (s *dsnStack) Calls() int             { return s.calls }
func (s *dsnStack) IsAuthErr(e error) bool { return mariadb.IsAuthenticationError(e) }
func (s *dsnStack) DSN() (string, error) {
	dsn := s.dsns[s.calls]
	s.calls++

	if err, ok := dsn.(error); ok {
		return "", err
	}
	return dsn.(string), nil
}
