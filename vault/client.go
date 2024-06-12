package vault

import (
	"io/ioutil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eapache/go-resiliency/breaker"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"gopkg/logging"
)

//nolint:gosec
const (
	defaultServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	defaultAuthPath                = "auth/kubernetes/login"
	defaultTimeout                 = time.Second * 60
	defaultMaxRetries              = 5
	defaultBreakerSuccessTH        = 1
	defaultBreakerErrorTH          = 3
	defaultBreakerTimeout          = defaultTimeout * defaultBreakerErrorTH
)

var log = logging.NewLogger()

type ConfigFn func(*config) error

type Client interface {
	Read(string) (*api.Secret, error)
	Write(string, map[string]interface{}) (*api.Secret, error)
	Delete(string) (*api.Secret, error)
	List(string) (*api.Secret, error)
	Mount(string, *api.MountInput) error
	Unmount(string) error
	ListMounts() (map[string]*api.MountOutput, error)
}

type client struct {
	lock        sync.RWMutex
	config      *config
	initialized uint32
	h           *vaultClientHolder
	breaker     *breaker.Breaker
}

type vaultClientHolder struct {
	setInstCh chan *api.Client
	getInstCh chan *api.Client
}

type config struct {
	AuthPath, JwtPath, VaultAddress, Role string
	Timeout                               time.Duration
	Token                                 string
	MaxRetries                            int
	BreakerTimeout                        time.Duration
	BreakerErrorTH                        int
	BreakerSuccessTH                      int
}

func (c *client) List(path string) (secret *api.Secret, err error) {
	err = c.connectIfNotInitialized()
	if err != nil {
		return nil, err
	}

	return c.tryOperationWithBreaker(func() (secret *api.Secret, err error) {
		return c.h.get().Logical().List(path)
	})
}

func (c *client) Read(path string) (secret *api.Secret, err error) {
	err = c.connectIfNotInitialized()
	if err != nil {
		return nil, err
	}

	return c.tryOperationWithBreaker(func() (secret *api.Secret, err error) {
		return c.h.get().Logical().Read(path)
	})
}

func (c *client) Write(path string, data map[string]interface{}) (secret *api.Secret, err error) {
	err = c.connectIfNotInitialized()
	if err != nil {
		return nil, err
	}

	return c.tryOperationWithBreaker(func() (secret *api.Secret, err error) {
		return c.h.get().Logical().Write(path, data)
	})
}

func (c *client) Delete(path string) (secret *api.Secret, err error) {
	err = c.connectIfNotInitialized()
	if err != nil {
		return nil, err
	}

	return c.tryOperationWithBreaker(func() (secret *api.Secret, err error) {
		return c.h.get().Logical().Delete(path)
	})
}

func (c *client) Mount(path string, input *api.MountInput) error {
	err := c.connectIfNotInitialized()
	if err != nil {
		return err
	}
	_, err = c.tryOperation(func() (secret *api.Secret, err error) {
		return nil, c.h.get().Sys().Mount(path, input)
	})
	return err
}

func (c *client) Unmount(path string) error {
	err := c.connectIfNotInitialized()
	if err != nil {
		return err
	}
	_, err = c.tryOperation(func() (secret *api.Secret, err error) {
		return nil, c.h.get().Sys().Unmount(path)
	})
	return err
}

func (c *client) ListMounts() (map[string]*api.MountOutput, error) {
	err := c.connectIfNotInitialized()
	if err != nil {
		return nil, err
	}
	var mountList map[string]*api.MountOutput

	_, err = c.tryOperationWithBreaker(func() (secret *api.Secret, err error) {
		mountList, err = c.h.get().Sys().ListMounts()
		return nil, err
	})
	return mountList, err
}

func (c *client) tryOperationWithBreaker(operation func() (secret *api.Secret, err error)) (secret *api.Secret, err error) {
	err = c.breaker.Run(func() (e error) {
		secret, e = c.tryOperation(operation)
		return e
	})
	if err == breaker.ErrBreakerOpen {
		log.Error("vault operation skipped due to open circuit breaker")
	}
	return secret, err
}

func (c *client) tryOperation(operation func() (secret *api.Secret, err error)) (secret *api.Secret, err error) {
	secret, err = operation()
	if err == nil {
		return
	}
	log.Debug("error performing request, retrying")

	c.lock.Lock()
	defer c.lock.Unlock()

	secret, err = operation()
	if err == nil {
		return
	}
	log.Debug("error performing request, reconnecting to vault server")

	if err = c.connectToVaultServerWithBreaker(); err != nil {
		return
	}

	return operation()
}

