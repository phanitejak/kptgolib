package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	osExecutable = os.Executable
	osGetwd      = os.Getwd
)

// ParseServiceName validates and parses the microservice name
func ParseServiceName(name string) (string, error) {
	trimmedName := strings.TrimSpace(name)
	if len(trimmedName) == 0 {
		return "", errors.New("microservice name is required. Use -h for help")
	}

	validNameRegex := regexp.MustCompile(`^[a-zA-Z0-9-\s]+$`)
	if !validNameRegex.MatchString(trimmedName) {
		return "", errors.New("only alphanumeric characters are allowed")
	}

	emptySpacesRegex := regexp.MustCompile(`[\s-]+`)
	parsedName := emptySpacesRegex.ReplaceAllString(trimmedName, "-")
	return strings.ToLower(parsedName), nil
}

// GetCurrentDirectoryPath returns path of the executable
func GetCurrentDirectoryPath() (string, error) {
	executablePath, err := osExecutable()
	if err != nil {
		return "", fmt.Errorf("could not get executable directory: %w", err)
	}
	return filepath.Dir(executablePath), nil
}

// GetSvcDirectoryPath returns "svc" directory path
func GetSvcDirectoryPath() (string, error) {
	workingDirectory, err := osGetwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory: %w", err)
	}
	svcPathFromWorkingDirectory := strings.SplitAfter(workingDirectory, "svc")[0]
	if svcPathFromWorkingDirectory != workingDirectory {
		return svcPathFromWorkingDirectory, nil
	}
	return filepath.Join(workingDirectory, "svc"), nil
}

// GetToolByName returns path to the tool executable by name
func GetToolByName(name string) (string, error) {
	workingDirectory, err := osGetwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory: %w", err)
	}
	toolsDirectory := filepath.Join(workingDirectory, ".tools")
	execPath := ""
	err = filepath.Walk(toolsDirectory, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, name) {
			execPath = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if execPath == "" {
		return "", errors.New("tool not found")
	}
	return execPath, nil
}

// Exists check if path exists or not
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

var execCommand = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output() //nolint:gosec
}

// GetGitInfo returns current user's name and email from git
func GetGitInfo() (string, string) {
	userNameOutput, userNameError := execCommand("git", "config", "user.name")
	userEmailOutput, userEmailError := execCommand("git", "config", "user.email")

	if userNameError != nil || userEmailError != nil {
		return "NOM", "firstname.lastname@nokia.com"
	}
	userName := strings.TrimSpace(string(userNameOutput))
	userEmail := strings.TrimSpace(string(userEmailOutput))

	return userName, userEmail
}
