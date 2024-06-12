package credentialreader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
)

//nolint:gosec
const appCredentialProviderURLPath = "%s/api/sch/v1/credentials"

// DefaultServiceAccountTokenPath ...
//
//nolint:gosec
var DefaultServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

type AppCredentialResponse struct {
	Username       string          `json:"username"`
	Password       string          `json:"password"`
	AdditionalData json.RawMessage `json:"additionalData"`
}

type AppCredentialRequest struct {
	Username       string             `json:"username"`
	EntityName     string             `json:"entityName"`
	AdditionalInfo json.RawMessage    `json:"additionalData"`
	AuthData       AuthenticationData `json:"authData"`
}

type AuthenticationData struct {
	Token string `json:"token"`
}

const httpTimeout = 30 * time.Second

// Static errors for credentials reader.
var (
	ErrEmptyUsernameOrInfo = errors.New("request has to contain either UserName or AdditionalInfo")
	ErrEmptyEntityName     = errors.New("empty entity name")
	ErrEmptyToken          = errors.New("empty authentication token")
	ErrParseConfigFromEnv  = errors.New("failed to read config from env")
)

// GetAppCredentials ... Get the specific OEM credentials of the service.
// The caller service should have kubernetes service account defined.
// Service should define
//
//		url - Central service URL
//	 credentialRequest - Credential Request
func GetAppCredentials(ctx context.Context, url string, credentialRequest AppCredentialRequest) (appCredential *AppCredentialResponse, err error) {
	if url == "" {
		return nil, fmt.Errorf("URL should not be empty")
	}

	if err = validate(credentialRequest); err != nil {
		return
	}

	httpClient := &http.Client{Timeout: httpTimeout}

	appCredentialProviderURL := fmt.Sprintf(appCredentialProviderURLPath, url)
	reqestBody, err := json.Marshal(credentialRequest)

	resp, err := httpClient.Post(appCredentialProviderURL, "application/json", bytes.NewBuffer(reqestBody))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		err = fmt.Errorf("Credentials could not be fetched for Request  : %#v ", credentialRequest)
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("App credential provider responded with http code: %d for %s", resp.StatusCode, url)
		return
	}

	appCredential = &AppCredentialResponse{}
	if err = json.NewDecoder(resp.Body).Decode(appCredential); err != nil {
		return nil, fmt.Errorf("could not decode credentials: %s", err)
	}

	return appCredential, nil
}

// ReadServiceAccountToken Read token from the path.
// path is optional parameter, if not provided token will be picked up from default path.
func ReadServiceAccountToken(paths ...string) (s string, err error) {
	tokenPath := os.Getenv("SERVICE_ACCOUNT_TOKEN_PATH")
	if tokenPath == "" {
		tokenPath = DefaultServiceAccountTokenPath
	}

	if len(paths) > 0 {
		tokenPath = paths[0]
	}

	b, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		return
	}

	s = string(b)
	return
}

// FetchCredentialsForUser reads required information from env and fetches credentials.
func FetchCredentialsForUser(username string) (AppCredentialResponse, error) {
	credConf := struct {
		EntityName               string `envconfig:"ENTITY_NAME" required:"true"`
		AppCredentialProviderURL string `envconfig:"APP_CREDENTIAL_PROVIDER_URL" required:"true"`
		ServiceAccountTokenPath  string `envconfig:"SERVICE_ACCOUNT_TOKEN_PATH" required:"false" default:"/var/run/secrets/kubernetes.io/serviceaccount/token"`
	}{}
	if err := envconfig.Process("", &credConf); err != nil {
		return AppCredentialResponse{}, fmt.Errorf("%w: %s", ErrParseConfigFromEnv, err)
	}

	jwt, err := ReadServiceAccountToken(credConf.ServiceAccountTokenPath)
	if err != nil {
		return AppCredentialResponse{}, fmt.Errorf("error reading service account token: %w", err)
	}

	req := AppCredentialRequest{
		Username:   username,
		EntityName: credConf.EntityName,
		AuthData:   AuthenticationData{Token: jwt},
	}

	cred, err := GetAppCredentials(context.Background(), credConf.AppCredentialProviderURL, req)
	if err != nil {
		return AppCredentialResponse{}, err
	}
	return *cred, nil
}

func validate(credentialRequest AppCredentialRequest) error {
	if credentialRequest.Username == "" && len(credentialRequest.AdditionalInfo) == 0 {
		return ErrEmptyUsernameOrInfo
	}
	if credentialRequest.EntityName == "" {
		return ErrEmptyEntityName
	}
	if credentialRequest.AuthData.Token == "" {
		return ErrEmptyToken
	}
	return nil
}