func (c *client) connectIfNotInitialized() (err error) {
	if atomic.LoadUint32(&c.initialized) == 1 {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.initialized == 0 {
		log.Debug("initializing vault client")
		if err = c.connectToVaultServerWithBreaker(); err != nil {
			return err
		}
	}

	return
}

func (c *client) connectToVaultServerWithBreaker() (err error) {
	err = c.breaker.Run(c.connectToVaultServer)
	if err == breaker.ErrBreakerOpen {
		log.Error("connect to vault skipped due to open circuit breaker")
	}
	return err
}

func (c *client) connectToVaultServer() (err error) {
	log.Debug("establishing connection to vault")

	jwt, err := readServiceAccountToken(c.config.JwtPath)
	if err != nil {
		log.Errorf("error reading json web token under %s", c.config.JwtPath)
		return
	}

	config := defaultConfig(c.config.VaultAddress)
	config.Timeout = c.config.Timeout
	config.MaxRetries = c.config.MaxRetries

	vaultClient, err := api.NewClient(config)
	if err != nil {
		return errors.WithMessage(err, "Could not configure Vault client with settings provided")
	}

	c.h.set(vaultClient)

	token := c.config.Token
	if c.config.Token == "" {
		authResponse, err := c.h.get().Logical().Write(c.config.AuthPath, createAuthData(jwt, c.config.Role))
		if err != nil {
			log.Errorf(errors.WithMessagef(err, "error authenticating to vault server. auth path: %s, role: %s", c.config.AuthPath, c.config.Role).Error())
			return err
		}

		token, err = authResponse.TokenID()
		if err != nil {
			log.Error("error extracting token from authentication response")
			return err
		}
	}

	c.h.get().SetToken(token)

	atomic.StoreUint32(&c.initialized, 1)

	log.Debugf("successfully connected to vault server %s", c.config.VaultAddress)
	return nil
}

// Token is Vault url path to make login request.
func Token(token string) ConfigFn {
	return func(c *config) (err error) {
		c.Token = token
		return
	}
}

// AuthPath is Vault url path to make login request.
func AuthPath(authPath string) ConfigFn {
	return func(c *config) (err error) {
		c.AuthPath = authPath
		return
	}
}

// JwtPath is a path to Service Account Token file.
func JwtPath(jwtPath string) ConfigFn {
	return func(c *config) (err error) {
		c.JwtPath = jwtPath
		return
	}
}

// Timeout for HTTPClient.
func Timeout(timeout time.Duration) ConfigFn {
	return func(c *config) (err error) {
		c.Timeout = timeout
		return
	}
}

// MaxRetries in case of 5xx errors from Vault server.
func MaxRetries(maxRetries int) ConfigFn {
	return func(c *config) (err error) {
		c.MaxRetries = maxRetries
		return
	}
}

// BreakerErrorTH amount of failures within BreakerTimeout period -> circuit breaker opens.
func BreakerErrorTH(bErrTH int) ConfigFn {
	return func(c *config) (err error) {
		c.BreakerErrorTH = bErrTH
		return
	}
}

// BreakerSuccessTH amount of consecutive successes -> circuit breaker moves from half-closed to closed.
func BreakerSuccessTH(bSuccessTH int) ConfigFn {
	return func(c *config) (err error) {
		c.BreakerSuccessTH = bSuccessTH
		return
	}
}

// BreakerTimeout error-free time -> circuit breaker moves from open to half-closed.
func BreakerTimeout(bTimeout time.Duration) ConfigFn {
	return func(c *config) (err error) {
		c.BreakerTimeout = bTimeout
		return
	}
}

//nolint:golint
func NewClient(vaultAddress, role string, options ...ConfigFn) (c *client, err error) {
	conf := config{
		AuthPath:         defaultAuthPath,
		JwtPath:          defaultServiceAccountTokenPath,
		VaultAddress:     vaultAddress,
		Role:             role,
		Timeout:          defaultTimeout,
		MaxRetries:       defaultMaxRetries,
		BreakerErrorTH:   defaultBreakerErrorTH,
		BreakerSuccessTH: defaultBreakerSuccessTH,
		BreakerTimeout:   defaultBreakerTimeout,
	}

	for _, option := range options {
		err = option(&conf)
		if err != nil {
			return
		}
	}

	b := breaker.New(conf.BreakerErrorTH, conf.BreakerSuccessTH, conf.BreakerTimeout)

	c = &client{
		config:  &conf,
		h:       newVaultClientHolder(),
		breaker: b,
	}

	return
}

func newVaultClientHolder() *vaultClientHolder {
	h := &vaultClientHolder{
		setInstCh: make(chan *api.Client),
		getInstCh: make(chan *api.Client),
	}
	go h.mux()
	return h
}

func (h vaultClientHolder) mux() {
	var instance *api.Client
	for {
		select {
		case instance = <-h.setInstCh:
		case h.getInstCh <- instance:
		}
	}
}

func (h vaultClientHolder) get() *api.Client {
	return <-h.getInstCh
}

func (h vaultClientHolder) set(inst *api.Client) {
	h.setInstCh <- inst
}

func createAuthData(jwt string, role string) (authData map[string]interface{}) {
	authData = make(map[string]interface{})

	authData["role"] = role
	authData["jwt"] = jwt

	return
}

func readServiceAccountToken(jwtPath string) (s string, err error) {
	b, err := ioutil.ReadFile(jwtPath)
	if err != nil {
		return
	}

	s = string(b)
	return
}

func defaultConfig(address string) *api.Config {
	config := api.DefaultConfig()
	config.Address = address
	config.Timeout = defaultTimeout
	config.Backoff = retryablehttp.DefaultBackoff
	// exponential backoff instead of linear set in vault by default
	config.MaxRetries = defaultMaxRetries
	return config
}
