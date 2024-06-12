# Vault client for NEO services in golang

TODO: Enable integration tests once vault supports vendoring.

## Usage

Start using vault client:

```go
package main

import (
	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/vault"
	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"testing"
	"fmt"
)

func main(){
    log := logging.NewLogger()

    client, err := vault.NewClient(
    	"https://vault-server-address",
    	"my-service-role",
    	vault.JwtPath("/var/run/secrets/kubernetes.io/serviceaccount/token")) // This is default mount path of JWT inside the pod
    if err != nil {
    	log.Errorf("unable to create vault client: %v", err)
        return
    }
    defer client.Close()

    // read, write and delete secrets using vault client
}
```

## Testing

You can use mock vault client implementation to test your application behavior using standard go testing library:


```go
package main

import (
	"github.com/phanitejak/kptgolib/vault"
	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"testing"
	"errors"
	"fmt"
)

func TestMyApp(t *testing.T) {
	c := vault.NewMockClient(t)
	c.WhenDelete("path/to/delete").ThenReturn(&api.Secret{RequestID: "delete request id"})
	c.WhenDelete("path/to/delete").ThenError(errors.New("delete error"))
	c.WhenRead("path/to/read").ThenReturn(&api.Secret{RequestID: "read request id"})

	secret, err := c.Delete("path/to/delete")
	assert.NoErrorf(t, err, "error should be nil")
	assert.Equal(t, "delete request id", secret.RequestID)

	secret, err = c.Delete("path/to/delete")
	assert.EqualError(t, err, "delete error")
	assert.Nil(t, secret)

	secret, err = c.Read("path/to/read")
	assert.NoErrorf(t, err, "error should be nil")
	assert.Equal(t, "read request id", secret.RequestID)
}
```
