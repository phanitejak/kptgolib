package foo

import (
	"context"
	"testing"

	"gopkg/logging/v2/loggingtest"
)

func TestStack(t *testing.T) {
	l := loggingtest.NewTestLogger(t)
	l.Error(context.Background(), "error")
	l.Fatal(context.Background(), "fatal")
}
