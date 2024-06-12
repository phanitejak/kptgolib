// Package generator wraps all the logic for generating code
package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/phanitejak/gopkg/tools/service-generator/templatefs"
	"github.com/phanitejak/gopkg/tools/service-generator/utils"
)

var serviceTemplate = templatefs.ServiceTemplate{}

func runGoFmt(svcPath string) error {
	goFumptPath, err := utils.GetToolByName("gofumports")
	if err != nil {
		utils.Log("gofumpt is not installed. Code is not formatted.", utils.INFO)
		return nil
	}
	_, err = exec.Command(goFumptPath, "-w", svcPath).Output() //nolint:gosec
	return err
}

func applyTemplateAndWriteFile(contents, dstFilePath string, config utils.Config) error {
	t, err := template.New("template").Delims("<<<", ">>>").Parse(contents)
	if err != nil {
		return err
	}
	fileWriter, err := os.Create(dstFilePath)
	if err != nil {
		return err
	}
	if err := t.Execute(fileWriter, config); err != nil {
		return err
	}
	return nil
}

func fillAndCopyTemplateFile(srcFilePath, dstFilePath string, config utils.Config) error {
	srcFileContents, err := serviceTemplate.GetFileContents(srcFilePath)
	if err != nil {
		return err
	}
	if err := applyTemplateAndWriteFile(srcFileContents, dstFilePath, config); err != nil {
		return err
	}
	return nil
}

// Generate generates a new microservice with user preferences
func Generate(svcDirectory string, config utils.Config) error {
	appName := config.ServiceName
	appDstPath := filepath.Join(svcDirectory, appName)

	utils.Log("Copying files and directories", utils.INFO)
	utils.Log("Creating microservice", utils.INFO)

	files, err := serviceTemplate.GetFilesByRoot("template")
	if err != nil {
		return err
	}

	for _, srcFile := range files {
		dstFile := filepath.Join(appDstPath, srcFile)
		if strings.Contains(dstFile, ".gotmpl") {
			dstFile = strings.Replace(dstFile, ".gotmpl", ".go", 1)
		}
		dstFile = strings.Replace(dstFile, "template"+string(os.PathSeparator), "", 1) // remove the template/ or template\ from the path based on OS
		dstFile = strings.Replace(dstFile, "service-name", appName, 1)                 // rename the "service-name" directory to actual service name inside "cmd" directory

		// if MySQL is not selected then we'll skip the storage.go module
		if strings.Contains(dstFile, "storage.go") && !config.UseMySQL {
			continue
		}

		if !utils.Exists(filepath.Dir(dstFile)) {
			if err := os.MkdirAll(filepath.Dir(dstFile), os.ModePerm); err != nil {
				return err
			}
		}
		if err := fillAndCopyTemplateFile(srcFile, dstFile, config); err != nil {
			return err
		}
	}

	if err := runGoFmt(appDstPath); err != nil {
		return err
	}

	utils.Log(fmt.Sprintf("\"%v\" microservice generated successfully", appName), utils.SUCCESS)
	return nil
}

// Regenerate updates the code with new options
func Regenerate(svcDirectory string, config utils.Config) error {
	appName := config.ServiceName

	utils.Log(fmt.Sprintf("Microservice \"%v\" already exists. Updating modules only...", appName), utils.INFO)

	appDstPath := filepath.Join(svcDirectory, appName)

	srcMainPath := "template/cmd/service-name/main.gotmpl" // embedded FS is following linux styles path so we don't need filepath.Join
	dstMainPath := filepath.Join(appDstPath, "cmd", appName, "main.go")

	if err := fillAndCopyTemplateFile(srcMainPath, dstMainPath, config); err != nil {
		return err
	}

	srcDepPath := "template/internal/app/modules.gotmpl"
	dstDepPath := filepath.Join(appDstPath, "internal", "app", "modules.go")

	if err := fillAndCopyTemplateFile(srcDepPath, dstDepPath, config); err != nil {
		return err
	}

	if err := runGoFmt(appDstPath); err != nil {
		return err
	}

	utils.Log(fmt.Sprintf("Modules updated successfully for \"%v\"", appName), utils.SUCCESS)
	return nil
}
