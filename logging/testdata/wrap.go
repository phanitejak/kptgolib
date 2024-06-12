package main

import (
	"os"

	"github.com/phanitejak/gopkg/logging"
	"github.com/phanitejak/gopkg/tracing"
)

func main() {
	os.Setenv("LOGGING_LEVEL", "debug") // nolint
	os.Setenv("LOGGING_FORMAT", "json") // nolint
	l := tracing.NewLogger(logging.NewLogger())
	l.Debug("debug msg")
	l.Info("info msg")
	l.Error("error msg")
	l.Fatal("fatal msg")
}
