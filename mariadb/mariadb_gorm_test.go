package mariadb_test

import (
	"testing"

	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/mariadb"
	"github.com/phanitejak/kptgolib/tracing"
	"github.com/stretchr/testify/require"
)

func TestNewDBClientFromENV(t *testing.T) {
	restore := clearEnv(dbNameKey, dbUserNameKey, dbAddressKey, dbPasswordKey, credProviderKey, tokenPathKey, entityNameKey)
	defer restore()

	t.Run("WithInvalidConfig", func(t *testing.T) {
		db, err := mariadb.NewDBClientFromENV(tracing.NewLogger(logging.NewLogger()))
		require.Error(t, err)
		require.Nil(t, db, "DB should be nil when error is returned")
	})
}
