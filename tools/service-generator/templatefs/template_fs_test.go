// Package templatefs wraps and provide the filesystem for text template
package templatefs

import (
	"embed"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

var ContainsAtLeast = []string{
	"template/cmd/service-name/main.gotmpl",
	"template/internal/app/app.gotmpl",
	"template/internal/app/modules.gotmpl",
	"template/internal/db/storage.gotmpl",
	"template/helm/Chart.yaml",
}

func TestServiceTemplate_embeddedFS(t *testing.T) {
	err := fstest.TestFS(embeddedFS, ContainsAtLeast...)
	assert.NoError(t, err)
}

func TestServiceTemplate_GetFileContents(t *testing.T) {
	st := ServiceTemplate{}

	_, err := st.GetFileContents("template/cmd/service-name/main.gotmpl")
	assert.NoError(t, err)

	_, err = st.GetFileContents("template/cmd/not-existing.gotmpl")
	assert.Error(t, err)
}

func TestServiceTemplate_GetFilesByRoot(t *testing.T) {
	st := ServiceTemplate{}

	files, err := st.GetFilesByRoot("template")
	assert.NoError(t, err)
	assert.Subset(t, files, ContainsAtLeast)

	_, err = st.GetFilesByRoot("notExisting")
	assert.Error(t, err)
}

func TestServiceTemplate_GetFS(t *testing.T) {
	st := ServiceTemplate{}

	gotFS := st.GetFS()
	assert.IsType(t, embed.FS{}, gotFS)
}
