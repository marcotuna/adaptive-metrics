package metrics

import (
	"testing"
	"time"
)

func TestUsageTracker_TrackMetric(t *testing.T) {
	// Create a tracker with a short retention period for testing
	tracker := NewUsageTracker(1 * time.Hour)

	// Track some metrics
	tracker.TrackMetric("test_metric", map[string]string{
		"label1": "value1",
		"label2": "value2",
	}, 42.0)

	tracker.TrackMetric("test_metric", map[string]string{
		"label1": "value1",
		"label2": "value3", // Different value for label2
	}, 43.0)

	// Track the same metric with same labels again
	tracker.TrackMetric("test_metric", map[string]string{
		"label1": "value1",
		"label2": "value2",
	}, 44.0)

	// Check that the summary info was updated correctly
	info := tracker.GetMetricInfo("test_metric")
	if info == nil {
		t.Fatal("Expected metric info but got nil")
	}

	if info.MetricName != "test_metric" {
		t.Errorf("MetricName = %v, want %v", info.MetricName, "test_metric")
	}

	if info.SampleCount != 3 {
		t.Errorf("SampleCount = %v, want %v", info.SampleCount, 3)
	}

	if info.Cardinality != 2 {
		t.Errorf("Cardinality = %v, want %v", info.Cardinality, 2)
	}

	if len(info.LabelCardinality) != 2 {
		t.Errorf("LabelCardinality has %v entries, want %v", len(info.LabelCardinality), 2)
	}

	if info.MinValue != 42.0 {
		t.Errorf("MinValue = %v, want %v", info.MinValue, 42.0)
	}

	if info.MaxValue != 44.0 {
		t.Errorf("MaxValue = %v, want %v", info.MaxValue, 44.0)
	}

	if info.SumValue != 129.0 {
		t.Errorf("SumValue = %v, want %v", info.SumValue, 129.0)
	}

	// Check that label cardinality was tracked
	if info.LabelCardinality["label1"] != 1 {
		t.Errorf("LabelCardinality[label1] = %v, want %v", info.LabelCardinality["label1"], 1)
	}

	if info.LabelCardinality["label2"] != 2 {
		t.Errorf("LabelCardinality[label2] = %v, want %v", info.LabelCardinality["label2"], 2)
	}
}

func TestUsageTracker_GetAllMetricsInfo(t *testing.T) {
	// Create a tracker with a short retention period for testing
	tracker := NewUsageTracker(1 * time.Hour)

	// Track multiple metrics
	tracker.TrackMetric("metric1", map[string]string{"app": "app1"}, 10.0)
	tracker.TrackMetric("metric2", map[string]string{"app": "app2"}, 20.0)
	tracker.TrackMetric("metric3", map[string]string{"app": "app3"}, 30.0)

	// Get all metrics info
	metricsInfo := tracker.GetAllMetricsInfo()

	// Verify we have info for all tracked metrics
	if len(metricsInfo) != 3 {
		t.Errorf("GetAllMetricsInfo() returned %v metrics, want %v", len(metricsInfo), 3)
	}

	// Check that each metric is present with correct name
	if info, exists := metricsInfo["metric1"]; !exists {
		t.Errorf("metric1 not found in GetAllMetricsInfo() result")
	} else if info.MetricName != "metric1" {
		t.Errorf("metricsInfo[metric1].MetricName = %v, want %v", info.MetricName, "metric1")
	}

	if info, exists := metricsInfo["metric2"]; !exists {
		t.Errorf("metric2 not found in GetAllMetricsInfo() result")
	} else if info.MetricName != "metric2" {
		t.Errorf("metricsInfo[metric2].MetricName = %v, want %v", info.MetricName, "metric2")
	}

	if info, exists := metricsInfo["metric3"]; !exists {
		t.Errorf("metric3 not found in GetAllMetricsInfo() result")
	} else if info.MetricName != "metric3" {
		t.Errorf("metricsInfo[metric3].MetricName = %v, want %v", info.MetricName, "metric3")
	}
}

