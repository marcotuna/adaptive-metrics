package models

import (
	"testing"
	"time"
)

func TestRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rule",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 60,
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			rule: Rule{
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 60,
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: true,
			errMsg:  "rule name is required",
		},
		{
			name: "missing metric names",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 60,
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: true,
			errMsg:  "at least one metric name must be specified",
		},
		{
			name: "invalid aggregation type",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "invalid",
					IntervalSeconds: 60,
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: true,
			errMsg:  "invalid aggregation type: invalid",
		},
		{
			name: "invalid interval",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 0,
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: true,
			errMsg:  "aggregation interval must be greater than 0",
		},
		{
			name: "missing output metric name",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 60,
				},
				Output: OutputConfig{
					MetricName: "",
				},
			},
			wantErr: true,
			errMsg:  "output metric name is required",
		},
		{
			name: "invalid segmentation rule - missing label",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 60,
					SegmentationRules: []SegmentationRule{
						{
							Label:     "",
							LimitType: "top",
							Limit:     10,
						},
					},
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: true,
			errMsg:  "segmentation rule label is required",
		},
		{
			name: "invalid segmentation rule - invalid limit type",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 60,
					SegmentationRules: []SegmentationRule{
						{
							Label:     "method",
							LimitType: "invalid",
							Limit:     10,
						},
					},
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: true,
			errMsg:  "invalid segmentation limit type: invalid",
		},
		{
			name: "invalid segmentation rule - top without limit",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 60,
					SegmentationRules: []SegmentationRule{
						{
							Label:     "method",
							LimitType: "top",
							Limit:     0,
						},
					},
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: true,
			errMsg:  "segmentation limit must be greater than 0 for type top",
		},
		{
			name: "invalid segmentation rule - include without values",
			rule: Rule{
				Name: "Test Rule",
				Matcher: MetricMatcher{
					MetricNames: []string{"http_requests_total"},
				},
				Aggregation: AggregationConfig{
					Type:            "sum",
					IntervalSeconds: 60,
					SegmentationRules: []SegmentationRule{
						{
							Label:     "method",
							LimitType: "include",
							Values:    []string{},
						},
					},
				},
				Output: OutputConfig{
					MetricName: "http_requests_aggregated",
				},
			},
			wantErr: true,
			errMsg:  "segmentation values must be specified for type include",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Rule.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("Rule.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestEstimatedImpact(t *testing.T) {
	impact := &EstimatedImpact{
		CardinalityReduction: 10.5,
		SavingsPercentage:    90.5,
		AffectedSeries:       1000,
		RetentionPeriod:      "30d",
	}

	if impact.CardinalityReduction != 10.5 {
		t.Errorf("EstimatedImpact.CardinalityReduction = %v, want %v", impact.CardinalityReduction, 10.5)
	}

	if impact.SavingsPercentage != 90.5 {
		t.Errorf("EstimatedImpact.SavingsPercentage = %v, want %v", impact.SavingsPercentage, 90.5)
	}

	if impact.AffectedSeries != 1000 {
		t.Errorf("EstimatedImpact.AffectedSeries = %v, want %v", impact.AffectedSeries, 1000)
	}

	if impact.RetentionPeriod != "30d" {
		t.Errorf("EstimatedImpact.RetentionPeriod = %v, want %v", impact.RetentionPeriod, "30d")
	}
}

func TestMetricSample(t *testing.T) {
	now := time.Now()
	sample := &MetricSample{
		Name:      "test_metric",
		Value:     42.0,
		Timestamp: now,
		Labels: map[string]string{
			"label1": "value1",
			"label2": "value2",
		},
	}

	if sample.Name != "test_metric" {
		t.Errorf("MetricSample.Name = %v, want %v", sample.Name, "test_metric")
	}

	if sample.Value != 42.0 {
		t.Errorf("MetricSample.Value = %v, want %v", sample.Value, 42.0)
	}

	if !sample.Timestamp.Equal(now) {
		t.Errorf("MetricSample.Timestamp = %v, want %v", sample.Timestamp, now)
	}

	if len(sample.Labels) != 2 {
		t.Errorf("len(MetricSample.Labels) = %v, want %v", len(sample.Labels), 2)
	}

	if v, ok := sample.Labels["label1"]; !ok || v != "value1" {
		t.Errorf("MetricSample.Labels[\"label1\"] = %v, want %v", v, "value1")
	}
}

func TestAggregatedMetric(t *testing.T) {
	start := time.Now()
	end := start.Add(60 * time.Second)
	
	metric := &AggregatedMetric{
		Name:       "test_aggregated",
		Value:      100.0,
		StartTime:  start,
		EndTime:    end,
		Labels:     map[string]string{"source": "test"},
		SourceRule: "rule-123",
		Count:      10,
	}

	if metric.Name != "test_aggregated" {
		t.Errorf("AggregatedMetric.Name = %v, want %v", metric.Name, "test_aggregated")
	}

	if metric.Value != 100.0 {
		t.Errorf("AggregatedMetric.Value = %v, want %v", metric.Value, 100.0)
	}

	if !metric.StartTime.Equal(start) {
		t.Errorf("AggregatedMetric.StartTime = %v, want %v", metric.StartTime, start)
	}

	if !metric.EndTime.Equal(end) {
		t.Errorf("AggregatedMetric.EndTime = %v, want %v", metric.EndTime, end)
	}

	if metric.SourceRule != "rule-123" {
		t.Errorf("AggregatedMetric.SourceRule = %v, want %v", metric.SourceRule, "rule-123")
	}

	if metric.Count != 10 {
		t.Errorf("AggregatedMetric.Count = %v, want %v", metric.Count, 10)
	}
}