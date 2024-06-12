package logging_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/phanitejak/gopkg/logging"
	"github.com/phanitejak/gopkg/logging/testutil"
	"github.com/phanitejak/gopkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleNewLogger_basic() {
	log := logging.NewLogger()
	log.Info("Message")
	// Example Output:
	// {"level":"info","logger":"logging_test.go:36","message":"Message","timestamp":"2018-03-01T10:43:31.269+02:00"}
}

func ExampleNewLogger_with() {
	log := logging.NewLogger()
	log = log.With("key", "value")
	log.Info("Message")
	// Example Output:
	// {"key":"value","level":"info","logger":"logging_test.go:47","message":"Message","timestamp":"2018-03-01T10:44:33.303+02:00"}
}

func ExampleNewLogger_withFields() {
	log := logging.NewLogger()
	context := map[string]interface{}{
		"trace_id": 12345,
		"user_id":  "haapis",
		"origin":   "127.0.0.1",
	}
	logWithContext := log.WithFields(context)
	logWithContext.Info("Message")
	// Example Output:
	// {"level":"info","logger":"logging_test.go:63","message":"Message","origin":"127.0.0.1","timestamp":"2018-03-01T11:01:13.415+02:00","trace_id":12345,"user_id":"haapis"}
}

func TestDefaultFieldsInInfo(t *testing.T) {
	expectedLogMessage := map[string]string{
		"level":   "info",
		"message": "huhuu",
	}

	logger, logOutput := getLogger(t)
	logger.Info("huhuu")
	logMessage := testutil.UnmarshalLogMessage(t, logOutput().Bytes())

	assertKeyNotInMap(t, "stack_trace", logMessage)
	assert.Contains(t, logMessage["logger"], "logging_test.go")
	delete(logMessage, "logger")

	if _, err := time.Parse(logging.ISO8601, logMessage["timestamp"]); err != nil {
		t.Fatalf("'timestamp' field missing or has wrong value %s", err)
	}

	// Delete timestamp so that we can assert other fields
	delete(logMessage, "timestamp")
	assert.Equal(t, expectedLogMessage, logMessage)
}

func TestError(t *testing.T) {
	expectedLevel := "error"
	expectedMessage := "This is error"

	logger, logOutput := getLogger(t)
	logger.Error(expectedMessage)
	logMessage := testutil.UnmarshalLogMessage(t, logOutput().Bytes())

	testutil.AssertKeyInMap(t, "stack_trace", logMessage)
	assert.Equal(t, expectedLevel, logMessage["level"])
	assert.Equal(t, expectedMessage, logMessage["message"])
}

func TestWithField(t *testing.T) {
	expectedLevel := "error"
	expectedMessage := "This is error"
	expectedContext := "test"

	logger, logOutput := getLogger(t)
	logger.With("context", expectedContext).Error(expectedMessage)
	logMessage := testutil.UnmarshalLogMessage(t, logOutput().Bytes())

	testutil.AssertKeyInMap(t, "stack_trace", logMessage)
	assert.Equal(t, expectedLevel, logMessage["level"])
	assert.Equal(t, expectedMessage, logMessage["message"])
	assert.Equal(t, expectedContext, logMessage["context"])
}

func TestWithFields(t *testing.T) {
	expectedLevel := "error"
	expectedMessage := "This is error"
	expectedContext := "test"
	newFields := map[string]interface{}{
		"context":    "test",
		"tracker_id": "1",
	}

	logger, logOutput := getLogger(t)
	logger.WithFields(newFields).Error(expectedMessage)
	logMessage := testutil.UnmarshalLogMessage(t, logOutput().Bytes())

	testutil.AssertKeyInMap(t, "stack_trace", logMessage)
	assert.Equal(t, expectedLevel, logMessage["level"])
	assert.Equal(t, expectedMessage, logMessage["message"])
	assert.Equal(t, expectedContext, logMessage["context"])
}

func TestConfigChanges(t *testing.T) {
	os.Setenv("LOGGING_FORMAT", "txt")
	os.Setenv("LOGGING_LEVEL", "debug")
	expectedMessage := "This is debug"

	logger, logOutput := getLogger(t)
	logger.Debug(expectedMessage)
	message := logOutput().String()
	assert.Contains(t, message, expectedMessage)
	assert.Contains(t, message, "level=debug")
}

