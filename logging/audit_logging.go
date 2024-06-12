// Package logging provides compatibility with Neo logging guidelines
package logging

import (
	"os"
	"time"

	"github.com/fatih/structs"
	"github.com/sirupsen/logrus"
)

const (
	auditFacility = "audit"
	authFacility  = "auth"
	logType       = "log"
	facilityName  = "facility"
	logTypeName   = "type"
	timeZoneName  = "timezone"
)

type SyslogLevels string

const (
	ALERT  SyslogLevels = "ALERT"
	CRIT   SyslogLevels = "CRIT"
	DEBUG  SyslogLevels = "DEBUG"
	EMERG  SyslogLevels = "EMERG"
	ERROR  SyslogLevels = "ERROR"
	INFO   SyslogLevels = "INFO"
	NOTICE SyslogLevels = "NOTICE"
	WARN   SyslogLevels = "WARN"
)

// EventTypes represents type of Audit Event.
type EventTypes string

// nolint
const (
	N_USER_AUTH          EventTypes = "N_USER_AUTH"
	N_USER_ACCS          EventTypes = "N_USER_ACCS"
	N_USER_LOGIN         EventTypes = "N_USER_LOGIN"
	N_USER_LOGOUT        EventTypes = "N_USER_LOGOUT"
	N_USER_MGMT          EventTypes = "N_USER_MGMT"
	N_GROUP_MGMT         EventTypes = "N_GROUP_MGMT"
	N_ADD_USER           EventTypes = "N_ADD_USER"
	N_ADD_GROUP          EventTypes = "N_ADD_GROUP"
	N_DEL_USER           EventTypes = "N_DEL_USER"
	N_DEL_GROUP          EventTypes = "N_DEL_GROUP"
	N_USER_OPER          EventTypes = "N_USER_OPER"
	N_OPER_RESULT        EventTypes = "N_OPER_RESULT"
	N_CM_UPLOAD          EventTypes = "N_CM_UPLOAD"
	N_CM_PROVISION       EventTypes = "N_CM_PROVISION"
	N_SWM_PROVISION      EventTypes = "N_SWM_PROVISION"
	N_IMPORT_PLAN        EventTypes = "N_IMPORT_PLAN"
	N_EXPORT_ACTUALS     EventTypes = "N_EXPORT_ACTUALS"
	N_NE_PASSWORD_RESET  EventTypes = "N_NE_PASSWORD_RESET"
	N_NE_CERTM_OPERATION EventTypes = "N_NE_CERTM_OPERATION"
	N_EXPORT_PLAN        EventTypes = "N_EXPORT_PLAN"
	N_EXPORT_INVENTORY   EventTypes = "N_EXPORT_INVENTORY"
)

type ErrorCodes string

//nolint:golint
const (
	// Operation failed, but the detailed cause is unknown.
	E_FAILURE ErrorCodes = "E_FAILURE"

	// Operation is incomplete.
	E_INCOMPLETE ErrorCodes = "E_INCOMPLETE"

	// Operation failed because the request was illegal.
	E_INVALID_PARAMETERS ErrorCodes = "E_INVALID_PARAMETERS"

	// Operation is denied because invoker haves no privileges to perform it.
	E_NO_PRIVIlEGES ErrorCodes = "E_NO_PRIVIlEGES"

	// Operation failed because it is not implemented or enabled.
	E_NOT_AVAILABLE ErrorCodes = "E_NOT_AVAILABLE"

	// Operation failed while the request is valid but performing it would violate some constraints.
	E_NOT_POSSIBLE ErrorCodes = "E_NOT_POSSIBLE"

	// Operation depends on some external system that cannot be reached.
	E_NOT_REACHABLE ErrorCodes = "E_NOT_REACHABLE"

	// Operation is denied because invoker has not enough resource quota.
	E_QUOTA_LIMIT ErrorCodes = "E_QUOTA_LIMIT"

	// Operation succeed.
	E_SUCCESS ErrorCodes = "E_SUCCESS"

	// Operation failed because something is wrong in the system.
	E_SYSTEM_FAILURE ErrorCodes = "E_SYSTEM_FAILURE"

	// Operation failed because some system limit has been exceed.
	E_SYSTEM_LIMIT ErrorCodes = "E_SYSTEM_LIMIT"

	// Operation failed because invoker tried using wrong credentials.
	E_WRONG_CREDENTIALS ErrorCodes = "E_WRONG_CREDENTIALS"

	// Operation failed because the target does not exist.
	E_DOES_NOT_EXIST ErrorCodes = "E_DOES_NOT_EXIST"

	// Operation failed due to unspecified reason (do not use this if at all possible!!)
	E_UNSPECIFIED ErrorCodes = "E_UNSPECIFIED"

	// Not appliciable.
	E_NA ErrorCodes = "E_NA"
)

