package httpmiddleware_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phanitejak/kptgolib/httpmiddleware"
	"github.com/phanitejak/kptgolib/logging"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	swaggerJSON = `{
	"components": {
		"schemas": {
			"mockData": {
				"description": "mockData description",
				"type": "string"
			}
		}
	},
	"info": {
		"description": "Provides API for mock-api",
		"title": "mock-api",
		"version": "1.0.0"
	},
	"openapi": "3.0.3",
	"paths": {
		"/api/v1/test": {
			"post": {
				"operationId": "postMockApiTest",
				"requestBody": {
					"content": {
						"application/json": {
							"schema": {
								"$ref": "#/components/schemas/mockData"
							}
						}
					},
					"required": true
				},
				"responses": {
					"200": {
						"description": "OK"
					},
					"400": {
						"content": {
							"application/json": {}
						},
						"description": "Bad request"
					}
				}
			}
		}
	},
	"servers": [{
		"url": "/"
	}]
}`
)

func getSwaggerFnc() (*openapi3.T, error) {
	return openapi3.NewLoader().LoadFromData([]byte(swaggerJSON))
}

func TestNew_validation(t *testing.T) {
	testCases := []struct {
		name           string
		method         string
		url            string
		contentType    string
		body           io.Reader
		expectedStatus int
	}{
		{
			name:           "invalid path",
			method:         http.MethodPost,
			url:            "/invalidUrl",
			contentType:    "application/json",
			body:           nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			url:            "/api/v1/test",
			contentType:    "application/json",
			body:           nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid payload",
			method:         http.MethodPost,
			url:            "/api/v1/test",
			contentType:    "application/json",
			body:           bytes.NewBuffer([]byte("5")),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid Content-Type",
			method:         http.MethodPost,
			url:            "/api/v1/test",
			contentType:    "text/csv",
			body:           bytes.NewBuffer([]byte(`"5"`)),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid request",
			method:         http.MethodPost,
			url:            "/api/v1/test",
			contentType:    "application/json",
			body:           bytes.NewBuffer([]byte(`"5"`)),
			expectedStatus: http.StatusOK,
		},
	}

	h, err := httpmiddleware.New(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
	}), getSwaggerFnc)
	require.NoError(t, err)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, tc.url, tc.body)
			r.Header.Add("Content-Type", tc.contentType)

			h.ServeHTTP(w, r)
			resp := w.Result()
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			assert.NoError(t, resp.Body.Close())
		})
	}
}

func TestNew_getSwaggerError(t *testing.T) {
	expectedErr := errors.New("swagger error")
	got, err := httpmiddleware.New(http.DefaultServeMux, func() (*openapi3.T, error) { return nil, expectedErr })
	assert.Nil(t, got)
	assert.ErrorIs(t, err, expectedErr)
}

func TestNewWithValidationOptions_skippingOfValidations(t *testing.T) {
	testCases := []struct {
		name              string
		method            string
		url               string
		contentType       string
		body              io.Reader
		validationOptions openapi3filter.Options
		expectedStatus    int
	}{
		{
			name:        "invalid body causes validation error",
			method:      http.MethodPost,
			url:         "/api/v1/test",
			contentType: "application/json",
			body:        bytes.NewBuffer([]byte("5")),
			validationOptions: openapi3filter.Options{
				ExcludeRequestBody:    false,
				ExcludeResponseBody:   false,
				IncludeResponseStatus: false,
				AuthenticationFunc:    nil,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "invalid body, but validation skipped",
			method:      http.MethodPost,
			url:         "/api/v1/test",
			contentType: "application/json",
			body:        bytes.NewBuffer([]byte("5")),
			validationOptions: openapi3filter.Options{
				ExcludeRequestBody:    true,
				ExcludeResponseBody:   false,
				IncludeResponseStatus: false,
				AuthenticationFunc:    nil,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		tc := tc
		h, err := httpmiddleware.NewWithValidationOptions(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		}), getSwaggerFnc, &tc.validationOptions)
		require.NoError(t, err)
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, tc.url, tc.body)
			r.Header.Add("Content-Type", tc.contentType)

			h.ServeHTTP(w, r)
			resp := w.Result()
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			assert.NoError(t, resp.Body.Close())
		})
	}
}

func TestGetApiDocs(t *testing.T) {
	log := logging.NewLogger()
	w := httptest.NewRecorder()

	httpmiddleware.GetApiDocs(w, nil, []byte(swaggerJSON), log)
	resp := w.Result()
	assertAPIDocsRespOK(t, resp)
	assert.NoError(t, resp.Body.Close())
}

func assertAPIDocsRespOK(t *testing.T, resp *http.Response) {
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	respBody, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.True(t, json.Valid(respBody), string(respBody))
	assert.NoError(t, resp.Body.Close())
}

func TestGetStatus(t *testing.T) {
	w := httptest.NewRecorder()

	httpmiddleware.GetStatus(w, nil)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, resp.Body.Close())
}

func TestGetSwaggerJSON(t *testing.T) {
	getSwaggerFncError := errors.New("GetSwaggerFnc error")

	testCases := []struct {
		name           string
		getSwaggerFunc httpmiddleware.GetSwaggerFnc
		wantErr        error
	}{
		{
			name:           "GetSwaggerJSON fails when calling GetSwaggerFnc",
			getSwaggerFunc: func() (*openapi3.T, error) { return nil, errors.New("GetSwaggerFnc error") },
			wantErr:        fmt.Errorf("unable to get swagger: %w", getSwaggerFncError),
		},
		{
			name:           "successful GetSwaggerJSON call",
			getSwaggerFunc: getSwaggerFnc,
			wantErr:        nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := httpmiddleware.GetSwaggerJSON(tc.getSwaggerFunc)
			if tc.wantErr != nil {
				assert.Equal(t, err.Error(), tc.wantErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
