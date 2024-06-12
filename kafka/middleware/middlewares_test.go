package middleware_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phanitejak/kptgolib/kafka"
	"github.com/phanitejak/kptgolib/kafka/middleware"
	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/tracing"
)

func TestTraceLog(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{{
		name: "NoError",
		err:  nil,
	}, {
		name: "WithError",
		err:  errors.New("err"),
	}}

	closer, err := tracing.InitGlobalTracer()
	require.NoError(t, err)
	defer closer.Close()

	log := tracing.NewLogger(logging.NewLogger())
	h, errCh := NewHandler()
	handler, ctxCh := NewCtxHandler(h)

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			getCtx := func() context.Context {
				select {
				case ctx := <-ctxCh:
					return ctx
				default:
					t.Fatal("Did not get ctx from channel")
					return nil
				}
			}

			errCh <- tt.err
			err := middleware.Trace(middleware.Log(log, handler))(&sarama.ConsumerMessage{}, func(string) {})
			require.Equal(t, tt.err, err)

			ctx := getCtx()
			span := tracing.SpanFromContext(ctx)
			require.NotNil(t, span, "span shouldn't be nil")
		})
	}
}

func TestMark(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{{
		name: "NoError",
		err:  nil,
	}, {
		name: "WithError",
		err:  errors.New("err"),
	}}

	handler, errCh := NewHandler()

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			errCh <- tt.err
			markCh := make(chan struct{}, 1)
			err := middleware.Mark(handler)(&sarama.ConsumerMessage{}, func(string) { markCh <- struct{}{} })
			require.Equal(t, tt.err, err)

			select {
			case <-markCh:
			default:
				t.Fatal("message was not marked")
			}
		})
	}
}

func TestMarkIfNoError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{{
		name: "NoError",
		err:  nil,
	}, {
		name: "WithError",
		err:  errors.New("err"),
	}, {
		name: "WithMarkableError",
		err:  middleware.Markable(errors.New("err")),
	}}

	handler, errCh := NewHandler()

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			errCh <- tt.err
			markCh := make(chan struct{}, 1)
			err := middleware.MarkIfNoError(handler)(&sarama.ConsumerMessage{}, func(string) { markCh <- struct{}{} })

			if tt.name == "WithMarkableError" {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.err, err)
			}

			select {
			case <-markCh:
				if tt.name == "WithError" {
					t.Fatal("message was marked")
				}
			default:
				if tt.name != "WithError" {
					t.Fatal("message was not marked")
				}
			}
		})
	}
}

func TestIgnoreErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{{
		name: "NoError",
		err:  nil,
	}, {
		name: "WithError",
		err:  errors.New("err"),
	}}

	handler, errCh := NewHandler()

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			errCh <- tt.err
			err := middleware.IgnoreErrors(handler)(&sarama.ConsumerMessage{}, func(string) {})
			require.NoError(t, err)
		})
	}
}

