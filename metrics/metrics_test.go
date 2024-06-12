package metrics_test

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/phanitejak/gopkg/metrics"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const plainMetricNameKey = "_plain_metric_name"

var (
	metricNamespace = "com_nokia_neo_metrics"
	statusEndPoint  = "/status"

	testServerURLPrefix = "http://localhost"
	testServerAddr      = ":8181"
	counterName         = "testCounter"
	counterSubsystem    = "mySubSystem1"
	counterDesc         = "testCounterDesc"
	counterValue        = 1

	gaugeName      = "testGauge"
	gaugeSubsystem = "mySubSystem2"
	gaugeDesc      = "testGaugeDesc"
	gaugeValue     = float64(7)

	summaryName      = "testSummary"
	summarySubsystem = "mySummarySubSystem"
	summaryDesc      = "testSummaryDesc"
	summaryValue     = float64(11)
	summaryCount     = 13

	summaryWithObjectivesName      = "testSummaryWithObjectives"
	summaryWithObjectivesSubsystem = "mySummaryWithObjectivesSubSystem"
	summaryWithObjectivesDesc      = "testSummaryWithObjectivesDesc"
	summaryWithObjectivesQuantile  = map[float64]float64{0: 0, 0.25: 0.025, 0.5: 0.05, 0.75: 0.025, 0.9: 0.01, 0.99: 0.001, 1: 0}

	summaryDurationName  = "testSummaryDuration"
	summaryDurationValue = 5

	counterVecName             = "testCounterVec"
	counterVecSubsystem        = "ss"
	counterVecDesc             = "desc"
	counterVecLabels           = []string{"code", "method"}
	counterVecElem1CodeLabel   = "404"
	counterVecElem1MethodLabel = "GET"
	counterVecElem1Value       = int64(2)
	counterVecElem2CodeLabel   = "200"
	counterVecElem2MethodLabel = "POST"
	counterVecElem2Value       = int64(1)

	gaugeVecName          = "testGaugeVec"
	gaugeVecSubsystem     = "ss"
	gaugeVecDesc          = "desc"
	gaugeVecLabels        = []string{"bar", "foo"}
	gaugeVecElem1FooLabel = "bar"
	gaugeVecElem1BarLabel = "foo"
	gaugeVecElem1Value    = float64(1)
	gaugeVecElem2FooLabel = "xxx"
	gaugeVecElem2BarLabel = "yyy"
	gaugeVecElem2Value    = float64(2)

	summaryVecName          = "testSummaryVec"
	summaryVecSubsystem     = "ssummarySubsystem"
	summaryVecDesc          = "summaryVec desc"
	summaryVecLabels        = []string{"kaa", "yyy"}
	summaryVecElem1YyyLabel = "blah"
	summaryVecElem1KaaLabel = "hello"
	summaryVecElem1Value    = float64(4)
	summaryVecElem1Count    = 7
	summaryVecElem2YyyLabel = "hey"
	summaryVecElem2KaaLabel = "you"
	summaryVecElem2Value    = float64(2)
	summaryVecElem2Count    = 3

	metricHTTPActiveRequestsName   = "http_server_active_requests_count"
	metricHTTPRequestsDurationName = "http_server_requests_duration_seconds"
	metricHTTPResponsesSizeName    = "http_server_responses_size_bytes"
	metricHTTPRequestsSizeName     = "http_server_requests_size_bytes"
	activeRequestName              = metricHTTPActiveRequestsName
	requestDurationSum             = metricHTTPRequestsDurationName + "_sum"
	requestDurationCount           = metricHTTPRequestsDurationName + "_count"
	requestSizeSum                 = metricHTTPRequestsSizeName + "_sum"
	requestSizeCount               = metricHTTPRequestsSizeName + "_count"
	responseSizeSum                = metricHTTPResponsesSizeName + "_sum"
	responseSizeCount              = metricHTTPResponsesSizeName + "_count"
	activeRequestCount             = 1

	requestMethod   = "GET"
	request200URI   = "/test"
	request200Code  = "200"
	request200Count = 10

	request404URI         = "/test2"
	request404URIResponse = "/test2"
	request404Code        = "404"
	request404Count       = 10

	request200AfterRuleURI = "/originalwas" + request200URI + "/butnowthis"
	endpointNameAfterRule  = "/application/notprometheus"

	goMetricsMetricName            = "fooGoMetric"
	goMetricsMetricValue           = int64(69)
	goMetricsMetricRegistry2Prefix = "prefix"
	testCases                      = []struct {
		str  string
		name string
	}{
		{fmt.Sprintf("%s_%s_%s %d", metricNamespace, counterSubsystem,
			counterName, counterValue), "Test counter"},
		{fmt.Sprintf("# HELP %s_%s_%s %s", metricNamespace, counterSubsystem,
			counterName, counterDesc), "Test counter help"},
		{fmt.Sprintf("# TYPE %s_%s_%s counter", metricNamespace,
			counterSubsystem, counterName), "Test counter type"},
		{fmt.Sprintf("%s_%s_%s %d", metricNamespace, gaugeSubsystem, gaugeName,
			int(gaugeValue)), "Test gauge"},
		{fmt.Sprintf("# HELP %s_%s_%s %s", metricNamespace, gaugeSubsystem,
			gaugeName, gaugeDesc), "Test gauge help"},
		{fmt.Sprintf("# TYPE %s_%s_%s gauge", metricNamespace, gaugeSubsystem,
			gaugeName), "Test gauge type"},
		{fmt.Sprintf("%s_%s_%s_sum %d", metricNamespace, summarySubsystem, summaryName,
			int(summaryValue)*summaryCount), "Test summary sum"},
		{fmt.Sprintf("%s_%s_%s_count %d", metricNamespace, summarySubsystem, summaryName,
			summaryCount), "Test summary count"},
		{fmt.Sprintf("# HELP %s_%s_%s %s", metricNamespace, summarySubsystem,
			summaryName, summaryDesc), "Test summary help"},
		{fmt.Sprintf("# TYPE %s_%s_%s summary", metricNamespace, summarySubsystem,
			summaryName), "Test summary type"},
		{fmt.Sprintf("%s_%s_%s_sum %d", metricNamespace, summarySubsystem, summaryDurationName,
			summaryDurationValue), "Test duration summary sum"},
		{fmt.Sprintf("%s_%s_%s_count %d", metricNamespace, summarySubsystem, summaryDurationName,
			1), "Test duration summary count"},
		{
			fmt.Sprintf("%s_%s_%s{%s=\"%s\",%s=\"%s\",%s=\"%s\"} %d", metricNamespace,
				counterVecSubsystem, counterVecName, plainMetricNameKey, counterVecName, counterVecLabels[0],
				counterVecElem1CodeLabel, counterVecLabels[1],
				counterVecElem1MethodLabel, counterVecElem1Value),
			"Test counter vector elem1",
		},
		{
			fmt.Sprintf("%s_%s_%s{%s=\"%s\",%s=\"%s\",%s=\"%s\"} %d", metricNamespace,
				counterVecSubsystem, counterVecName, plainMetricNameKey, counterVecName, counterVecLabels[0],
				counterVecElem2CodeLabel, counterVecLabels[1],
				counterVecElem2MethodLabel, counterVecElem2Value),
			"Test counter vector elem2",
		},
		{
			fmt.Sprintf("# HELP %s_%s_%s %s", metricNamespace,
				counterVecSubsystem, counterVecName, counterVecDesc),
			"Test counter vector help",
		},
		{fmt.Sprintf("# TYPE %s_%s_%s counter", metricNamespace,
			counterVecSubsystem, counterVecName), "Test counter vector type"},
		{fmt.Sprintf("%s_%s_%s{%s=\"%s\",%s=\"%s\",%s=\"%s\"} %d", metricNamespace,
			gaugeVecSubsystem, gaugeVecName, plainMetricNameKey, gaugeVecName, gaugeVecLabels[0],
			gaugeVecElem1FooLabel, gaugeVecLabels[1], gaugeVecElem1BarLabel,
			int(gaugeVecElem1Value)), "Test gauge vector elem1"},
		{fmt.Sprintf("%s_%s_%s{%s=\"%s\",%s=\"%s\",%s=\"%s\"} %d", metricNamespace,
			gaugeVecSubsystem, gaugeVecName, plainMetricNameKey, gaugeVecName, gaugeVecLabels[0],
			gaugeVecElem2FooLabel, gaugeVecLabels[1], gaugeVecElem2BarLabel,
			int(gaugeVecElem2Value)), "Test gauge vector elem2"},
		{fmt.Sprintf("# HELP %s_%s_%s %s", metricNamespace, gaugeVecSubsystem,
			gaugeVecName, gaugeVecDesc), "Test gauge vector help"},
		{fmt.Sprintf("# TYPE %s_%s_%s gauge", metricNamespace,
			gaugeVecSubsystem, gaugeVecName), "Test gauge vector type"},
		{
			fmt.Sprintf("%s_%s_%s_sum{%s=\"%s\",%s=\"%s\",%s=\"%s\"} %d", metricNamespace,
				summaryVecSubsystem, summaryVecName, plainMetricNameKey, summaryVecName, summaryVecLabels[0],
				summaryVecElem1YyyLabel, summaryVecLabels[1],
				summaryVecElem1KaaLabel, int(summaryVecElem1Value)*summaryVecElem1Count),
			"Test summary vector elem1",
		},
		{
			fmt.Sprintf("%s_%s_%s_sum{%s=\"%s\",%s=\"%s\",%s=\"%s\"} %d", metricNamespace,
				summaryVecSubsystem, summaryVecName, plainMetricNameKey, summaryVecName, summaryVecLabels[0],
				summaryVecElem2YyyLabel, summaryVecLabels[1],
				summaryVecElem2KaaLabel, int(summaryVecElem2Value)*summaryVecElem2Count),
			"Test summary vector elem2",
		},
		{
			fmt.Sprintf("%s_%s_%s_count{%s=\"%s\",%s=\"%s\",%s=\"%s\"} %d", metricNamespace,
				summaryVecSubsystem, summaryVecName, plainMetricNameKey, summaryVecName, summaryVecLabels[0],
				summaryVecElem1YyyLabel, summaryVecLabels[1],
				summaryVecElem1KaaLabel, summaryVecElem1Count),
			"Test summary vector elem1",
		},
		{
			fmt.Sprintf("%s_%s_%s_count{%s=\"%s\",%s=\"%s\",%s=\"%s\"} %d", metricNamespace,
				summaryVecSubsystem, summaryVecName, plainMetricNameKey, summaryVecName, summaryVecLabels[0],
				summaryVecElem2YyyLabel, summaryVecLabels[1],
				summaryVecElem2KaaLabel, summaryVecElem2Count),
			"Test summary vector elem2",
		},
		{
			fmt.Sprintf("# HELP %s_%s_%s %s", metricNamespace,
				summaryVecSubsystem, summaryVecName, summaryVecDesc),
			"Test summary vector help",
		},
		{fmt.Sprintf("# TYPE %s_%s_%s summary", metricNamespace,
			summaryVecSubsystem, summaryVecName), "Test summary vector type"},
		{
			fmt.Sprintf("go_info{version=\"%s\"} 1", runtime.Version()),
			"Built-in go_info metric",
		},
	}

	httpTestCases = []struct {
		str  string
		name string
	}{
		{
			fmt.Sprintf("%s{method=\"%s\",uri=\"%s\"} %d", activeRequestName, requestMethod, metrics.DefaultEndPoint, activeRequestCount),
			"Test http_active_requests",
		},
		{
			fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
				requestDurationSum, requestMethod, request200Code, request200URI),
			"Test " + requestDurationSum + " " + request200URI,
		},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
			requestDurationSum, requestMethod, request404Code,
			request404URIResponse), "Test " + requestDurationSum + " " +
			request404URI},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
			requestDurationSum, requestMethod, request200Code,
			metrics.DefaultEndPoint), "Test " + requestDurationSum + " " +
			metrics.DefaultEndPoint},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			requestDurationCount, requestMethod, request200Code, request200URI,
			request200Count), "Test " + requestDurationCount + " " +
			request200URI},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			requestDurationCount, requestMethod, request404Code, request404URIResponse,
			request404Count), "Test " + requestDurationCount + " " +
			request404URI},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			requestDurationCount, requestMethod, request200Code,
			metrics.DefaultEndPoint, request200Count*2), "Test " +
			requestDurationCount + " " + metrics.DefaultEndPoint},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			requestSizeCount, requestMethod, request200Code, request200URI,
			request200Count), "Test " + requestSizeCount + " " + request200URI},
		{
			fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
				requestSizeSum, requestMethod, request404Code, request404URIResponse),
			"Test " + requestSizeSum + " " + request404URI,
		},
		{
			fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
				requestSizeSum, requestMethod, request200Code, metrics.DefaultEndPoint),
			"Test " + requestSizeSum + " " + metrics.DefaultEndPoint,
		},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			responseSizeCount, requestMethod, request200Code, request200URI,
			request200Count), "Test " + responseSizeSum + " " + request200Code},
		{
			fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
				responseSizeSum, requestMethod, request404Code, request404URIResponse),
			"Test " + responseSizeSum + " " + request404URI,
		},
		{
			fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
				responseSizeSum, requestMethod, request200Code, metrics.DefaultEndPoint),
			"Test " + responseSizeSum + " " + metrics.DefaultEndPoint,
		},
	}

	httpTestCasesForRules = []struct {
		str  string
		name string
	}{
		{
			fmt.Sprintf("%s{method=\"%s\",uri=\"%s\"} %d", activeRequestName, requestMethod, endpointNameAfterRule, 1),
			"Test http_active_requests",
		},
		{
			fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
				requestDurationSum, requestMethod, request200Code, request200AfterRuleURI),
			"Test " + requestDurationSum + " " + request200AfterRuleURI,
		},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
			requestDurationSum, requestMethod, request200Code,
			endpointNameAfterRule), "Test " + requestDurationSum + " " +
			endpointNameAfterRule},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			requestDurationCount, requestMethod, request200Code, request200AfterRuleURI,
			request200Count), "Test " + requestDurationCount + " " +
			request200AfterRuleURI},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			requestDurationCount, requestMethod, request200Code,
			endpointNameAfterRule, request200Count*2), "Test " +
			requestDurationCount + " " + endpointNameAfterRule},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			requestSizeCount, requestMethod, request200Code, request200AfterRuleURI,
			request200Count), "Test " + requestSizeCount + " " + request200AfterRuleURI},
		{
			fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
				requestSizeSum, requestMethod, request200Code, endpointNameAfterRule),
			"Test " + requestSizeSum + " " + endpointNameAfterRule,
		},
		{fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"} %d",
			responseSizeCount, requestMethod, request200Code, request200AfterRuleURI,
			request200Count), "Test " + responseSizeSum + " " + request200Code},
		{
			fmt.Sprintf("%s{method=\"%s\",status=\"%s\",uri=\"%s\"}",
				responseSizeSum, requestMethod, request200Code, endpointNameAfterRule),
			"Test " + responseSizeSum + " " + endpointNameAfterRule,
		},
	}

	managementServerContentTestCases = []struct {
		str  string
		name string
	}{
		{
			fmt.Sprintf("uri=\"%s\"", statusEndPoint),
			fmt.Sprintf("Test %s endpoint is found from the metrics response", statusEndPoint),
		},
		{
			fmt.Sprintf("uri=\"%s\"", metrics.DefaultEndPoint),
			fmt.Sprintf("Test %s endpoint is found from the metrics response", metrics.DefaultEndPoint),
		},
	}

	httpRuleTestCases = []struct {
		str  string
		name string
	}{
		{"/credentials/v1/{id}/{type}", "Test api1 match"},
		{"/credentials/v1/123/fooType/not_mapped", "Test api1 not match"},
		{"/credentials/v1/alarm/{dn}/{systemAlarmId}", "Test api match when / is in path variable"},
	}

	neoCollectorTestCases = []struct {
		str  string
		name string
	}{
		{"timezone_offset_milliseconds", "Test timezone_offset_milliseconds metric expose"},
		{"process_cpu_count", "Test process_cpu_count metric expose"},
		{"process_cgo_calls", "Test process_cgo_calls metric expose"},
	}

	SwaggerJSON = json.RawMessage([]byte(`{
		"consumes": [
		  "application/json"
		],
		"produces": [
		  "application/json"
		],
		"schemes": [
		  "https"
		],
		"swagger": "2.0",
		"info": {
		  "description": "Provides credentials to NEO services.",
		  "title": "NEO Credentials Provider Service",
		  "version": "1.0.0"
		},
		"basePath": "/credentials/v1",
		"paths": {
		  "/": {
			"get": {
			  "description": "Fetches credentials of the specified type for the specified object.\n\nFor example:\n` + "`" + `` + "`" + `` + "`" + `\nGET /credentials/v1/?id=PLMN-PLMN/SBTS-1\u0026type=NE3S/WS\n` + "`" + `` + "`" + `` + "`" + `\n\nReturns NE3S/WS credentials:\n` + "`" + `` + "`" + `` + "`" + `\n{\"username\":\"string\",\"password\":\"string\"}\n` + "`" + `` + "`" + `` + "`" + `\n",
			  "tags": [
				"credentials-provider"
			  ],
			  "summary": "Fetch credentials using query",
			  "parameters": [
				{
				  "type": "string",
				  "description": "Object identifier, for example PLMN-PLMN/SBTS-1.",
				  "name": "id",
				  "in": "query",
				  "required": true
				},
				{
				  "type": "string",
				  "description": "Credentials type, for example NE3S/WS.",
				  "name": "type",
				  "in": "query",
				  "required": true
				}
			  ],
			  "responses": {
				"200": {
				  "$ref": "#/responses/Credentials"
				},
				"401": {
				  "$ref": "#/responses/Unauthorized"
				},
				"404": {
				  "$ref": "#/responses/CredentialsNotFound"
				}
			  }
			}
		  },
		  "/{id}/{type}": {
			"get": {
			  "description": "Fetches credentials of the specified type for the specified object.\n\nFor example:\n` + "`" + `` + "`" + `` + "`" + `\nGET /credentials/v1/PLMN-PLMN%2FSBTS-1/NE3S%2FWS\n` + "`" + `` + "`" + `` + "`" + `\n\nReturns NE3S/WS credentials:\n` + "`" + `` + "`" + `` + "`" + `\n{\"username\":\"string\",\"password\":\"string\"}\n` + "`" + `` + "`" + `` + "`" + `\n",
			  "tags": [
				"credentials-provider"
			  ],
			  "summary": "Fetch credentials using path",
			  "parameters": [
				{
				  "type": "string",
				  "description": "Percent encoded object identifier, for example PLMN-PLMN%2FSBTS-1.",
				  "name": "id",
				  "in": "path",
				  "required": true
				},
				{
				  "type": "string",
				  "description": "Percent encoded credentials type, for example NE3S%2FWS.",
				  "name": "type",
				  "in": "path",
				  "required": true
				}
			  ],
			  "responses": {
				"200": {
				  "$ref": "#/responses/Credentials"
				},
				"401": {
				  "$ref": "#/responses/Unauthorized"
				},
				"404": {
				  "$ref": "#/responses/CredentialsNotFound"
				}
			  }
			}
		  },
		  "/alarm/{dn}/{systemAlarmId}": {
			"get": {
			  "description": "Get an active alarm by key\n\nFor example:\n` + "`" + `` + "`" + `` + "`" + `\nGET /api/aal/v1/alarm/PLMN-PLMN%2FBSC-1/12347\n` + "`" + `` + "`" + `` + "`" + `\n\nReturns an active alarm for a given alarm distinguished name and system alarm identifier:\n` + "`" + `` + "`" + `` + "`" + `\n{\n    \"dn\": \"PLMN-PLMN/BSC-1\",\n    \"systemAlarmId\": \"12347\"\n    ...\n    ...\n}\n` + "`" + `` + "`" + `` + "`" + `\n",
			  "produces": [
				"application/json"
			  ],
			  "tags": [
				"active-alarms-list"
			  ],
			  "summary": "Get an active alarm by key",
			  "parameters": [
				{
				  "type": "string",
				  "description": "Encoded distinguished name of the alarm.",
				  "name": "dn",
				  "in": "path",
				  "required": true
				},
				{
				  "type": "string",
				  "description": "System alarm identifier of the alarm.",
				  "name": "systemAlarmId",
				  "in": "path",
				  "required": true
				}
			  ],
			  "responses": {
				"200": {
				  "description": "Fetching of an active alarm for the given dn and system alarm id is successful.",
				  "schema": {
					"$ref": "#/definitions/Alarm"
				  }
				},
				"404": {
				  "description": "Alarms for the specified dn and system alarm id was not found.",
				  "schema": {
					"$ref": "#/definitions/Error"
				  }
				},
				"500": {
				  "description": "Internal server error (e.g. lost database connection)",
				  "schema": {
					"$ref": "#/definitions/Error"
				  }
				}
			  }
			}
		  }
	    },
		"definitions": {
		  "Credentials": {
			"description": "The requested credentials in JSON format, for example {\"username\":\"string\",\"password\":\"string\"} for credentials of type NE3S/WS.",
			"type": "object"
		  },
		  "Error": {
			"description": "The request did not complete successfully.",
			"type": "object",
			"properties": {
			  "code": {
				"description": "Error code, for example 401.",
				"type": "integer"
			  },
			  "message": {
				"description": "Error message, for example unauthorized.",
				"type": "string"
			  }
			}
		  }
		},
		"responses": {
		  "Credentials": {
			"description": "The credentials of the specified type for the specified object.",
			"schema": {
			  "$ref": "#/definitions/Credentials"
			}
		  },
		  "CredentialsNotFound": {
			"description": "The credentials of the specified type for the specified object were not found.",
			"schema": {
			  "$ref": "#/definitions/Error"
			}
		  },
		  "Unauthorized": {
			"description": "Client was not properly authenticated.",
			"schema": {
			  "$ref": "#/definitions/Error"
			}
		  }
		}
	  }`))
)

