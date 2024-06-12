package vault

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_tokenSetAndDeleteMethodUsed_whenDeletingSecret(t *testing.T) {
	handler := &mockVaultHandler{}
	mockVaultServer := httptest.NewServer(handler)
	mockVaultServerURL := mockVaultServer.URL
	defer mockVaultServer.Close()

	tokenClient, err := NewSimpleTokenClient(mockVaultServerURL, "helloToken")
	require.NoError(t, err)

	_, err = tokenClient.Delete("toBeDeleted")
	require.NoError(t, err)

	assert.NotNil(t, handler.capturedRequest, "Request should have been captured, but nothing came")
	assert.Equal(t, "DELETE", handler.capturedRequest.Method)
	assert.Equal(t, "helloToken", handler.capturedRequest.Header.Get("X-Vault-Token"))
	assert.Equal(t, "/v1/toBeDeleted", handler.capturedRequest.URL.Path)
}

func Test_tokenSetAndPutMethodUsed_whenUpdatingSecret(t *testing.T) {
	handler := &mockVaultHandler{}
	mockVaultServer := httptest.NewServer(handler)
	mockVaultServerURL := mockVaultServer.URL
	defer mockVaultServer.Close()

	tokenClient, err := NewSimpleTokenClient(mockVaultServerURL, "helloToken")
	require.NoError(t, err)

	_, err = tokenClient.Write("/my/path", map[string]interface{}{"hello": "world"})
	require.NoError(t, err)

	assert.NotNil(t, handler.capturedRequest, "Request should have been captured, but nothing came")
	assert.Equal(t, "PUT", handler.capturedRequest.Method)
	assert.Equal(t, "helloToken", handler.capturedRequest.Header.Get("X-Vault-Token"))
	assert.Equal(t, "/v1/my/path", handler.capturedRequest.URL.Path)
}

func Test_tokenSetAndGetMethodUsed_whenReadingSecret(t *testing.T) {
	mockVaultServerHandler := &mockVaultHandler{}
	mockVaultServer := httptest.NewServer(mockVaultServerHandler)
	mockVaultServerURL := mockVaultServer.URL
	defer mockVaultServer.Close()

	tokenClient, err := NewSimpleTokenClient(mockVaultServerURL, "helloToken")
	require.NoError(t, err)

	mockVaultServerHandler.bodyToWrite = []byte(`{"data":{"hello":"world"}}`)

	secret, err := tokenClient.Read("/my/path")
	require.NoError(t, err)

	assert.NotNil(t, mockVaultServerHandler.capturedRequest, "Request should have been captured, but nothing came")
	assert.Equal(t, "GET", mockVaultServerHandler.capturedRequest.Method)
	assert.Equal(t, "helloToken", mockVaultServerHandler.capturedRequest.Header.Get("X-Vault-Token"))
	assert.Equal(t, "/v1/my/path", mockVaultServerHandler.capturedRequest.URL.Path)
	assert.Equal(t, "world", secret.Data["hello"])
}

func Test_tokenSetGetMethodUsedAndSpecialListQueryParamSet_whenListingSecrets(t *testing.T) {
	mockVaultServerHandler := &mockVaultHandler{}
	mockVaultServer := httptest.NewServer(mockVaultServerHandler)
	mockVaultServerURL := mockVaultServer.URL
	defer mockVaultServer.Close()

	tokenClient, err := NewSimpleTokenClient(mockVaultServerURL, "helloToken")
	require.NoError(t, err)

	mockVaultServerHandler.bodyToWrite = []byte(`{"data":{"hello":"world"}}`)

	secret, err := tokenClient.List("/my/path")
	require.NoError(t, err)

	assert.NotNil(t, mockVaultServerHandler.capturedRequest, "Request should have been captured, but nothing came")
	assert.Equal(t, "GET", mockVaultServerHandler.capturedRequest.Method)
	assert.Equal(t, "helloToken", mockVaultServerHandler.capturedRequest.Header.Get("X-Vault-Token"))
	assert.Equal(t, "/v1/my/path", mockVaultServerHandler.capturedRequest.URL.Path)
	assert.Equal(t, "world", secret.Data["hello"])
	assert.Equal(t, "list=true", mockVaultServerHandler.capturedRequest.URL.RawQuery)
}

type mockVaultHandler struct {
	capturedRequest *http.Request
	bodyToWrite     []byte
}

func (mh *mockVaultHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	mh.capturedRequest = req
	rw.WriteHeader(200)

	if len(mh.bodyToWrite) > 0 {
		_, _ = rw.Write(mh.bodyToWrite)
	}
}
