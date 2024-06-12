package metrics

import (
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// PUSH_ENDPOINT is push endpoint's environment variable name.
const PUSH_ENDPOINT = "METRICS_PUSH_ENDPOINT"

// Pusher manages a push to the push gateway. Use NewPusher to create one, configure it
// with its methods, and finally use the Add or Push method to push.
type Pusher struct {
	pusher *push.Pusher
}

// PushConfig configuration.
type PushConfig struct {

	// Push job name, mandatory
	JobName string

	// Push endpoint, optional in case push endpoint's environment variable is set
	EndPoint string
}

// NewPusher creates a new Pusher to push to the provided URL with the provided job
// name. You can use just host:port or ip:port as url, in which case “http://”
// is added automatically. Alternatively, include the schema in the
// URL.
func NewPusher(config PushConfig) *Pusher {
	if config.EndPoint != "" {
		return &Pusher{push.New(config.EndPoint, config.JobName)}
	}
	if os.Getenv(PUSH_ENDPOINT) == "" {
		panic(fmt.Sprintf("EndPoint is not given and env %s is not set or it's empty!", PUSH_ENDPOINT))
	}
	return &Pusher{push.New(os.Getenv(PUSH_ENDPOINT), config.JobName)}
}

// Collector adds a Collector to the Pusher, from which metrics will be
// collected to push them to the push gateway. The collected metrics must not
// contain a job label of their own.
//
// For convenience, this method returns a pointer to the Pusher itself.
func (p *Pusher) Collector(c CustomMetric) *Pusher {
	p.pusher.Collector(c.GetCollector())
	return p
}

// Push collects/gathers all metrics from all Collectors added to
// this Pusher. Then, it pushes them to the push gateway configured while
// creating this Pusher, using the configured job name and any added grouping
// labels as grouping key. All previously pushed metrics with the same job and
// other grouping labels will be replaced with the metrics pushed by this
// call. (It uses HTTP method “PUT” to push to the push gateway.)
//
// Push returns the first error encountered by any method call (including this
// one) in the lifetime of the Pusher.
func (p *Pusher) Push() error {
	return p.pusher.Push()
}

// Add works like push, but only previously pushed metrics with the same name
// (and the same job and other grouping labels) will be replaced. (It uses HTTP
// method “POST” to push to the push gateway.)
func (p *Pusher) Add() error {
	return p.pusher.Add()
}

// CollectAll adds a default gatherer to the Pusher, from which metrics will be gathered
// to push them to the push gateway. The gathered metrics must not contain a job
// label of their own.
//
// For convenience, this method returns a pointer to the Pusher itself.
func (p *Pusher) CollectAll() *Pusher {
	p.pusher.Gatherer(prometheus.DefaultGatherer)
	return p
}

// Grouping adds a label pair to the grouping key of the Pusher, replacing any
// previously added label pair with the same label name. Note that setting any
// labels in the grouping key that are already contained in the metrics to push
// will lead to an error.
//
// For convenience, this method returns a pointer to the Pusher itself.
func (p *Pusher) Grouping(name, value string) *Pusher {
	p.pusher.Grouping(name, value)
	return p
}

// Client sets a custom HTTP client for the Pusher. For convenience, this method
// returns a pointer to the Pusher itself.
func (p *Pusher) Client(c *http.Client) *Pusher {
	p.pusher.Client(c)
	return p
}

// BasicAuth configures the Pusher to use HTTP Basic Authentication with the
// provided username and password. For convenience, this method returns a
// pointer to the Pusher itself.
func (p *Pusher) BasicAuth(username, password string) *Pusher {
	p.pusher.BasicAuth(username, password)
	return p
}