// Only for examples.
type exampleConfig struct {
	MetricRegistry gometrics.Registry
}

type testRouter interface {
	Handle(pattern string, handler http.Handler)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type testCustomMetricVec interface {
	Reset()
	Unregister() bool
	DeleteSerie(labelValues ...string) bool
}

type testCustomMetric interface {
	Unregister() bool
}

var (
	kafkaConsumerConfig = &exampleConfig{gometrics.NewRegistry()}
	kafkaProducerConfig = &exampleConfig{gometrics.NewRegistry()}
	serveMux            = http.NewServeMux()
	goMetricsRegistry   = gometrics.NewRegistry()
)

// Use standalone server when your service doesn't have other (than health-check)
// HTTP endpoints to be exposed or you want metrics to be exposed on separate HTTP server.
func Example_standaloneMetricsServer() {
	// Start management server. Health-check is simply empty response (status code 200)
	managementServer := metrics.StartManagementServer(":9876", func(http.ResponseWriter, *http.Request) {})

	// If applicable, cross-register Kafka consumer metrics from Kafka consumer config
	metrics.MustCrossRegisterKafkaConsumerMetrics(kafkaConsumerConfig.MetricRegistry)

	// If applicable, cross-register Kafka producer metrics from Kafka producer config
	metrics.MustCrossRegisterKafkaProducerMetrics(kafkaProducerConfig.MetricRegistry)

	// If applicable, register some custom metrics
	testCounter := metrics.RegisterCounter("fooCounter", "my_service", "lorem ipsum...")
	testGauge := metrics.RegisterGauge("fooGauge", "my_service", "lorem ipsum...")
	testSummary := metrics.RegisterSummary("fooSummary", "my_service", "lorem ipsum...")
	testDurationSummary := metrics.RegisterSummary("fooDurationSummary", "my_service", "lorem ipsum...")
	testCounterVec := metrics.RegisterCounterVec("fooCounterWithTags", "my_service", "lorem ipsum...", "ctag1", "ctag2")
	testGaugeVec := metrics.RegisterGaugeVec("fooGaugeWithTags", "my_service", "lorem ipsum...", "gtag1", "gtag2")
	testSummaryVec := metrics.RegisterSummaryVec("fooSummaryWithTags", "my_service", "lorem ipsum...", "stag1", "stag2")

	start := time.Now()

	// Update custom metrics by this way
	testCounter.Inc()
	testCounter.Add(10)

	testGauge.Set(50.4)
	testGauge.Add(5)
	testGauge.Sub(5)

	testSummary.Observe(11.45)
	testSummary.Observe(12)

	// Observers elapsed time from setting start until this
	testDurationSummary.ObserveDuration(start)

	testCounterVec.GetCustomCounter("ctag1Value", "ctag2Value").Inc()
	testCounterVec.GetCustomCounter("ctag1Value2", "ctag2Value2").Inc()
	testCounterVec.GetCustomCounter("ctag1Value", "ctag2Value").Add(9)
	testCounterVec.GetCustomCounter("ctag1Value2", "ctag2Value2").Add(4)

	testGaugeVec.GetCustomGauge("gtag1Value", "gtag2Value").Set(6)
	testGaugeVec.GetCustomGauge("gtag1Value2", "gtag2Value2").Set(7)
	testGaugeVec.GetCustomGauge("gtag1Value", "gtag2Value").Add(3)
	testGaugeVec.GetCustomGauge("gtag1Value2", "gtag2Value2").Add(2)

	testSummaryVec.GetCustomSummary("stag1Value", "stag2Value").Observe(5)
	testSummaryVec.GetCustomSummary("stag1Value2", "stag2Value2").Observe(4)
	testSummaryVec.GetCustomSummary("stag1Value", "stag2Value").Observe(3)
	testSummaryVec.GetCustomSummary("stag1Value2", "sgtag2Value2").Observe(8)

	// At the shutdown of service, close management server gracefully
	managementServer.Close()
}

// Use embedded metrics server when your service has HTTP endpoints to
// be exposed and you want also embed metrics endpoint to this same HTTP server.
func Example_embeddedMetricsServer() {
	// Assign a default endpoint for metrics handler to your existing servemux
	serveMux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())

	// Instrument your servemux with pprof profiling endpoints
	metrics.InstrumentWithPprof(serveMux)

	// Assing your business endpoints to your handler normally using Handle or HandleFunc...
	serveMux.HandleFunc("/my_endpoint", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello!")
	})
	// ...

	// If applicable, cross-register Kafka consumer metrics from Kafka consumer config
	metrics.MustCrossRegisterKafkaConsumerMetrics(kafkaConsumerConfig.MetricRegistry)

	// If applicable, cross-register Kafka producer metrics from Kafka producer config
	metrics.MustCrossRegisterKafkaProducerMetrics(kafkaProducerConfig.MetricRegistry)

	// If applicable, register some custom metrics
	testCounter := metrics.RegisterCounter("fooCounter", "my_service", "lorem ipsum...")
	testGauge := metrics.RegisterGauge("fooGauge", "my_service", "lorem ipsum...")
	testSummary := metrics.RegisterSummary("fooSummary", "my_service", "lorem ipsum...")
	testDurationSummary := metrics.RegisterSummary("fooDurationSummary", "my_service", "lorem ipsum...")
	testCounterVec := metrics.RegisterCounterVec("fooCounterWithTags", "my_service", "lorem ipsum...", "ctag1", "ctag2")
	testGaugeVec := metrics.RegisterGaugeVec("fooGaugeWithTags", "my_service", "lorem ipsum...", "gtag1", "gtag2")
	testSummaryVec := metrics.RegisterSummaryVec("fooSummaryWithTags", "my_service", "lorem ipsum...", "stag1", "stag2")

	start := time.Now()

	// Update custom metrics by this way
	testCounter.Inc()
	testCounter.Add(10)

	testGauge.Set(50.4)
	testGauge.Add(5)
	testGauge.Sub(5)

	testSummary.Observe(11.45)
	testSummary.Observe(12)

	// Observers elapsed time from setting start until this
	testDurationSummary.ObserveDuration(start)

	testCounterVec.GetCustomCounter("ctag1Value", "ctag2Value").Inc()
	testCounterVec.GetCustomCounter("ctag1Value2", "ctag2Value2").Inc()
	testCounterVec.GetCustomCounter("ctag1Value", "ctag2Value").Add(9)
	testCounterVec.GetCustomCounter("ctag1Value2", "ctag2Value2").Add(4)

	testGaugeVec.GetCustomGauge("gtag1Value", "gtag2Value").Set(6)
	testGaugeVec.GetCustomGauge("gtag1Value2", "gtag2Value2").Set(7)
	testGaugeVec.GetCustomGauge("gtag1Value", "gtag2Value").Add(3)
	testGaugeVec.GetCustomGauge("gtag1Value2", "gtag2Value2").Add(2)

	testSummaryVec.GetCustomSummary("stag1Value", "stag2Value").Observe(5)
	testSummaryVec.GetCustomSummary("stag1Value2", "stag2Value2").Observe(4)
	testSummaryVec.GetCustomSummary("stag1Value", "stag2Value").Observe(3)
	testSummaryVec.GetCustomSummary("stag1Value2", "sgtag2Value2").Observe(8)

	// If service has not REST-API endpoints, instrument handler and start server
	if err := http.ListenAndServe(":9876", metrics.InstrumentHTTPHandler(serveMux)); err != nil {
		log.Fatal(err)
	}

	// In case of REST-API endpoints, provide also SwaggerJSON as an argument to get dynamic URL paths handled correctly
	// http.ListenAndServe(":9876", metrics.InstrumentHTTPHandlerUsingSwaggerSpec(serveMux, SwaggerJSON))
}

