// Package testutils provides different convenience functions for testing - feel free to extend it
package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const unset = `UNSET_THIS_ENVIRONMENT_VARIABLE_ON_CLEANUP`

// SetEnv Sets necessary environment variables for the test function execution and cleans up afterwards
//
// Usage:
//
//
// cleanUpFunc := SetEnv(t,  map[string]string{
//   "LOGGING_LEVEL": "debug",
// })
// defer cleanUpFunc()
//
// ...
func SetEnv(t testing.TB, vars map[string]string) (cleanUp func()) {
	existingVars := make(map[string]string)
	for k := range vars {
		val, exists := os.LookupEnv(k)
		if exists {
			existingVars[k] = val
		} else {
			existingVars[k] = unset
		}
	}

	cleanUp = func() {
		for k, v := range existingVars {
			if v == unset {
				require.NoError(t, os.Unsetenv(k))
			} else {
				require.NoError(t, os.Setenv(k, v))
			}
		}
	}

	for k, v := range vars {
		require.NoError(t, os.Setenv(k, v))
	}

	return
}
