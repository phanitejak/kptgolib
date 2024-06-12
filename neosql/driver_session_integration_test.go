//go:build integration
// +build integration

package neosql_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/phanitejak/gopkg/mariadb"
	"github.com/phanitejak/gopkg/neosql"
	"github.com/stretchr/testify/require"
)

func TestIntegrationAuthRefreshDriverSession(t *testing.T) {
	adminDB := getAdminDB(t)

	removeUser(t, adminDB, "user1")
	removeUser(t, adminDB, "user2")

	defer removeUser(t, adminDB, "user1")
	defer removeUser(t, adminDB, "user2")

	c := &credentials{user: "user1", passwd: "password1"}

	// Set short lifetime for connections so we can test them faster
	userDB := getUserDB(t, c)
	userDB.SetConnMaxLifetime(time.Second)

	err := userDB.Ping()
	require.Error(t, err, "user isn't created yet so ping should fail")

	addUser(t, adminDB, c.user, c.passwd)

	err = userDB.Ping()
	require.NoError(t, err, "user is created so ping should succeed")

	changePassword(t, userDB, c.user, "newpassword")
	time.Sleep(2 * time.Second)

	err = userDB.Ping()
	require.Error(t, err, "password is changed in DB but provided to AuthLoader so ping should fail")

	c.passwd = "newpassword"

	err = userDB.Ping()
	require.NoError(t, err, "password is updated so ping should succeed")

	removeUser(t, adminDB, "user1")
	time.Sleep(2 * time.Second)

	err = userDB.Ping()
	require.Error(t, err, "user is removed so ping should fail")

	c.user = "user2"
	c.passwd = "password2"
	addUser(t, adminDB, c.user, c.passwd)

	err = userDB.Ping()
	require.NoError(t, err, "new user is created so ping should succeed")

	rows, err := userDB.Query("SHOW GRANTS")
	require.NoError(t, err)
	for rows.Next() {
		s := ""
		require.NoError(t, rows.Scan(&s))
	}

	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())

	addDBForUser(t, adminDB, "foo", c.user)
	_, err = userDB.Exec("USE foo")
	require.NoError(t, err)

	_, err = userDB.Exec("DROP TABLE IF EXISTS bar")
	require.NoError(t, err)

	_, err = userDB.Exec("CREATE TABLE bar ( a integer )")
	require.NoError(t, err)

	_, err = userDB.Exec("DROP TABLE bar")
	require.NoError(t, err)
}

func addUser(t testing.TB, db *sql.DB, name, password string) {
	_, err := db.Exec(fmt.Sprintf("CREATE USER IF NOT EXISTS %s IDENTIFIED BY '%s';", name, password))
	require.NoError(t, err, "couldn't create a user")
}

func changePassword(t testing.TB, db *sql.DB, name, password string) {
	_, err := db.Exec(fmt.Sprintf("SET PASSWORD FOR %s = PASSWORD('%s');", name, password))
	require.NoError(t, err, "couldn't set password")
}

func addDBForUser(t testing.TB, db *sql.DB, dbName, username string) {
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", dbName))
	require.NoError(t, err, "couldn't create db")

	_, err = db.Exec(fmt.Sprintf("GRANT ALL ON %s.* TO '%s';", dbName, username))
	require.NoError(t, err, "couldn't grant access to db")
}

func removeUser(t testing.TB, db *sql.DB, name string) {
	_, err := db.Exec(fmt.Sprintf("DROP USER IF EXISTS %s;", name))
	require.NoError(t, err, "couldn't drop user")
}

func getAdminDB(t testing.TB) *sql.DB {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/")
	require.NoError(t, err)
	return db
}

func getUserDB(t testing.TB, a neosql.AuthProvider) *sql.DB {
	driverName := fmt.Sprintf("test-driver-%d", time.Now().UnixNano())
	sql.Register(driverName, neosql.NewAuthRefreshDriver(&mysql.MySQLDriver{}, a))

	db, err := sql.Open(driverName, "")
	require.NoError(t, err, "couldn't open DB with custom driver")

	return db
}

type credentials struct {
	user, passwd, db string
}

func (c *credentials) IsAuthErr(e error) bool { return mariadb.IsAuthenticationError(e) }
func (c *credentials) DSN() (string, error) {
	return fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s", c.user, c.passwd, c.db), nil
}
