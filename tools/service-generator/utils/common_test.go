package utils

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Valid name with one space",
			args:    args{name: "Some Service"},
			want:    "some-service",
			wantErr: false,
		},
		{
			name: "Valid name with multiple spaces",
			args: args{name: "   Some 	Service	"},
			want:    "some-service",
			wantErr: false,
		},
		{
			name: "Valid name with multiple spaces and dashes",
			args: args{name: "   Some 	--- Service	"},
			want:    "some-service",
			wantErr: false,
		},
		{
			name:    "Valid name without space",
			args:    args{name: "Some-Service"},
			want:    "some-service",
			wantErr: false,
		},
		{
			name:    "Empty spaces only",
			args:    args{name: "    "},
			wantErr: true,
		},
		{
			name:    "Invalid name",
			args:    args{name: "some-name-!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServiceName(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseServiceName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseServiceName() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCurrentDirectoryPathSuccess(t *testing.T) {
	// Save current function and restore at the end:
	old := osExecutable
	defer func() { osExecutable = old }()

	osExecutable = func() (string, error) {
		return filepath.Join("path", "to", "some", "file.go"), nil
	}

	got, _ := GetCurrentDirectoryPath()
	want := filepath.Join("path", "to", "some")

	if got != want {
		t.Errorf("GetCurrentDirectoryPath() got = %v, want %v", got, want)
	}
}

func TestGetCurrentDirectoryPathFail(t *testing.T) {
	// Save current function and restore at the end:
	old := osExecutable
	defer func() { osExecutable = old }()

	osExecutable = func() (string, error) {
		return "", errors.New("some error occurred")
	}

	_, err := GetCurrentDirectoryPath()

	if err == nil {
		t.Errorf("GetCurrentDirectoryPath() must've returned an error")
	}
}

func TestGetSvcDirectoryPathSuccess(t *testing.T) {
	// Save current function and restore at the end:
	old := osGetwd
	defer func() { osGetwd = old }()

	osGetwd = func() (string, error) {
		return filepath.Join("path", "to", "some", "directory"), nil
	}

	got, _ := GetSvcDirectoryPath()
	want := filepath.Join("path", "to", "some", "directory", "svc")

	if got != want {
		t.Errorf("GetSvcDirectoryPath() got = %v, want %v", got, want)
	}
}

func TestGetSvcDirectoryPathFail(t *testing.T) {
	// Save current function and restore at the end:
	old := osGetwd
	defer func() { osGetwd = old }()

	osGetwd = func() (string, error) {
		return "", errors.New("some error occurred")
	}

	_, err := GetSvcDirectoryPath()

	if err == nil {
		t.Errorf("GetSvcDirectoryPath() must've returned an error")
	}
}

func makeTree(t *testing.T, dir string) {
	oapiPath := filepath.Join(dir, "github.com", "deepmap", "oapi-codegen", "cmd")
	err := os.MkdirAll(oapiPath, os.ModePerm)
	assert.NoError(t, err)

	gofumptPath := filepath.Join(dir, "mvdan.cc", "gofumpt", "cmd")
	err = os.MkdirAll(gofumptPath, os.ModePerm)
	assert.NoError(t, err)

	content := []byte("temporary file's content")
	tmpfile, err := ioutil.TempFile(oapiPath, "oapi-codegen")
	assert.NoError(t, err)

	_, err = tmpfile.Write(content)
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	content = []byte("temporary file's content")
	tmpfile, err = ioutil.TempFile(gofumptPath, "gofumports")
	assert.NoError(t, err)

	_, err = tmpfile.Write(content)
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)
}

func TestGetToolByName(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "TestGetToolByName")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tempDir)
		assert.NoError(t, err)
	}()

	toolsDir := filepath.Join(tempDir, ".tools")
	err = os.Mkdir(toolsDir, os.ModePerm)
	assert.NoError(t, err)

	old := osGetwd
	defer func() { osGetwd = old }()

	osGetwd = func() (string, error) {
		return "", errors.New("some error occurred")
	}

	_, err = GetToolByName("oapi-codegen")
	assert.Error(t, err)

	osGetwd = func() (string, error) {
		return tempDir, nil
	}

	makeTree(t, toolsDir)

	_, err = GetToolByName("oapi-codegen")
	assert.NoError(t, err)

	_, err = GetToolByName("gofumports")
	assert.NoError(t, err)

	_, err = GetToolByName("notexisting")
	assert.Error(t, err)
}

func TestExists(t *testing.T) {
	want := true
	pathThatExists := filepath.Clean("../")

	got := Exists(pathThatExists)

	if got != want {
		t.Errorf("Exist() must've returned true")
	}

	want = false
	pathThatDoestNotExist := filepath.Clean("./not-existing")

	got = Exists(pathThatDoestNotExist)

	if got != want {
		t.Errorf("Exist() must've returned false")
	}
}

func TestGetGitInfo(t *testing.T) {
	old := execCommand
	defer func() { execCommand = old }()

	execCommand = func(name string, args ...string) ([]byte, error) {
		return []byte("something"), nil
	}

	userName, userEmail := GetGitInfo()

	assert.Equal(t, "something", userName)
	assert.Equal(t, "something", userEmail)

	execCommand = func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("'git' command not found")
	}

	userName, userEmail = GetGitInfo()

	assert.Equal(t, "NOM", userName)
	assert.Equal(t, "firstname.lastname@nokia.com", userEmail)
}
