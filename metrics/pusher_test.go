package metrics_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"gopkg/metrics"
)

var (
	lastMethod string
	lastBody   []byte
	lastPath   string
)

func TestPusher(t *testing.T) {
	// Fake a Pushgateway that always responds with 202.
	pgwServer := startTestPushGateway(t)
	defer pgwServer.Close()

	g := metrics.RegisterGauge("pushertest_gauge", "neo", "desc")
	g.Set(1.0)
	gv := metrics.RegisterGaugeVec("pushertest_gaugeVec", "neo", "desc", "label1", "label2")
	gv.GetCustomGauge("foo1", "bar1").Set(2)
	gv.GetCustomGauge("foo2", "bar2").Set(5)

	c := metrics.RegisterCounter("pushertest_counter", "neo", "desc")
	c.Add(1.0)
	cv := metrics.RegisterCounterVec("pushertest_counterVec", "neo", "desc", "label1", "label2")
	cv.GetCustomCounter("foo1", "bar1").Add(2)
	cv.GetCustomCounter("foo2", "bar2").Add(5)

	s := metrics.RegisterSummary("pushertest_summary", "neo", "desc")
	s.Observe(1.0)
	sv := metrics.RegisterSummaryVec("pushertest_summaryVec", "neo", "desc", "label1", "label2")
	sv.GetCustomSummary("foo1", "bar1").Observe(2)
	sv.GetCustomSummary("foo2", "bar2").Observe(5)
	p := metrics.NewPusher(metrics.PushConfig{EndPoint: pgwServer.URL, JobName: "testjob"})
	p.Collector(g).Collector(gv).Collector(c).Collector(cv).Collector(s).Collector(sv)

	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	enc := expfmt.NewEncoder(buf, expfmt.NewFormat(expfmt.TypeProtoDelim))

	for _, mf := range mfs {
		if strings.HasPrefix(mf.GetName(), "com_nokia_neo_metrics_neo_pushertest_") {
			if err := enc.Encode(mf); err != nil {
				t.Fatal(err)
			}
		}
	}
	wantBody := buf.Bytes()
	if err := p.Push(); err != nil {
		t.Fatal(err)
	}
	assert.True(t, bytes.Equal(lastBody, wantBody), "Test all metrics are pushed properly")
	assert.Equal(t, lastMethod, "PUT", "Test method is PUT")
	assert.Equal(t, lastPath, "/metrics/job/testjob", "Test metrics path")
}

// Example how to use pusher.
func ExamplePusher() {
	// Define your metrics -
	firstMetric := metrics.RegisterSummaryVec("summary_metric1", "MyGreatJobName", "Lorem Ipsum Desc", "label1", "label2")
	secondMetric := metrics.RegisterSummary("summary_metric2", "MyGreatJobName", "Lorem Ipsum Desc")

	// Create pusher. In case EndPoint is not given in PushConfig, it has to be set in env by this way:
	// METRICS_PUSH_ENDPOINT=http://cpro-pushgateway.default:9091
	pusher := metrics.NewPusher(metrics.PushConfig{JobName: "MyGreatJobName"})

	// Register your metrics to be pushed
	pusher.Collector(firstMetric).Collector(secondMetric)

	// Collect your observations
	start := time.Now()
	// Do something
	firstMetric.GetCustomSummary("label1_value", "label2_value").ObserveDuration(start)
	secondMetric.Observe(1234567890)

	// Push metrics to gateway
	if err := pusher.Push(); err != nil {
		panic(err)
	}
}

func startTestPushGateway(t *testing.T) *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lastMethod = r.Method
			var err error
			lastBody, err = ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			lastPath = r.URL.EscapedPath()
			w.Header().Set("Content-Type", `text/plain; charset=utf-8`)
			w.WriteHeader(http.StatusAccepted)
		}),
	)
}
