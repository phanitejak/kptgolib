package foo

import (
	"context"
	"testing"

	"github.com/phanitejak/gopkg/logging/v2/loggingtest"
)

func TestStack(t *testing.T) {
	l := loggingtest.NewTestLogger(t)
	l.Error(context.Background(), "error")
	l.Fatal(context.Background(), "fatal")
}
