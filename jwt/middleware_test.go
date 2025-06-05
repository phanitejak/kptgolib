package jwt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var jwtPayloadJSON = `{
  "exp": 0,
  "nbf": 0,
  "resource_access": {
    "UM_SCOPE_WorkingSets": {
      "roles": [
        "WS-1",
        "WS-2",
        "WS-3",
        "WS-4"
      ]
    },
    "Some_Other_Resource": {
      "roles": [
        "manage-account",
        "manage-account-links",
        "view-profile"
      ]
    }
  }
}`

var jwtSingleClaimPayloadJSON = `{
  "exp": 0,
  "nbf": 0,
  "resource_access": {
    "UM_SCOPE_WorkingSets": {
      "roles": [
        "WS-1"
      ]
    }
  }
}`

var jwtEmptyClaimListPayloadJSON = `{
  "exp": 0,
  "nbf": 0,
  "resource_access": {
    "UM_SCOPE_WorkingSets": {
      "roles": [

      ]
    }
  }
}`

var jwtStringSigned = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjAsIm5iZiI6MCwicmVzb3VyY2VfYWNjZXNzIjp7IlVNX1NDT1BFX1dvcmtpbmdTZXRzIjp7InJvbGVzIjpbIldTLTEiLCJXUy0yIiwiV1MtMyIsIldTLTQiXX0sIlNvbWVfT3RoZXJfUmVzb3VyY2UiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX19.KD9BZ_pD1f0XwHl24zFhAVP_wMWuWQAzJdCb9vMlv_tkWWlqJ1f98cTRgcv8T48BSAluhBVDW4vY7JU0bZ7tNvn-09lpvNdhz8w1Gn3syY641pRBBkDB-dt_60QQTNjzZ_GNiRE-cs3rg5JcT590RPqgqx643SgNC6jnl7JmHEjor0OxyJxztJO1DbEUk09eEKArG8cGL6jRAnPBwl-dtUMiHlg7uhkeYdOwCTXbKeLMPUoBq-Hqd4phfhbZit8D_UNLh_tv9kY41azmNiBK5ofuy2C-AU_tE9V5oiITEj6MI0EwRo9vizmYZ5P-WykjbBF474jUer5RrfwG0JkTJQ`

type contextKey struct {
	name string
}

type tokenKey struct{}

var (
	otherKey        = contextKey{"other"}
	scopesKey       = contextKey{"scopes"}
	tokenContextKey = tokenKey{}
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name                       string
		options                    []func(conf) (conf, error)
		givenKey                   any
		authHeader                 string
		expectedResponseBody       any
		expectedResponseStatusCode int
		expectTokenStoredInContext bool
		assertValue                func(t *testing.T, value any)
	}{
		{
			name:                       "with defaults",
			options:                    nil,
			givenKey:                   nil,
			authHeader:                 "",
			expectedResponseBody:       "no Authorization header found in request",
			expectedResponseStatusCode: http.StatusBadRequest,
			assertValue: func(t *testing.T, value any) {
				assert.Nil(t, value)
			},
			expectTokenStoredInContext: false,
		},
		{
			name:                       "given invalid token",
			options:                    nil,
			givenKey:                   nil,
			authHeader:                 "Bearer invalid_jwt",
			expectedResponseBody:       "failed to decode a bearer token",
			expectedResponseStatusCode: http.StatusBadRequest,
			assertValue: func(t *testing.T, value any) {
				assert.Nil(t, value)
			},
			expectTokenStoredInContext: false,
		},
		{
			name:                       "given token is not a valid json",
			options:                    nil,
			givenKey:                   nil,
			authHeader:                 "Bearer ignored_header_value..ignored_signature",
			expectedResponseBody:       "token is not a valid json",
			expectedResponseStatusCode: http.StatusBadRequest,
			assertValue: func(t *testing.T, value any) {
				assert.Nil(t, value)
			},
			expectTokenStoredInContext: false,
		},
		{
			name: "ignoring token",
			options: []func(conf) (conf, error){
				WithRequiredToken(false),
			},
			givenKey:                   nil,
			authHeader:                 "Bearer ignored_header_value..ignored_signature",
			expectedResponseBody:       "",
			expectedResponseStatusCode: http.StatusOK,
			assertValue: func(t *testing.T, value any) {
				assert.Nil(t, value)
			},
			expectTokenStoredInContext: false,
		},
		{
			name: "with custom error handler",
			options: []func(conf) (conf, error){
				WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
					w.WriteHeader(http.StatusUnauthorized)
					b, _ := json.Marshal(map[string]any{
						"error": err.Error(),
					})
					_, _ = w.Write(b)
				}),
			},
			givenKey:                   nil,
			authHeader:                 "Bearer invalid_jwt",
			expectedResponseBody:       `{"error":"failed to decode a bearer token"}`,
			expectedResponseStatusCode: http.StatusUnauthorized,
			assertValue: func(t *testing.T, value any) {
				assert.Nil(t, value)
			},
			expectTokenStoredInContext: false,
		},
		{
			name: "with custom error handler and non existing claims",
			options: []func(conf) (conf, error){
				WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
					w.WriteHeader(http.StatusUnauthorized)
					b, _ := json.Marshal(map[string]any{
						"error": err.Error(),
					})
					_, _ = w.Write(b)
				}),
				WithClaimsToExtract(map[string]any{"non.existing.json.path": otherKey}),
				WithStoredTokenInContext(tokenContextKey),
			},
			givenKey:                   "some_claim",
			authHeader:                 "Bearer ignored." + base64.RawURLEncoding.EncodeToString([]byte(`{"some_claim": "some_value"}`)) + ".ignored",
			expectedResponseBody:       `{"error":"expected claim does not exist in the token"}`,
			expectedResponseStatusCode: http.StatusUnauthorized,
			assertValue: func(t *testing.T, value any) {
				assert.Nil(t, value)
			},
			expectTokenStoredInContext: true,
		},
		{
			name: "with other non existing claims",
			options: []func(conf) (conf, error){
				WithClaimsToExtract(map[string]any{"non.existing.json.path": otherKey}),
				WithStoredTokenInContext(tokenContextKey),
			},
			givenKey:                   "some_other_claim",
			authHeader:                 "Bearer ignored." + base64.RawURLEncoding.EncodeToString([]byte(`{"some_claim": "some_value"}`)) + ".ignored",
			expectedResponseBody:       `expected claim does not exist in the token`,
			expectedResponseStatusCode: http.StatusBadRequest,
			assertValue: func(t *testing.T, value any) {
				assert.Nil(t, value)
			},
			expectTokenStoredInContext: true,
		},
		{
			name: "with existing claims",
			options: []func(conf) (conf, error){
				WithClaimsToExtract(map[string]any{"resource_access.UM_SCOPE_WorkingSets.roles": scopesKey}),
				WithStoredTokenInContext(tokenContextKey),
			},
			givenKey:                   scopesKey,
			authHeader:                 "Bearer ignored." + base64.RawURLEncoding.EncodeToString([]byte(jwtPayloadJSON)) + ".ignored",
			expectedResponseBody:       "",
			expectedResponseStatusCode: http.StatusOK,
			assertValue: func(t *testing.T, value any) {
				require.NotNil(t, value)

				var actual []string
				for _, v := range value.([]any) {
					actual = append(actual, v.(string))
				}

				assert.Equal(t, []string{"WS-1", "WS-2", "WS-3", "WS-4"}, actual)
			},
			expectTokenStoredInContext: true,
		},
		{
			name: "with single claim im the list",
			options: []func(conf) (conf, error){
				WithClaimsToExtract(map[string]any{"resource_access.UM_SCOPE_WorkingSets.roles": scopesKey}),
			},
			givenKey:                   scopesKey,
			authHeader:                 "Bearer ignored." + base64.RawURLEncoding.EncodeToString([]byte(jwtSingleClaimPayloadJSON)) + ".ignored",
			expectedResponseBody:       "",
			expectedResponseStatusCode: http.StatusOK,
			assertValue: func(t *testing.T, value any) {
				require.NotNil(t, value)

				var actual []string
				for _, v := range value.([]any) {
					actual = append(actual, v.(string))
				}

				assert.Equal(t, []string{"WS-1"}, actual)
			},
			expectTokenStoredInContext: false,
		},
		{
			name: "with empty claim list",
			options: []func(conf) (conf, error){
				WithClaimsToExtract(map[string]any{"resource_access.UM_SCOPE_WorkingSets.roles": scopesKey}),
				WithStoredTokenInContext(tokenContextKey),
			},
			givenKey:                   scopesKey,
			authHeader:                 "Bearer ignored." + base64.RawURLEncoding.EncodeToString([]byte(jwtEmptyClaimListPayloadJSON)) + ".ignored",
			expectedResponseBody:       "",
			expectedResponseStatusCode: http.StatusOK,
			assertValue: func(t *testing.T, value any) {
				require.NotNil(t, value)

				var actual []string
				for _, v := range value.([]any) {
					actual = append(actual, v.(string))
				}

				assert.Equal(t, []string(nil), actual)
			},
			expectTokenStoredInContext: true,
		},
		{
			name: "with lower case Bearer string",
			options: []func(conf) (conf, error){
				WithClaimsToExtract(map[string]any{"resource_access.UM_SCOPE_WorkingSets.roles": scopesKey}),
			},
			givenKey:                   scopesKey,
			authHeader:                 "bearer ignored." + base64.RawURLEncoding.EncodeToString([]byte(jwtPayloadJSON)) + ".ignored",
			expectedResponseBody:       "",
			expectedResponseStatusCode: http.StatusOK,
			assertValue: func(t *testing.T, value any) {
				require.NotNil(t, value)

				var actual []string
				for _, v := range value.([]any) {
					actual = append(actual, v.(string))
				}

				assert.Equal(t, []string{"WS-1", "WS-2", "WS-3", "WS-4"}, actual)
			},
			expectTokenStoredInContext: false,
		},
		{
			name: "with RS256 signature verification",
			options: []func(conf) (conf, error){
				WithClaimsToExtract(map[string]any{"resource_access.UM_SCOPE_WorkingSets.roles": scopesKey}),
				WithCertificatePem(certPem),
				func(c conf) (conf, error) {
					c.signatureVerificationIsEnabled = true
					return c, nil
				},
			},
			givenKey:                   scopesKey,
			authHeader:                 fmt.Sprintf("bearer %s", jwtStringSigned),
			expectedResponseBody:       "",
			expectedResponseStatusCode: http.StatusOK,
			assertValue: func(t *testing.T, value any) {
				require.NotNil(t, value)
				var actual []string
				for _, v := range value.([]any) {
					actual = append(actual, v.(string))
				}

				assert.Equal(t, []string{"WS-1", "WS-2", "WS-3", "WS-4"}, actual)
			},
			expectTokenStoredInContext: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mw, err := NewMiddleware(test.options...)
			require.NoError(t, err)

			var hf http.HandlerFunc = func(writer http.ResponseWriter, r *http.Request) {
				value := r.Context().Value(test.givenKey)
				test.assertValue(t, value)
				if test.expectTokenStoredInContext {
					tokenValue := r.Context().Value(tokenContextKey)
					require.NotNil(t, tokenValue)
					assert.Equal(t, bytes.TrimSpace([]byte(test.authHeader)[6:]), []byte(tokenValue.(string)))
				}
			}

			router := httprouter.New()
			router.Handler("GET", "/someUrl", mw.Handler(hf))

			server := httptest.NewServer(router)
			defer server.Close()

			request, err := http.NewRequest("GET", server.URL+"/someUrl", nil)
			require.NoError(t, err)

			if test.authHeader != "" {
				request.Header.Add("Authorization", test.authHeader)
			}

			response, err := server.Client().Do(request)
			require.NoError(t, err)

			assert.Equal(t, test.expectedResponseStatusCode, response.StatusCode)
			b, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)

			assert.Equal(t, test.expectedResponseBody, string(b))
			err = response.Body.Close()
			require.NoError(t, err)
		})
	}
}

const certPem = `-----BEGIN CERTIFICATE-----
MIIE2DCCAsCgAwIBAgIUbb/8tz2Hcbwko7xnSPSdDXOHyPMwDQYJKoZIhvcNAQEL
BQAwbTELMAkGA1UEBhMCRkkxEDAOBgNVBAgMB0ZpbmxhbmQxEDAOBgNVBAcMB1Rh
bXBlcmUxDjAMBgNVBAoMBU5va2lhMQ0wCwYDVQQLDARjbm1zMRswGQYDVQQDDBJO
b2tpYSBjbm1zIFJvb3QgQ0EwHhcNMTkxMDE2MDkyNTUyWhcNMjQxMDEzMjIyNjIx
WjAlMSMwIQYDVQQDExpuZW8wMDg2LmR5bi5uZXNjLm5va2lhLm5ldDCCASIwDQYJ
KoZIhvcNAQEBBQADggEPADCCAQoCggEBALF1eQ7RU9OlLoBCB7Gdy/u13U7CrIk8
gdFQOF6TMX4A7nL/ERefiXLRiE50kRyVKmKT5jXPQpWy8LzKQcIRN7gGfxZcMJoq
pZQr1m4Zy7tNv8OQ3h4r5eN51ufprZr0F1DWdyzb8A0Edw4JcFknblzTXS666Ddt
Nr2CMTVoZDQp03yG7a/jhcMb4zr01X+waJs/tMBnQScJAEHBWM+CaiurflWbhgni
MJTh2zEX0jiWfyQETf66smmxkfhmZokC2y0qWmIc3Y1M3CVQSVNn4RGurTWxejAk
ErsldRkj3dPVaTAxhsSxhpHOzuhK618IBc1zk1FIN1C+hjGEjWnJR1sCAwEAAaOB
tzCBtDAOBgNVHQ8BAf8EBAMCA6gwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUF
BwMCMB0GA1UdDgQWBBR3aMdAViAoql3lffEWj0Q0M1RNSzAfBgNVHSMEGDAWgBT1
PoBNiSMHbp/HuNSfqPESqGL2bTBDBgNVHREEPDA6ghpuZW8wMDg2LmR5bi5uZXNj
Lm5va2lhLm5ldIIcKi5uZW8wMDg2LmR5bi5uZXNjLm5va2lhLm5ldDANBgkqhkiG
9w0BAQsFAAOCAgEAdfh9eapzzniMSbLGZWEe0HHrhvZIXIJNy6tYHY7BB/fruYY8
HBlLZ3NB1LWaeqL+exy8gtkr4IrlzEz2Ygwn25My7SFo3tfcmH6/wyfrDwyWaXV/
eNmJ7Bc25R1nbLCB1z3jK+vtQ5I4q4AeLL3PiN7UDnTOzmfRTkHGHemP1M+NT8k+
tkyaUxJNwU/cMnwj0+QgwIcVQjWRLdei+UnfqoWoMQwuunqn1d/p7tviPO1G1yL6
9Yl1j9rfs1K/tvVY4v5q3yrvaRd0T7YmFWl4gjNy2nK551e7Ivna2bzysRVtxWZ4
7zdcesmQIQYDrjOBhpv/oHMGmyUDfxYL5KQtVp4051aPUlCwpmaQtgnGP/ZBrxde
Gg/rO+fs5TBADCkMdm9aOEaJVUSqaYFJY8XyEX2zIfuE7VFx+dHQPwixOyHmxDWa
d63WgZRGOHpgTQbfCDXmQEBgU4XQW8SqmzUby0P8rCNPjqn4gH6BcSNCO6h2GBYp
bT8hVKPOVEqI2J0PP4c5Q7/17D7wvblCQqyIHXnxl3mQDfDgxXIf0n7XVbejaPiO
+2KtsOohSmjJqb/2bwkQqUbYHSPg9iTHgo22MnoY41pleHerpuCbRc5N7zLn9sIP
yD2aGAXVuVa+69AWGZaGuuGNLLkHpfv4M6elkn9vlx3Hx6HdIRjsrpiB+vA=
-----END CERTIFICATE-----`
