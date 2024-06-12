package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/phanitejak/gopkg/logging/v2"
	"github.com/phanitejak/gopkg/logging/v2/testutil"
	"github.com/phanitejak/gopkg/metrics"
	"github.com/phanitejak/gopkg/testutils"
	"github.com/phanitejak/gopkg/tracing"
	"github.com/phanitejak/gopkg/tracing/tracingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleNewLogger_withValidRequestContext() {
	log := logging.NewLogger()
	_, err := tracing.InitGlobalTracer()
	if err != nil {
		// Error handle somehow
	}

	// Either create a span from existing request context or create a fresh one
	span, ctx := tracing.StartSpanFromContext(context.Background(), "testOperation")
	defer span.Finish()

	log.Info(ctx, "Message")
	// Example Output:
	// {"is_sampled":"false","level":"info","logger":"logging_test.go:70","message":"Message","parent_id":"0","span_id":"54b168451e541edd","timestamp":"2020-12-11T12:02:00.370+02:00","trace_id":"54b168451e541edd"}
}

func ExampleNewLogger_withEmptyContext() {
	log := logging.NewLogger()
	log.Info(context.Background(), "Message")
	log.Info(context.TODO(), "Other message")
	// Example Output:
	// {"level":"info","logger":"logging_test.go:74","message":"Message","timestamp":"2020-12-11T13:00:31.316+02:00"}
	// {"level":"info","logger":"logging_test.go:75","message":"Other message","timestamp":"2020-12-11T13:00:31.316+02:00"}
}

func ExampleNewLogger_with() {
	log := logging.NewLogger()
	log = log.With("key", "value")
	log.Info(context.Background(), "Message")
	// Example Output:
	// {"key":"value","level":"info","logger":"logging_test.go:47","message":"Message","timestamp":"2018-03-01T10:44:33.303+02:00"}
}

func ExampleNewLogger_withCustomInterfaceFields() {
	log := logging.NewLogger()
	fields := map[string]interface{}{
		"trace_id": 12345,
		"user_id":  "haapis",
		"origin":   "127.0.0.1",
	}
	logWithContext := log.WithFields(fields)
	logWithContext.Info(context.Background(), "Message")
	// Example Output:
	// {"level":"info","logger":"logging_test.go:63","message":"Message","origin":"127.0.0.1","timestamp":"2018-03-01T11:01:13.415+02:00","trace_id":12345,"user_id":"haapis"}
}

func TestDefaultFieldsInInfo(t *testing.T) {
	expectedLogMessage := map[string]string{
		"level":   "info",
		"message": "huhuu",
	}

	logger, logOutput := getLogger(t)
	logger.Info(context.Background(), "huhuu")
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
	logger.Error(context.Background(), expectedMessage)
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
	logger.With("context", expectedContext).Error(context.Background(), expectedMessage)
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
	logger.WithFields(newFields).Error(context.Background(), expectedMessage)
	logMessage := testutil.UnmarshalLogMessage(t, logOutput().Bytes())

	testutil.AssertKeyInMap(t, "stack_trace", logMessage)
	assert.Equal(t, expectedLevel, logMessage["level"])
	assert.Equal(t, expectedMessage, logMessage["message"])
	assert.Equal(t, expectedContext, logMessage["context"])
}

func TestConfigChanges(t *testing.T) {
	cleanUp := testutils.SetEnv(t, map[string]string{
		"LOGGING_FORMAT": "txt",
		"LOGGING_LEVEL":  "debug",
	})
	defer cleanUp()

	expectedMessage := "This is debug"

	logger, logOutput := getLogger(t)
	logger.Debug(context.Background(), expectedMessage)
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
			log.Info(context.Background(), "logging ... ")
		}()
	}
	wg.Wait()
}

