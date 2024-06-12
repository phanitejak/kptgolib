package runner_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/phanitejak/gopkg/logging/loggingtest"
	"github.com/phanitejak/gopkg/runner"
	"github.com/phanitejak/gopkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestAppRunner(t *testing.T) {
	tests := []struct {
		name                   string
		app                    *CounterApp
		wantModulesInitCalled  []int
		wantModulesRunCalled   []int
		wantModulesCloseCalled []int
		wantExitCode           int
	}{
		{
			name: "ErrorInFirstModuleInit",
			app: &CounterApp{
				modules: []*Module{
					{initErr: errors.New("module init err")},
					{},
				},
			},
			wantModulesInitCalled:  []int{1, 0},
			wantModulesRunCalled:   []int{0, 0},
			wantModulesCloseCalled: []int{0, 0},
			wantExitCode:           1,
		}, {
			name: "ErrorInSecontModuleInit",
			app: &CounterApp{
				modules: []*Module{
					{},
					{initErr: errors.New("module init err")},
				},
			},
			wantModulesInitCalled:  []int{1, 1},
			wantModulesRunCalled:   []int{0, 0},
			wantModulesCloseCalled: []int{0, 0},
			wantExitCode:           1,
		}, {
			name: "ErrorInFirstModuleRun",
			app: &CounterApp{
				modules: []*Module{
					{runErr: errors.New("module run err")},
					{},
				},
			},
			wantModulesInitCalled:  []int{1, 1},
			wantModulesRunCalled:   []int{1, 1},
			wantModulesCloseCalled: []int{1, 1},
			wantExitCode:           1,
		}, {
			name: "ErrorInSecondModuleRun",
			app: &CounterApp{
				modules: []*Module{
					{},
					{runErr: errors.New("module run err")},
				},
			},
			wantModulesInitCalled:  []int{1, 1},
			wantModulesRunCalled:   []int{1, 1},
			wantModulesCloseCalled: []int{1, 1},
			wantExitCode:           1,
		}, {
			name: "ErrorInAllModuleRuns",
			app: &CounterApp{
				modules: []*Module{
					{runErr: errors.New("module run err")},
					{runErr: errors.New("module run err")},
				},
			},
			wantModulesInitCalled:  []int{1, 1},
			wantModulesRunCalled:   []int{1, 1},
			wantModulesCloseCalled: []int{1, 1},
			wantExitCode:           1,
		}, {
			name: "ErrorInModuleClose",
			app: &CounterApp{
				modules: []*Module{
					{closeErr: errors.New("close error")},
					{},
				},
			},
			wantModulesInitCalled:  []int{1, 1},
			wantModulesRunCalled:   []int{1, 1},
			wantModulesCloseCalled: []int{1, 1},
			wantExitCode:           1,
		}, {
			name: "ErrorInModuleRunAndClose",
			app: &CounterApp{
				modules: []*Module{
					{
						runErr:   errors.New("run error"),
						closeErr: errors.New("close error"),
					},
					{},
				},
			},
			wantModulesInitCalled:  []int{1, 1},
			wantModulesRunCalled:   []int{1, 1},
			wantModulesCloseCalled: []int{1, 1},
			wantExitCode:           1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, stop := context.WithCancel(context.Background())
			defer stop()

			log := tracing.NewLogger(loggingtest.NewTestLogger(t))
			r := runner.NewRunner(ctx, log)
			exitCode := r.Run(tt.app)

			r.Ready() // Make sure that ready doesn't block

			assert.Equal(t, tt.wantExitCode, exitCode, "unexpected exit code")

			for i, mod := range tt.app.modules {
				assert.Equalf(t, tt.wantModulesInitCalled[i], mod.initCalled, "modules Init called unexpected number of times for module with index %d", i)
				assert.Equalf(t, tt.wantModulesRunCalled[i], mod.runCalled, "modules Run called unexpected number of times for module with index %d", i)
				assert.Equalf(t, tt.wantModulesCloseCalled[i], mod.closeCalled, "modules Close called unexpected number of times for module with index %d", i)
			}

			assert.Equal(t, 1, tt.app.modulesCalled, "app.Modules called unexpected number of times")
		})
	}
}

