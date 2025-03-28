package rules

import (
	"regexp"
	"strings"

	"github.com/marcotuna/adaptive-metrics/internal/models"
)

// Matcher is responsible for determining which rules apply to metrics
type Matcher struct {
	engine *Engine
	regexCache map[string]*regexp.Regexp
}

// NewMatcher creates a new rule matcher
func NewMatcher(engine *Engine) *Matcher {
	return &Matcher{
		engine: engine,
		regexCache: make(map[string]*regexp.Regexp),
	}
}

// MatchingRules returns all rules that match a given metric sample
func (m *Matcher) MatchingRules(sample *models.MetricSample) []*models.Rule {
	m.engine.ruleMu.RLock()
	defer m.engine.ruleMu.RUnlock()
	
	var matchingRules []*models.Rule
	
	for _, rule := range m.engine.rules {
		if !rule.Enabled {
			continue
		}
		
		if m.matchesRule(sample, rule) {
			matchingRules = append(matchingRules, rule)
		}
	}
	
	return matchingRules
}

// matchesRule checks if a metric sample matches a specific rule
func (m *Matcher) matchesRule(sample *models.MetricSample, rule *models.Rule) bool {
	// Check metric name
	nameMatched := false
	for _, metricName := range rule.Matcher.MetricNames {
		if metricName == sample.Name || metricName == "*" {
			nameMatched = true
			break
		}
		
		// Check for glob patterns in metric name
		if strings.Contains(metricName, "*") {
			pattern := "^" + strings.ReplaceAll(metricName, "*", ".*") + "$"
			re, exists := m.regexCache[pattern]
			if !exists {
				re = regexp.MustCompile(pattern)
				m.regexCache[pattern] = re
			}
			
			if re.MatchString(sample.Name) {
				nameMatched = true
				break
			}
		}
	}
	
	if !nameMatched {
		return false
	}
	
	// Check label matchers
	for labelKey, labelValue := range rule.Matcher.Labels {
		sampleValue, exists := sample.Labels[labelKey]
		if !exists || sampleValue != labelValue {
			return false
		}
	}
	
	// Check regex label matchers
	for labelKey, regexStr := range rule.Matcher.LabelRegex {
		sampleValue, exists := sample.Labels[labelKey]
		if !exists {
			return false
		}
		
		cacheKey := labelKey + ":" + regexStr
		re, exists := m.regexCache[cacheKey]
		if !exists {
			re = regexp.MustCompile(regexStr)
			m.regexCache[cacheKey] = re
		}
		
		if !re.MatchString(sampleValue) {
			return false
		}
	}
	
	return true
}

// GetRulesByMetricName returns all rules that might apply to metrics with the given name
func (m *Matcher) GetRulesByMetricName(metricName string) []*models.Rule {
	m.engine.ruleMu.RLock()
	defer m.engine.ruleMu.RUnlock()
	
	var matchingRules []*models.Rule
	
	for _, rule := range m.engine.rules {
		if !rule.Enabled {
			continue
		}
		
		for _, ruleMetricName := range rule.Matcher.MetricNames {
			if ruleMetricName == metricName || ruleMetricName == "*" {
				matchingRules = append(matchingRules, rule)
				break
			}
			
			// Check for glob patterns in metric name
			if strings.Contains(ruleMetricName, "*") {
				pattern := "^" + strings.ReplaceAll(ruleMetricName, "*", ".*") + "$"
				re, exists := m.regexCache[pattern]
				if !exists {
					re = regexp.MustCompile(pattern)
					m.regexCache[pattern] = re
				}
				
				if re.MatchString(metricName) {
					matchingRules = append(matchingRules, rule)
					break
				}
			}
		}
	}
	
	return matchingRules
}