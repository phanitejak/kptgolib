//nolint:golint
package integration_tests

import (
	"context"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type k8sBackend struct {
	*framework.Backend
	allowedLogin *loginInfo
	loginCount   int
}

type loginInfo struct {
	jwt       string
	role      string
	renewable bool
	ttl       time.Duration
}

func NewKubernetesBackendMock(ttl time.Duration) *k8sBackend {
	b := &k8sBackend{}

	b.Backend = &framework.Backend{
		BackendType: logical.TypeCredential,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{"login"},
		},
		Paths: framework.PathAppend(
			[]*framework.Path{{
				Pattern: "login$",
				Fields: map[string]*framework.FieldSchema{
					"role": {Type: framework.TypeString},
					"jwt":  {Type: framework.TypeString},
				},

				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.UpdateOperation: b.loginFunc,
				},
			}}),
	}
	b.allowedLogin = &loginInfo{jwt: "test_token", role: "default", ttl: ttl}
	return b
}

func (b *k8sBackend) loginFunc(context context.Context, request *logical.Request, data *framework.FieldData) (resp *logical.Response, err error) {
	jwt, ok := request.Data["jwt"].(string)
	if !ok {
		return logical.ErrorResponse("malformed request: jwt missing"), nil
	}
	role, ok := request.Data["role"].(string)
	if !ok {
		return logical.ErrorResponse("malformed request: role missing"), nil
	}

	if b.allowedLogin != nil && b.allowedLogin.jwt == jwt && b.allowedLogin.role == role {
		resp = authenticate(jwt, b.allowedLogin.role, b.allowedLogin.renewable, b.allowedLogin.ttl)
	} else {
		err = logical.ErrPermissionDenied
	}

	b.loginCount++
	return
}

func authenticate(jwt, roleName string, renewable bool, ttl time.Duration) *logical.Response {
	return &logical.Response{
		Auth: &logical.Auth{
			Policies: []string{roleName},
			Metadata: map[string]string{
				"role": roleName, "jwt": jwt,
			},
			InternalData: map[string]interface{}{
				"role": roleName, "jwt": jwt,
			},
			LeaseOptions: logical.LeaseOptions{
				Renewable: renewable,
				TTL:       ttl,
				MaxTTL:    2 * ttl,
			},
		},
	}
}
