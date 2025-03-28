package metrics

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/marcotuna/adaptive-metrics/internal/models"
)

// RecommendationEngine analyzes metric usage to generate aggregation rule recommendations
type RecommendationEngine struct {
	usageTracker       *UsageTracker
	minSampleThreshold int64
	minCardinalityThreshold int
	minConfidence     float64
}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine(usageTracker *UsageTracker, minSampleThreshold int64, minCardinalityThreshold int, minConfidence float64) *RecommendationEngine {
	return &RecommendationEngine{
		usageTracker:       usageTracker,
		minSampleThreshold: minSampleThreshold,
		minCardinalityThreshold: minCardinalityThreshold,
		minConfidence:     minConfidence,
	}
}

// GenerateRecommendations analyzes metric usage to generate aggregation rule recommendations
func (re *RecommendationEngine) GenerateRecommendations() []models.Recommendation {
	var recommendations []models.Recommendation
	metricsInfo := re.usageTracker.GetAllMetricsInfo()

	// Filter metrics that meet the criteria for recommendation
	for _, metricInfo := range metricsInfo {
		// Skip metrics with low cardinality or sample count
		if metricInfo.Cardinality < re.minCardinalityThreshold || metricInfo.SampleCount < re.minSampleThreshold {
			continue
		}

		// Generate recommendations for high-cardinality metrics
		recommendation := re.generateRecommendationForMetric(metricInfo)
		if recommendation != nil {
			recommendations = append(recommendations, *recommendation)
		}
	}

	return recommendations
}

// generateRecommendationForMetric creates a recommendation for a specific metric
func (re *RecommendationEngine) generateRecommendationForMetric(metricInfo *MetricUsageInfo) *models.Recommendation {
	// Analyze label cardinality to determine which labels to segment by
	segmentationLabels := re.determineSegmentationLabels(metricInfo)
	if len(segmentationLabels) == 0 {
		return nil // No good segmentation labels found
	}

	// Determine the best aggregation type based on metric behavior
	aggregationType := re.determineAggregationType(metricInfo)

	// Estimate the impact of aggregation
	estimatedImpact := re.estimateImpact(metricInfo, segmentationLabels)
	if estimatedImpact.CardinalityReduction < 2.0 {
		return nil // Not worth aggregating
	}

	// Calculate confidence score
	confidence := re.calculateConfidence(metricInfo, estimatedImpact)
	if confidence < re.minConfidence {
		return nil // Low confidence recommendation
	}

	// Create a rule based on the analysis
	rule := models.Rule{
		ID:          fmt.Sprintf("autogen-%s", uuid.New().String()[:8]),
		Name:        fmt.Sprintf("Recommended aggregation for %s", metricInfo.MetricName),
		Description: fmt.Sprintf("Automatically generated rule to aggregate high-cardinality metric %s based on usage patterns", metricInfo.MetricName),
		Enabled:     false, // Default to disabled until user confirms
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Matcher: models.MetricMatcher{
			MetricNames: []string{metricInfo.MetricName},
			Labels:      make(map[string]string),
			LabelRegex:  make(map[string]string),
		},
		Aggregation: models.AggregationConfig{
			Type:            aggregationType,
			IntervalSeconds: re.determineAggregationInterval(metricInfo),
			Segmentation:    segmentationLabels,
			DelayMs:         5000, // Default delay of 5 seconds
		},
		Output: models.OutputConfig{
			MetricName: fmt.Sprintf("%s_aggregated", metricInfo.MetricName),
			AdditionalLabels: map[string]string{
				"aggregated_by": "adaptive_metrics",
				"source":        "usage_based_recommendation",
			},
			DropOriginal: false, // Default to keeping original metrics
		},
		Source:     "usage_analysis",
		Confidence: confidence,
		EstimatedImpact: estimatedImpact,
	}

	return &models.Recommendation{
		ID:              uuid.New().String(),
		CreatedAt:       time.Now(),
		Rule:            rule,
		Confidence:      confidence,
		EstimatedImpact: estimatedImpact,
		Source:          "usage_analysis",
		Status:          "pending",
	}
}

