// Package generator wraps all the logic for generating code
package generator

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/phanitejak/gopkg/tools/service-generator/utils"

	"github.com/stretchr/testify/assert"
)

var conf = utils.Config{
	ServiceName:      "awesome-gopher-service",
	UseMySQL:         true,
	UseKafkaConsumer: true,
}

const wantGeneratorDirective = "//go:generate go run ../../../../tools/service-generator/cmd/service-generator/main.go -name awesome-gopher-service -mysql -kafkac"

func Test_applyTemplateAndWriteFile(t *testing.T) {
	config := utils.Config{
		ServiceName: "test-app",
	}

	testTemplate := `
	package main

	import "fmt"

	func main() {
		fmt.Println("Hello world from <<<.ServiceName>>>")
	}
	`

	want := `
	package main

	import "fmt"

	func main() {
		fmt.Println("Hello world from test-app")
	}
	`

	dstDir, err := ioutil.TempDir("", "dstDir")
	assert.Nil(t, err)
	defer func() {
		err := os.RemoveAll(dstDir)
		assert.NoError(t, err)
	}()

	dstFilePath := filepath.Join(dstDir, "dstFileName")

	assert.NoFileExists(t, dstFilePath)

	err = applyTemplateAndWriteFile(testTemplate, dstFilePath, config)
	assert.Nil(t, err)

	assert.FileExists(t, dstFilePath)

	got, err := ioutil.ReadFile(filepath.Clean(dstFilePath))
	assert.NoError(t, err)

	assert.True(t, bytes.Equal(got, []byte(want)))
}

func Test_fillAndCopyTemplateFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tmpDir")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tmpDir)
		assert.NoError(t, err)
	}()

	dstFile := filepath.Join(tmpDir, "main.go")

	err = fillAndCopyTemplateFile("template/cmd/service-name/main.gotmpl", dstFile, conf)
	assert.NoError(t, err)

	contents, err := os.ReadFile(dstFile)
	assert.NoError(t, err)
	assert.Contains(t, string(contents), wantGeneratorDirective)
}

func Test_Generate(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tmpDir")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tmpDir)
		assert.NoError(t, err)
	}()
	doGenerateService(t, tmpDir)
}

func doGenerateService(t *testing.T, tmpDir string) {
	mainFilePath := filepath.Join(tmpDir, conf.ServiceName, "cmd", conf.ServiceName, "main.go")
	assert.NoFileExists(t, mainFilePath)

	err := Generate(tmpDir, conf)
	assert.NoError(t, err)

	assert.FileExists(t, mainFilePath)

	contents, err := os.ReadFile(mainFilePath)
	assert.NoError(t, err)
	assert.Contains(t, string(contents), wantGeneratorDirective)
}

func Test_ReGenerate(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tmpDir")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tmpDir)
		assert.NoError(t, err)
	}()
	doGenerateService(t, tmpDir)

	newConf := conf
	newConf.UseMySQL = false
	newConf.UseKafkaProducer = true

	err = Regenerate(tmpDir, newConf)
	assert.NoError(t, err)

	mainFilePath := filepath.Join(tmpDir, newConf.ServiceName, "cmd", newConf.ServiceName, "main.go")
	contents, err := os.ReadFile(mainFilePath)
	assert.NoError(t, err)

	wantGeneratorDirective := "//go:generate go run ../../../../tools/service-generator/cmd/service-generator/main.go -name awesome-gopher-service -kafkac -kafkap"
	assert.Contains(t, string(contents), wantGeneratorDirective)
}
