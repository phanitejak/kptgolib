package ecache

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) (a *assert.Assertions, c *ECache) {
	a = assert.New(t)
	c = NewDefault(1 * time.Minute)
	return
}

func assertKey(a *assert.Assertions, c *ECache, key string, eV any, eE error, eF bool) {
	v, f, e := c.Get(key)
	a.Equal(eF, f, "Failed for key %s", key)
	a.Equal(eE, e, "Failed for key %s", key)
	a.Equal(eV, v, "Failed for key %s", key)
}

func TestStoreValidValues(t *testing.T) {
	key := "random"
	a, c := setup(t)
	assertKey(a, c, key, nil, nil, false)
	c.SetIfData(key, "AA", nil)
	assertKey(a, c, key, "AA", nil, true)
	c.SetIfData(key, nil, ErrDataNotFound)
	assertKey(a, c, key, nil, ErrDataNotFound, true)
	c.SetIfData(key, nil, nil)
	assertKey(a, c, key, nil, nil, true)
}

func TestStoreInvalidValidValues(t *testing.T) {
	key := "random"
	a, c := setup(t)
	c.SetIfData(key, nil, fmt.Errorf("Generic error"))
	assertKey(a, c, key, nil, nil, false)
}
