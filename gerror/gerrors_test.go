package gerror_test

import (
	"errors"
	"testing"

	"github.com/phanitejak/gopkg/gerror"
)

func TestGetErrorMessage(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"standard error",
			args{errors.New("message")},
			"message",
		},
		{
			"gerror with message",
			args{gerror.New(gerror.InternalError, "message")},
			"Internal Error: message",
		},
		{
			"gerror from error with message",
			args{gerror.NewFromError(gerror.InternalError, errors.New("message"))},
			"Internal Error: message",
		},
		{
			"gerror from error with gerror",
			args{gerror.NewFromError(gerror.InternalError, gerror.New(gerror.InternalError, "message"))},
			"Internal Error: Internal Error: message",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := gerror.GetErrorMessage(tt.args.err); got != tt.want {
				t.Errorf("GetErrorMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