// Embed metrics endpoint to your existing HTTP server.
func ExampleGetMetricsHandler() {
	// Assign a default endpoint for metrics handler
	serveMux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
}

// Instrument HTTP handler to expose metrics related to endpoints request/response count, size and times.
func ExampleInstrumentHTTPHandler() {
	// Instrument handler and start server
	if err := http.ListenAndServe(":9876", metrics.InstrumentHTTPHandler(serveMux)); err != nil {
		log.Fatal(err)
	}
}

// Instrument handler and start server using Swagger spec JSON.
func ExampleInstrumentHTTPHandlerWithSwaggerSpec() {
	// Instrument handler and start server
	h, err := metrics.InstrumentHTTPHandlerWithSwaggerSpec(serveMux, SwaggerJSON)
	if err != nil {
		panic(err)
	}
	if err := http.ListenAndServe(":9876", h); err != nil {
		log.Fatal(err)
	}
}

// Instrument handler and start server using Swagger spec JSON. Panic in case JSON is not valid.
func ExampleMustInstrumentHTTPHandlerWithSwaggerSpec() {
	// Instrument handler and start server
	h := metrics.MustInstrumentHTTPHandlerWithSwaggerSpec(serveMux, SwaggerJSON)

	if err := http.ListenAndServe(":9876", h); err != nil {
		log.Fatal(err)
	}
}