func TestRecover(t *testing.T) {
	tests := []struct {
		name        string
		createPanic bool
	}{{
		name:        "No Panic",
		createPanic: false,
	}, {
		name:        "panic",
		createPanic: true,
	}}

	plog := tracing.NewLogger(logging.NewLogger())
	for _, tt := range tests {
		tt := tt
		handler := NewPanicHandler(tt.createPanic)
		t.Run(tt.name, func(t *testing.T) {
			err := middleware.Recover(plog, handler)(&sarama.ConsumerMessage{}, func(string) {})
			if tt.createPanic {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRetry(t *testing.T) {
	tests := []struct {
		name    string
		retries uint
		errors  uint
	}{{
		name:    "SuccessWithZeroRetries",
		retries: 0,
		errors:  0,
	}, {
		name:    "SuccessAtFirstRetry",
		retries: 1,
		errors:  1,
	}, {
		name:    "SuccessAtSecondRetry",
		retries: 3,
		errors:  2,
	}, {
		name:    "FailNoRetries",
		retries: 0,
		errors:  1,
	}, {
		name:    "FailAllRetries",
		retries: 10,
		errors:  11,
	}}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			handler, errCh := NewHandler()
			done := make(chan struct{})
			defer close(done)

			e := middleware.Retryable(errors.New("error"))
			go waitLoop(tt.errors, e, errCh, done)

			err := middleware.Retry(tt.retries, 0, handler)(&sarama.ConsumerMessage{}, func(string) {})
			if tt.retries < tt.errors {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRetryNonRetryable(t *testing.T) {
	handler, errCh := NewHandler()

	go func() {
		errCh <- middleware.Retryable(errors.New("error"))
		errCh <- errors.New("error") // Break retryable chain of errors
		errCh <- middleware.Retryable(errors.New("error"))
		errCh <- nil
	}()

	err := middleware.Retry(10, 0, handler)(&sarama.ConsumerMessage{}, func(string) {})
	require.Error(t, err)
}

func TestCommonDefaultsWithRetry(t *testing.T) {
	tests := []struct {
		name               string
		retries            uint
		errors             uint
		errorsNotRetryable bool
		marked             bool
	}{
		{
			name:    "SuccessWithZeroRetries",
			retries: 0,
			errors:  0,
			marked:  true,
		}, {
			name:    "SuccessAtFirstRetry",
			retries: 1,
			errors:  1,
			marked:  true,
		}, {
			name:    "SuccessAtSecondRetry",
			retries: 3,
			errors:  2,
			marked:  true,
		}, {
			name:    "FailNoRetries",
			retries: 0,
			errors:  1,
			marked:  false,
		}, {
			name:    "FailAllRetries",
			retries: 10,
			errors:  11,
			marked:  false,
		}, {
			name:               "MarkNonRetryableWithRetries",
			retries:            5,
			errors:             1,
			errorsNotRetryable: true,
			marked:             true,
		}, {
			name:               "MarkNonRetryableNoRetries",
			retries:            0,
			errors:             1,
			errorsNotRetryable: true,
			marked:             true,
		},
	}

	for _, tt := range tests {
		tt := tt
		marked := false
		t.Run(tt.name, func(t *testing.T) {
			handler, errCh := NewHandler()
			done := make(chan struct{})
			defer close(done)

			e := errors.New("error")
			if !tt.errorsNotRetryable {
				e = middleware.Retryable(e)
			}

			go waitLoop(tt.errors, e, errCh, done)

			err := middleware.CommonDefaultsWithRetry(tt.retries, 0, handler)(&sarama.ConsumerMessage{}, func(string) { marked = true })
			if tt.retries < tt.errors && !tt.errorsNotRetryable {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.marked, marked, "message marked")
		})
	}
}

func waitLoop(ttErrors uint, e error, errCh chan<- error, done chan struct{}) {
	for i := uint(0); i < ttErrors; i++ {
		select {
		case errCh <- e:
		case <-done:
			return
		}
	}
	for {
		select {
		case errCh <- nil:
		case <-done:
			return
		}
	}
}

// NewHandler returns handler func which will return next error from channel
func NewHandler() (kafka.HandlerFunc, chan<- error) {
	errCh := make(chan error, 1)
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		err := <-errCh
		return err
	}, errCh
}

// NewPanicHandler returns handler func which will cause panic
func NewPanicHandler(createPanic bool) kafka.HandlerFunc {
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		if createPanic {
			var m *sync.Mutex
			m.Lock()
		}
		return nil
	}
}

// NewCtxHandler wraps handler func with context and allows inspecting it via provided channel.
func NewCtxHandler(next kafka.HandlerFunc) (middleware.CtxHandlerFunc, <-chan context.Context) {
	ctxCh := make(chan context.Context, 1)
	return func(ctx context.Context, msg *sarama.ConsumerMessage, mark func(string)) error {
		ctxCh <- ctx
		return next(msg, mark)
	}, ctxCh
}

func TestSerial(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{{
		name: "NoError",
		err:  nil,
	}, {
		name: "WithError",
		err:  errors.New("err"),
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Go's race detector should detect non synchronised access to shared resource if one occurs.
			var counter int
			handler := middleware.Serial(func(msg *sarama.ConsumerMessage, mark func(string)) error {
				counter++
				return tt.err
			})

			const iterations = 100

			wg := sync.WaitGroup{}
			wg.Add(iterations)
			for i := 0; i < iterations; i++ {
				go func() {
					defer wg.Done()
					err := handler(&sarama.ConsumerMessage{Value: []byte{'d', 'a', 't', 'a'}}, func(string) {})
					assert.Equal(t, tt.err, err)
				}()
			}
			wg.Wait()

			require.Equal(t, iterations, counter)
		})
	}
}

func TestMultiPartitionMark(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{{
		name: "NoError",
		err:  nil,
	}, {
		name: "WithError",
		err:  errors.New("err"),
	}}

	const (
		partitions int32 = 5
		messages   int   = 10
	)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			markCounters := map[int32]int{
				0: messages,
				1: messages,
				2: messages,
				3: messages - 1,
				4: messages - 1,
			}

			handler := middleware.MultiPartitionMark(func(msg *sarama.ConsumerMessage, mark func(string)) error {
				if msg.Partition == 2 {
					mark("")
				}
				return tt.err
			})

			for i := 0; i < messages; i++ {
				for p := int32(0); p < partitions; p++ {
					p := p
					msg := &sarama.ConsumerMessage{Partition: p}
					mark := func(string) { markCounters[p]-- }
					assert.Equal(t, tt.err, handler(msg, mark))
				}
			}

			for k, v := range markCounters {
				assert.Equalf(t, 0, v, "partition %d should have zero unmarked messages", k)
			}
		})
	}
}
