package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_logger(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		level logLevel
	}{
		{
			name:  "logs INFO messages",
			want:  fmt.Sprintf("%v[ℹ]  %v %v\n", ColorBlue, "Something", ColorReset),
			level: INFO,
		},
		{
			name:  "logs SUCCESS messages",
			want:  fmt.Sprintf("%v[✔]  %v %v\n", ColorGreen, "Something", ColorReset),
			level: SUCCESS,
		},
		{
			name:  "logs ERROR messages",
			want:  fmt.Sprintf("%v[✖]  %v %v\n", ColorRed, "Something", ColorReset),
			level: ERROR,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rescueStdout := os.Stdout
			defer func() {
				os.Stdout = rescueStdout
			}()

			r, w, _ := os.Pipe()

			os.Stdout = w

			Log("Something", tt.level)

			err := w.Close()
			assert.NoError(t, err)

			got, _ := ioutil.ReadAll(r)

			if !reflect.DeepEqual(string(got), tt.want) {
				t.Errorf("Log() = %v, want %v", got, tt.want)
			}
		})
	}
}
