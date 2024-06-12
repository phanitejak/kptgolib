package mariadb_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/phanitejak/kptgolib/credentialreader"
	"github.com/phanitejak/kptgolib/mariadb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tokenPathKey    = "SERVICE_ACCOUNT_TOKEN_PATH"
	entityNameKey   = "ENTITY_NAME"
	credProviderKey = "APP_CREDENTIAL_PROVIDER_URL"

	dbNameKey     = "DATABASE_NAME"
	dbAddressKey  = "DATABASE_ADDRESS"
	dbUserNameKey = "DATABASE_USERNAME"
	dbPasswordKey = "DATABASE_PASSWORD"
)

func TestNewDBMapFromEnv(t *testing.T) {
	restore := clearEnv(dbNameKey, dbUserNameKey, dbAddressKey, dbPasswordKey, credProviderKey, tokenPathKey, entityNameKey)
	defer restore()

	t.Run("WithInvalidConfig", func(t *testing.T) {
		dbMap, err := mariadb.NewDBMapFromEnv()
		require.Error(t, err)
		require.Nil(t, dbMap, "dbMap should be nil when error is returned")
	})
}

func TestNewDBFromEnv(t *testing.T) {
	restore := clearEnv(dbNameKey, dbUserNameKey, dbAddressKey, dbPasswordKey, credProviderKey, tokenPathKey, entityNameKey)
	defer restore()

	t.Run("WithInvalidConfig", func(t *testing.T) {
		db, err := mariadb.NewDBFromEnv()
		require.Error(t, err)
		require.Nil(t, db, "DB should be nil when error is returned")
	})
}

func TestParseConfigFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		input  mariadb.Config
		output mariadb.Config
	}{{
		name:   "NoCredentialsProvider",
		input:  mariadb.Config{Username: "local", Password: "local", Address: "local"},
		output: mariadb.Config{Username: "local", Password: "local", Address: "local"},
	}, {
		name:   "WithCredentialsProvider",
		input:  mariadb.Config{Username: "local", Address: "local"},
		output: mariadb.Config{Username: "remote", Password: "remote", Address: "local"},
	}}

	restore := clearEnv(dbNameKey, dbUserNameKey, dbAddressKey, dbPasswordKey, credProviderKey, tokenPathKey, entityNameKey)
	defer restore()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				req := &credentialreader.AppCredentialRequest{}
				err := json.NewDecoder(r.Body).Decode(req)
				assert.NoError(t, err)
				err = json.NewEncoder(w).Encode(credentialreader.AppCredentialResponse{Username: "remote", Password: "remote"})
				assert.NoError(t, err)
			}))
			defer server.Close()

			clearEnv(dbNameKey, dbUserNameKey, dbAddressKey, dbPasswordKey, credProviderKey)
			os.Setenv(credProviderKey, server.URL)
			os.Setenv(tokenPathKey, "testdata/token")
			os.Setenv(entityNameKey, "entity")
			os.Setenv(dbNameKey, "some-db")
			os.Setenv(dbAddressKey, tt.input.Address)
			os.Setenv(dbUserNameKey, tt.input.Username)
			os.Setenv(dbPasswordKey, tt.input.Password)

			conf, err := mariadb.ParseConfigFromEnv()
			require.NoError(t, err)
			require.NotNil(t, conf)
			assert.Equal(t, tt.output.Username, conf.Username)
			assert.Equal(t, tt.output.Password, conf.Password)
		})
	}
}

// clearEnv unsets all given keys and returns function to restore it.
func clearEnv(keys ...string) func() {
	env := make(map[string]string, len(keys))
	for _, key := range keys {
		env[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	return func() {
		for k, v := range env {
			os.Setenv(k, v)
		}
	}
}