// Start management server on port 9876 with "/application/prometheus", "/debug/pprof/*" and "/status" endpoints.
func ExampleStartManagementServer_withHealtCheck() {
	// Start management server. Health-check is simply empty response (status code 200)
	managementServer := metrics.StartManagementServer(":9876", func(http.ResponseWriter, *http.Request) {})

	// Add your business logic here...

	// At the shutdown of service, close management server gracefully
	managementServer.Close()
}

// Start management server on port 9876 with "/application/prometheus" and "/debug/pprof/*" endpoints.
func ExampleStartManagementServer_withoutHealtCheck() {
	// Start management server with metrics endpoint only
	managementServer := metrics.StartManagementServer(":9876", nil)

	// Add your business logic here...

	// At the shutdown of service, close management server gracefully
	managementServer.Close()
}

func ExampleMustCrossRegisterMetrics() {
	// Cross-register metrics from the given go-metrics registry as-is
	metrics.MustCrossRegisterMetrics(goMetricsRegistry)
}

func ExampleMustCrossRegisterMetricsWithPrefix() {
	// Cross-register metrics from the given go-metrics registry by using "go_metrics" prefix
	metrics.MustCrossRegisterMetricsWithPrefix("go_metrics", goMetricsRegistry)
}

func ExampleMustCrossRegisterKafkaConsumerMetrics() {
	// Cross-register Kafka consumer metrics from Kafka consumer config
	metrics.MustCrossRegisterKafkaConsumerMetrics(kafkaConsumerConfig.MetricRegistry)
}

