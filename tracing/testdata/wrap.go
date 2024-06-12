package main

import (
	"context"
	"os"

	"gopkg/logging"

	"gopkg/tracing"
)

func main() {
	os.Setenv("LOGGING_LEVEL", "debug") // nolint
	os.Setenv("LOGGING_FORMAT", "json") // nolint

	closer, err := tracing.InitGlobalTracer()
	if err != nil {
		panic(err)
	}
	defer closer.Close() // nolint

	l := tracing.NewLogger(logging.NewLogger())
	_, ctx := tracing.StartSpan("my-span")

	l.Debug("debug msg")
	l.Info("info msg")
	l.Error("error msg")
	l.For(context.Background()).Debugf("%s", "debug msg")
	l.For(context.Background()).Infof("%s", "info msg")
	l.For(context.Background()).Errorf("%s", "error msg")
	l.For(ctx).Debugf("%s", "debug msg")
	l.For(ctx).Infof("%s", "info msg")
	l.For(ctx).Errorf("%s", "error msg")
}
