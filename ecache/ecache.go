package ecache

import (
	"fmt"
	"time"

	cache "github.com/patrickmn/go-cache"
)

// ECache is abstracting go-cache offering caching of ErrEmptyValue errors for empty values.
type ECache struct {
	cache *cache.Cache
}

// ErrEmptyValue is used to define an error as constant.
type ErrEmptyValue string

func (e ErrEmptyValue) Error() string { return string(e) }

// IsErrEmptyValueType tests if given error is of type ErrEmptyValue.
func IsErrEmptyValueType(err error) bool {
	_, ok := err.(ErrEmptyValue)
	return ok
}

// ErrDataNotFound is used to designate data not found results.
var ErrDataNotFound = ErrEmptyValue("Data not found error")

type valueWrapper struct {
	err   error
	value interface{}
}

// NewDefault instantiates the cache with given expiration time for key and cleanup duration of 2x expiration.
func NewDefault(expiration time.Duration) *ECache {
	return New(cache.New(expiration, 2*expiration))
}

// New instantiates new cache with given, already configured go-cache object.
func New(cache *cache.Cache) *ECache {
	return &ECache{
		cache: cache,
	}
}

// SetIfData sets the key in cache only if err is nil or it is of type ErrEmptyValue.
func (c *ECache) SetIfData(key string, value interface{}, err error) {
	if err == nil || IsErrEmptyValueType(err) {
		c.cache.Set(key, valueWrapper{err: err, value: value}, cache.DefaultExpiration)
	}
}

// Get gets the value and its associated error for given key, and found is true. If there was no such key in cache, value and err are nil and found is false.
func (c *ECache) Get(key string) (value interface{}, found bool, err error) {
	data, found := c.cache.Get(key)
	if !found {
		return nil, false, nil
	}
	w, ok := data.(valueWrapper)
	if ok {
		return w.value, found, w.err
	}
	return nil, true, fmt.Errorf("Wrong cache wrapper data for key %s", key)
}

// Delete from cache if key exist.
func (c *ECache) Delete(key string) {
	c.cache.Delete(key)
}

// Flush deletes all item from cache
func (c *ECache) Flush() {
	c.cache.Flush()
}
