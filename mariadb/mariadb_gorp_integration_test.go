//go:build integration
// +build integration

package mariadb

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const newPassword = "newpassword"

func TestIntegrationNewNeoDBChangePassword(t *testing.T) {
	adminDB := getAdminDB(t)

	db, err := NewNeoDB()
	require.NoError(t, err)

	// Set short lifetime for connections so we can test them faster
	db.SetConnMaxLifetime(time.Second)

	err = db.Ping()
	require.NoError(t, err, "user is created so ping should succeed")

	// Store the current password and reset back after the testcased finished
	restorePassword := os.Getenv("DATABASE_USERNAME")
	changeUserPassword(t, adminDB, newPassword)
	defer func() {
		changeUserPassword(t, adminDB, restorePassword)
	}()
	time.Sleep(2 * time.Second)

	err = db.Ping()
	require.Error(t, err, "password is changed in DB but not updated to env so ping should fail")

	err = os.Setenv("DATABASE_PASSWORD", newPassword)
	require.NoError(t, err, "couldn't set new pasword to env")

	err = db.Ping()
	require.NoError(t, err, "password is updated so ping should succeed")
}

func getAdminDB(t testing.TB) *sql.DB {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/")
	require.NoError(t, err)
	return db
}

func changeUserPassword(t *testing.T, adminDB *sql.DB, password string) {
	// Change password and wait for connection lifetime to end
	log.Infof("changing %s user password to %s", os.Getenv("DATABASE_USERNAME"), password)
	_, err := adminDB.Exec(fmt.Sprintf("SET PASSWORD FOR %s = PASSWORD('%s');", os.Getenv("DATABASE_USERNAME"), password))
	require.NoError(t, err, "couldn't set password")
}
