//nolint:dupl,errcheck,nakedret,gosec
package integration_tests //nolint:golint

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/helper/testhelpers"
	vaulthttp "github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/sdk/logical"
	vaultserver "github.com/hashicorp/vault/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg/vault"
)

type testParams struct {
	testCluster *vaultserver.TestCluster
	backend     *k8sBackend
	client      vault.Client
	vaultURL    string
	tokenFile   string
}

func startVault(t testing.TB, k8sMock *k8sBackend) (testCluster *vaultserver.TestCluster) {
	base := &vaultserver.CoreConfig{
		Logger: hclog.NewNullLogger(),
		CredentialBackends: map[string]logical.Factory{
			"kubernetes": func(ctx context.Context, config *logical.BackendConfig) (bk logical.Backend, err error) {
				bk = k8sMock
				err = bk.Setup(ctx, config)
				return
			},
		},
	}
	opts := &vaultserver.TestClusterOptions{
		HandlerFunc: vaulthttp.Handler,
	}
	testCluster = vaultserver.NewTestCluster(t, base, opts)
	testCluster.Start()

	vaultserver.TestWaitActive(t, testCluster.Cores[0].Core)
	err := testCluster.Cores[0].Client.Sys().EnableAuthWithOptions("custom", &api.EnableAuthOptions{Type: "kubernetes"})
	require.NoError(t, err, "unable to start vault test cluster")

	return testCluster
}

func createTokenFile(tb testing.TB, tokenValue string) string {
	tmpfile, err := ioutil.TempFile("", "token")
	require.NoError(tb, err, "unable to create a token file")

	_, err = tmpfile.Write([]byte(tokenValue))
	require.NoError(tb, err, "unable to write a token file")

	err = tmpfile.Close()
	require.NoError(tb, err, "unable to close a token file")

	return tmpfile.Name()
}

func TestVaultClient(t *testing.T) {
	tests := []struct {
		testFn     func(t *testing.T, params testParams)
		clientConf []vault.ConfigFn
		backend    *k8sBackend
	}{{
		testFn:     connectivity,
		clientConf: []vault.ConfigFn{},
		backend:    NewKubernetesBackendMock(time.Hour),
	}, {
		testFn:     automaticallyReestablishConnectionIfVaultGoesDown,
		clientConf: []vault.ConfigFn{vault.MaxRetries(0), vault.BreakerErrorTH(10)},
		backend:    NewKubernetesBackendMock(time.Hour),
	}, {
		testFn:     reestablishConnectionIfK8STokenExpires,
		clientConf: []vault.ConfigFn{},
		backend:    NewKubernetesBackendMock(time.Second),
	}, {
		testFn:     reuseTokenOnConcurrentWriteRequests,
		clientConf: []vault.ConfigFn{},
		backend:    NewKubernetesBackendMock(time.Second * 2),
	}, {
		testFn:     reuseTokenOnConcurrentReadRequests,
		clientConf: []vault.ConfigFn{},
		backend:    NewKubernetesBackendMock(time.Second * 2),
	}, {
		testFn:     reuseTokenOnConcurrentListRequests,
		clientConf: []vault.ConfigFn{},
		backend:    NewKubernetesBackendMock(time.Second * 2),
	}, {
		testFn:     reuseTokenOnConcurrentDeleteRequests,
		clientConf: []vault.ConfigFn{},
		backend:    NewKubernetesBackendMock(time.Second * 2),
	}, {
		testFn:     contextDeadlineExceededWhenTimeoutShorterThanLogicalOperationExecutionTime,
		clientConf: []vault.ConfigFn{vault.Timeout(time.Nanosecond)},
		backend:    NewKubernetesBackendMock(time.Hour),
	}, {
		testFn:     concurrentRequestsWhenTogglingVault,
		clientConf: []vault.ConfigFn{},
		backend:    NewKubernetesBackendMock(time.Second),
	}, {
		testFn:     openBreakerOnFailureTHAndCloseAfterTimeout,
		clientConf: []vault.ConfigFn{vault.MaxRetries(0), vault.BreakerTimeout(time.Second), vault.BreakerErrorTH(2)},
		backend:    NewKubernetesBackendMock(time.Hour),
	}}

	os.Setenv("VAULT_SKIP_VERIFY", "true")
	for _, tc := range tests {
		tc := tc
		fnFullName := runtime.FuncForPC(reflect.ValueOf(tc.testFn).Pointer()).Name()
		fnName := strings.TrimPrefix(fnFullName[strings.LastIndex(fnFullName, "."):], ".")
		t.Run(fnName, func(t *testing.T) {
			t.Parallel()
			testCluster := startVault(t, tc.backend)
			defer testCluster.Cleanup()

			tokenFile := createTokenFile(t, tc.backend.allowedLogin.jwt)
			defer os.Remove(tokenFile)

			defaultClientConfs := []vault.ConfigFn{vault.JwtPath(tokenFile), vault.AuthPath("auth/custom/login")}
			clientConfigs := append(defaultClientConfs, tc.clientConf...)

			vaultURL := "https://" + testCluster.Cores[0].Listeners[0].Addr().String()
			client, err := vault.NewClient(vaultURL, tc.backend.allowedLogin.role, clientConfigs...)
			require.NoError(t, err, "should create client")
			require.Equal(t, 0, tc.backend.loginCount, "should not login before any request is initiated")

			tc.testFn(t, testParams{
				testCluster: testCluster,
				backend:     tc.backend,
				client:      client,
				vaultURL:    vaultURL,
				tokenFile:   tokenFile,
			})
		})
	}
}