func ExampleMustCrossRegisterKafkaProducerMetrics() {
	// Cross-register Kafka producer metrics from Kafka producer config
	metrics.MustCrossRegisterKafkaProducerMetrics(kafkaProducerConfig.MetricRegistry)
}

func ExampleRegisterCounter() {
	// Registers new custom counter metric named com_nokia_neo_metrics_my_service_foo_counter
	testCounter := metrics.RegisterCounter("foo_counter", "my_service", "lorem ipsum...")

	// Increment / add counter value
	testCounter.Inc()
	testCounter.Add(10)
}

func ExampleRegisterCounterVec() {
	// Registers new custom counter vector metric named com_nokia_neo_metrics_my_service_foo_counter_with_tags{key1=<value>,key2=<value>}
	testCounterVec := metrics.RegisterCounterVec("foo_counter_with_tags", "my_service", "lorem ipsum...", "key1", "key2")

	// Increment / add counter value
	testCounterVec.GetCustomCounter("key1Value", "key2Value").Inc()
	testCounterVec.GetCustomCounter("key1Value2", "key2Value2").Inc()
	testCounterVec.GetCustomCounter("key1Value", "key2Value").Add(9)
	testCounterVec.GetCustomCounter("key1Value2", "key2Value2").Add(4)
}

func ExampleRegisterGauge() {
	// Registers new custom gauge metric named com_nokia_neo_metrics_my_service_foo_gauge
	testGauge := metrics.RegisterGauge("foo_gauge", "my_service", "lorem ipsum...")

	// Set / add / subtract gauge value
	testGauge.Set(50.4)
	testGauge.Add(5)
	testGauge.Sub(5)
}

func ExampleRegisterGaugeVec() {
	// Registers new custom gauge vector metric named com_nokia_neo_metrics_my_service_foo_gauge_with_tags{key1=<value>,key2=<value>}
	testGaugeVec := metrics.RegisterGaugeVec("foo_gauge_with_tags", "my_service", "lorem ipsum...", "key1", "key2")

	// Set / add / subtract gauge value
	testGaugeVec.GetCustomGauge("key1Value", "key2Value").Set(6)
	testGaugeVec.GetCustomGauge("key1Value2", "key2Value2").Set(7)
	testGaugeVec.GetCustomGauge("key1Value", "key2Value").Add(3)
	testGaugeVec.GetCustomGauge("key1Value2", "key2Value2").Add(2)
	testGaugeVec.GetCustomGauge("key1Value", "key2Value").Sub(3)
	testGaugeVec.GetCustomGauge("key1Value2", "key2Value2").Sub(2)
}

func ExampleRegisterSummaryWithObjectives() {
	// Registers new custom summary metric named com_nokia_neo_metrics_my_service_foo_summary
	testSummary := metrics.RegisterSummaryWithObjectives("foo_summary", "my_service", "lorem ipsum...", map[float64]float64{0: 0, 0.25: 0.025, 0.5: 0.05, 0.75: 0.025, 0.9: 0.01, 0.99: 0.001, 1: 0})

	// Observe summary value
	testSummary.Observe(55.5)
}

func ExampleRegisterSummary_observe() {
	// Registers new custom summary metric named com_nokia_neo_metrics_my_service_foo_summary
	testSummary := metrics.RegisterSummary("foo_summary", "my_service", "lorem ipsum...")

	// Observe summary value
	testSummary.Observe(66.7)
}

func ExampleRegisterSummary_observeDuration() {
	// Registers new custom summary metric named com_nokia_neo_metrics_my_service_foo_time
	testSummary := metrics.RegisterSummary("foo_time", "my_service", "lorem ipsum...")
	// Get start time of business logic execution
	start := time.Now()

	// ...Business logic to measure...

	// Observe elapsed time
	testSummary.ObserveDuration(start)
}

func ExampleRegisterSummaryVec() {
	// Registers new custom summary vector metric named com_nokia_neo_metrics_my_service_foo_summary_with_tags{key1=<value>,key2=<value>}
	testSummaryVec := metrics.RegisterSummaryVec("foo_summary_with_tags", "my_service", "lorem ipsum...", "key1", "key2")

	// Observe summary value
	testSummaryVec.GetCustomSummary("key1Value", "key2Value").Observe(6.1)
}

func TestResponseContent(t *testing.T) {
	testCounter := metrics.RegisterCounter(counterName, counterSubsystem, counterDesc)
	testGauge := metrics.RegisterGauge(gaugeName, gaugeSubsystem, gaugeDesc)
	testSummary := metrics.RegisterSummary(summaryName, summarySubsystem, summaryDesc)
	testDurationSummary := metrics.RegisterSummary(summaryDurationName, summarySubsystem, summaryDesc)
	testSummaryWithObjectives := metrics.RegisterSummaryWithObjectives(summaryWithObjectivesName, summaryWithObjectivesSubsystem, summaryWithObjectivesDesc, summaryWithObjectivesQuantile)
	testCounterVec := metrics.RegisterCounterVec(counterVecName, counterVecSubsystem,
		counterVecDesc, counterVecLabels[0], counterVecLabels[1])
	testGaugeVec := metrics.RegisterGaugeVec(gaugeVecName, gaugeVecSubsystem,
		gaugeVecDesc, gaugeVecLabels[0], gaugeVecLabels[1])
	testSummaryVec := metrics.RegisterSummaryVec(summaryVecName, summaryVecSubsystem,
		summaryVecDesc, summaryVecLabels[0], summaryVecLabels[1])

	testCounter.Inc()
	testGauge.Set(gaugeValue)
	for i := 0; i < summaryCount; i++ {
		testSummary.Observe(summaryValue)
		testSummaryWithObjectives.Observe(summaryValue)
	}

	testDurationSummary.ObserveDuration(time.Now().Add(time.Duration(-summaryDurationValue) * time.Millisecond))

	testCounterVec.GetCustomCounter(counterVecElem1CodeLabel,
		counterVecElem1MethodLabel).Add(counterVecElem1Value)
	testCounterVec.GetCustomCounter(counterVecElem2CodeLabel,
		counterVecElem2MethodLabel).Add(counterVecElem2Value)
	testGaugeVec.GetCustomGauge(gaugeVecElem1FooLabel,
		gaugeVecElem1BarLabel).Set(gaugeVecElem1Value)
	testGaugeVec.GetCustomGauge(gaugeVecElem2FooLabel,
		gaugeVecElem2BarLabel).Set(gaugeVecElem2Value)

	for i := 0; i < summaryVecElem1Count; i++ {
		testSummaryVec.GetCustomSummary(summaryVecElem1YyyLabel,
			summaryVecElem1KaaLabel).Observe(summaryVecElem1Value)
	}

	for i := 0; i < summaryVecElem2Count; i++ {
		testSummaryVec.GetCustomSummary(summaryVecElem2YyyLabel,
			summaryVecElem2KaaLabel).Observe(summaryVecElem2Value)
	}
	req := httptest.NewRequest("GET", testServerURLPrefix+metrics.DefaultEndPoint, nil)
	w := httptest.NewRecorder()
	metrics.GetMetricsHandler().(http.HandlerFunc)(w, req)
	resp := w.Result()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	body := string(buf)
	for _, testCase := range testCases {
		assert.Contains(t, body, testCase.str, testCase.name)
	}
}

