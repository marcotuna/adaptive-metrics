package metrics

import (
	"testing"
	"time"

	"github.com/marcotuna/adaptive-metrics/internal/models"
)

func TestRecommendationEngine_DetermineSegmentationLabels(t *testing.T) {
	usageTracker := NewUsageTracker(90 * 24 * time.Hour)
	engine := NewRecommendationEngine(usageTracker, 1000, 100, 0.5)

	tests := []struct {
		name      string
		metricInfo *MetricUsageInfo
		wantLabels int
	}{
		{
			name: "normal labels distribution",
			metricInfo: &MetricUsageInfo{
				MetricName: "http_requests_total",
				LabelCardinality: map[string]int{
					"method":      2,    // Low cardinality
					"path":        1000, // High cardinality (should be excluded)
					"status_code": 5,    // Low cardinality
					"handler":     20,   // Medium cardinality
				},
				Cardinality: 1500, // Total cardinality
			},
			wantLabels: 3, // Should include method, status_code, handler (path excluded as too high)
		},
		{
			name: "empty labels",
			metricInfo: &MetricUsageInfo{
				MetricName:       "simple_metric",
				LabelCardinality: map[string]int{},
				Cardinality:      1,
			},
			wantLabels: 0,
		},
		{
			name: "all high cardinality labels",
			metricInfo: &MetricUsageInfo{
				MetricName: "high_cardinality_metric",
				LabelCardinality: map[string]int{
					"id":       1000,
					"user_id":  800,
					"trace_id": 1500,
				},
				Cardinality: 3000,
			},
			wantLabels: 0, // All labels have too high cardinality
		},
		{
			name: "all very low cardinality labels",
			metricInfo: &MetricUsageInfo{
				MetricName: "low_cardinality_metric",
				LabelCardinality: map[string]int{
					"region":    1,
					"datacenter": 1,
				},
				Cardinality: 1,
			},
			wantLabels: 0, // All labels have too low cardinality (below 2)
		},
		{
			name: "mix of usable labels",
			metricInfo: &MetricUsageInfo{
				MetricName: "mixed_metric",
				LabelCardinality: map[string]int{
					"service":     5,
					"environment": 3,
					"pod":         100, // Higher but still below threshold
					"container":   2,
				},
				Cardinality: 600,
			},
			wantLabels: 3, // Should pick 3 labels
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segmentationLabels := engine.determineSegmentationLabels(tt.metricInfo)
			if len(segmentationLabels) != tt.wantLabels {
				t.Errorf("determineSegmentationLabels() returned %d labels, want %d", 
					len(segmentationLabels), tt.wantLabels)
			}
		})
	}
}