func connectivity(t *testing.T, p testParams) {
	secretPath1 := "cubbyhole/secret1"
	secretData1 := map[string]interface{}{"key": "value"}
	secretPath2 := "cubbyhole/secret2"
	secretData2 := map[string]interface{}{"key": "value"}

	_, err := p.client.Write(secretPath1, secretData1)
	require.NoError(t, err, "could not write secret")

	_, err = p.client.Write(secretPath2, secretData2)
	require.NoError(t, err, "could not write secret")

	secretList, err := p.client.List("cubbyhole")
	require.NoError(t, err, "could not list secrets")
	require.NotNil(t, secretList)
	require.NotNil(t, secretList.Data)
	require.EqualValues(t, map[string]interface{}{"keys": []interface{}{"secret1", "secret2"}}, secretList.Data)

	secret, err := p.client.Read(secretPath1)
	require.NoError(t, err, "could not read secretData")
	require.NotNil(t, secret, "secretData is not found")
	require.EqualValues(t, secretData1, secret.Data, "data is not the same")

	_, err = p.client.Delete(secretPath1)
	require.NoError(t, err, "could not remove secret")
	require.Equal(t, 1, p.backend.loginCount, "should use same token for all requests")
}

func automaticallyReestablishConnectionIfVaultGoesDown(t *testing.T, p testParams) {
	secretData := map[string]interface{}{"key": "value"}

	_, err := p.client.Write("cubbyhole/someSecret", secretData)
	require.NoError(t, err, "could not write secret")

	// Reset vault
	testhelpers.EnsureCoresSealed(t, p.testCluster)
	_, err = p.client.Read("cubbyhole/someSecret")
	require.Error(t, err, "vault server should not work at this point")
	testhelpers.EnsureCoresUnsealed(t, p.testCluster)

	// Write should reconnect automatically
	_, err = p.client.Write("cubbyhole/someNewSecret", secretData)
	require.NoError(t, err, "could not write secret")

	// Reset vault
	testhelpers.EnsureCoresSealed(t, p.testCluster)
	_, err = p.client.Read("cubbyhole/someSecret")
	require.Error(t, err, "vault server should not work at this point")
	testhelpers.EnsureCoresUnsealed(t, p.testCluster)

	// Read should reconnect automatically
	_, err = p.client.Read("cubbyhole/someNewSecret")
	require.NoError(t, err, "could not read secret")

	// Reset vault
	testhelpers.EnsureCoresSealed(t, p.testCluster)
	_, err = p.client.Read("cubbyhole/someSecret")
	require.Error(t, err, "vault server should not work at this point")
	testhelpers.EnsureCoresUnsealed(t, p.testCluster)

	// Delete should reconnect automatically
	_, err = p.client.Delete("cubbyhole/someNewSecret")
	require.NoError(t, err, "could not delete secret")

	// Reset vault
	testhelpers.EnsureCoresSealed(t, p.testCluster)
	_, err = p.client.Read("cubbyhole/someSecret")
	require.Error(t, err, "vault server should not work at this point")
	testhelpers.EnsureCoresUnsealed(t, p.testCluster)

	// List should reconnect automatically
	_, err = p.client.List("cubbyhole")
	require.NoError(t, err, "could not list secrets")

	require.Equal(t, 5, p.backend.loginCount, "should reconnect after each vault reset")
}

