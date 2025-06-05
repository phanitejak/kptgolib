package openapi

import (
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/phanitejak/kptgolib/metrics"
)

var rulePattern = regexp.MustCompile(`(?s)({[^}]*})`)

// ToInstrumentRules converts OpenAPI path templates into regex rules for instrumentation.
// Example: "/users/{id}" → "^/users/[^/]+$"
func ToInstrumentRules(swagger *openapi3.T) []metrics.InstrumentRule {
	var rules []metrics.InstrumentRule
	pathsMap := swagger.Paths.Map()
	for uri := range pathsMap {
		if strings.Contains(uri, "{") {
			pattern := "^" + rulePattern.ReplaceAllString(uri, `[^/]+`) + "$"
			rules = append(rules, metrics.InstrumentRule{
				Condition: regexp.MustCompile(pattern),
				URIPath:   uri,
			})
		}
	}
	return rules
}

// ToInstrumentRulesV2 is an alias to ToInstrumentRules for forward compatibility.
func ToInstrumentRulesV2(swagger *openapi3.T) []metrics.InstrumentRule {
	return ToInstrumentRules(swagger)
}
