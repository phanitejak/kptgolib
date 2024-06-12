package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getSvcDirectoryPathInCurrentDirectory() (string, error) {
	_, cwd, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("Colud not get current directory path")
	}

	cwd = filepath.Dir(cwd)
	svcDirectoryPath := filepath.Join(cwd, "svc")
	return svcDirectoryPath, nil
}

func Test_noMonorepo(t *testing.T) {
	t.Cleanup(func() { osExit = os.Exit })

	var exitCode int
	osExit = func(code int) { exitCode = code }

	main()
	assert.Equal(t, 1, exitCode)
}

func Test_invalidFlags(t *testing.T) {
	t.Cleanup(func() { osExit = os.Exit })

	var exitCode int
	osExit = func(code int) { exitCode = code }

	svcDirectoryPath, err := getSvcDirectoryPathInCurrentDirectory()
	assert.NoError(t, err)

	err = os.Mkdir(svcDirectoryPath, os.ModePerm)
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(svcDirectoryPath)
		assert.NoError(t, err)
	}()

	oldOsArgs := os.Args
	defer func() {
		os.Args = oldOsArgs
	}()

	os.Args = []string{
		"",
		"-name",
	}

	main()
	assert.Equal(t, 1, exitCode)
}

func Test_invalidPrompts(t *testing.T) {
	t.Cleanup(func() { osExit = os.Exit })

	var exitCode int
	osExit = func(code int) { exitCode = code }

	svcDirectoryPath, err := getSvcDirectoryPathInCurrentDirectory()
	assert.NoError(t, err)

	err = os.Mkdir(svcDirectoryPath, os.ModePerm)
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(svcDirectoryPath)
		assert.NoError(t, err)
	}()

	oldOsArgs := os.Args
	defer func() {
		os.Args = oldOsArgs
	}()

	os.Args = []string{
		"",
		"-name",
	}

	main()
	assert.Equal(t, 1, exitCode)
}

func Test_mainFromFlags(t *testing.T) {
	svcDirectoryPath, err := getSvcDirectoryPathInCurrentDirectory()
	assert.NoError(t, err)

	err = os.Mkdir(svcDirectoryPath, os.ModePerm)
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(svcDirectoryPath)
		assert.NoError(t, err)
	}()

	oldOsArgs := os.Args
	defer func() {
		os.Args = oldOsArgs
	}()

	os.Args = []string{
		"",
		"-name",
		"my-awesome-service",
		"-mysql",
		"-kafkac",
	}

	servicePath := filepath.Join(svcDirectoryPath, "my-awesome-service")
	assert.NoDirExists(t, servicePath)

	main()

	assert.DirExists(t, servicePath)

	os.Args = []string{
		"",
		"-name",
		"my-awesome-service",
		"-mysql",
		"-kafkap",
	}

	main()

	assert.DirExists(t, servicePath)
}

func Test_mainFromPrompts(t *testing.T) {
	svcDirectoryPath, err := getSvcDirectoryPathInCurrentDirectory()
	assert.NoError(t, err)

	err = os.Mkdir(svcDirectoryPath, os.ModePerm)
	assert.NoError(t, err)

	defer func() {
		err := os.RemoveAll(svcDirectoryPath)
		assert.NoError(t, err)
	}()

	servicePath := filepath.Join(svcDirectoryPath, "my-awesome-service")
	assert.NoDirExists(t, servicePath)

	promptInput := []byte(strings.Join([]string{"My awesome service", "y", "", "y", "", ""}, "\n"))

	tmpfile, err := ioutil.TempFile("", "input")
	assert.NoError(t, err)

	defer func() {
		err := os.Remove(tmpfile.Name())
		assert.NoError(t, err)
	}()

	_, err = tmpfile.Write(promptInput)
	assert.NoError(t, err)

	_, err = tmpfile.Seek(0, 0)
	assert.NoError(t, err)

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	os.Stdin = tmpfile

	oldOsArgs := os.Args
	defer func() {
		os.Args = oldOsArgs
	}()

	os.Args = []string{"my-awesome-service"}

	main()

	assert.DirExists(t, servicePath)

	promptInput = []byte(strings.Join([]string{"My awesome service", "y", "y", "y", "y", "y"}, "\n"))

	_, err = tmpfile.Write(promptInput)
	assert.NoError(t, err)

	_, err = tmpfile.Seek(0, 0)
	assert.NoError(t, err)

	main()

	assert.DirExists(t, servicePath)
}