func TestAppRunner_CloseWithSignal(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	log := tracing.NewLogger(loggingtest.NewTestLogger(t))

	initCh := make(chan struct{})
	closeCh := make(chan struct{})
	mod := &FnModule{
		initFn: func() error {
			close(initCh)
			return nil
		},
		runFn: func() error {
			stop()
			<-closeCh
			return nil
		},
		closeFn: func() error {
			close(closeCh)
			return nil
		},
	}

	exitCode := runner.NewRunner(ctx, log).Run(&App{modules: []runner.Module{mod}})
	assert.Equal(t, 0, exitCode, "unexpected exit code")
}

func TestModuleWrappers(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	log := tracing.NewLogger(loggingtest.NewTestLogger(t))

	mod1 := runner.InitializerAsModule(&FnModule{initFn: func() error { return nil }})
	mod2 := runner.CloserAsModule(&FnModule{closeFn: func() error { return nil }})
	mod3 := &FnModule{
		initFn:  func() error { return nil },
		closeFn: func() error { return nil },
		runFn: func() error {
			stop()
			return nil
		},
	}
	exitCode := runner.NewRunner(ctx, log).Run(&App{modules: []runner.Module{mod1, mod2, mod3}})
	assert.Equal(t, 0, exitCode, "unexpected exit code")
}

func TestRunnerOrdering(t *testing.T) {
	const (
		numOfMods       = 3
		funcCallsPerMod = 3
	)

	app := &App{}
	ctx, stop := context.WithCancel(context.Background())
	ch := make(chan string, funcCallsPerMod*numOfMods)
	for i := 0; i < numOfMods; i++ {
		i := i
		app.modules = append(app.modules, &FnModule{
			initFn: func() error {
				ch <- fmt.Sprintf("init-%d", i)
				return nil
			},
			runFn: func() error {
				ch <- fmt.Sprintf("run-%d", i)
				<-ctx.Done()
				return nil
			},
			closeFn: func() error {
				ch <- fmt.Sprintf("close-%d", i)
				return nil
			},
		})
	}

	log := tracing.NewLogger(loggingtest.NewTestLogger(t))
	r := runner.NewRunner(ctx, log)

	exitCode := make(chan int)
	go func() { exitCode <- r.Run(app) }()

	r.Ready()
	for i := 0; i < numOfMods; i++ {
		select {
		case val := <-ch:
			assert.Equal(t, fmt.Sprintf("init-%d", i), val)
		default:
			t.Fatalf("did not receive 'init-%d' in time", i)
		}
	}

	for i := 0; i < numOfMods; i++ {
		select {
		case val := <-ch:
			assert.Equal(t, fmt.Sprintf("run-%d", i), val)
		case <-time.After(time.Millisecond * 500):
			t.Fatalf("did not receive 'run-%d' in time", i)
		}
	}
	stop()

	for i := numOfMods - 1; i >= 0; i-- {
		select {
		case val := <-ch:
			assert.Equal(t, fmt.Sprintf("close-%d", i), val)
		case <-time.After(time.Millisecond * 500):
			t.Fatalf("did not receive 'close-%d' in time", i)
		}
	}

	assert.Equal(t, 0, <-exitCode, "unexpected exit code")
}

type App struct {
	modules []runner.Module
}

func (a *App) Name() string { return "test-app" }

func (a *App) Modules() []runner.Module {
	return a.modules
}

type CounterApp struct {
	modules       []*Module
	modulesCalled int
}

func (a *CounterApp) Name() string { return "test-app" }

func (a *CounterApp) Modules() []runner.Module {
	a.modulesCalled++
	mods := make([]runner.Module, 0, len(a.modules))
	for i := range a.modules {
		mods = append(mods, a.modules[i])
	}
	return mods
}

type FnModule struct {
	initFn  func() error
	runFn   func() error
	closeFn func() error
}

func (a *FnModule) Init(*tracing.Logger) error { return a.initFn() }
func (a *FnModule) Run() error                 { return a.runFn() }
func (a *FnModule) Close() error               { return a.closeFn() }

type Module struct {
	initErr     error
	runErr      error
	closeErr    error
	initCalled  int
	runCalled   int
	closeCalled int
}

func (m *Module) Init(*tracing.Logger) error {
	m.initCalled++
	return m.initErr
}

func (m *Module) Run() error {
	m.runCalled++
	return m.runErr
}

func (m *Module) Close() error {
	m.closeCalled++
	return m.closeErr
}
