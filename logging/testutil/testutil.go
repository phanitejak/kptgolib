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

func AssertKeyInMap(t *testing.T, key string, m map[string]string) {
	if _, ok := m[key]; !ok {
		t.Fatalf("key '%s' missing from map '%v'", key, m)
	}
}

func UnmarshalLogMessage(t *testing.T, buf []byte) (logMessage map[string]string) {
	if err := json.Unmarshal(buf, &logMessage); err != nil {
		t.Fatalf("Could not unmarshal log message: '%s', error: %s", buf, err)
	}
	return
}

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
		w.Close()
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
