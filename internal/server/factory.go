// Factory functions for creating server components
package server

import (
	"github.com/marcotuna/adaptive-metrics/internal/aggregator"
	"github.com/marcotuna/adaptive-metrics/internal/api"
	"github.com/marcotuna/adaptive-metrics/internal/config"
	"github.com/marcotuna/adaptive-metrics/internal/rules"
	"github.com/marcotuna/adaptive-metrics/internal/types"
)

// createMetricTracker creates a new MetricTracker (API handler) instance
func createMetricTracker(cfg *config.Config) (types.MetricTracker, error) {
	return api.NewHandler(cfg)
}

// createMetricProcessor creates a new MetricProcessor instance
func createMetricProcessor(cfg *config.Config, ruleEngine interface{}, tracker types.MetricTracker) (types.MetricProcessor, error) {
	// Convert the rule engine to its concrete type
	concreteRuleEngine, ok := ruleEngine.(*rules.Engine)
	if !ok {
		// Create a new rule engine if not the right type
		var err error
		concreteRuleEngine, err = rules.NewEngine(cfg)
		if err != nil {
			return nil, err
		}
	}
	// Create an adapter for the tracker to implement the aggregator's MetricTracker interface
	trackerAdapter := &metricTrackerAdapter{tracker: tracker}
	return aggregator.NewProcessor(cfg, concreteRuleEngine, trackerAdapter)
}

// metricTrackerAdapter adapts types.MetricTracker to the interface needed by aggregator
type metricTrackerAdapter struct {
	tracker types.MetricTracker
}

// TrackMetric delegates to the underlying tracker
func (a *metricTrackerAdapter) TrackMetric(name string, labels map[string]string, value float64) {
	a.tracker.TrackMetric(name, labels, value)
}
