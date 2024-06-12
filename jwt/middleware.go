// Package jwt_middleware provides HTTP middleware, able to extract and validate JWT from HTTP Headers.
// Middleware will extract provided claims and insert those into request context
package jwt

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/tidwall/gjson"
)

const numberOfJWTParts = 3

var (
	ErrNoAuthHeader   = errors.New("no Authorization header found in request")
	ErrNoBearerToken  = errors.New("no bearer token found in Authorization header")
	ErrDecodingBearer = errors.New("failed to decode a bearer token")
	ErrNotValidJSON   = errors.New("token is not a valid json")
	ErrClaimNotExists = errors.New("expected claim does not exist in the token")
)

type Middleware struct {
	c conf
}

type conf struct {
	// Claims, which will be extracted from JWT and put into request context.
	// Keys in the map are json path strings, parsable by github.com/tidwall/gjson library.
	// Values in the map will be used as keys for putting values into context.
	// Refer to https://golang.org/pkg/context/#WithValue for more details on context keys.
	claimsToExtract map[string]interface{}

	// set to true, if token must always be present in a request
	requireToken bool

	// set to true, if request should proceed, even when token processing fails
	ignoreErrors bool

	// if set to true, not existing claims will be ignored
	ignoreNotExistingClaim bool

	// error handling function
	// will be called when token processing fails
	errorHandle func(w http.ResponseWriter, r *http.Request, err error)

	// Trusted public key to verify JWT signature
	publicKey *rsa.PublicKey

	// Flag to verify key signature
	signatureVerificationIsEnabled bool

	// Context Key to store extracted bearer token in the request context
	// If tokenContextKey is nil - token will not be stored in the request context
	tokenContextKey interface{}
}

func WithClaimsToExtract(claimsToExtract map[string]interface{}) func(conf) (conf, error) {
	return func(c conf) (conf, error) {
		c.claimsToExtract = claimsToExtract
		return c, nil
	}
}

// A trusted certificate to verify JWT signature
// If present, token signature will be automatically verified.
func WithCertificatePem(certificatePem string) func(conf) (conf, error) {
	return func(c conf) (conf, error) {
		block, _ := pem.Decode([]byte(certificatePem))
		if block == nil {
			return c, errors.New("error parsing certificate pem")
		}

		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return c, err
		}

		publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
		if !ok {
			return c, errors.New("only rsa keys are supported")
		}

		c.publicKey = publicKey
		return c, nil
	}
}

func WithRequiredToken(requireToken bool) func(conf) (conf, error) {
	return func(c conf) (conf, error) {
		c.requireToken = requireToken
		return c, nil
	}
}

func WithIgnoreErrors(ignoreErrors bool) func(conf) (conf, error) {
	return func(c conf) (conf, error) {
		c.ignoreErrors = ignoreErrors
		return c, nil
	}
}

func WithErrorHandler(errorHandler func(w http.ResponseWriter, r *http.Request, err error)) func(conf) (conf, error) {
	return func(c conf) (conf, error) {
		c.errorHandle = errorHandler
		return c, nil
	}
}

func WithStoredTokenInContext(tokenContextKey interface{}) func(conf) (conf, error) {
	return func(c conf) (conf, error) {
		c.tokenContextKey = tokenContextKey
		return c, nil
	}
}

func NewMiddleware(options ...func(conf) (conf, error)) (Middleware, error) {
	c := conf{
		claimsToExtract:        map[string]interface{}{},
		requireToken:           true,
		ignoreErrors:           false,
		ignoreNotExistingClaim: false,
		errorHandle: func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, err)
		},
		publicKey: nil,
		// TODO: Add support for signature verification - use some library, write more tests and enable this flag
		signatureVerificationIsEnabled: false,
		tokenContextKey:                nil,
	}

	for _, option := range options {
		cTemp, err := option(c)
		if err != nil {
			return Middleware{}, err
		}
		c = cTemp
	}

	return Middleware{c: c}, nil
}

func (m Middleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := m.processToken(w, r)

		if err != nil && !m.c.ignoreErrors {
			m.c.errorHandle(w, r, err)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (m Middleware) Handle(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		err := m.processToken(w, r)

		if err != nil && !m.c.ignoreErrors {
			m.c.errorHandle(w, r, err)
			return
		}

		h(w, r, params)
	}
}

func (m Middleware) processToken(_ http.ResponseWriter, r *http.Request) error {
	if !m.c.requireToken {
		return nil
	}

	authHeader := []byte(r.Header.Get("Authorization"))
	if len(authHeader) == 0 {
		return ErrNoAuthHeader
	}

	if !bytes.HasPrefix(authHeader, []byte("bearer ")) && !bytes.HasPrefix(authHeader, []byte("Bearer ")) {
		return ErrNoBearerToken
	}

	bearer := bytes.TrimSpace(authHeader[6:])
	parts := bytes.Split(bearer, []byte{'.'})
	if len(parts) != numberOfJWTParts {
		return ErrDecodingBearer
	}

	tokenJSONBytes := make([]byte, base64.RawURLEncoding.DecodedLen(len(parts[1])))
	n, err := base64.RawURLEncoding.Decode(tokenJSONBytes, parts[1])
	if err != nil {
		return err
	}
	tokenJSONBytes = tokenJSONBytes[:n]

	if !json.Valid(tokenJSONBytes) {
		return ErrNotValidJSON
	}

	if m.c.signatureVerificationIsEnabled {
		if err := validateTokenSignature(bearer[:len(parts[0])+len(parts[1])+1], parts[2], m.c.publicKey); err != nil {
			return err
		}
	}

	for path, key := range m.c.claimsToExtract {
		claim := gjson.GetBytes(tokenJSONBytes, path)

		if !claim.Exists() && !m.c.ignoreNotExistingClaim {
			return ErrClaimNotExists
		}

		newR := r.WithContext(context.WithValue(r.Context(), key, claim.Value()))
		*r = *newR
	}

	if m.c.tokenContextKey != nil {
		newR := r.WithContext(context.WithValue(r.Context(), m.c.tokenContextKey, string(bearer)))
		*r = *newR
	}

	return nil
}

func validateTokenSignature(signedToken, signature []byte, key *rsa.PublicKey) error {
	// TODO: use some library to verify all kinds of signatures
	h := crypto.SHA256.New()
	_, err := h.Write(signedToken)
	if err != nil {
		return err
	}

	sigBytes := make([]byte, base64.RawURLEncoding.DecodedLen(len(signature)))
	n, err := base64.RawURLEncoding.Decode(sigBytes, signature)
	if err != nil {
		return err
	}
	sigBytes = sigBytes[:n]

	return rsa.VerifyPKCS1v15(key, crypto.SHA256, h.Sum(nil), sigBytes)
}
