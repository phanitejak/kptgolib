package openapi

import (
	"regexp"
	"sort"
	"testing"

	"github.com/phanitejak/kptgolib/metrics"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestToInstrumentRules(t *testing.T) {
	tests := []struct {
		name    string
		swagger *openapi3.T
		want    []metrics.InstrumentRule
	}{
		{
			name: "Without path params",
			swagger: &openapi3.T{
				Paths: openapi3.NewPaths(openapi3.WithPath("/v1/somepath/", nil)),
			},
			want: nil,
		},
		{
			name: "With single path param",
			swagger: &openapi3.T{
				Paths: openapi3.NewPaths(openapi3.WithPath("/v1/somepath/{parameter}/details", nil)),
			},
			want: []metrics.InstrumentRule{
				{
					Condition: regexp.MustCompile("^/v1/somepath/[^/]+/details$"),
					URIPath:   "/v1/somepath/{parameter}/details",
				},
			},
		},
		{
			name: "With multiple paths",
			swagger: &openapi3.T{
				Paths: openapi3.NewPaths(
					openapi3.WithPath("/v1/somepath/{parameter}/details", nil),
					openapi3.WithPath("/v1/otherpath/{parameter}/details", nil),
				),
			},
			want: []metrics.InstrumentRule{
				{
					Condition: regexp.MustCompile("^/v1/somepath/[^/]+/details$"),
					URIPath:   "/v1/somepath/{parameter}/details",
				},
				{
					Condition: regexp.MustCompile("^/v1/otherpath/[^/]+/details$"),
					URIPath:   "/v1/otherpath/{parameter}/details",
				},
			},
		},
		{
			name: "With multiple path params",
			swagger: &openapi3.T{
				Paths: openapi3.NewPaths(openapi3.WithPath("/v1/somepath/{parameter}/details/{subparam}", nil)),
			},
			want: []metrics.InstrumentRule{
				{
					Condition: regexp.MustCompile("^/v1/somepath/[^/]+/details/[^/]+$"),
					URIPath:   "/v1/somepath/{parameter}/details/{subparam}",
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rules := ToInstrumentRules(tt.swagger)

			sort.Slice(rules, func(i, j int) bool { return rules[i].URIPath < rules[j].URIPath })
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].URIPath < tt.want[j].URIPath })

			require.Equal(t, tt.want, rules)
		})
	}
}