func TestUsageTracker_Cleanup(t *testing.T) {
	// Create a tracker with a very short retention period
	tracker := NewUsageTracker(10 * time.Millisecond)

	// Track metrics
	tracker.TrackMetric("old_metric", map[string]string{"age": "old"}, 1.0)

	// Wait for retention period to expire
	time.Sleep(20 * time.Millisecond)

	// Track another metric
	tracker.TrackMetric("new_metric", map[string]string{"age": "new"}, 2.0)

	// Force a cleanup by tracking another metric
	tracker.TrackMetric("trigger_cleanup", map[string]string{}, 3.0)

	// Verify the old metric was cleaned up
	oldInfo := tracker.GetMetricInfo("old_metric")
	if oldInfo != nil {
		t.Errorf("Expected old_metric to be cleaned up but it still exists")
	}

	// Verify new metrics are still there
	newInfo := tracker.GetMetricInfo("new_metric")
	if newInfo == nil {
		t.Errorf("Expected new_metric to exist but it was cleaned up")
	}

	triggerInfo := tracker.GetMetricInfo("trigger_cleanup")
	if triggerInfo == nil {
		t.Errorf("Expected trigger_cleanup to exist but it was cleaned up")
	}
}

func TestUsageTracker_DetailedUsage(t *testing.T) {
	tracker := NewUsageTracker(1 * time.Hour)

	// Track a metric with different label combinations
	tracker.TrackMetric("detailed_metric", map[string]string{
		"region": "us-west", 
		"status": "success",
	}, 10.0)

	tracker.TrackMetric("detailed_metric", map[string]string{
		"region": "us-east",
		"status": "success",
	}, 20.0)

	tracker.TrackMetric("detailed_metric", map[string]string{
		"region": "us-west",
		"status": "error",
	}, 5.0)

	// Get metric info and check cardinality
	info := tracker.GetMetricInfo("detailed_metric")
	if info == nil {
		t.Fatal("Expected metric info but got nil")
	}

	// Should have 3 different label combinations
	if info.Cardinality != 3 {
		t.Errorf("Cardinality = %v, want %v", info.Cardinality, 3)
	}

	// Should have 2 different values for region label
	if info.LabelCardinality["region"] != 2 {
		t.Errorf("LabelCardinality[region] = %v, want %v", info.LabelCardinality["region"], 2)
	}

	// Should have 2 different values for status label
	if info.LabelCardinality["status"] != 2 {
		t.Errorf("LabelCardinality[status] = %v, want %v", info.LabelCardinality["status"], 2)
	}
}

func TestUsageTracker_EdgeCases(t *testing.T) {
	tracker := NewUsageTracker(1 * time.Hour)

	// Track a metric with no labels
	tracker.TrackMetric("no_labels_metric", map[string]string{}, 42.0)

	info := tracker.GetMetricInfo("no_labels_metric")
	if info == nil {
		t.Fatal("Expected metric info but got nil")
	}

	if info.Cardinality != 1 {
		t.Errorf("Cardinality = %v, want %v", info.Cardinality, 1)
	}

	if len(info.LabelCardinality) != 0 {
		t.Errorf("LabelCardinality has %v entries, want %v", len(info.LabelCardinality), 0)
	}

	// Get info for non-existent metric
	nonExistentInfo := tracker.GetMetricInfo("i_dont_exist")
	if nonExistentInfo != nil {
		t.Errorf("Expected nil for non-existent metric but got %v", nonExistentInfo)
	}

	// Track a metric with extreme values
	tracker.TrackMetric("extreme_metric", map[string]string{"extreme": "true"}, -1000000.0)
	tracker.TrackMetric("extreme_metric", map[string]string{"extreme": "true"}, 1000000.0)

	extremeInfo := tracker.GetMetricInfo("extreme_metric")
	if extremeInfo == nil {
		t.Fatal("Expected metric info but got nil")
	}

	if extremeInfo.MinValue != -1000000.0 {
		t.Errorf("MinValue = %v, want %v", extremeInfo.MinValue, -1000000.0)
	}

	if extremeInfo.MaxValue != 1000000.0 {
		t.Errorf("MaxValue = %v, want %v", extremeInfo.MaxValue, 1000000.0)
	}
}