func TestStdLogger(t *testing.T) {
	cleanUp := testutils.SetEnv(t, map[string]string{
		"LOGGING_FORMAT": "txt",
		"LOGGING_LEVEL":  "debug",
	})
	defer cleanUp()

	expectedMessage := "This is debug"

	logger, logOutput := getLogger(t)
	stdLogger := logger.(logging.StdLogger)

	stdLogger.Println(context.Background(), expectedMessage)
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

	logger.Info(context.Background(), "This is a random log.. with some sensitive data here -- # ~ * ^ "+encodedSensitiveUser+" from location: "+logging.PrivacyDataFormatter(ipAddress))

	actualLog := logOutput().String()

	assert.Equal(t, expectedEncodedUserName, encodedSensitiveUser)
	assert.Equal(t, expectedEncodedIPAddress, logging.PrivacyDataFormatter(ipAddress))

	assert.Contains(t, actualLog, expectedEncodedUserName)
	assert.Contains(t, actualLog, expectedEncodedIPAddress)

	re := regexp.MustCompile(`(\[_priv_]).*?(\[/_priv_])`)

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
	cleanUp := testutils.SetEnv(t, map[string]string{
		"LOGGING_LEVEL": "debug",
	})
	defer cleanUp()

	managementServer := metrics.StartManagementServer(":9876", nil)
	defer managementServer.Close()
	logger, _ := getLogger(t)
	for i := 0; i < debugMsgCount; i++ {
		logger.Debug(context.Background(), "debug msg")
	}
	for i := 0; i < errorMsgCount; i++ {
		logger.Error(context.Background(), "error msg")
	}
	for i := 0; i < infoMsgCount; i++ {
		logger.Info(context.Background(), "info msg")
	}
	resp, err := http.Get("http://localhost:9876" + metrics.DefaultEndPoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

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

// --- Traceable logging tests ---

func TestLoggingForBackgroundContextShouldWork(t *testing.T) {
	logger := logging.NewLogger()

	logger.Info(context.Background(), "Test")
}

func TestLoggingForEmptyContextShouldWork(t *testing.T) {
	logger := logging.NewLogger()

	logger.Info(context.TODO(), "Test")
}

func TestLoggingForContext(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()
	span, ctx := tracing.StartSpanFromContext(context.Background(), "testSpan")
	defer span.Finish()

	logger := logging.NewLogger()

	logger.Info(ctx, "Test")
	logger.Infof(ctx, "Test")
	logger.Infoln(ctx, "Test")
	logger.Debug(ctx, "Test")
	logger.Debugf(ctx, "Test")
	logger.Debugln(ctx, "Test")
	logger.Error(ctx, "Test")
	logger.Errorf(ctx, "Test")
	logger.Errorln(ctx, "Test")
}

func TestShouldLogIsSampledAsString(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	_, ctx := tracing.StartSpanFromContext(context.Background(), "testSpan")

	r, w, err := os.Pipe()
	require.NoError(t, err)
	stderr := os.Stderr
	defer func() {
		os.Stderr = stderr
	}()

	os.Stderr = w
	logger := logging.NewLogger()

	logger.Info(ctx, "Test")

	err = w.Close()
	require.NoError(t, err)

	decoder := json.NewDecoder(r)

	logEntry := struct {
		IsSampled string `json:"is_sampled"`
		TraceID   string `json:"trace_id"`
		SpanID    string `json:"span_id"`
		ParentID  string `json:"parent_id"`
		Level     string `json:"level"`
		Logger    string `json:"logger"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
	}{}

	err = decoder.Decode(&logEntry)
	require.NoError(t, err)
}

func TestLogFatal(t *testing.T) {
	if os.Getenv("CRASH_APPLICATION") == "1" {
		_, ctx := tracing.StartSpanFromContext(context.Background(), "crashingSpan")

		logger := logging.NewLogger()

		logger.Fatal(ctx, "Crashing application")
		return
	}
	runTest("TestLogFatal", t)
}

func TestLogFatalf(t *testing.T) {
	if os.Getenv("CRASH_APPLICATION") == "1" {
		_, ctx := tracing.StartSpanFromContext(context.Background(), "crashingSpan")

		logger := logging.NewLogger()

		logger.Fatalf(ctx, "Crashing application")
		return
	}
	runTest("TestLogFatalf", t)
}

func TestLogFatalln(t *testing.T) {
	if os.Getenv("CRASH_APPLICATION") == "1" {
		_, ctx := tracing.StartSpanFromContext(context.Background(), "crashingSpan")

		logger := logging.NewLogger()

		logger.Fatalln(ctx, "Crashing application")
		return
	}
	runTest("TestLogFatalln", t)
}

func runTest(testName string, t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run="+testName) //nolint:gosec
	cmd.Env = append(os.Environ(), "CRASH_APPLICATION=1")
	err := cmd.Run()
	var e *exec.ExitError
	ok := errors.As(err, &e)
	require.True(t, ok, "error should be of type ExitError")
	assert.False(t, e.Success(), "error should not have status success")
	assert.Equal(t, "exit status 1", e.String())
}

type logMsg struct {
	Level      string    `json:"level"`
	Logger     string    `json:"logger"`
	Message    string    `json:"message"`
	StackTrace string    `json:"stack_trace"`
	Timestamp  time.Time `json:"timestamp"`
}

func TestLoggerImplementsIncDepth(t *testing.T) {
	type incremental interface {
		IncDepth(depth int) logging.Logger
	}

	log := logging.NewLogger()
	l, ok := log.(incremental)
	require.True(t, ok, "logger doesn't implement IncDepth(depth int) Logger method")
	log = l.IncDepth(0)
	log.Debug(context.Background())
}

func TestLoggerImplementsStdLogger(t *testing.T) {
	log := logging.NewLogger()
	l, ok := log.(logging.StdLogger)
	require.True(t, ok, "logger doesn't implement StdLogger")
	l.Print(context.Background(), "")
	l.Printf(context.Background(), "")
	l.Println(context.Background(), "")
}

func TestLoggingFromExecutable(t *testing.T) {
	for _, file := range []string{"basic", "wrap"} {
		file := file
		t.Run(file, func(t *testing.T) {
			cmd := exec.Command("go", "run", "../testdata/"+file+".go") //nolint: gosec
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