type Results string

const (
	Success Results = "success"
	Failed  Results = "failure"
)

type AuditRecord struct {
	User                    string       `structs:"user,omitempty"`
	Result                  Results      `structs:"result,omitempty"`
	Operation               string       `structs:"operation,omitempty"`
	Msg                     string       `structs:"-"`
	EventType               EventTypes   `structs:"eventtype,omitempty"`
	ErrorCode               ErrorCodes   `structs:"errorcode,omitempty"`
	Level                   SyslogLevels `structs:"level,omitempty"`
	Process                 string       `structs:"process,omitempty"`
	Service                 string       `structs:"service,omitempty"`
	System                  string       `structs:"system,omitempty"`
	NeID                    string       `structs:"neid,omitempty"`
	TimeZone                string       `structs:"timezone,omitempty"`
	Container               string       `structs:"container,omitempty"`
	Host                    string       `structs:"host,omitempty"`
	TargetUserIdentity      string       `structs:"target-user-identity,omitempty"`
	TargetSessionIdentifier string       `structs:"target-session-identity,omitempty"`
	SourcePort              string       `structs:"source-port,omitempty"`
	SourceIP                string       `structs:"source-ip,omitempty"`
	SourceUserIdentity      string       `structs:"source-user-identity,omitempty"`
	SourceSessionIdentifier string       `structs:"source-session-identity,omitempty"`
	SessionID               string       `structs:"session-id,omitempty"`
	ProxySourcePort         string       `structs:"proxy-source-port,omitempty"`
	ProxySourceIP           string       `structs:"proxy-source-ip,omitempty"`
	ProxyBackendPort        string       `structs:"proxy-backend-port,omitempty"`
	ProxyBackendIP          string       `structs:"proxy-backend-ip,omitempty"`
	ObjectID                string       `structs:"object-id,omitempty"`
	ObjectAttributes        string       `structs:"object-attributes,omitempty"`
	Object                  string       `structs:"object,omitempty"`
	Interface               string       `structs:"interface,omitempty"`
	BackendPort             string       `structs:"backend-port,omitempty"`
	BackendIP               string       `structs:"backend-ip,omitempty"`
}

// AuditLogger is the interface for audit loggers.
type AuditLogger interface {
	Audit(AuditRecord)
	Auth(AuditRecord)
}

type auditLogger struct {
	entry *logrus.Entry
}

// Audit logs a message at level Info on the standard logger.
func (l auditLogger) Audit(args AuditRecord) {
	auditLog := structs.Map(args)
	auditLog[facilityName] = auditFacility
	auditLog[logTypeName] = logType
	auditLog[timeZoneName] = time.Local.String()
	l.entry.WithFields(auditLog).Info(args.Msg)
}

func (l auditLogger) Auth(args AuditRecord) {
	authLog := structs.Map(args)
	authLog[facilityName] = authFacility
	authLog[logTypeName] = logType
	authLog[timeZoneName] = time.Local.String()
	l.entry.WithFields(authLog).Info(args.Msg)
}

// NewAuditLogger returns a new Logger logging to stderr.
//
// Logger configuration is done in a way that it complies
// with Neo logging standards, configuration can be changed with
// environment variables as follows:
//
//	Variable            | Values
//	-----------------------------------------------------------
//	LOGGING_FORMAT      | 'json' (default), 'txt'
//
// If invalid configuration is given NewLogger will return Logger
// with default configuration and handle error by logging it.
// Log events contains following fields by default:
//
//	timestamp
//	message
//	logger
//
// # Log metrics
//
// Logger will automatically collect metrics (log event counters) for Prometheus.
// Metrics will be exposed only if you run metrics.ManagementServer in your application.
func NewAuditLogger() AuditLogger {
	_, format, _ := parseConfig()
	level, _ := logrus.ParseLevel("info")
	l := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: format,
		Hooks:     make(logrus.LevelHooks),
		Level:     level,
	}
	l.Hooks.Add(auditHook)
	return auditLogger{entry: logrus.NewEntry(l)}
}
