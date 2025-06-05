package vault

import (
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

func NewSimpleTokenClient(address, token string) (Client, error) {
	return NewSimpleTokenClientWithConfig(defaultConfig(address), token)
}

func NewSimpleTokenClientWithConfig(config *api.Config, token string) (Client, error) {
	vaultClient, err := api.NewClient(config)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not configure Vault client with settings provided")
	}
	vaultClient.SetToken(token)

	return &simpleTokenClient{vaultClient: vaultClient}, nil
}

type simpleTokenClient struct {
	vaultClient *api.Client
}

func (c *simpleTokenClient) List(path string) (secret *api.Secret, err error) {
	return c.vaultClient.Logical().List(path)
}

func (c *simpleTokenClient) Read(path string) (secret *api.Secret, err error) {
	return c.vaultClient.Logical().Read(path)
}

func (c *simpleTokenClient) Write(path string, data map[string]any) (secret *api.Secret, err error) {
	return c.vaultClient.Logical().Write(path, data)
}

func (c *simpleTokenClient) Delete(path string) (secret *api.Secret, err error) {
	return c.vaultClient.Logical().Delete(path)
}

func (c *simpleTokenClient) Mount(path string, input *api.MountInput) error {
	return c.vaultClient.Sys().Mount(path, input)
}

func (c *simpleTokenClient) Unmount(path string) error {
	return c.vaultClient.Sys().Unmount(path)
}

func (c *simpleTokenClient) ListMounts() (map[string]*api.MountOutput, error) {
	return c.vaultClient.Sys().ListMounts()
}
