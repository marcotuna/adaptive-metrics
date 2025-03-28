package metrics

import (
	"time"

	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// InputMetricsCounter counts the number of input metrics received
	InputMetricsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "adaptive_metrics_input_total",
			Help: "Total number of input metrics received",
		},
		[]string{"metric_name"},
	)

	// AggregatedMetricsCounter counts the number of aggregated metrics produced
	AggregatedMetricsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "adaptive_metrics_aggregated_total",
			Help: "Total number of aggregated metrics produced",
		},
		[]string{"metric_name", "rule_id"},
	)

	// DiscardedSamplesCounter counts the number of discarded raw samples
	DiscardedSamplesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "adaptive_metrics_discarded_samples_total",
			Help: "Total number of discarded raw samples",
		},
		[]string{"metric_name", "reason"},
	)

	// ProcessingDurationHistogram tracks the duration of metric processing
	ProcessingDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "adaptive_metrics_processing_duration_seconds",
			Help:    "Time taken to process metrics in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// RuleMatchingHistogram tracks the time taken for rule matching
	RuleMatchingHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "adaptive_metrics_rule_matching_duration_seconds",
			Help:    "Time taken to match rules to metrics in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"},
	)

	// ActiveRulesGauge tracks the number of active rules
	ActiveRulesGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "adaptive_metrics_active_rules",
			Help: "Number of active aggregation rules",
		},
	)

	// AggregationBucketsGauge tracks the number of active aggregation buckets
	AggregationBucketsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "adaptive_metrics_aggregation_buckets",
			Help: "Number of active aggregation buckets",
		},
	)
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(InputMetricsCounter)
	prometheus.MustRegister(AggregatedMetricsCounter)
	prometheus.MustRegister(DiscardedSamplesCounter)
	prometheus.MustRegister(ProcessingDurationHistogram)
	prometheus.MustRegister(RuleMatchingHistogram)
	prometheus.MustRegister(ActiveRulesGauge)
	prometheus.MustRegister(AggregationBucketsGauge)
}

// TrackDuration is a helper to measure and record the duration of operations
func TrackDuration(operation string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start).Seconds()
		ProcessingDurationHistogram.WithLabelValues(operation).Observe(duration)
	}
}

// RecordMetricReceived records that a metric was received
func RecordMetricReceived(sample *models.MetricSample) {
	InputMetricsCounter.WithLabelValues(sample.Name).Inc()
}

// RecordMetricAggregated records that a metric was aggregated
func RecordMetricAggregated(metric *models.AggregatedMetric) {
	AggregatedMetricsCounter.WithLabelValues(metric.Name, metric.SourceRule).Inc()
}

// RecordDiscardedSample records that a sample was discarded
func RecordDiscardedSample(metricName, reason string) {
	DiscardedSamplesCounter.WithLabelValues(metricName, reason).Inc()
}

// RecordRuleMatching records the duration of a rule matching operation
func RecordRuleMatching(duration time.Duration, matched bool) {
	result := "no_match"
	if matched {
		result = "matched"
	}
	RuleMatchingHistogram.WithLabelValues(result).Observe(duration.Seconds())
}

// UpdateActiveRulesCount updates the count of active rules
func UpdateActiveRulesCount(count int) {
	ActiveRulesGauge.Set(float64(count))
}

// UpdateAggregationBucketsCount updates the count of active aggregation buckets
func UpdateAggregationBucketsCount(count int) {
	AggregationBucketsGauge.Set(float64(count))
}