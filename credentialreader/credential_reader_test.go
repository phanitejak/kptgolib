package credentialreader_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg/credentialreader"
)

func TestFetchCredentialsForUser(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		entity      string
		tokenPath   string
		respBody    string
		expectResp  credentialreader.AppCredentialResponse
		expectedErr error
	}{
		{
			name:      "Success",
			username:  "username",
			entity:    "entity",
			tokenPath: "testdata/token",
			respBody:  `{"username":"name", "password":"passwd", "additionalData":"data"}`,
			expectResp: credentialreader.AppCredentialResponse{
				Username:       "name",
				Password:       "passwd",
				AdditionalData: json.RawMessage(`"data"`),
			},
		},
		{
			name:        "EmptyEntityName",
			username:    "username",
			entity:      "",
			tokenPath:   "testdata/token",
			respBody:    `{"username":"name", "password":"passwd", "additionalData":"data"}`,
			expectedErr: credentialreader.ErrEmptyEntityName,
		},
		{
			name:        "EmptyUsername",
			username:    "",
			entity:      "entity",
			tokenPath:   "testdata/token",
			respBody:    `{"username":"name", "password":"passwd", "additionalData":"data"}`,
			expectedErr: credentialreader.ErrEmptyUsernameOrInfo,
		},
		{
			name:        "EmptyToken",
			username:    "username",
			entity:      "entity",
			tokenPath:   "testdata/empty",
			respBody:    `{"username":"name", "password":"passwd", "additionalData":"data"}`,
			expectedErr: credentialreader.ErrEmptyToken,
		},
		{
			name:        "TokenNotFound",
			username:    "username",
			entity:      "entity",
			tokenPath:   "testdata/not_a_file",
			respBody:    `{"username":"name", "password":"passwd", "additionalData":"data"}`,
			expectedErr: os.ErrNotExist,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, tt.respBody) })) //nolint
			require.NoError(t, os.Setenv("APP_CREDENTIAL_PROVIDER_URL", srv.URL))
			require.NoError(t, os.Setenv("SERVICE_ACCOUNT_TOKEN_PATH", tt.tokenPath))
			require.NoError(t, os.Setenv("ENTITY_NAME", tt.entity))

			resp, err := credentialreader.FetchCredentialsForUser(tt.username)
			assert.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.expectResp, resp)
		})
	}
}

func TestFetchCredentialsForUser_ErrParseConfigFromEnv(t *testing.T) {
	require.NoError(t, os.Unsetenv("APP_CREDENTIAL_PROVIDER_URL"))
	require.NoError(t, os.Unsetenv("SERVICE_ACCOUNT_TOKEN_PATH"))
	require.NoError(t, os.Unsetenv("ENTITY_NAME"))

	resp, err := credentialreader.FetchCredentialsForUser("username")
	assert.ErrorIs(t, err, credentialreader.ErrParseConfigFromEnv)
	assert.Equal(t, credentialreader.AppCredentialResponse{}, resp)
}
