package vault

import (
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestMockClientShouldReturnErrorsOnList(t *testing.T) {
	tt := &testing.T{}
	mockVaultClient := NewMockClient(tt)

	mockVaultClient.WhenList("some/path").ThenError(errors.New("some error"))

	secret, err := mockVaultClient.List("some/path")
	assert.False(t, tt.Failed())
	assert.Nil(t, secret)
	assert.EqualError(t, err, "some error")
}

func TestMockClientShouldReturnErrorsOnRead(t *testing.T) {
	tt := &testing.T{}
	mockVaultClient := NewMockClient(tt)

	mockVaultClient.WhenRead("some/path").ThenError(errors.New("some error"))

	secret, err := mockVaultClient.Read("some/path")
	assert.False(t, tt.Failed())
	assert.Nil(t, secret)
	assert.EqualError(t, err, "some error")
}

func TestMockClientShouldFailWhenNoExpectedReturnsOrErrorsAreDefined(t *testing.T) {
	tt := &testing.T{}
	mockVaultClient := NewMockClient(tt)

	mockVaultClient.WhenRead("some/path")

	secret, err := mockVaultClient.Read("some/path")
	assert.True(t, tt.Failed())
	assert.Nil(t, secret)
	assert.EqualError(t, err, "unexpected invocation of read operation, you should define expected behavior")
}

func TestMockClientShouldBeAbleToReturnTwoNils(t *testing.T) {
	tt := &testing.T{}
	mockVaultClient := NewMockClient(tt)

	mockVaultClient.WhenRead("some/path").ThenReturn(nil)

	secret, err := mockVaultClient.Read("some/path")
	assert.False(t, tt.Failed())
	assert.Nil(t, secret)
	assert.NoError(t, err)
}

func TestMockClientShouldFailIfParametersNotMatch(t *testing.T) {
	tt := &testing.T{}
	mockVaultClient := NewMockClient(tt)

	mockVaultClient.WhenRead("some/path").ThenError(errors.New("some error"))

	secret, err := mockVaultClient.Read("some/different/path")
	assert.True(t, tt.Failed())
	assert.Nil(t, secret)
	assert.EqualError(t, err, `parameter mismatch on read operation. expected: some/path, actual: some/different/path`)
}

func TestMockClientShouldNotFailTest(t *testing.T) {
	tt := &testing.T{}
	NewMockClient(tt)

	assert.False(t, tt.Failed())
}

func TestMockClientShouldFailIfNoStubsDefinedForReadOperation(t *testing.T) {
	tt := &testing.T{}
	v := NewMockClient(tt)

	secret, err := v.Read("some/path")

	assert.True(t, tt.Failed())
	assert.Nil(t, secret)
	assert.EqualError(t, err, "unexpected invocation of read operation, you should define expected behavior")
}

func TestMockClientShouldFailIfNoStubsDefinedForWriteOperation(t *testing.T) {
	tt := &testing.T{}
	v := NewMockClient(tt)

	secret, err := v.Write("some/path", nil)

	assert.True(t, tt.Failed())
	assert.Nil(t, secret)
	assert.EqualError(t, err, "unexpected invocation of write operation, you should define expected behavior")
}

func TestMockClientShouldFailIfNoStubsDefinedForDeleteOperation(t *testing.T) {
	tt := &testing.T{}
	v := NewMockClient(tt)

	secret, err := v.Delete("some/path")

	assert.True(t, tt.Failed())
	assert.Nil(t, secret)
	assert.EqualError(t, err, "unexpected invocation of delete operation, you should define expected behavior")
}

func TestShouldPreserveOrder(t *testing.T) {
	tt := &testing.T{}
	v := NewMockClient(tt)
	v.WhenDelete("first/delete/request").ThenReturn(&api.Secret{RequestID: "first delete request"})
	v.WhenDelete("second/delete/request").ThenError(errors.New("first delete error"))
	v.WhenRead("first/read/request").ThenReturn(&api.Secret{RequestID: "first read request"})
	v.WhenList("first/read").ThenReturn(&api.Secret{RequestID: "list of keys"})

	dataToWrite := make(map[string]any)
	dataToWrite["first_key"] = "first_value"

	v.WhenWrite("first/write/request", dataToWrite).ThenError(errors.New("first write error"))
	v.WhenWrite("second/write/request", nil).ThenError(errors.New("second write error"))
	v.WhenRead("second/read/request").ThenError(errors.New("first read error"))
	v.WhenList("second/read").ThenError(errors.New("first list error"))
	v.WhenDelete("third/delete/request").ThenReturn(&api.Secret{RequestID: "third delete request"})

	s, e := v.Delete("first/delete/request")
	assert.NoError(t, e)
	assert.Equal(t, "first delete request", s.RequestID)

	s, e = v.Delete("second/delete/request")
	assert.EqualError(t, e, "first delete error")
	assert.Nil(t, s)

	s, e = v.Read("first/read/request")
	assert.NoError(t, e)
	assert.Equal(t, "first read request", s.RequestID)

	s, e = v.List("first/read")
	assert.NoError(t, e)
	assert.Equal(t, "list of keys", s.RequestID)

	s, e = v.Write("first/write/request", dataToWrite)
	assert.EqualError(t, e, "first write error")
	assert.Nil(t, s)

	s, e = v.Write("second/write/request", nil)
	assert.EqualError(t, e, "second write error")
	assert.Nil(t, s)

	s, e = v.Read("second/read/request")
	assert.EqualError(t, e, "first read error")
	assert.Nil(t, s)

	s, e = v.List("second/read")
	assert.EqualError(t, e, "first list error")
	assert.Nil(t, s)

	s, e = v.Delete("third/delete/request")
	assert.NoError(t, e)
	assert.Equal(t, "third delete request", s.RequestID)

	assert.False(t, tt.Failed())
}

func TestShouldFailIfOrderIsBroken(t *testing.T) {
	tt := &testing.T{}
	v := NewMockClient(tt)
	v.WhenDelete("first/delete/request").ThenReturn(&api.Secret{RequestID: "first delete request"})
	v.WhenRead("first/read/request").ThenReturn(&api.Secret{RequestID: "first read request"})

	s, e := v.Read("first/read/request")
	assert.EqualError(t, e, "call order of read operation is different than expected")
	assert.Nil(t, s)

	s, e = v.Delete("first/delete/request")
	assert.NoError(t, e)
	assert.Equal(t, "first delete request", s.RequestID)

	assert.True(t, tt.Failed())
}
