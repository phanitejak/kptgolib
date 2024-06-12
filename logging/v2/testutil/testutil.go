// Package testutil provides convenience functions for testing
package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// AssertKeyInMap checks if the key exists in the given map
func AssertKeyInMap(t *testing.T, key string, m map[string]string) {
	if _, ok := m[key]; !ok {
		t.Fatalf("key '%s' missing from map '%v'", key, m)
	}
}

// UnmarshalLogMessage unmarshalls a byte buffer into a string-to-string map
func UnmarshalLogMessage(t *testing.T, buf []byte) (logMessage map[string]string) {
	if err := json.Unmarshal(buf, &logMessage); err != nil {
		t.Fatalf("Could not unmarshal log message: '%s', error: %s", buf, err)
	}
	return
}

// PipeStderr returns function, which is piping stderr to a buffer
// This function has serious problems with concurrency.
// Don't use it in any production code!
func PipeStderr(t *testing.T) func() *bytes.Buffer {
	old := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	return func() *bytes.Buffer {
		defer func() { os.Stderr = old }()
		_ = w.Close()
		buf := bytes.NewBuffer([]byte{})
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := io.Copy(buf, r)
			assert.NoError(t, err)
		}()
		wg.Wait()
		return buf
	}
}
