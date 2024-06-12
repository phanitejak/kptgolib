package httpmiddleware

import (
	"fmt"
	"net/http"

	"gopkg/logging"
)

// This file contains ready made handling of common endpoints, one can use them directly

// GetStatus serves /status.
func GetStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// GetApiDocs serves /api-docs
// nolint:golint,stylecheck
func GetApiDocs(w http.ResponseWriter, r *http.Request, swaggerJSON []byte, log logging.Logger) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(swaggerJSON)
	if err != nil {
		log.Fatalf("failed to write swagger to response: %s", err)
	}
}

// GetSwaggerJSON returns swagger as bytes, used in GetApiDocs.
func GetSwaggerJSON(fnc GetSwaggerFnc) ([]byte, error) {
	swagger, err := fnc()
	if err != nil {
		return []byte{}, fmt.Errorf("unable to get swagger: %s", err)
	}
	swaggerJSON, err := swagger.MarshalJSON()
	if err != nil {
		return []byte{}, fmt.Errorf("failed to marshall swagger: %s", err)
	}
	return swaggerJSON, nil
}

// GetApiDocsV2 serves /api-docs
// nolint
func GetApiDocsV2(w http.ResponseWriter, apiGetSwaggerFn GetSwaggerFnc, logFatal func(string, ...interface{})) {
	swaggerJSON, err := GetSwaggerJSON(apiGetSwaggerFn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeAPIDocs(w, swaggerJSON, logFatal)
}

func writeAPIDocs(w http.ResponseWriter, swaggerJSON []byte, logErrorf func(string, ...interface{})) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(swaggerJSON)
	if err != nil {
		logErrorf("failed to write swagger to response: %s", err)
	}
}