func TestCustomMetricUnregister(t *testing.T) {
	t.Run("CustomCounterUnregister", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetric := metrics.RegisterCounter(metric, "test", "test")
		customMetric.Inc()
		verifyCustomCustomMetricUnregister(t, customMetric, metric)
	})
	t.Run("CustomGaugeUnregister", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetric := metrics.RegisterGauge(metric, "test", "test")
		customMetric.Set(1)
		verifyCustomCustomMetricUnregister(t, customMetric, metric)
	})
	t.Run("CustomSummaryUnregister", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetric := metrics.RegisterSummary(metric, "test", "test")
		customMetric.Observe(1)
		verifyCustomCustomMetricUnregister(t, customMetric, metric)
	})
}

func TestCustomMetricVecReset(t *testing.T) {
	t.Run("CustomCounterVecReset", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetricVec := metrics.RegisterCounterVec(metric, "test", "test", "key1", "key2")
		customMetricVec.GetCustomCounter("val1", "val2").Inc()
		verifyCustomCustomMetricVecReset(t, customMetricVec, metric)
	})
	t.Run("CustomGaugeVecReset", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetricVec := metrics.RegisterGaugeVec(metric, "test", "test", "key1", "key2")
		customMetricVec.GetCustomGauge("val1", "val2").Set(1)
		verifyCustomCustomMetricVecReset(t, customMetricVec, metric)
	})
	t.Run("CustomSummaryVecReset", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetricVec := metrics.RegisterSummaryVec(metric, "test", "test", "key1", "key2")
		customMetricVec.GetCustomSummary("val1", "val2").Observe(1)
		verifyCustomCustomMetricVecReset(t, customMetricVec, metric)
	})
}

func TestCustomMetricVecUnregister(t *testing.T) {
	t.Run("CustomCounterVecUnregister", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetricVec := metrics.RegisterCounterVec(metric, "test", "test", "key1")
		customMetricVec.GetCustomCounter("val1").Inc()
		verifyCustomCustomMetricUnregister(t, customMetricVec, metric)
	})
	t.Run("CustomGaugeVecUnregister", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetricVec := metrics.RegisterGaugeVec(metric, "test", "test", "key1")
		customMetricVec.GetCustomGauge("val1").Set(1)
		verifyCustomCustomMetricUnregister(t, customMetricVec, metric)
	})
	t.Run("CustomSummaryVecUnregister", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		customMetricVec := metrics.RegisterSummaryVec(metric, "test", "test", "key1")
		customMetricVec.GetCustomSummary("val1").Observe(1)
		verifyCustomCustomMetricUnregister(t, customMetricVec, metric)
	})
}

func TestCustomMetricVecDeleteSerie(t *testing.T) {
	t.Run("CustomCounterVecDeleteSerie", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		label1value := uuid.New().String()
		customMetricVec := metrics.RegisterCounterVec(metric, "test", "test", "key1", "key2")
		customMetricVec.GetCustomCounter(label1value, "val2").Inc()
		customMetricVec.GetCustomCounter("val3", "val4").Inc()
		verifyCustomCustomMetricVecDeleteSerie(t, customMetricVec, metric, label1value, "val2")
	})
	t.Run("CustomGaugeVecDeleteSerie", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		label1value := uuid.New().String()
		customMetricVec := metrics.RegisterGaugeVec(metric, "test", "test", "key1", "key2")
		customMetricVec.GetCustomGauge(label1value, "val2").Set(1)
		customMetricVec.GetCustomGauge("val3", "val4").Set(1)
		verifyCustomCustomMetricVecDeleteSerie(t, customMetricVec, metric, label1value, "val2")
	})
	t.Run("CustomSummaryVecDeleteSerie", func(t *testing.T) {
		metric := strings.ReplaceAll(uuid.New().String(), "-", "_")
		label1value := uuid.New().String()
		customMetricVec := metrics.RegisterSummaryVec(metric, "test", "test", "key1", "key2")
		customMetricVec.GetCustomSummary(label1value, "val2").Observe(1)
		customMetricVec.GetCustomSummary("val3", "val4").Observe(1)
		verifyCustomCustomMetricVecDeleteSerie(t, customMetricVec, metric, label1value, "val2")
	})
}

func TestMetricsHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, nil))

	buf, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, len(buf) > 0, "Buffer has data")
}

func TestDefaultCollectorMetrics(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, nil))

	buf, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	for _, testCase := range neoCollectorTestCases {
		assert.Contains(t, body, testCase.str, testCase.name)
	}
}

func TestPprof(t *testing.T) {
	t.Run("WithServeMux", func(t *testing.T) {
		checkPprofEndpoints(t, http.NewServeMux())
	})

	t.Run("WithChiMux", func(t *testing.T) {
		checkPprofEndpoints(t, chi.NewMux())
	})
}

func checkPprofEndpoints(t *testing.T, router testRouter) {
	metrics.InstrumentWithPprof(router)
	checkPprofEndpoint(t, router, "/debug/pprof/allocs")
	checkPprofEndpoint(t, router, "/debug/pprof/block")
	checkPprofEndpoint(t, router, "/debug/pprof/cmdline")
	checkPprofEndpoint(t, router, "/debug/pprof/goroutine")
	checkPprofEndpoint(t, router, "/debug/pprof/heap")
	checkPprofEndpoint(t, router, "/debug/pprof/mutex")
	checkPprofEndpoint(t, router, "/debug/pprof/profile?seconds=1")
	checkPprofEndpoint(t, router, "/debug/pprof/threadcreate")
	checkPprofEndpoint(t, router, "/debug/pprof/trace?seconds=1")
}

func checkPprofEndpoint(t *testing.T, handler http.Handler, endpoint string) {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, endpoint, nil))

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode, fmt.Sprintf("Checking status code of pprof endpoint %s", endpoint))

	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestInstrumentHttpHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(request200URI, func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	})

	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	server := metrics.InstrumentHTTPHandler(mux)

	for i := 0; i < request200Count; i++ {
		server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+request200URI, nil))
	}

	for i := 0; i < request404Count; i++ {
		server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+request404URI, nil))
	}

	for i := 0; i < request200Count*2; i++ {
		server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, nil))
	}

	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, nil))

	buf, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	for _, testCase := range httpTestCases {
		assert.Contains(t, body, testCase.str, testCase.name)
	}
}

func TestInstrumentHttpHandlerWithRules(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(request200URI, func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	})

	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())

	rules := []metrics.InstrumentRule{
		{
			Condition: regexp.MustCompile(`\/test$`),
			URIPath:   request200AfterRuleURI,
		},
		{
			Condition: regexp.MustCompile(`\/application\/prometheus$`),
			URIPath:   endpointNameAfterRule,
		},
	}

	server := metrics.InstrumentHTTPHandlerWithRules(mux, rules)

	for i := 0; i < request200Count; i++ {
		server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+request200URI, nil))
	}

	for i := 0; i < request200Count*2; i++ {
		server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, nil))
	}

	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, nil))

	buf, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	for _, testCase := range httpTestCasesForRules {
		assert.Contains(t, body, testCase.str, testCase.name)
	}
}

