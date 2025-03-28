package aggregator

import (
	"fmt"
	"sync"
	"time"

	"github.com/marcotuna/adaptive-metrics/internal/api"
	"github.com/marcotuna/adaptive-metrics/internal/config"
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/marcotuna/adaptive-metrics/internal/rules"
	"github.com/marcotuna/adaptive-metrics/pkg/remote"
)

// Processor handles metric aggregation based on rules
type Processor struct {
	cfg          *config.Config
	ruleEngine   *rules.Engine
	buckets      map[string]*aggregationBucket
	bucketMu     sync.RWMutex
	inputCh      chan *models.MetricSample
	outputCh     chan *models.AggregatedMetric
	workerWg     sync.WaitGroup
	stopCh       chan struct{}
	apiHandler   *api.Handler // Reference to API handler for usage tracking
	remoteWriter *remote.Client // Remote write client
}

// aggregationBucket represents a collection of metrics being aggregated
type aggregationBucket struct {
	rule      *models.Rule
	metrics   map[string][]*models.MetricSample // key is the segmentation key
	startTime time.Time
	endTime   time.Time
}

// NewProcessor creates a new metrics aggregation processor
func NewProcessor(cfg *config.Config, ruleEngine *rules.Engine, apiHandler *api.Handler) (*Processor, error) {
	processor := &Processor{
		cfg:        cfg,
		ruleEngine: ruleEngine,
		buckets:    make(map[string]*aggregationBucket),
		inputCh:    make(chan *models.MetricSample, cfg.Aggregator.BatchSize),
		outputCh:   make(chan *models.AggregatedMetric, cfg.Aggregator.BatchSize),
		stopCh:     make(chan struct{}),
		apiHandler: apiHandler,
	}

	// Initialize remote write client if enabled
	if cfg.RemoteWrite.Enabled && len(cfg.RemoteWrite.Endpoints) > 0 {
		var err error
		processor.remoteWriter, err = remote.NewClient(&cfg.RemoteWrite)
		if err != nil {
			fmt.Printf("Warning: Failed to initialize remote write client: %v\n", err)
			// Continue without remote write
		}
	}

	return processor, nil
}

// Start starts the aggregation processor
func (p *Processor) Start() {
	// Start the remote write client if configured
	if p.remoteWriter != nil {
		p.remoteWriter.Start()
	}

	// Start worker goroutines
	for i := 0; i < p.cfg.Aggregator.WorkerCount; i++ {
		p.workerWg.Add(1)
		go p.worker()
	}
	// Start aggregator goroutine
	go p.aggregator()
}

// Stop stops the aggregation processor
func (p *Processor) Stop() {
	close(p.stopCh)
	p.workerWg.Wait()

	// Stop the remote write client if configured
	if p.remoteWriter != nil {
		p.remoteWriter.Stop()
	}
}

// ProcessMetric submits a metric for processing
func (p *Processor) ProcessMetric(sample *models.MetricSample) {
	// Track the metric's usage before processing
	if p.apiHandler != nil {
		p.apiHandler.TrackMetric(sample.Name, sample.Labels, sample.Value)
	}

	select {
	case p.inputCh <- sample:
		// Metric submitted successfully
	default:
		// Channel is full, log and drop
		fmt.Printf("Warning: Input channel full, dropping metric: %s\n", sample.Name)
	}
}

// RegisterRecommendationRule registers a rule as coming from a recommendation with the remote write client
func (p *Processor) RegisterRecommendationRule(ruleID string) {
	if p.remoteWriter != nil {
		p.remoteWriter.RegisterRecommendationRule(ruleID)
	}
}

// GetOutputChannel returns the channel for aggregated metrics
func (p *Processor) GetOutputChannel() <-chan *models.AggregatedMetric {
	return p.outputCh
}

// worker processes incoming metrics
func (p *Processor) worker() {
	defer p.workerWg.Done()
	for {
		select {
		case <-p.stopCh:
			return
		case sample := <-p.inputCh:
			p.processSample(sample)
		}
	}
}

// processSample processes a single metric sample
func (p *Processor) processSample(sample *models.MetricSample) {
	// Find matching rules
	matchingRules := p.ruleEngine.Matcher.MatchingRules(sample)
	for _, rule := range matchingRules {
		// Create bucket key from rule ID and interval
		bucketKey := fmt.Sprintf("%s-%d", rule.ID, rule.Aggregation.IntervalSeconds)
		// Get current interval
		intervalSeconds := rule.Aggregation.IntervalSeconds
		interval := time.Duration(intervalSeconds) * time.Second

		// Calculate bucket boundaries
		now := time.Now()
		bucketStart := now.Truncate(interval)
		bucketEnd := bucketStart.Add(interval)
		// Add to appropriate bucket
		p.bucketMu.Lock()
		bucket, exists := p.buckets[bucketKey]
		if !exists || bucket.endTime.Before(now) {
			// Create new bucket if it doesn't exist or the existing one is expired
			bucket = &aggregationBucket{
				rule:      rule,
				metrics:   make(map[string][]*models.MetricSample),
				startTime: bucketStart,
				endTime:   bucketEnd,
			}
			p.buckets[bucketKey] = bucket
		}
		// Generate segmentation key from sample labels
		segmentKey := p.generateSegmentKey(sample, rule.Aggregation.Segmentation)
		// Add the sample to the bucket
		bucket.metrics[segmentKey] = append(bucket.metrics[segmentKey], sample)

		p.bucketMu.Unlock()
	}
}

