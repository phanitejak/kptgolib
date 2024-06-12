package main

import (
	"os"

	// empty line to match line numbers
	"github.com/phanitejak/kptgolib/logging"
)

func main() {
	os.Setenv("LOGGING_FORMAT", "json") // nolint
	os.Setenv("LOGGING_LEVEL", "debug") // nolint
	l := logging.NewLogger()
	l.Debug("debug msg")
	l.Info("info msg")
	l.Error("error msg")
	l.Fatal("fatal msg")
}