func TestLoggingConcurrency(t *testing.T) {
	wg := sync.WaitGroup{}
	for index := 0; index < 100; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log := logging.NewLogger()
			log.Info("logging ... ")
		}()
	}
	wg.Wait()
}

func TestStdLogger(t *testing.T) {
	os.Setenv("LOGGING_FORMAT", "txt")
	os.Setenv("LOGGING_LEVEL", "debug")
	expectedMessage := "This is debug"

	logger, logOutput := getLogger(t)
	stdLogger := logger.(logging.StdLogger)

	stdLogger.Println(expectedMessage)
	message := logOutput().String()
	assert.Contains(t, message, expectedMessage)
	assert.Contains(t, message, "level=debug")
}

func TestPrivacyDataFormatter(t *testing.T) {
	logger, logOutput := getLogger(t)

	userName := "userName"
	ipAddress := "192.168.0.1"
	expectedEncodedUserName := fmt.Sprintf("[_priv_]%s[/_priv_]", userName)
	expectedEncodedIPAddress := fmt.Sprintf("[_priv_]%s[/_priv_]", ipAddress)

	encodedSensitiveUser := logging.PrivacyDataFormatter(userName)

	logger.Info("This is a random log.. with some sensitive data here -- # ~ * ^ " + encodedSensitiveUser + " from location: " + logging.PrivacyDataFormatter(ipAddress))

	actualLog := logOutput().String()

	assert.Equal(t, expectedEncodedUserName, encodedSensitiveUser)
	assert.Equal(t, expectedEncodedIPAddress, logging.PrivacyDataFormatter(ipAddress))

	assert.Contains(t, actualLog, expectedEncodedUserName)
	assert.Contains(t, actualLog, expectedEncodedIPAddress)

	re := regexp.MustCompile("(\\[_priv_\\]).*?(\\[\\/_priv_\\])")

	sensitiveDataStrings := re.FindAllString(actualLog, -1)

	assert.Equal(t, len(sensitiveDataStrings), 2)
	assert.Equal(t, expectedEncodedUserName, sensitiveDataStrings[0])
	assert.Equal(t, expectedEncodedIPAddress, sensitiveDataStrings[1])
}

func TestLoggingMetrics(t *testing.T) {
	var (
		debugMsgCount                  = 1
		errorMsgCount                  = 2
		infoMsgCount                   = 3
		levelDebug                     = "debug"
		levelInfo                      = "info"
		levelError                     = "error"
		loggingMetricsContentTestCases = []struct {
			str  string
			name string
		}{
			{
				fmt.Sprintf("logger_events_total{_plain_metric_name=\"events_total\",level=\"%s\"}", levelDebug),
				fmt.Sprintf("Test %s log events found from the metrics endpoint response", levelDebug),
			},
			{
				fmt.Sprintf("logger_events_total{_plain_metric_name=\"events_total\",level=\"%s\"}", levelInfo),
				fmt.Sprintf("Test %s log events found from the metrics endpoint response", levelInfo),
			},
			{
				fmt.Sprintf("logger_events_total{_plain_metric_name=\"events_total\",level=\"%s\"}", levelError),
				fmt.Sprintf("Test %s log events found from the metrics endpoint response", levelError),
			},
		}

		loggingMetricsContentNonZeroTestCases = []struct {
			str  string
			name string
		}{
			{
				fmt.Sprintf("logger_events_total{_plain_metric_name=\"events_total\",level=\"%s\"} 0", levelDebug),
				fmt.Sprintf("Test %s log events count is non-zero in the metrics endpoint response", levelDebug),
			},
			{
				fmt.Sprintf("logger_events_total{_plain_metric_name=\"events_total\",level=\"%s\"} 0", levelInfo),
				fmt.Sprintf("Test %s log events count is non-zero in the metrics endpoint response", levelInfo),
			},
			{
				fmt.Sprintf("logger_events_total{_plain_metric_name=\"events_total\",level=\"%s\"} 0", levelError),
				fmt.Sprintf("Test %s log events count is non-zero in the metrics endpoint response", levelError),
			},
		}
	)
	os.Setenv("LOGGING_LEVEL", "debug")
	managementServer := metrics.StartManagementServer(":9876", nil)
	defer managementServer.Close()
	logger, _ := getLogger(t)
	for i := 0; i < debugMsgCount; i++ {
		logger.Debug("debug msg")
	}
	for i := 0; i < errorMsgCount; i++ {
		logger.Error("error msg")
	}
	for i := 0; i < infoMsgCount; i++ {
		logger.Info("info msg")
	}
	resp, err := http.Get("http://localhost:9876" + metrics.DefaultEndPoint)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, resp.StatusCode, "Status code 200 is returned from "+
		metrics.DefaultEndPoint+" endpoint")
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	for _, testCase := range loggingMetricsContentTestCases {
		assert.Contains(t, body, testCase.str, testCase.name)
	}

	for _, testCase := range loggingMetricsContentNonZeroTestCases {
		assert.NotContains(t, body, testCase.str, testCase.name)
	}
}