func TestInstrumentHttpHandlerUsingSwaggerJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/credentials/v1/123/fooType", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	})
	mux.HandleFunc("/credentials/v1/123/fooType/not_mapped", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	})
	mux.HandleFunc("/credentials/v1/alarm/PLMN-PLMN/NEO-1/ELEM-123/12345789", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		require.NoError(t, err)
	})
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	server := metrics.MustInstrumentHTTPHandlerWithSwaggerSpec(mux, SwaggerJSON)

	req := httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+"/credentials/v1/123/fooType", nil)
	server.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+"/credentials/v1/123/fooType/not_mapped", nil)
	server.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+"/credentials/v1/alarm/"+url.PathEscape("PLMN-PLMN/NEO-1/ELEM-123")+"/12345789", nil)
	server.ServeHTTP(httptest.NewRecorder(), req)

	req = httptest.NewRequest("GET", testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	buf, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	for _, testCase := range httpRuleTestCases {
		assert.Contains(t, body, testCase.str, testCase.name)
	}
}

func TestMustCrossRegisterConcurrency(t *testing.T) {
	routines := 500
	wg := &sync.WaitGroup{}
	wg.Add(routines)
	rands := make([]string, routines)
	for i := 0; i < routines; i++ {
		rands[i] = generateRandString(t)
	}
	for i := 0; i < routines; i++ {
		go func(wg *sync.WaitGroup, i int) {
			defer wg.Done()
			goRegistry := gometrics.NewRegistry()
			counter := gometrics.NewCounter()
			require.NoError(t, goRegistry.Register("counter", counter))
			if err := metrics.CrossRegisterMetricsWithPrefix(rands[i], goRegistry); err != nil {
				t.Fatal(err)
			}
		}(wg, i)
	}
	wg.Wait()
	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func(wg *sync.WaitGroup, i int) {
			defer wg.Done()
			metrics.UnregisterMetricsWithPrefix(rands[i])
		}(wg, i)
	}
	wg.Wait()
}

func generateRandString(t *testing.T) string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("m-%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func TestMustCrossRegisterUnregisterMetric(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	// This is how to access go-metrics registry for sarama Kafka
	// goRegistry := sarama.NewConfig().MetricRegistry
	goRegistry := gometrics.NewRegistry()
	counter := gometrics.NewCounter()
	require.NoError(t, goRegistry.Register(goMetricsMetricName, counter))
	counter.Inc(goMetricsMetricValue)
	metrics.MustCrossRegisterMetrics(goRegistry)

	// Create metric with the same name and avoid conflict by defining a prefix
	goRegistry2 := gometrics.NewRegistry()
	counter2 := gometrics.NewCounter()
	require.NoError(t, goRegistry2.Register(goMetricsMetricName, counter2))
	counter2.Inc(goMetricsMetricValue + 1)
	metrics.MustCrossRegisterMetricsWithPrefix(goMetricsMetricRegistry2Prefix, goRegistry2)

	// Cross-register registry as Kafka consumer registry
	goRegistryKafkaConsumer := gometrics.NewRegistry()
	consumerCounter := gometrics.NewCounter()
	require.NoError(t, goRegistryKafkaConsumer.Register(goMetricsMetricName, consumerCounter))
	consumerCounter.Inc(goMetricsMetricValue + 2)
	metrics.MustCrossRegisterKafkaConsumerMetrics(goRegistryKafkaConsumer)

	// Cross-register registry as Kafka producer registry
	goRegistryKafkaProducer := gometrics.NewRegistry()
	producerCounter := gometrics.NewCounter()
	require.NoError(t, goRegistryKafkaProducer.Register(goMetricsMetricName, producerCounter))
	producerCounter.Inc(goMetricsMetricValue + 3)
	metrics.MustCrossRegisterKafkaProducerMetrics(goRegistryKafkaProducer)

	// Wait until cross-registering flush interval is exceeded
	time.Sleep(1100 * time.Millisecond)

	checkResponse(getMetricsResponse(testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, mux, t), true, true, true, true, false, false, t)

	metrics.UnregisterKafkaConsumerMetrics()
	checkResponse(getMetricsResponse(testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, mux, t), true, true, false, true, false, false, t)

	metrics.UnregisterKafkaProducerMetrics()
	metrics.UnregisterMetricsWithPrefix(goMetricsMetricRegistry2Prefix)
	checkResponse(getMetricsResponse(testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, mux, t), true, false, false, false, false, false, t)

	// Cross-register registry as Kafka consumer registry
	goRegistryKafkaPrefixConsumer := gometrics.NewRegistry()
	consumerPrefixCounter := gometrics.NewCounter()
	require.NoError(t, goRegistryKafkaPrefixConsumer.Register(goMetricsMetricName, consumerPrefixCounter))
	consumerPrefixCounter.Inc(goMetricsMetricValue + 5)
	metrics.MustCrossRegisterKafkaConsumerMetricsPrefix(goRegistryKafkaPrefixConsumer, "consumerprefix")

	// Cross-register registry as Kafka producer registry
	goRegistryKafkaPrefixProducer := gometrics.NewRegistry()
	producerPrefixCounter := gometrics.NewCounter()
	require.NoError(t, goRegistryKafkaPrefixProducer.Register(goMetricsMetricName, producerPrefixCounter))
	producerPrefixCounter.Inc(goMetricsMetricValue + 6)
	metrics.MustCrossRegisterKafkaProducerMetricsPrefix(goRegistryKafkaPrefixProducer, "producerprefix")

	// Wait until cross-registering flush interval is exceeded
	time.Sleep(1100 * time.Millisecond)
	checkResponse(getMetricsResponse(testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, mux, t), true, false, false, false, true, true, t)

	metrics.UnregisterKafkaProducerMetricsPrefix("producerprefix")
	checkResponse(getMetricsResponse(testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, mux, t), true, false, false, false, true, false, t)

	metrics.UnregisterKafkaConsumerMetricsPrefix("consumerprefix")
	checkResponse(getMetricsResponse(testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, mux, t), true, false, false, false, false, false, t)

	metrics.UnregisterMetrics()
	checkResponse(getMetricsResponse(testServerURLPrefix+testServerAddr+metrics.DefaultEndPoint, mux, t), false, false, false, false, false, false, t)
}

func getMetricsResponse(uri string, handle http.Handler, t *testing.T) string {
	req := httptest.NewRequest("GET", uri, nil)
	w := httptest.NewRecorder()
	handle.ServeHTTP(w, req)
	buf, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(buf)
}

