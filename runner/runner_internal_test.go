package runner

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg/tracing"
)

func TestRunAppSuccess(t *testing.T) {
	t.Cleanup(func() { exitFn = os.Exit })

	var exitCode int
	exitFn = func(code int) { exitCode = code }

	RunApp(&TestApp{err: nil})
	assert.Equal(t, 0, exitCode)
}

func TestRunAppFail(t *testing.T) {
	t.Cleanup(func() { exitFn = os.Exit })

	var exitCode int
	exitFn = func(code int) { exitCode = code }

	RunApp(&TestApp{err: errors.New("error")})
	assert.Equal(t, 1, exitCode)
}

type TestApp struct {
	err error
}

func (a *TestApp) Name() string               { return "test-app" }
func (a *TestApp) Modules() []Module          { return []Module{a} }
func (a *TestApp) Init(*tracing.Logger) error { return a.err }
func (a *TestApp) Run() error                 { return a.err }
func (a *TestApp) Close() error               { return a.err }
