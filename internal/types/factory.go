package types

import (
	"github.com/marcotuna/adaptive-metrics/internal/config"
)

// NewMetricTracker creates a new MetricTracker instance
func NewMetricTracker(cfg *config.Config) (MetricTracker, error) {
	// This will be implemented by a factory function that imports and uses api.NewHandler
	// The implementation will be in a separate file to avoid import cycles
	return nil, nil
}

// NewMetricProcessor creates a new MetricProcessor instance
func NewMetricProcessor(cfg *config.Config, ruleEngine interface{}, tracker MetricTracker) (MetricProcessor, error) {
	// This will be implemented by a factory function that imports and uses aggregator.NewProcessor
	// The implementation will be in a separate file to avoid import cycles
	return nil, nil
}