func TestRecommendationEngine_DetermineAggregationType(t *testing.T) {
	usageTracker := NewUsageTracker(90 * 24 * time.Hour)
	engine := NewRecommendationEngine(usageTracker, 1000, 100, 0.5)

	tests := []struct {
		name      string
		metricInfo *MetricUsageInfo
		want      string
	}{
		{
			name: "counter-like metric (always increasing)",
			metricInfo: &MetricUsageInfo{
				MetricName: "http_requests_total",
				MinValue:   0,
				MaxValue:   1000,
				SumValue:   1000,
			},
			want: "sum",
		},
		{
			name: "gauge-like metric (goes up and down)",
			metricInfo: &MetricUsageInfo{
				MetricName: "memory_usage_bytes",
				MinValue:   -10,
				MaxValue:   100,
				SumValue:   500,
			},
			want: "avg",
		},
		{
			name: "another counter-like metric",
			metricInfo: &MetricUsageInfo{
				MetricName: "logged_in_users_total",
				MinValue:   0,
				MaxValue:   50,
				SumValue:   150,
			},
			want: "sum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.determineAggregationType(tt.metricInfo)
			if got != tt.want {
				t.Errorf("determineAggregationType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecommendationEngine_EstimateImpact(t *testing.T) {
	usageTracker := NewUsageTracker(90 * 24 * time.Hour)
	engine := NewRecommendationEngine(usageTracker, 1000, 100, 0.5)

	tests := []struct {
		name      string
		metricInfo *MetricUsageInfo
		segLabels  []string
		wantReduc  float64
		wantSavings float64
	}{
		{
			name: "high cardinality reduction",
			metricInfo: &MetricUsageInfo{
				MetricName: "http_requests_total",
				Cardinality: 1000,
				LabelCardinality: map[string]int{
					"method":      4,
					"status_code": 5,
				},
			},
			segLabels:   []string{"method", "status_code"},
			wantReduc:   50.0,    // 1000 / (4*5) = 50
			wantSavings: 98.0,    // (1 - 1/50) * 100 = 98%
		},
		{
			name: "moderate cardinality reduction",
			metricInfo: &MetricUsageInfo{
				MetricName: "api_latency_seconds",
				Cardinality: 500,
				LabelCardinality: map[string]int{
					"endpoint": 20,
					"method":   4,
				},
			},
			segLabels:   []string{"endpoint", "method"},
			wantReduc:   6.25,    // 500 / (20*4) = 6.25
			wantSavings: 84.0,    // (1 - 1/6.25) * 100 = 84%
		},
		{
			name: "single label segmentation",
			metricInfo: &MetricUsageInfo{
				MetricName: "queue_size",
				Cardinality: 100,
				LabelCardinality: map[string]int{
					"queue": 10,
				},
			},
			segLabels:   []string{"queue"},
			wantReduc:   10.0,    // 100 / 10 = 10
			wantSavings: 90.0,    // (1 - 1/10) * 100 = 90%
		},
		{
			name: "no labels - edge case",
			metricInfo: &MetricUsageInfo{
				MetricName: "simple_metric",
				Cardinality: 50,
				LabelCardinality: map[string]int{},
			},
			segLabels:   []string{},
			wantReduc:   50.0,    // 50 / 1 = 50
			wantSavings: 98.0,    // (1 - 1/50) * 100 = 98%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impact := engine.estimateImpact(tt.metricInfo, tt.segLabels)
			
			// Check cardinality reduction with tolerance for floating point comparison
			if !almostEqual(impact.CardinalityReduction, tt.wantReduc, 0.1) {
				t.Errorf("estimateImpact() CardinalityReduction = %v, want %v", 
					impact.CardinalityReduction, tt.wantReduc)
			}
			
			// Check savings percentage with tolerance
			if !almostEqual(impact.SavingsPercentage, tt.wantSavings, 0.1) {
				t.Errorf("estimateImpact() SavingsPercentage = %v, want %v", 
					impact.SavingsPercentage, tt.wantSavings)
			}
			
			// Check affected series
			if impact.AffectedSeries != tt.metricInfo.Cardinality {
				t.Errorf("estimateImpact() AffectedSeries = %v, want %v", 
					impact.AffectedSeries, tt.metricInfo.Cardinality)
			}
			
			// Check retention period exists
			if impact.RetentionPeriod != "30d" {
				t.Errorf("estimateImpact() RetentionPeriod = %v, want %v", 
					impact.RetentionPeriod, "30d")
			}
		})
	}
}

func TestRecommendationEngine_CalculateConfidence(t *testing.T) {
	usageTracker := NewUsageTracker(90 * 24 * time.Hour)
	engine := NewRecommendationEngine(usageTracker, 1000, 100, 0.5)

	tests := []struct {
		name      string
		metricInfo *MetricUsageInfo
		impact     *models.EstimatedImpact
		want      float64
	}{
		{
			name: "high confidence recommendation",
			metricInfo: &MetricUsageInfo{
				MetricName:  "http_requests_total",
				SampleCount: 10000,  // Lots of samples
				Cardinality: 1000,   // High cardinality
			},
			impact: &models.EstimatedImpact{
				CardinalityReduction: 50.0,  // High impact
				SavingsPercentage:    98.0,
			},
			want: 0.85, // High confidence (between 0.8 and 0.9)
		},
		{
			name: "medium confidence recommendation",
			metricInfo: &MetricUsageInfo{
				MetricName:  "api_latency_seconds",
				SampleCount: 5000,   // Moderate samples
				Cardinality: 500,    // Moderate cardinality
			},
			impact: &models.EstimatedImpact{
				CardinalityReduction: 10.0,  // Moderate impact
				SavingsPercentage:    90.0,
			},
			want: 0.6, // Medium confidence (between 0.5 and 0.7)
		},
		{
			name: "low confidence recommendation",
			metricInfo: &MetricUsageInfo{
				MetricName:  "rarely_seen_metric",
				SampleCount: 1200,   // Just over threshold
				Cardinality: 120,    // Just over threshold
			},
			impact: &models.EstimatedImpact{
				CardinalityReduction: 2.0,   // Low impact
				SavingsPercentage:    50.0,
			},
			want: 0.3, // Low confidence (between 0.2 and 0.4)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.calculateConfidence(tt.metricInfo, tt.impact)
			
			// Use a tolerance for floating point comparison
			if !almostEqual(got, tt.want, 0.15) {
				t.Errorf("calculateConfidence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecommendationEngine_GenerateRecommendationForMetric(t *testing.T) {
	usageTracker := NewUsageTracker(90 * 24 * time.Hour)
	engine := NewRecommendationEngine(usageTracker, 1000, 100, 0.5)

	tests := []struct {
		name      string
		metricInfo *MetricUsageInfo
		wantRec   bool  // Whether a recommendation should be generated
	}{
		{
			name: "good candidate for recommendation",
			metricInfo: &MetricUsageInfo{
				MetricName:  "http_requests_total",
				SampleCount: 10000,
				Cardinality: 1000,
				LabelCardinality: map[string]int{
					"method":      4,
					"status_code": 5,
					"path":        500,  // Too high to be used as segment
				},
				MinValue:   0,
				MaxValue:   1000,
				SumValue:   50000,
			},
			wantRec: true,
		},
		{
			name: "no good segmentation labels",
			metricInfo: &MetricUsageInfo{
				MetricName:  "unique_ids",
				SampleCount: 5000,
				Cardinality: 5000,
				LabelCardinality: map[string]int{
					"id":      5000,  // All labels have high cardinality
					"user_id": 4000,
				},
				MinValue:   0,
				MaxValue:   5000,
				SumValue:   10000,
			},
			wantRec: false,
		},
		{
			name: "low cardinality reduction",
			metricInfo: &MetricUsageInfo{
				MetricName:  "simple_metric",
				SampleCount: 2000,
				Cardinality: 100,
				LabelCardinality: map[string]int{
					"type": 90,  // Almost as high as total cardinality
				},
				MinValue:   0,
				MaxValue:   100,
				SumValue:   5000,
			},
			wantRec: false, // Cardinality reduction would be too low
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendation := engine.generateRecommendationForMetric(tt.metricInfo)
			
			if tt.wantRec && recommendation == nil {
				t.Errorf("generateRecommendationForMetric() returned nil, expected a recommendation")
			} else if !tt.wantRec && recommendation != nil {
				t.Errorf("generateRecommendationForMetric() returned a recommendation, expected nil")
			}
			
			if recommendation != nil {
				// Check that essential fields are set
				if recommendation.ID == "" {
					t.Errorf("Recommendation ID is empty")
				}
				
				if recommendation.Rule.Matcher.MetricNames[0] != tt.metricInfo.MetricName {
					t.Errorf("Rule MetricNames[0] = %v, want %v", 
						recommendation.Rule.Matcher.MetricNames[0], tt.metricInfo.MetricName)
				}
				
				// Output metric name should be derived from original
				expectedOutputName := tt.metricInfo.MetricName + "_aggregated"
				if recommendation.Rule.Output.MetricName != expectedOutputName {
					t.Errorf("Rule Output.MetricName = %v, want %v", 
						recommendation.Rule.Output.MetricName, expectedOutputName)
				}
				
				// Check that confidence and estimated impact are set
				if recommendation.Confidence <= 0 {
					t.Errorf("Recommendation confidence is <= 0")
				}
				
				if recommendation.EstimatedImpact == nil {
					t.Errorf("EstimatedImpact is nil")
				} else if recommendation.EstimatedImpact.CardinalityReduction <= 1.0 {
					t.Errorf("CardinalityReduction = %v, should be > 1.0", 
						recommendation.EstimatedImpact.CardinalityReduction)
				}
			}
		})
	}
}

// Helper function for approximate floating point comparison
func almostEqual(a, b, tolerance float64) bool {
	return (a-b) < tolerance && (b-a) < tolerance
}