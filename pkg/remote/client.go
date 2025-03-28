package remote

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/marcotuna/adaptive-metrics/internal/config"
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/prometheus/prometheus/prompb"
)

// Client is a Prometheus remote write client
type Client struct {
	cfg           *config.RemoteWriteConfig
	httpClient    *http.Client
	endpoints     []string
	headers       map[string]string
	basicAuth     *BasicAuth
	queue         chan *models.AggregatedMetric
	done          chan struct{}
	wg            sync.WaitGroup
	// Track which metrics came from recommendations
	recommendationMetrics map[string]bool
	recommendationMu      sync.RWMutex
}

// BasicAuth contains basic authentication credentials
type BasicAuth struct {
	Username string
	Password string
}

// NewClient creates a new remote write client
func NewClient(cfg *config.RemoteWriteConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("remote write config cannot be nil")
	}

	if !cfg.Enabled {
		return nil, fmt.Errorf("remote write is not enabled")
	}

	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one remote write endpoint must be configured")
	}

	var basicAuth *BasicAuth
	if cfg.Username != "" {
		basicAuth = &BasicAuth{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	client := &Client{
		cfg:                  cfg,
		endpoints:            cfg.Endpoints,
		headers:              cfg.Headers,
		basicAuth:            basicAuth,
		queue:                make(chan *models.AggregatedMetric, cfg.BatchSize),
		done:                 make(chan struct{}),
		recommendationMetrics: make(map[string]bool),
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}

	return client, nil
}

// Start starts the remote write client
func (c *Client) Start() {
	c.wg.Add(1)
	go c.worker()
}

// Stop stops the remote write client
func (c *Client) Stop() {
	close(c.done)
	c.wg.Wait()
}

// Write queues a metric for remote write
func (c *Client) Write(metric *models.AggregatedMetric) {
	// If recommendation_metrics_only is set to true, only write metrics from recommendations
	if c.cfg.RecommendationMetricsOnly {
		c.recommendationMu.RLock()
		_, isFromRecommendation := c.recommendationMetrics[metric.SourceRule]
		c.recommendationMu.RUnlock()
		
		if !isFromRecommendation {
			return
		}
	}

	select {
	case c.queue <- metric:
		// Successfully queued
	default:
		// Queue is full, log and drop
		fmt.Printf("Warning: Remote write queue is full, dropping metric: %s\n", metric.Name)
	}
}

// RegisterRecommendationRule registers a rule as coming from a recommendation
func (c *Client) RegisterRecommendationRule(ruleID string) {
	c.recommendationMu.Lock()
	defer c.recommendationMu.Unlock()
	c.recommendationMetrics[ruleID] = true
}

// worker processes the queue and sends metrics to remote endpoints
func (c *Client) worker() {
	defer c.wg.Done()

	batch := make([]*models.AggregatedMetric, 0, c.cfg.BatchSize)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			// Flush any remaining metrics before exiting
			if len(batch) > 0 {
				c.sendBatch(batch)
			}
			return
		case metric := <-c.queue:
			batch = append(batch, metric)
			// Send immediately if batch is full
			if len(batch) >= c.cfg.BatchSize {
				c.sendBatch(batch)
				batch = make([]*models.AggregatedMetric, 0, c.cfg.BatchSize)
			}
		case <-ticker.C:
			// Send periodically even if batch is not full
			if len(batch) > 0 {
				c.sendBatch(batch)
				batch = make([]*models.AggregatedMetric, 0, c.cfg.BatchSize)
			}
		}
	}
}

// sendBatch sends a batch of metrics to all configured remote write endpoints
func (c *Client) sendBatch(metrics []*models.AggregatedMetric) {
	if len(metrics) == 0 {
		return
	}

	// Convert to Prometheus write request
	req := c.buildWriteRequest(metrics)

	// Serialize and compress
	data, err := proto.Marshal(req)
	if err != nil {
		fmt.Printf("Error marshaling write request: %v\n", err)
		return
	}

	compressed := snappy.Encode(nil, data)

	// Send to all endpoints
	for _, endpoint := range c.endpoints {
		for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
			if err := c.sendToEndpoint(endpoint, compressed); err != nil {
				fmt.Printf("Error sending to endpoint %s (attempt %d/%d): %v\n", 
					endpoint, attempt+1, c.cfg.MaxRetries+1, err)
				
				if attempt < c.cfg.MaxRetries {
					// Wait before retrying
					time.Sleep(time.Duration(c.cfg.RetryInterval) * time.Second)
					continue
				}
			} else {
				// Success
				break
			}
		}
	}
}

// sendToEndpoint sends compressed data to a specific endpoint
func (c *Client) sendToEndpoint(endpoint string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.cfg.Timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	// Add custom headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Add basic auth if configured
	if c.basicAuth != nil {
		req.SetBasicAuth(c.basicAuth.Username, c.basicAuth.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("non-200 status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// buildWriteRequest converts aggregated metrics to a Prometheus write request
func (c *Client) buildWriteRequest(metrics []*models.AggregatedMetric) *prompb.WriteRequest {
	request := &prompb.WriteRequest{
		Timeseries: make([]prompb.TimeSeries, 0, len(metrics)),
	}

	for _, metric := range metrics {
		// Create labels including the metric name
		labels := make([]prompb.Label, 0, len(metric.Labels)+1)
		
		// Add the __name__ label
		labels = append(labels, prompb.Label{
			Name:  "__name__",
			Value: metric.Name,
		})

		// Add all other labels
		for k, v := range metric.Labels {
			labels = append(labels, prompb.Label{
				Name:  k,
				Value: v,
			})
		}

		// Create a sample
		sample := prompb.Sample{
			Value:     metric.Value,
			Timestamp: metric.EndTime.UnixNano() / int64(time.Millisecond),
		}

		// Add to timeseries
		ts := prompb.TimeSeries{
			Labels:  labels,
			Samples: []prompb.Sample{sample},
		}

		request.Timeseries = append(request.Timeseries, ts)
	}

	return request
}