package rules

import (
	"reflect"
	"testing"

	"github.com/marcotuna/adaptive-metrics/internal/models"
)

func TestMatcher_matchesRule(t *testing.T) {
	// Create a simple engine for testing
	engine := &Engine{
		rules: make(map[string]*models.Rule),
	}
	
	matcher := NewMatcher(engine)

	tests := []struct {
		name   string
		sample *models.MetricSample
		rule   *models.Rule
		want   bool
	}{
		{
			name: "exact metric name match",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
					"path":   "/api/v1/users",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
			},
			want: true,
		},
		{
			name: "wildcard metric name match",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"*"},
				},
			},
			want: true,
		},
		{
			name: "glob pattern metric name match",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"http_*"},
				},
			},
			want: true,
		},
		{
			name: "no metric name match",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"node_cpu_seconds_total"},
				},
			},
			want: false,
		},
		{
			name: "label match",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
					"path":   "/api/v1/users",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"http_requests_total"},
					Labels: map[string]string{
						"method": "GET",
					},
				},
			},
			want: true,
		},
		{
			name: "label mismatch",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
					"path":   "/api/v1/users",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"http_requests_total"},
					Labels: map[string]string{
						"method": "POST",
					},
				},
			},
			want: false,
		},
		{
			name: "label missing",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"http_requests_total"},
					Labels: map[string]string{
						"service": "api",
					},
				},
			},
			want: false,
		},
		{
			name: "regex label match",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"path": "/api/v1/users",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"http_requests_total"},
					LabelRegex: map[string]string{
						"path": "^/api/.*",
					},
				},
			},
			want: true,
		},
		{
			name: "regex label mismatch",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"path": "/ui/dashboard",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"http_requests_total"},
					LabelRegex: map[string]string{
						"path": "^/api/.*",
					},
				},
			},
			want: false,
		},
		{
			name: "complex match - name, label and regex",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method":  "GET",
					"path":    "/api/v1/users",
					"service": "user-service",
				},
			},
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"http_requests_total"},
					Labels: map[string]string{
						"method": "GET",
					},
					LabelRegex: map[string]string{
						"path": "^/api/.*",
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matcher.matchesRule(tt.sample, tt.rule); got != tt.want {
				t.Errorf("Matcher.matchesRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatcher_MatchingRules(t *testing.T) {
	// Create rules for testing
	rule1 := &models.Rule{
		ID:      "rule1",
		Enabled: true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_requests_total"},
			Labels: map[string]string{
				"method": "GET",
			},
		},
	}
	
	rule2 := &models.Rule{
		ID:      "rule2",
		Enabled: true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_requests_total"},
			LabelRegex: map[string]string{
				"path": "^/api/.*",
			},
		},
	}
	
	rule3 := &models.Rule{
		ID:      "rule3",
		Enabled: false, // Disabled rule
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_requests_total"},
		},
	}
	
	rule4 := &models.Rule{
		ID:      "rule4",
		Enabled: true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"node_cpu_seconds_total"},
		},
	}

	// Create engine with rules
	engine := &Engine{
		rules: map[string]*models.Rule{
			"rule1": rule1,
			"rule2": rule2,
			"rule3": rule3,
			"rule4": rule4,
		},
	}
	
	matcher := NewMatcher(engine)

	tests := []struct {
		name   string
		sample *models.MetricSample
		want   []*models.Rule
	}{
		{
			name: "match single rule",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
					"path":   "/ui/dashboard",
				},
			},
			want: []*models.Rule{rule1},
		},
		{
			name: "match multiple rules",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{
					"method": "GET",
					"path":   "/api/v1/users",
				},
			},
			want: []*models.Rule{rule1, rule2},
		},
		{
			name: "no match",
			sample: &models.MetricSample{
				Name: "node_memory_bytes",
				Labels: map[string]string{
					"instance": "localhost:9090",
				},
			},
			want: []*models.Rule{},
		},
		{
			name: "should skip disabled rules",
			sample: &models.MetricSample{
				Name: "http_requests_total",
				Labels: map[string]string{},
			},
			want: []*models.Rule{}, // rule3 is disabled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.MatchingRules(tt.sample)
			
			// Create maps for easier comparison (order doesn't matter)
			gotMap := make(map[string]*models.Rule)
			for _, rule := range got {
				gotMap[rule.ID] = rule
			}
			
			wantMap := make(map[string]*models.Rule)
			for _, rule := range tt.want {
				wantMap[rule.ID] = rule
			}
			
			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("Matcher.MatchingRules() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatcher_GetRulesByMetricName(t *testing.T) {
	// Create rules for testing
	rule1 := &models.Rule{
		ID:      "rule1",
		Enabled: true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_requests_total"},
		},
	}
	
	rule2 := &models.Rule{
		ID:      "rule2",
		Enabled: true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_*"},
		},
	}
	
	rule3 := &models.Rule{
		ID:      "rule3",
		Enabled: true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"*"},
		},
	}
	
	rule4 := &models.Rule{
		ID:      "rule4",
		Enabled: false, // Disabled rule
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_requests_total"},
		},
	}

	// Create engine with rules
	engine := &Engine{
		rules: map[string]*models.Rule{
			"rule1": rule1,
			"rule2": rule2,
			"rule3": rule3,
			"rule4": rule4,
		},
	}
	
	matcher := NewMatcher(engine)

	tests := []struct {
		name       string
		metricName string
		want       []*models.Rule
	}{
		{
			name:       "exact match",
			metricName: "http_requests_total",
			want:       []*models.Rule{rule1, rule2, rule3}, // rule4 is disabled
		},
		{
			name:       "wildcard pattern match",
			metricName: "http_response_size_bytes",
			want:       []*models.Rule{rule2, rule3},
		},
		{
			name:       "global wildcard match",
			metricName: "node_cpu_seconds_total",
			want:       []*models.Rule{rule3},
		},
		{
			name:       "no match",
			metricName: "nonexistent_metric",
			want:       []*models.Rule{rule3}, // Only matched by global wildcard
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.GetRulesByMetricName(tt.metricName)
			
			// Create maps for easier comparison (order doesn't matter)
			gotMap := make(map[string]*models.Rule)
			for _, rule := range got {
				gotMap[rule.ID] = rule
			}
			
			wantMap := make(map[string]*models.Rule)
			for _, rule := range tt.want {
				wantMap[rule.ID] = rule
			}
			
			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("Matcher.GetRulesByMetricName() = %v, want %v", got, tt.want)
			}
		})
	}
}