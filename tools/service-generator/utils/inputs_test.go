// Package utils wraps all the utility methods
package utils

import (
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func TestConfigFromFlags(t *testing.T) {
	tests := []struct {
		name    string
		want    Config
		wantErr bool
		osArgs  []string
	}{
		{
			name:    "handles the required inputs correctly",
			want:    Config{ServiceName: "app-name"},
			wantErr: false,
			osArgs:  []string{"generator", "-name", "App Name"},
		},
		{
			name:    "handles the partial inputs correctly",
			want:    Config{ServiceName: "app-name-123", UseMySQL: true, UseKafkaProducer: true},
			wantErr: false,
			osArgs:  []string{"generator", "-name", "App Name 123", "-mysql", "-kafkap"},
		},
		{
			name:    "handles all the inputs correctly",
			want:    Config{ServiceName: "app-name-123", UseMySQL: true, UsePgSQL: true, UseKafkaConsumer: true, UseKafkaProducer: true},
			wantErr: false,
			osArgs:  []string{"generator", "-name", "App Name 123", "-mysql", "-pgsql", "-kafkac", "-kafkap"},
		},
		{
			name:    "throws error without arguments",
			wantErr: true,
			osArgs:  []string{"generator"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConfigFromFlags(tt.osArgs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigFromFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigFromFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigFromPrompt(t *testing.T) {
	type args struct {
		stdin io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    Config
		wantErr bool
	}{
		{
			name:    "handles service name testing-service with two dependencies",
			args:    args{stdin: strings.NewReader(strings.Join([]string{"Testing Service", "y", "", "y", "", ""}, "\n"))},
			want:    Config{ServiceName: "testing-service", UseMySQL: true, UseKafkaConsumer: true},
			wantErr: false,
		},
		{
			name:    "handles service name testing-service-123 with all dependencies",
			args:    args{stdin: strings.NewReader(strings.Join([]string{"Testing Service 123", "y", "y", "y", "y", ""}, "\n"))},
			want:    Config{ServiceName: "testing-service-123", UseMySQL: true, UsePgSQL: true, UseKafkaConsumer: true, UseKafkaProducer: true},
			wantErr: false,
		},
		{
			name:    "returns the error",
			args:    args{stdin: strings.NewReader(strings.Join([]string{""}, "\n"))},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConfigFromPrompt(tt.args.stdin, ioutil.Discard)

			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigFromPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigFromPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}
