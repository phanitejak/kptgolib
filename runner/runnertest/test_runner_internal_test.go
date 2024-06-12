package runnertest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg/runner"
)

func TestRun_CtxDone(t *testing.T) {
	tt := &testing.T{}
	a := &TestApp{
		mods: []runner.Module{
			runner.CloserAsModule(&TestModule{}),
		},
	}

	tr := Run(tt, a)
	tr.Stop()

	ctx, stop := context.WithCancel(context.Background())
	stop()

	code := tr.ExitCodeCtx(ctx)
	assert.Equal(tt, 1, code)
}

type TestApp struct {
	mods []runner.Module
}

func (a *TestApp) Name() string             { return "test-app" }
func (a *TestApp) Modules() []runner.Module { return a.mods }

type TestModule struct {
	done chan struct{}
}

func (m *TestModule) Close() error {
	<-m.done
	return nil
}
