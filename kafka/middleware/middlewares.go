package middleware

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/IBM/sarama"

	"github.com/phanitejak/kptgolib/kafka"
	"github.com/phanitejak/kptgolib/tracing"
)

// CtxHandlerFunc message handler function type.
type CtxHandlerFunc func(ctx context.Context, msg *sarama.ConsumerMessage, mark func(metadata string)) error

// MessageHandlerFunc defines kafka message handling function for Middleware.
type MessageHandlerFunc func(ctx context.Context, msg *sarama.ConsumerMessage) error

// Trace will create span from message and in case of error it will add error log field in span.
func Trace(next CtxHandlerFunc) kafka.HandlerFunc {
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		span, ctx := tracing.StartSpanFromMessage(msg, "MessageReceived")
		defer span.Finish()

		if err := next(ctx, msg, mark); err != nil {
			span.LogFields(tracing.Error(err))
			return err
		}
		return nil
	}
}

// Log will handle debug and error logging.
func Log(logger *tracing.Logger, next CtxHandlerFunc) CtxHandlerFunc {
	return func(ctx context.Context, msg *sarama.ConsumerMessage, mark func(string)) error {
		logger.For(ctx).Debugf("message %s:%d:%d with key %s received, create time duration %v",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Key), time.Since(msg.Timestamp))

		if err := next(ctx, msg, mark); err != nil {
			logger.For(ctx).Errorf("message handling failed: %s", err)
			return err
		}
		return nil
	}
}

// Duration will print how long it took to execute passed handler function.
func Duration(logger *tracing.Logger, next kafka.HandlerFunc) kafka.HandlerFunc {
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		started := time.Now()
		err := next(msg, mark)
		logger.Debugf("message %s:%d:%d duration %v", msg.Topic, msg.Partition, msg.Offset, time.Since(started))
		return err
	}
}

// Mark will always mark offset after HandlerFunc gets executed.
func Mark(next kafka.HandlerFunc) kafka.HandlerFunc {
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		err := next(msg, mark)
		mark("")
		return err
	}
}

// MarkIfNoError will mark offset after HandlerFunc gets executed if no error or markable error was returned.
func MarkIfNoError(next kafka.HandlerFunc) kafka.HandlerFunc {
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		if err := next(msg, mark); err != nil {
			if !errors.As(err, &markable{}) {
				return err
			}
		}
		mark("")
		return nil
	}
}

// IgnoreErrors will silently ignore all returned errors from handler and continue.
func IgnoreErrors(next kafka.HandlerFunc) kafka.HandlerFunc {
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		if err := next(msg, mark); err != nil {
			return nil
		}
		return nil
	}
}

// Recover will recover panics in message processing and logs.
func Recover(logger *tracing.Logger, next kafka.HandlerFunc) kafka.HandlerFunc {
	return func(msg *sarama.ConsumerMessage, mark func(string)) (err error) {
		defer func() {
			if r := recover(); r != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				err = fmt.Errorf("%v panic stack, %s", r, string(buf))
				logger.Errorf("message handling failed: %s", err)
			}
		}()
		err = next(msg, mark)
		return err
	}
}

// CommonDefaultsWithRetry implements common default middlewares Mark, IgnoreErrors and Retry.
// Retry functionality will call given HandlerFunc the amount of maxRetries waiting wait duration between each retry.
// If the error is retryable and it does not succeed in given amount of maxRetries, error is returned.
// If error is not retryable, the offset will be marked and no error is returned.
// If HandlerFunc succeeds, the offset will be marked.
func CommonDefaultsWithRetry(maxRetries uint, wait time.Duration, next kafka.HandlerFunc) kafka.HandlerFunc {
	return MarkIfNoError(func(msg *sarama.ConsumerMessage, mark func(string)) error {
		var err error
		tries := uint(1)
		for {
			if err = next(msg, mark); errors.As(err, &retryable{}) && tries <= maxRetries {
				time.Sleep(wait)
				tries++
				continue
			}
			break
		}
		if err != nil && !errors.As(err, &retryable{}) {
			return nil
		}
		return err
	})
}

// Retry will try to call given HandlerFunc maxRetries times if returned error is retryable.
func Retry(maxRetries uint, wait time.Duration, next kafka.HandlerFunc) kafka.HandlerFunc {
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		var err error
		tries := uint(1)
		for {
			if err = next(msg, mark); errors.As(err, &retryable{}) && tries <= maxRetries {
				time.Sleep(wait)
				tries++
				continue
			}
			break
		}

		return err
	}
}

type retryable struct {
	error
}

func (*retryable) retry() {}

// Retryable will return retryable error if given error is not nil.
// If given error is nil then nil will be returned.
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return retryable{err}
}

type markable struct {
	error
}

// Markable will return markable error if given error is markable.
func Markable(err error) error {
	if err == nil {
		return nil
	}
	return markable{err}
}

// Serial will synchronise access to next handler.
func Serial(next kafka.HandlerFunc) kafka.HandlerFunc {
	lock := &sync.Mutex{}
	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		lock.Lock()
		defer lock.Unlock()
		return next(msg, mark)
	}
}

// MultiPartitionMark stores internally latest mark function for each partition and creates mark function which executes those stored mark functions.
// This allows users to buffer messages and not to worry about partitions.
func MultiPartitionMark(next kafka.HandlerFunc) kafka.HandlerFunc {
	lock := &sync.Mutex{}
	markers := make(map[int32]func(string))
	markAll := func(metadata string) {
		lock.Lock()
		defer lock.Unlock()
		for partition, mark := range markers {
			mark(metadata)
			delete(markers, partition)
		}
	}

	return func(msg *sarama.ConsumerMessage, mark func(string)) error {
		lock.Lock()
		markers[msg.Partition] = mark
		lock.Unlock()
		return next(msg, markAll)
	}
}

// Middleware returns CtxHandlerFunc type for services requiring middleware.
func Middleware(next MessageHandlerFunc) CtxHandlerFunc {
	return func(ctx context.Context, msg *sarama.ConsumerMessage, _ func(string)) error {
		return next(ctx, msg)
	}
}
