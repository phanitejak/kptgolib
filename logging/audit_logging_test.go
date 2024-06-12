package logging_test

import (
	"bytes"
	"testing"

	"github.com/phanitejak/gopkg/logging"
	"github.com/phanitejak/gopkg/logging/testutil"
	"github.com/stretchr/testify/assert"
)

func TestDefaultFieldsAudit(t *testing.T) {
	logger, logOutput := getAuditLogger(t)
	s := logging.AuditRecord{
		User:                    "test",
		Result:                  logging.Success,
		Operation:               "upload",
		Msg:                     "test operation",
		EventType:               logging.N_OPER_RESULT,
		ErrorCode:               logging.E_NOT_POSSIBLE,
		Level:                   "info",
		Process:                 "upload process",
		Service:                 "upload service",
		System:                  "neo",
		NeID:                    "123",
		Container:               "uploadcnt",
		Host:                    "neohost",
		TargetUserIdentity:      "test",
		TargetSessionIdentifier: "1111",
		SourcePort:              "8080",
		SourceIP:                "10.1.1.1",
		SourceUserIdentity:      "srctest",
		SourceSessionIdentifier: "2222",
		SessionID:               "abc",
		ProxySourcePort:         "8082",
		ProxySourceIP:           "10.1.1.2",
		ProxyBackendPort:        "8083",
		ProxyBackendIP:          "10.1.1.3",
		ObjectID:                "OBJ1",
		ObjectAttributes:        "testa",
		Object:                  "objA",
		Interface:               "intf1",
		BackendPort:             "8056",
		BackendIP:               "10.1.2.3",
	}

	logger.Audit(s)

	auditLogData := `{
		"operation": "upload",
		"proxy-backend-ip": "10.1.1.3",
		"proxy-backend-port": "8083",
		"proxy-source-port": "8082",
		"source-session-identity": "2222",
		"errorcode": "E_NOT_POSSIBLE",
		"object": "objA",
		"object-attributes": "testa",
		"system": "neo",
		"timezone": "Local",
		"user": "test",
		"interface": "intf1",
		"level": "info",
		"proxy-source-ip": "10.1.1.2",
		"type": "log",
		"backend-port": "8056",
		"host": "neohost",
		"session-id": "abc",
		"neid": "123",
		"result": "success",
		"source-port": "8080",
		"facility": "audit",
		"source-user-identity": "srctest",
		"target-session-identity": "1111",
		"target-user-identity": "test",
		"backend-ip": "10.1.2.3",
		"container": "uploadcnt",
		"eventtype": "N_OPER_RESULT",
		"fields.level": "info",
		"process": "upload process",
		"service": "upload service",
		"message": "test operation",
		"object-id": "OBJ1",
		"source-ip": "10.1.1.1"
	  }`

	authLogs := testutil.UnmarshalLogMessage(t, logOutput().Bytes())
	delete(authLogs, "timestamp")
	expectedAuthLogs := testutil.UnmarshalLogMessage(t, []byte(auditLogData))

	assert.Equal(t, expectedAuthLogs, authLogs, "audit message attributes not matched")
}

func TestDefaultFieldsAuth(t *testing.T) {
	logger, logOutput := getAuditLogger(t)
	s := logging.AuditRecord{
		User:      "test",
		Result:    logging.Failed,
		Operation: "Ack",
		Msg:       "testst",
		Process:   "Logging Lib",
	}
	logger.Auth(s)
	authLogs := testutil.UnmarshalLogMessage(t, logOutput().Bytes())
	assert.Equal(t, "auth", authLogs["facility"])
	assert.Equal(t, "test", authLogs["user"])
	assert.Equal(t, "Ack", authLogs["operation"])
	assert.Equal(t, "testst", authLogs["message"])
	assert.Equal(t, "Logging Lib", authLogs["process"])
}

func getAuditLogger(t *testing.T) (logging.AuditLogger, func() *bytes.Buffer) {
	logOutput := testutil.PipeStderr(t)
	logger := logging.NewAuditLogger()
	return logger, logOutput
}