func reestablishConnectionIfK8STokenExpires(t *testing.T, p testParams) {
	secretData := map[string]interface{}{"key": "value"}

	_, err := p.client.Write("cubbyhole/someSecret", secretData)
	require.NoError(t, err, "could not write secret")
	require.Equal(t, 1, p.backend.loginCount, "should establish first connection")

	// Wait for token to expire
	time.Sleep(p.backend.allowedLogin.ttl + time.Second)

	_, err = p.client.Write("cubbyhole/someNewSecret", secretData)
	require.NoError(t, err, "could not write secret")
	require.Equal(t, 2, p.backend.loginCount, "should reconnect after token expiration")
}

func reuseTokenOnConcurrentWriteRequests(t *testing.T, p testParams) {
	secretData := map[string]interface{}{"key": "value"}

	_, err := p.client.Write("cubbyhole/someSecret", secretData)
	require.NoError(t, err, "could not write secret")

	// Wait till token expires
	time.Sleep(p.backend.allowedLogin.ttl + time.Second)

	numInvocations := 4
	var wg sync.WaitGroup
	wg.Add(numInvocations)

	for i := 0; i < numInvocations; i++ {
		go func() {
			defer wg.Done()
			p.client.Write(fmt.Sprintf("cubbyhole/someSecret%d", rand.Int()), secretData)
		}()
	}

	wg.Wait()

	require.Equal(t, 2, p.backend.loginCount, "should reuse single token for all requests")
}

func reuseTokenOnConcurrentReadRequests(t *testing.T, p testParams) {
	secretData := map[string]interface{}{"key": "value"}

	_, err := p.client.Write("cubbyhole/someSecret", secretData)
	require.NoError(t, err, "could not write secret")

	// Wait till token expires
	time.Sleep(p.backend.allowedLogin.ttl + time.Second)

	numInvocations := 4
	var wg sync.WaitGroup
	wg.Add(numInvocations)

	for i := 0; i < numInvocations; i++ {
		go func() {
			defer wg.Done()
			p.client.Read("cubbyhole/someSecret")
		}()
	}

	wg.Wait()

	require.Equal(t, 2, p.backend.loginCount, "should reuse single token for all requests")
}

func reuseTokenOnConcurrentListRequests(t *testing.T, p testParams) {
	_, err := p.client.List("cubbyhole/someSecret")
	require.NoError(t, err, "could not list secret")

	// Wait till token expires
	time.Sleep(p.backend.allowedLogin.ttl + time.Second)

	numInvocations := 4
	var wg sync.WaitGroup
	wg.Add(numInvocations)

	for i := 0; i < numInvocations; i++ {
		go func() {
			defer wg.Done()
			p.client.List("cubbyhole/someSecret")
		}()
	}

	wg.Wait()

	require.Equal(t, 2, p.backend.loginCount, "should reuse single token for all requests")
}

func reuseTokenOnConcurrentDeleteRequests(t *testing.T, p testParams) {
	secretData := map[string]interface{}{"key": "value"}

	_, err := p.client.Write("cubbyhole/someSecret", secretData)
	require.NoError(t, err, "could not write secret")

	// Wait till token expires
	time.Sleep(p.backend.allowedLogin.ttl + time.Second)

	numInvocations := 4
	var wg sync.WaitGroup
	wg.Add(numInvocations)

	for i := 0; i < numInvocations; i++ {
		go func() {
			defer wg.Done()
			p.client.Delete("cubbyhole/someSecret")
		}()
	}

	wg.Wait()

	require.Equal(t, 2, p.backend.loginCount, "should reuse single token for all requests")
}

func contextDeadlineExceededWhenTimeoutShorterThanLogicalOperationExecutionTime(t *testing.T, p testParams) {
	_, err := p.client.Read("cubbyhole/secret1")
	require.Error(t, err, "timeout expected but did not happen")
}

func concurrentRequestsWhenTogglingVault(t *testing.T, p testParams) {
	for i := 0; i < 10; i++ {
		go func() {
			for i := 0; i < 50; i++ {
				p.client.List("cubbyhole/someSecret")
				time.Sleep(time.Millisecond * 50)
			}
		}()
	}

	// This test will fail when a data race occurs
	for i := 0; i < 3; i++ {
		testhelpers.EnsureCoresSealed(t, p.testCluster)
		time.Sleep(time.Millisecond * 50)
		testhelpers.EnsureCoresUnsealed(t, p.testCluster)
	}
}

