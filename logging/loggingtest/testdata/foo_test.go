package foo

import (
	"testing"

	"gopkg/logging/loggingtest"
	"gopkg/tracing"
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
