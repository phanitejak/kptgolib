//nolint:golint
package vault

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/vault/api"
)

type MockClient struct {
	t             *testing.T
	expectedCalls []*expectedCall
}

type expectedCall struct {
	addExpectedCall func(call *expectedCall)
	operation       string
	error           error
	expectedParams  any
	result          any
}

var _ Client = &MockClient{}

// Provides mock implementation for vault client.
// Will fail test, if any function is called unexpectedly.
// You can define expected mock behaviors using When***(...).Then***(...) methods.
// Expected calls are order sensitive. If call order is broken, test will fail.
func NewMockClient(t *testing.T) *MockClient {
	return &MockClient{t: t}
}

func (m *MockClient) addExpectedCall(call *expectedCall) {
	m.expectedCalls = append(m.expectedCalls, call)
}

func (m *MockClient) Read(path string) (result *api.Secret, err error) {
	checkResult, err := m.checkCallIsCorrect("read", path)
	if err != nil {
		return nil, err
	}

	if checkResult == nil {
		return
	}

	result = checkResult.(*api.Secret)
	return
}

func (m *MockClient) List(path string) (result *api.Secret, err error) {
	checkResult, err := m.checkCallIsCorrect("list", path)
	if err != nil {
		return nil, err
	}

	if checkResult == nil {
		return
	}

	result = checkResult.(*api.Secret)
	return
}

func (m *MockClient) Write(path string, data map[string]any) (result *api.Secret, err error) {
	writeParameters := struct {
		p string
		d map[string]any
	}{p: path, d: data}

	callResult, err := m.checkCallIsCorrect("write", writeParameters)
	if err != nil {
		return nil, err
	}

	if callResult == nil {
		return
	}

	result = callResult.(*api.Secret)
	return
}

func (m *MockClient) Delete(path string) (result *api.Secret, err error) {
	checkResult, err := m.checkCallIsCorrect("delete", path)
	if err != nil {
		return nil, err
	}

	if checkResult == nil {
		return
	}

	result = checkResult.(*api.Secret)
	return
}

// Mount is not implemented.
func (m *MockClient) Mount(string, *api.MountInput) error {
	panic("not implemented")
}

// Unmount is not implemented.
func (m *MockClient) Unmount(string) error {
	panic("not implemented")
}

// ListMounts is not implemented.
func (m *MockClient) ListMounts() (map[string]*api.MountOutput, error) {
	panic("not implemented")
}

func (m *MockClient) WhenList(path string) *expectedCall {
	c := &expectedCall{operation: "list", expectedParams: path, addExpectedCall: m.addExpectedCall}
	return c
}

func (m *MockClient) WhenRead(path string) *expectedCall {
	c := &expectedCall{operation: "read", expectedParams: path, addExpectedCall: m.addExpectedCall}
	return c
}

func (m *MockClient) WhenWrite(path string, data map[string]any) *expectedCall {
	writeParameters := struct {
		p string
		d map[string]any
	}{p: path, d: data}

	c := &expectedCall{operation: "write", expectedParams: writeParameters, addExpectedCall: m.addExpectedCall}
	return c
}

func (m *MockClient) WhenDelete(path string) *expectedCall {
	c := &expectedCall{operation: "delete", expectedParams: path, addExpectedCall: m.addExpectedCall}
	return c
}

func (ec *expectedCall) ThenReturn(result any) {
	ec.result = result
	ec.addExpectedCall(ec)
}

func (ec *expectedCall) ThenError(err error) {
	ec.error = err
	ec.addExpectedCall(ec)
}

func (m *MockClient) checkCallIsCorrect(methodName string, actualParams any) (expectedResult any, err error) {
	for k, v := range m.expectedCalls {
		if v.operation == methodName {
			if k != 0 {
				err = m.failWithBrokenOrderError(methodName)
				return
			}

			if !reflect.DeepEqual(v.expectedParams, actualParams) {
				err = m.failWithParameterMismatchError(methodName, v.expectedParams, actualParams)
				return
			}

			expectedResult = v.result
			err = v.error
			// Delete from expected calls
			m.expectedCalls = append(m.expectedCalls[:k], m.expectedCalls[k+1:]...)
			return
		}
	}
	err = m.failWithNoStubsDefinedError(methodName)
	return
}

func (m *MockClient) failWithNoStubsDefinedError(operation string) error {
	e := fmt.Errorf("unexpected invocation of %s operation, you should define expected behavior", operation)
	m.t.Error(e)
	return e
}

func (m *MockClient) failWithBrokenOrderError(operation string) error {
	e := fmt.Errorf("call order of %s operation is different than expected", operation)
	m.t.Error(e)
	return e
}

func (m *MockClient) failWithParameterMismatchError(operation string, expectedParams any, actualParams any) error {
	e := fmt.Errorf("parameter mismatch on %s operation. expected: %v, actual: %v", operation, expectedParams, actualParams)
	m.t.Error(e)
	return e
}