// determineSegmentationLabels analyzes label usage to determine which labels to segment by
func (re *RecommendationEngine) determineSegmentationLabels(metricInfo *MetricUsageInfo) []string {
	type labelInfo struct {
		name        string
		cardinality int
	}

	// Create a list of labels sorted by their cardinality
	var labels []labelInfo
	for label, cardinality := range metricInfo.LabelCardinality {
		labels = append(labels, labelInfo{name: label, cardinality: cardinality})
	}

	// Sort labels by cardinality from lowest to highest
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].cardinality < labels[j].cardinality
	})

	// Select labels with moderate cardinality for segmentation
	// High cardinality labels are filtered out as they would defeat the purpose of aggregation
	// Very low cardinality labels might be too coarse for meaningful aggregation
	var segmentationLabels []string
	for _, label := range labels {
		// Skip labels with extremely high cardinality (more than 20% of total cardinality)
		if float64(label.cardinality) > float64(metricInfo.Cardinality)*0.2 {
			continue
		}

		// Skip labels with extremely low cardinality (less than 2)
		if label.cardinality < 2 {
			continue
		}

		segmentationLabels = append(segmentationLabels, label.name)

		// Limit to 3 segmentation labels for efficiency
		if len(segmentationLabels) >= 3 {
			break
		}
	}

	return segmentationLabels
}

// determineAggregationType determines the best aggregation type based on metric behavior
func (re *RecommendationEngine) determineAggregationType(metricInfo *MetricUsageInfo) string {
	// Default to sum for most metrics
	// In a real implementation, this would include more complex analysis
	// of the metric's behavior over time
	
	// Simple heuristic: counter-like metrics (always increasing) -> sum
	// Gauge-like metrics -> avg
	if metricInfo.MinValue >= 0 && metricInfo.SumValue >= 0 {
		return "sum"
	}
	return "avg"
}

// determineAggregationInterval determines the best aggregation interval
func (re *RecommendationEngine) determineAggregationInterval(metricInfo *MetricUsageInfo) int {
	// Simple implementation - in a real system, analyze the scrape interval
	// and typical query patterns to determine the best interval
	return 60 // Default to 60 seconds
}

// estimateImpact estimates the impact of applying a recommended aggregation
func (re *RecommendationEngine) estimateImpact(metricInfo *MetricUsageInfo, segmentationLabels []string) *models.EstimatedImpact {
	// Estimate cardinality reduction 
	// (total cardinality / estimated post-aggregation cardinality)
	
	// For each segmentation label, estimate its unique values
	// This is a simplified calculation - in a real system would be more precise
	estimatedPostAggregationCardinality := 1
	for _, label := range segmentationLabels {
		if cardinality, exists := metricInfo.LabelCardinality[label]; exists {
			estimatedPostAggregationCardinality *= cardinality
		}
	}
	
	// Ensure we don't divide by zero
	if estimatedPostAggregationCardinality == 0 {
		estimatedPostAggregationCardinality = 1
	}
	
	// Calculate reduction ratio
	cardinalityReduction := float64(metricInfo.Cardinality) / float64(estimatedPostAggregationCardinality)
	
	// Calculate savings percentage (simple estimate)
	savingsPercentage := (1.0 - (1.0 / cardinalityReduction)) * 100.0
	
	return &models.EstimatedImpact{
		CardinalityReduction: cardinalityReduction,
		SavingsPercentage:    savingsPercentage,
		AffectedSeries:       metricInfo.Cardinality,
		RetentionPeriod:      "30d", // Default assumption
	}
}

// calculateConfidence calculates a confidence score for the recommendation
func (re *RecommendationEngine) calculateConfidence(metricInfo *MetricUsageInfo, impact *models.EstimatedImpact) float64 {
	// Simple formula: confidence is based on:
	// 1. Sample count (more samples = more confidence)
	// 2. Cardinality (higher cardinality = higher confidence)
	// 3. Impact (higher impact = higher confidence)
	
	// Normalize sample count (0.0 - 1.0)
	sampleScore := min(float64(metricInfo.SampleCount)/10000.0, 1.0)
	
	// Normalize cardinality (0.0 - 1.0)
	cardinalityScore := min(float64(metricInfo.Cardinality)/1000.0, 1.0)
	
	// Impact score based on cardinality reduction
	impactScore := min(impact.CardinalityReduction/100.0, 1.0)
	
	// Combined confidence score (weighted average)
	confidence := (sampleScore*0.3 + cardinalityScore*0.4 + impactScore*0.3)
	
	return confidence
}