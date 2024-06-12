package openapi

import (
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/phanitejak/kptgolib/metrics"
)

var rulePattern = regexp.MustCompile(`(?s)({[^}]*})`)

// ToInstrumentRules ...
func ToInstrumentRules(swagger *openapi3.T) []metrics.InstrumentRule {
	var rules []metrics.InstrumentRule
	for uri := range swagger.Paths.Map() {
		if strings.Contains(uri, "{") {
			rules = append(rules, metrics.InstrumentRule{Condition: regexp.MustCompile(rulePattern.ReplaceAllString("^"+uri+"$", `[^/]+`)), URIPath: uri})
		}
	}
	return rules
}

// ToInstrumentRulesV2 uses never openapi implementation.
func ToInstrumentRulesV2(swagger *openapi3.T) []metrics.InstrumentRule {
	return ToInstrumentRules(swagger)
}
