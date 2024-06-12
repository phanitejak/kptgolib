package gerror

import "fmt"

// ErrorCode service error constants.
type ErrorCode string

const (
	// ConfigurationsError Configurations reading failed.
	ConfigurationsError ErrorCode = "Environment Error"
	// JobPollingError Job Polling  failed.
	JobPollingError ErrorCode = "Poller Failed"
	// InternalError internal service error.
	InternalError ErrorCode = "Internal Error"
	// ObjectNotExist Object not exist.
	ObjectNotExist ErrorCode = "Object Not Exists"
	// NoError No error case.
	NoError ErrorCode = "No Error"
	// DuplicateRecord - Record already exists
	DuplicateRecord ErrorCode = "Record already Exist"
	// AuthenticationFailed - Authentication failed
	AuthenticationFailed ErrorCode = "Authentication Failed"
	// MariaDBError MariaDB Error
	MariaDBError ErrorCode = "Maria DB Error"
	// InvalidQuery Query formation is wrong.
	InvalidQuery ErrorCode = "Invalid Query"
	// MarshallingError Marshalling/UnMarshalling failed.
	MarshallingError ErrorCode = "Marshaling/UnMarshaling failed"
	// ValidationFailed json validation failed, Bad Request.
	ValidationFailed ErrorCode = "Bad Request"
)

func (e ErrorCode) String() string {
	return string(e)
}

// GetErrorType ...
func GetErrorType(err error) ErrorCode {
	gerr, ok := err.(Gerror)
	if ok {
		return gerr.Tag().(ErrorCode)
	}
	return InternalError
}

// GetErrorMessage ...
func GetErrorMessage(err error) string {
	if gerr, ok := err.(Gerror); ok {
		if cause := gerr.Cause(); cause != nil {
			return fmt.Sprintf("%s: %s", gerr.Tag(), GetErrorMessage(cause))
		}
		return fmt.Sprintf("%s: %s", gerr.Tag(), gerr.Message())
	}
	return err.Error()
}
