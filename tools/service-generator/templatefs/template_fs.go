// Package templatefs wraps and provide the filesystem for text template
package templatefs

import (
	"embed"
	"io/fs"
	"os"
)

//go:embed template/*
var embeddedFS embed.FS

// TemplateFS provides an interface for template filesystem
type TemplateFS interface {
	GetFileContents(string) (string, error)
	GetFS() fs.FS
}

// ServiceTemplate provides access to the embedded filesystem
type ServiceTemplate struct{}

// GetFileContents returns the string file contents from templateFS
func (st ServiceTemplate) GetFileContents(path string) (string, error) {
	byteContents, err := embeddedFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(byteContents), nil
}

// GetFS returns the embedded template filesystem
func (st ServiceTemplate) GetFS() fs.FS {
	return embeddedFS
}

// GetFilesByRoot returns all the files inside root directory including
// files recursively from child directories in sorted order by directory.
func (st ServiceTemplate) GetFilesByRoot(root string) ([]string, error) {
	files := []string{}

	fileAggregator := func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	}

	if err := fs.WalkDir(embeddedFS, root, fileAggregator); err != nil {
		return files, err
	}
	return files, nil
}
