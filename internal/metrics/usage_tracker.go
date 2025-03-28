package metrics

import (
	"sync"
	"time"
)

// MetricUsageInfo stores usage information for a specific metric
type MetricUsageInfo struct {
	MetricName       string
	Labels           map[string]string
	SampleCount      int64
	LastSeen         time.Time
	FirstSeen        time.Time
	Cardinality      int
	LabelCardinality map[string]int // Maps label keys to their cardinality
	MinValue         float64
	MaxValue         float64
	SumValue         float64
}

// UsageTracker tracks usage information for metrics
type UsageTracker struct {
	mu              sync.RWMutex
	metricsUsage    map[string]*MetricUsageInfo            // Tracks usage by metric name
	detailedUsage   map[string]map[string]*MetricUsageInfo // Tracks usage by metric name + label hash
	retentionPeriod time.Duration
	lastCleanup     time.Time
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker(retentionPeriod time.Duration) *UsageTracker {
	return &UsageTracker{
		metricsUsage:    make(map[string]*MetricUsageInfo),
		detailedUsage:   make(map[string]map[string]*MetricUsageInfo),
		retentionPeriod: retentionPeriod,
		lastCleanup:     time.Now(),
	}
}

// TrackMetric records usage information for a metric
func (ut *UsageTracker) TrackMetric(name string, labels map[string]string, value float64) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	// Track summary usage by metric name
	if _, exists := ut.metricsUsage[name]; !exists {
		ut.metricsUsage[name] = &MetricUsageInfo{
			MetricName:       name,
			SampleCount:      0,
			FirstSeen:        time.Now(),
			LastSeen:         time.Now(),
			Cardinality:      0,
			LabelCardinality: make(map[string]int),
			MinValue:         value,
			MaxValue:         value,
			SumValue:         0,
		}
	}

	info := ut.metricsUsage[name]
	info.SampleCount++
	info.LastSeen = time.Now()
	info.MinValue = min(info.MinValue, value)
	info.MaxValue = max(info.MaxValue, value)
	info.SumValue += value

	// Track detailed usage with label combinations
	labelHash := hashLabels(labels)
	if _, exists := ut.detailedUsage[name]; !exists {
		ut.detailedUsage[name] = make(map[string]*MetricUsageInfo)
	}

	if _, exists := ut.detailedUsage[name][labelHash]; !exists {
		info.Cardinality++
		ut.detailedUsage[name][labelHash] = &MetricUsageInfo{
			MetricName:  name,
			Labels:      copyLabels(labels),
			SampleCount: 0,
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
			MinValue:    value,
			MaxValue:    value,
			SumValue:    0,
		}

		// Update label cardinality only when we see a new unique combination
		for k, v := range labels {
			// Initialize tracking structures for this label if needed
			if _, exists := info.LabelCardinality[k]; !exists {
				info.LabelCardinality[k] = 0
			}

			// Check if this is a new value for this label
			isNewValue := true
			for existingHash, existingInfo := range ut.detailedUsage[name] {
				if existingHash != labelHash && existingInfo.Labels[k] == v {
					isNewValue = false
					break
				}
			}

			if isNewValue {
				info.LabelCardinality[k]++
			}
		}
	}

	detailedInfo := ut.detailedUsage[name][labelHash]
	detailedInfo.SampleCount++
	detailedInfo.LastSeen = time.Now()
	detailedInfo.MinValue = min(detailedInfo.MinValue, value)
	detailedInfo.MaxValue = max(detailedInfo.MaxValue, value)
	detailedInfo.SumValue += value

	// Periodically clean up old metrics
	if time.Since(ut.lastCleanup) > ut.retentionPeriod/10 {
		ut.cleanup()
	}
}

// GetMetricInfo returns usage information for a metric
func (ut *UsageTracker) GetMetricInfo(name string) *MetricUsageInfo {
	ut.mu.RLock()
	defer ut.mu.RUnlock()

	return ut.metricsUsage[name]
}

// GetAllMetricsInfo returns usage information for all metrics
func (ut *UsageTracker) GetAllMetricsInfo() map[string]*MetricUsageInfo {
	ut.mu.RLock()
	defer ut.mu.RUnlock()

	result := make(map[string]*MetricUsageInfo, len(ut.metricsUsage))
	for k, v := range ut.metricsUsage {
		result[k] = v
	}

	return result
}

// cleanup removes metrics that haven't been seen for the retention period
func (ut *UsageTracker) cleanup() {
	cutoff := time.Now().Add(-ut.retentionPeriod)
	ut.lastCleanup = time.Now()

	for metricName, metricInfo := range ut.metricsUsage {
		if metricInfo.LastSeen.Before(cutoff) {
			delete(ut.metricsUsage, metricName)
			delete(ut.detailedUsage, metricName)
			continue
		}

		// Clean up individual label combinations
		if details, exists := ut.detailedUsage[metricName]; exists {
			for labelHash, detailInfo := range details {
				if detailInfo.LastSeen.Before(cutoff) {
					delete(details, labelHash)
					metricInfo.Cardinality--
				}
			}
		}
	}
}

// helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func hashLabels(labels map[string]string) string {
	// Simple implementation - for production, use a more robust hashing mechanism
	result := ""
	for k, v := range labels {
		result += k + ":" + v + ";"
	}
	return result
}

func copyLabels(labels map[string]string) map[string]string {
	result := make(map[string]string, len(labels))
	for k, v := range labels {
		result[k] = v
	}
	return result
}