func openBreakerOnFailureTHAndCloseAfterTimeout(t *testing.T, p testParams) {
	secretData := map[string]interface{}{"key": "value"}

	_, err := p.client.Write("cubbyhole/someSecret", secretData)
	require.NoError(t, err, "could not write secret")

	// Reset vault
	testhelpers.EnsureCoresSealed(t, p.testCluster)

	// 1 failing read causes 2 breaker errors due to retry
	_, err = p.client.Read("cubbyhole/someSecret")
	require.Error(t, err, "vault server should not work at this point")
	assert.Contains(t, err.Error(), "Vault is sealed")

	_, err = p.client.Read("cubbyhole/someNewSecret")
	require.Error(t, err, "BreakerErrorTH amount of failures within BreakerTimeout should trip circuit breaker")
	assert.Equal(t, "circuit breaker is open", err.Error())

	testhelpers.EnsureCoresUnsealed(t, p.testCluster)

	// After waiting for more than BreakerTimeout, circuit breaker should be closed
	time.Sleep(2 * time.Second)

	_, err = p.client.Read("cubbyhole/someNewSecret")
	require.NoError(t, err, "circuit breaker should be closed")
}

func TestShouldNotOpenBreakerOnMissingPaths(t *testing.T) {
	t.Parallel()

	os.Setenv("VAULT_SKIP_VERIFY", "true")

	k8sMock := NewKubernetesBackendMock(time.Hour)
	k8sMock.allowedLogin = &loginInfo{jwt: "some_token", role: "default", ttl: time.Hour}
	testVaultCluster := startVault(t, k8sMock)
	defer testVaultCluster.Cleanup()

	tokenFile := createTokenFile(t, "some_token")
	defer os.Remove(tokenFile)

	breakerTH := 1
	vaultURL := "https://" + testVaultCluster.Cores[0].Listeners[0].Addr().String()
	client, err := vault.NewClient(vaultURL, "default", vault.AuthPath("auth/custom/login"),
		vault.JwtPath(tokenFile),
		vault.MaxRetries(0),
		vault.BreakerErrorTH(breakerTH))
	require.NoError(t, err, "should create client")

	for i := 0; i < breakerTH+1; i++ {
		_, err = client.Read("cubbyhole/missingSecret")
		require.NoError(t, err, "breaker should be closed")
	}
}

func TestBreakerOnHangingRequests(t *testing.T) {
	t.Parallel()
	tokenFile := createTokenFile(t, "some_token")
	defer os.Remove(tokenFile)

	timeout := 2 * time.Second
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { time.Sleep(2 * timeout) }))
	mockServer.Config.ReadTimeout = timeout
	mockServer.Config.WriteTimeout = timeout
	mockServer.Start()
	defer mockServer.Close()

	breakerErrTH := 3
	client, err := vault.NewClient(mockServer.URL, "default", vault.AuthPath("auth/custom/login"),
		vault.JwtPath(tokenFile),
		vault.MaxRetries(0),
		vault.Timeout(1*time.Second),
		vault.BreakerErrorTH(breakerErrTH),
		vault.BreakerSuccessTH(1),
		vault.BreakerTimeout(3*time.Second),
	)
	require.NoError(t, err, "should create client")
	wg := sync.WaitGroup{}
	for i := 0; i < breakerErrTH; i++ {
		wg.Add(1)
		go func() {
			_, err := client.Read("cubbyhole/missingSecret")
			require.Error(t, err, "breaker should be closed")
			assert.Contains(t, err.Error(), "context deadline exceeded")
			wg.Done()
		}()
	}
	wg.Wait()
	wg.Add(1)
	go func() {
		_, err := client.Read("cubbyhole/missingSecret")
		require.Error(t, err, "breaker should be closed")
		assert.Contains(t, err.Error(), "breaker is open")
		wg.Done()
	}()
	wg.Wait()
}

func TestShouldOpenBreakerOnConnectionFailures(t *testing.T) {
	t.Parallel()

	os.Setenv("VAULT_SKIP_VERIFY", "true")

	k8sMock := NewKubernetesBackendMock(time.Hour)
	testVaultCluster := startVault(t, k8sMock)
	defer testVaultCluster.Cleanup()

	tokenFile := createTokenFile(t, "wrong_token")
	defer os.Remove(tokenFile)

	vaultURL := "https://" + testVaultCluster.Cores[0].Listeners[0].Addr().String()
	client, err := vault.NewClient(vaultURL, "default", vault.AuthPath("auth/custom/login"),
		vault.JwtPath(tokenFile),
		vault.MaxRetries(0),
		vault.Timeout(60*time.Second),
		vault.BreakerErrorTH(1))
	require.NoError(t, err, "should create client")

	secretData := map[string]interface{}{"key": "value"}

	_, err = client.Write("cubbyhole/someSecret", secretData)
	require.Error(t, err, "should fail due to login error")
	assert.Contains(t, err.Error(), "permission denied")

	_, err = client.Write("cubbyhole/someSecret", secretData)
	require.Error(t, err, "breaker should be open")
	assert.Equal(t, "circuit breaker is open", err.Error())
}