// generateSegmentKey creates a key for segmenting metrics during aggregation
func (p *Processor) generateSegmentKey(sample *models.MetricSample, segmentBy []string) string {
	if len(segmentBy) == 0 {
		return "_all_" // No segmentation
	}
	var keyParts []string
	for _, label := range segmentBy {
		if value, exists := sample.Labels[label]; exists {
			keyParts = append(keyParts, fmt.Sprintf("%s=%s", label, value))
		} else {
			keyParts = append(keyParts, fmt.Sprintf("%s=", label))
		}
	}
	return fmt.Sprintf("%s", keyParts)
}

// aggregator periodically aggregates metrics in buckets
func (p *Processor) aggregator() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.aggregateBuckets()
		}
	}
}

// aggregateBuckets aggregates metrics in completed buckets
func (p *Processor) aggregateBuckets() {
	now := time.Now()

	// Calculate the delay for aggregation
	delayDuration := time.Duration(p.cfg.Aggregator.AggregationDelayMs) * time.Millisecond
	p.bucketMu.Lock()
	defer p.bucketMu.Unlock()
	// Check for buckets that are ready for aggregation
	for key, bucket := range p.buckets {
		// Skip if not yet expired or not past the delay
		if now.Before(bucket.endTime.Add(delayDuration)) {
			continue
		}
		// Process each segment in the bucket
		for segmentKey, samples := range bucket.metrics {
			if len(samples) == 0 {
				continue
			}
			// Aggregate the samples
			aggValue := p.aggregateSamples(samples, bucket.rule.Aggregation.Type)

			// Create labels map from segmentation key
			labels := p.parseSegmentKey(segmentKey)

			// Add any additional labels from the rule
			for k, v := range bucket.rule.Output.AdditionalLabels {
				labels[k] = v
			}
			// Create aggregated metric
			aggMetric := &models.AggregatedMetric{
				Name:       bucket.rule.Output.MetricName,
				Value:      aggValue,
				StartTime:  bucket.startTime,
				EndTime:    bucket.endTime,
				Labels:     labels,
				SourceRule: bucket.rule.ID,
				Count:      len(samples),
			}

			// Also track the aggregated metric for usage patterns
			if p.apiHandler != nil {
				p.apiHandler.TrackMetric(aggMetric.Name, aggMetric.Labels, aggMetric.Value)
			}

			// Send to remote write if enabled
			if p.remoteWriter != nil {
				p.remoteWriter.Write(aggMetric)
			}

			// Send to output channel
			select {
			case p.outputCh <- aggMetric:
				// Sent successfully
			default:
				// Channel full, log and drop
				fmt.Printf("Warning: Output channel full, dropping aggregated metric: %s\n", aggMetric.Name)
			}
		}
		// Remove the processed bucket
		delete(p.buckets, key)
	}
}

// aggregateSamples aggregates metric samples based on the specified type
func (p *Processor) aggregateSamples(samples []*models.MetricSample, aggType string) float64 {
	if len(samples) == 0 {
		return 0
	}
	switch aggType {
	case "sum":
		var sum float64
		for _, sample := range samples {
			sum += sample.Value
		}
		return sum
	case "avg":
		var sum float64
		for _, sample := range samples {
			sum += sample.Value
		}
		return sum / float64(len(samples))
	case "min":
		min := samples[0].Value
		for _, sample := range samples {
			if sample.Value < min {
				min = sample.Value
			}
		}
		return min
	case "max":
		max := samples[0].Value
		for _, sample := range samples {
			if sample.Value > max {
				max = sample.Value
			}
		}
		return max
	case "count":
		return float64(len(samples))
	default:
		// Default to sum if unrecognized
		var sum float64
		for _, sample := range samples {
			sum += sample.Value
		}
		return sum
	}
}

// parseSegmentKey parses a segment key back into a labels map
func (p *Processor) parseSegmentKey(segmentKey string) map[string]string {
	if segmentKey == "_all_" {
		return make(map[string]string)
	}
	// Placeholder implementation - in practice would need to parse the segment key format
	// This is simplified for demonstration
	labels := make(map[string]string)
	// Parse segment key format "key1=value1,key2=value2"
	// Actual implementation would depend on the format of segmentKey

	return labels
}