type logMsg struct {
	Level      string    `json:"level"`
	Logger     string    `json:"logger"`
	Message    string    `json:"message"`
	StackTrace string    `json:"stack_trace"`
	Timestamp  time.Time `json:"timestamp"`
}

func TestLoggingFromExecutable(t *testing.T) {
	for _, file := range []string{"basic", "wrap"} {
		file := file
		t.Run(file, func(t *testing.T) {
			cmd := exec.Command("go", "run", "testdata/"+file+".go") //nolint: gosec
			out, err := cmd.CombinedOutput()

			e := &exec.ExitError{}
			require.ErrorAs(t, err, &e)
			assert.Equal(t, 1, e.ExitCode())

			var msg logMsg
			lines := bytes.Split(out, []byte("\n"))
			require.NoError(t, json.Unmarshal(lines[0], &msg))
			assert.Equal(t, "debug", msg.Level)
			assert.Equal(t, file+".go:14", msg.Logger)
			assert.Equal(t, "debug msg", msg.Message)
			require.NoError(t, json.Unmarshal(lines[1], &msg))
			assert.Equal(t, "info", msg.Level)
			assert.Equal(t, file+".go:15", msg.Logger)
			assert.Equal(t, "info msg", msg.Message)
			require.NoError(t, json.Unmarshal(lines[2], &msg))
			assert.Equal(t, "error", msg.Level)
			assert.Equal(t, file+".go:16", msg.Logger)
			assert.Equal(t, "error msg", msg.Message)
			require.NoError(t, json.Unmarshal(lines[3], &msg))
			assert.Equal(t, "error", msg.Level)
			assert.Equal(t, file+".go:17", msg.Logger)
			assert.Equal(t, "fatal msg", msg.Message)
		})
	}
}

func TestLoggerImplementsIncDepth(t *testing.T) {
	type incremental interface {
		IncDepth(depth int) logging.Logger
	}

	log := logging.NewLogger()
	l, ok := log.(incremental)
	require.True(t, ok, "logger doesn't implement IncDepth(depth int) Logger method")
	log = l.IncDepth(0)
	log.Debug()
}

func TestLoggerImplementsStdLogger(t *testing.T) {
	log := logging.NewLogger()
	l, ok := log.(logging.StdLogger)
	require.True(t, ok, "logger doesn't implement StdLogger")
	l.Print("")
	l.Printf("")
	l.Println("")
}

func TestLogger(t *testing.T) {
	log := logging.NewLogger()
	log.Debugln("")
	log.Debugf("")
	log.Infoln("")
	log.Infof("")
	log.Errorln("")
	log.Errorf("")
}

func assertKeyNotInMap(t *testing.T, key string, m map[string]string) {
	if _, ok := m[key]; ok {
		t.Fatalf("key '%s' is in map '%v'", key, m)
	}
}

func getLogger(t *testing.T) (logging.Logger, func() *bytes.Buffer) {
	logOutput := testutil.PipeStderr(t)
	logger := logging.NewLogger()
	return logger, logOutput
}
