//nolint:dupl,errcheck,nakedret,gosec
package integration_tests //nolint:golint

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
	"gopkg/vault"
)

func BenchmarkList(b *testing.B) {
	builtInClient, ourClient, cleanupFunc := prepareDifferentClientsForBenchmark(b)
	defer cleanupFunc()

	secretPath := "cubbyhole/secret"
	secretData := map[string]interface{}{"key": "value"}
	ourClient.Write(secretPath, secretData)

	b.Run("BuiltInClient", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builtInClient.Logical().List(secretPath)
		}
	})
	b.Run("BuiltInClientParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				builtInClient.Logical().List(secretPath)
			}
		})
	})

	b.Run("OurClient", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ourClient.List(secretPath)
		}
	})
	b.Run("OurClientParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ourClient.List(secretPath)
			}
		})
	})
}

func BenchmarkRead(b *testing.B) {
	builtInClient, ourClient, cleanupFunc := prepareDifferentClientsForBenchmark(b)
	defer cleanupFunc()

	secretPath := "cubbyhole/secret"
	secretData := map[string]interface{}{"key": "value"}
	ourClient.Write(secretPath, secretData)

	b.Run("BuiltInClient", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builtInClient.Logical().Read(secretPath)
		}
	})
	b.Run("BuiltInClientParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				builtInClient.Logical().Read(secretPath)
			}
		})
	})

	b.Run("OurClient", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ourClient.Read(secretPath)
		}
	})
	b.Run("OurClientParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ourClient.Read(secretPath)
			}
		})
	})
}

func BenchmarkWrite(b *testing.B) {
	builtInClient, ourClient, cleanupFunc := prepareDifferentClientsForBenchmark(b)
	defer cleanupFunc()

	secretPath := "cubbyhole/secret"
	secretData := map[string]interface{}{"key": "value"}
	ourClient.Write(secretPath, secretData)

	b.Run("BuiltInClient", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builtInClient.Logical().Write(fmt.Sprintf("%s%s%d", secretPath, "builtInClient", rand.Int()), secretData)
		}
	})
	b.Run("BuiltInClientParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				builtInClient.Logical().Write(fmt.Sprintf("%s%s%d", secretPath, "builtInClientParallel", rand.Int()), secretData)
			}
		})
	})

	b.Run("OurClient", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ourClient.Write(fmt.Sprintf("%s%s%d", secretPath, "ourClient", rand.Int()), secretData)
		}
	})
	b.Run("OurClientParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ourClient.Write(fmt.Sprintf("%s%s%d", secretPath, "ourClientParallel", rand.Int()), secretData)
			}
		})
	})
}

func BenchmarkDelete(b *testing.B) {
	builtInClient, ourClient, cleanupFunc := prepareDifferentClientsForBenchmark(b)
	defer cleanupFunc()

	secretPath := "cubbyhole/secret"
	secretData := map[string]interface{}{"key": "value"}
	ourClient.Write(secretPath, secretData)

	b.Run("BuiltInClient", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builtInClient.Logical().Delete(fmt.Sprintf("%s%d", secretPath, rand.Int()))
		}
	})
	b.Run("BuiltInClientParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				builtInClient.Logical().Delete(fmt.Sprintf("%s%d", secretPath, rand.Int()))
			}
		})
	})

	b.Run("OurClient", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ourClient.Delete(fmt.Sprintf("%s%d", secretPath, rand.Int()))
		}
	})
	b.Run("OurClientParallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ourClient.Delete(fmt.Sprintf("%s%d", secretPath, rand.Int()))
			}
		})
	})
}

func prepareDifferentClientsForBenchmark(b *testing.B) (builtInClient *api.Client, ourClient vault.Client, cleanupFunc func()) {
	os.Setenv("VAULT_SKIP_VERIFY", "true")

	k8sMock := NewKubernetesBackendMock(time.Hour)
	testVaultCluster := startVault(b, k8sMock)
	builtInClient = testVaultCluster.Cores[0].Client
	tokenFile := createTokenFile(b, k8sMock.allowedLogin.jwt)

	cleanupFunc = func() {
		testVaultCluster.Cleanup()
		os.Remove(tokenFile)
	}

	vaultURL := "https://" + testVaultCluster.Cores[0].Listeners[0].Addr().String()
	ourClient, err := vault.NewClient(vaultURL, k8sMock.allowedLogin.role, vault.AuthPath("auth/custom/login"), vault.JwtPath(tokenFile))
	require.NoError(b, err, "could not create new client")

	return
}
