//go:build integration
// +build integration

package mariadb

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/tracing"
	"github.com/stretchr/testify/require"
)

const (
	//nolint:gosec
	passwordResposne  = `{"username": "%s","password": "%s"}`
	newPasswordPrefix = "newpassword%d"

	user1 = "User1"
	user2 = "User2"
)

var (
	tc  = testContext{}
	log = tracing.NewLogger(logging.NewLogger())

	model1 = &TestUser{
		Name: user1,
		Age:  35,
		ID:   "test",
	}

	model2 = &TestUser{
		Name: user2,
		Age:  38,
		ID:   "test2",
	}
)

type TestUser struct {
	Name string `json:"name" gorm:"column:name;not null;primary_key"`
	Age  int    `json:"age" gorm:"column:age"`
	ID   string `json:"id" gorm:"column:id"`
}

type testContext struct {
	adminDB           *sql.DB
	session           *gorm.DB
	isNewPasswordSet  bool
	userName          string
	currentPassword   string
	address           string
	dbName            string
	scoService        *httptest.Server
	scoMockServiceURL string
	nextPassword      string
	pc                int
}

func setup(t *testing.T, log *tracing.Logger) {
	tc.adminDB = getAdminDB(t)

	// Store the current password and reset back after the testcase finished
	tc.currentPassword = os.Getenv("DATABASE_PASSWORD")
	tc.userName = os.Getenv("DATABASE_USERNAME")
	tc.address = os.Getenv("DATABASE_ADDRESS")
	tc.dbName = os.Getenv("DATABASE_NAME")
	log.Infof("DATABASE Info : Username %v, Password %v, Address %v", tc.userName, tc.currentPassword, tc.address)

	tc.scoService = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.isNewPasswordSet {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(fmt.Sprintf(passwordResposne, tc.userName, tc.nextPassword)))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(fmt.Sprintf(passwordResposne, tc.userName, tc.currentPassword)))
			}
		}),
	)

	setGormClientEnvConfig(t)
	testCreateTable(t, log)
}

func teardown(t *testing.T) {
	changeUserPassword(t, tc.adminDB, tc.currentPassword)
	tc.isNewPasswordSet = false
	_ = tc.session.Close()
	tc.scoService.Close()

	// Restore the password
	err := os.Setenv("DATABASE_PASSWORD", tc.currentPassword)
	require.NoError(t, err, "couldn't set entity name to env")
}

func setGormClientEnvConfig(t *testing.T) {
	err := os.Setenv("DATABASE_CONN_MAX_LIFETIME_SECONDS", "1")
	require.NoError(t, err, "couldn't set database connection max life time to env")

	err = os.Setenv("SERVICE_ACCOUNT_TOKEN_PATH", "./testdata/token")
	require.NoError(t, err, "couldn't set service account token path to env")

	err = os.Setenv("APP_CREDENTIAL_PROVIDER_URL", tc.scoService.URL)
	require.NoError(t, err, "couldn't set SCO service URL to env")

	err = os.Setenv("ENTITY_NAME", "cmdb")
	require.NoError(t, err, "couldn't set entity name to env")

	err = os.Setenv("DATABASE_ADDRESS", "127.0.0.1:3306")
	require.NoError(t, err, "failed to set db address")
}

func testCreateTable(t *testing.T, log *tracing.Logger) {
	session, err := NewDBClientFromENV(log)
	if err != nil {
		require.NoError(t, err)
	}

	tc.session = session

	if tc.session.HasTable(&TestUser{}) {
		tc.session.DropTable(&TestUser{})
	}
	require.NoError(t, tc.session.CreateTable(&TestUser{}).Error)
}

func changeGormUserPassword(t *testing.T) {
	newPassword := fmt.Sprintf(newPasswordPrefix, tc.pc)
	changeUserPassword(t, tc.adminDB, newPassword)
	tc.isNewPasswordSet = true
	_ = os.Setenv("DATABASE_PASSWORD", newPassword)
	tc.nextPassword = newPassword
	time.Sleep(1100 * time.Millisecond)
	tc.pc++
}

// Steps to test in local
// docker run --name mariadbtest -e MYSQL_ROOT_PASSWORD=root -p 3306:3306 -d mariadb:10.3
// CREATE USER IF NOT EXISTS pkg_mariadb_integration IDENTIFIED BY 'pkg_mariadb_integration';
// SET PASSWORD FOR pkg_mariadb_integration = PASSWORD('pkg_mariadb_integration');
// CREATE DATABASE IF NOT EXISTS pkg_mariadb_integration;
// GRANT ALL ON pkg_mariadb_integration.* TO 'pkg_mariadb_integration';
// go clean --testcache; env DATABASE_USERNAME=pkg_mariadb_integration DATABASE_PASSWORD=pkg_mariadb_integration DATABASE_ADDRESS=127.0.0.1:3306 DATABASE_NAME=pkg_mariadb_integration go test -v -race -cover -tags=integration -run='^TestIntegration' ./...

func TestIntegrationNewGormClient(t *testing.T) {
	_ = os.Setenv("DATABASE_ADDRESS", "127.0.0.1:3306")
	setup(t, log)
	defer teardown(t)

	changeGormUserPassword(t)
	client, err := NewDBClientFromENV(log)
	require.NoError(t, err)
	defer func() {
		_ = client.Close()
	}()
	err = client.Create(model1).Error
	require.NoError(t, err)

	changeGormUserPassword(t)
	err = client.Save(model1).Error
	require.NoError(t, err)

	changeGormUserPassword(t)
	err = client.Save(model2).Error
	require.NoError(t, err)

	changeGormUserPassword(t)
	err = client.Delete(&TestUser{}, "name = ?", user2).Error
	require.NoError(t, err)
}
