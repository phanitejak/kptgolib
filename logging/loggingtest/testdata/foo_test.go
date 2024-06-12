package foo

import (
	"testing"

	"github.com/phanitejak/gopkg/logging/loggingtest"
	"github.com/phanitejak/gopkg/tracing"
)

func TestStack(t *testing.T) {
	l := loggingtest.NewTestLogger(t)
	l.Error("error")
	l.Fatal("fatal")
}

func TestStackWrapped(t *testing.T) {
	l := tracing.NewLogger(loggingtest.NewTestLogger(t))
	l.Error("error")
	l.Fatal("fatal")
}