func checkResponse(response string, noPrefixContains bool, prefixContains bool,
	kafkaConsumerContains bool, kafkaProducerContains bool, kafkaConsumerPrefixContains bool, kafkaProducerPrefixContains bool, t *testing.T) {
	if noPrefixContains {
		assert.Contains(t, response, fmt.Sprintf("%s %d", goMetricsMetricName,
			goMetricsMetricValue), fmt.Sprintf("go-metric %s is cross-registered",
			goMetricsMetricName))
	} else {
		assert.NotContains(t, response, fmt.Sprintf("%s %d", goMetricsMetricName,
			goMetricsMetricValue), fmt.Sprintf("go-metric %s is not cross-registered to prometheus registry",
			goMetricsMetricName))
	}
	if prefixContains {
		assert.Contains(t, response, fmt.Sprintf("%s_%s %d",
			goMetricsMetricRegistry2Prefix, goMetricsMetricName, goMetricsMetricValue+1),
			fmt.Sprintf("go-metric %s_%s is cross-registered",
				goMetricsMetricRegistry2Prefix, goMetricsMetricName))
	} else {
		assert.NotContains(t, response, fmt.Sprintf("%s_%s %d",
			goMetricsMetricRegistry2Prefix, goMetricsMetricName, goMetricsMetricValue+1),
			fmt.Sprintf("go-metric %s_%s is not cross-registered to prometheus registry",
				goMetricsMetricRegistry2Prefix, goMetricsMetricName))
	}
	if kafkaConsumerContains {
		assert.Contains(t, response, fmt.Sprintf("%s_%s %d",
			metrics.KafkaConsumerPrefix, goMetricsMetricName, goMetricsMetricValue+2),
			fmt.Sprintf("go-metric %s_%s is cross-registered",
				metrics.KafkaConsumerPrefix, goMetricsMetricName))
	} else {
		assert.NotContains(t, response, fmt.Sprintf("%s_%s %d",
			metrics.KafkaConsumerPrefix, goMetricsMetricName, goMetricsMetricValue+2),
			fmt.Sprintf("go-metric %s_%s is not cross-registered to prometheus registry",
				metrics.KafkaConsumerPrefix, goMetricsMetricName))
	}
	if kafkaProducerContains {
		assert.Contains(t, response, fmt.Sprintf("%s_%s %d",
			metrics.KafkaProducerPrefix, goMetricsMetricName, goMetricsMetricValue+3),
			fmt.Sprintf("go-metric %s_%s is cross-registered",
				metrics.KafkaProducerPrefix, goMetricsMetricName))
	} else {
		assert.NotContains(t, response, fmt.Sprintf("%s_%s %d",
			metrics.KafkaProducerPrefix, goMetricsMetricName, goMetricsMetricValue+3),
			fmt.Sprintf("go-metric %s_%s is not cross-registered to prometheus registry",
				metrics.KafkaProducerPrefix, goMetricsMetricName))
	}
	if kafkaProducerPrefixContains {
		assert.Contains(t, response, fmt.Sprintf("%s_%s %d",
			metrics.KafkaProducerPrefix+"_producerprefix", goMetricsMetricName, goMetricsMetricValue+6),
			fmt.Sprintf("go-metric %s_%s is cross-registered",
				metrics.KafkaProducerPrefix, goMetricsMetricName))
	} else {
		assert.NotContains(t, response, fmt.Sprintf("%s_%s %d",
			metrics.KafkaProducerPrefix+"_producerprefix", goMetricsMetricName, goMetricsMetricValue+6),
			fmt.Sprintf("go-metric %s_%s is not cross-registered to prometheus registry",
				metrics.KafkaProducerPrefix, goMetricsMetricName))
	}
	if kafkaConsumerPrefixContains {
		assert.Contains(t, response, fmt.Sprintf("%s_%s %d",
			metrics.KafkaConsumerPrefix+"_consumerprefix", goMetricsMetricName, goMetricsMetricValue+5),
			fmt.Sprintf("go-metric %s_%s is cross-registered",
				metrics.KafkaConsumerPrefix, goMetricsMetricName))
	} else {
		assert.NotContains(t, response, fmt.Sprintf("%s_%s %d",
			metrics.KafkaConsumerPrefix+"_consumerprefix", goMetricsMetricName, goMetricsMetricValue+5),
			fmt.Sprintf("go-metric %s_%s is not cross-registered to prometheus registry",
				metrics.KafkaConsumerPrefix, goMetricsMetricName))
	}
}

func TestInstrumentingMultipleServers(t *testing.T) {
	// Start serving metrics data
	mux := http.NewServeMux()
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	server := &http.Server{Addr: fmt.Sprintf(":%d", 29122), Handler: metrics.InstrumentHTTPHandler(mux)}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Fatal(err)
		}
	}()

	time.Sleep(time.Millisecond)
	defer server.Close()

	// Start, query and close servers
	serverCount := 10
	wg := &sync.WaitGroup{}
	wg.Add(serverCount)
	go func() {
		// Start servers
		for i := 0; i < serverCount; i++ {
			mux := http.NewServeMux()
			mux.HandleFunc(fmt.Sprintf("/status-%d", i), func(w http.ResponseWriter, r *http.Request) {})
			server := &http.Server{Addr: fmt.Sprintf(":%d", 29123+i), Handler: metrics.InstrumentHTTPHandler(mux)}
			defer server.Close()

			go func(server *http.Server) {
				defer wg.Done()
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					t.Fatal(err)
				}
			}(server)
			time.Sleep(time.Millisecond)
		}

		// Call all server instanses
		for i := 0; i < serverCount; i++ {
			if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/status-%d", 29123+i, i)); err != nil {
				t.Fatal(err)
			}
		}
	}()
	wg.Wait()

	// Check metrics
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:29122%s", metrics.DefaultEndPoint))
	if err != nil {
		t.Fatal(err)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	for i := 0; i < serverCount; i++ {
		endPoint := fmt.Sprintf("/status-%d", i)
		assert.Contains(t, body, endPoint, fmt.Sprintf("Metrics contains data for %s route", endPoint))
	}
}

func TestManagementServer(t *testing.T) {
	managementServer := metrics.StartManagementServer(testServerAddr,
		func(http.ResponseWriter, *http.Request) {})
	defer managementServer.Close()
	resp, err := http.Get(testServerURLPrefix + testServerAddr + statusEndPoint)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, resp.StatusCode, "Status code 200 is returned from "+
		statusEndPoint+" endpoint")

	resp, err = http.Get(testServerURLPrefix + testServerAddr + "/debug/pprof/trace?seconds=1")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, resp.StatusCode, "Status code 200 is returned from /debug/pprof/trace endpoint")

	// Extra invoke of metrics endpoint is needed to get it reported to avoid
	// chicken-egg problem
	if resp, err = http.Get(testServerURLPrefix + testServerAddr +
		metrics.DefaultEndPoint); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, resp.StatusCode, "Status code 200 is returned from "+
		metrics.DefaultEndPoint+" endpoint")
	if resp, err = http.Get(testServerURLPrefix + testServerAddr +
		metrics.DefaultEndPoint); err != nil {
		t.Fatal(err)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	for _, testCase := range managementServerContentTestCases {
		assert.Contains(t, body, testCase.str, testCase.name)
	}
}

func TestManagementServerNoHealthCheck(t *testing.T) {
	managementServer := metrics.StartManagementServer(testServerAddr, nil)
	defer managementServer.Close()

	resp, err := http.Get(testServerURLPrefix + testServerAddr + statusEndPoint)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 404, resp.StatusCode, "Status code 404 is returned from "+
		statusEndPoint+" endpoint as it is not started")
	// Extra invoke of metrics endpoint is needed to get it reported to avoid
	// chicken-egg problem
	if resp, err = http.Get(testServerURLPrefix + testServerAddr +
		metrics.DefaultEndPoint); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 200, resp.StatusCode, "Status code 200 is returned from "+
		metrics.DefaultEndPoint+" endpoint")
	if resp, err = http.Get(testServerURLPrefix + testServerAddr +
		metrics.DefaultEndPoint); err != nil {
		t.Fatal(err)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(buf)
	for _, testCase := range managementServerContentTestCases {
		assert.Contains(t, body, testCase.str, testCase.name)
	}
}

func verifyCustomCustomMetricVecReset(t *testing.T, vec testCustomMetricVec, metric string) {
	url := testServerURLPrefix + metrics.DefaultEndPoint
	assert.Contains(t, getMetricResponse(t, url), metric, "Metric exists")
	vec.Reset()
	assert.NotContains(t, getMetricResponse(t, url), metric, "Metric is reseted")
}

func verifyCustomCustomMetricUnregister(t *testing.T, vec testCustomMetric, metric string) {
	url := testServerURLPrefix + metrics.DefaultEndPoint
	assert.Contains(t, getMetricResponse(t, url), metric, "Metric exists")
	vec.Unregister()
	assert.NotContains(t, getMetricResponse(t, url), metric, "Metric is unregistered")
}

func verifyCustomCustomMetricVecDeleteSerie(t *testing.T, vec testCustomMetricVec, metric string, labelValues ...string) {
	url := testServerURLPrefix + metrics.DefaultEndPoint
	resp := getMetricResponse(t, url)
	assert.Contains(t, resp, metric, "Metric exists")
	assert.Contains(t, resp, labelValues[0], "Serie exists")
	vec.DeleteSerie(labelValues...)
	resp = getMetricResponse(t, url)
	assert.Contains(t, resp, metric, "Metric exists")
	assert.NotContains(t, resp, labelValues[0], "Metric serie is deleted")
}

func getMetricResponse(t *testing.T, url string) string {
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	metrics.GetMetricsHandler().(http.HandlerFunc)(w, req)
	resp := w.Result()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return string(buf)
}
