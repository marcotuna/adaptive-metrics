package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/marcotuna/adaptive-metrics/pkg/logger"
	"github.com/prometheus/prometheus/prompb"
)

// generateRequestID creates a unique identifier for tracking requests in logs
func generateRequestID() string {
	return time.Now().Format("20060102-150405") + "-" + fmt.Sprintf("%06d", time.Now().Nanosecond()/1000)
}

// PrometheusRemoteWrite handles incoming remote write requests from Prometheus
func (h *Handler) PrometheusRemoteWrite(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	remoteAddr := r.RemoteAddr
	logger.LogDebugWithFields("Received remote write request", logger.Fields{
		"request_id":     requestID,
		"remote_addr":    remoteAddr,
		"content_length": r.ContentLength,
	})

	startTime := time.Now()
	compressed, err := io.ReadAll(r.Body)
	if err != nil {
		logger.LogErrorWithFields("Failed to read request body", logger.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.LogDebugWithFields("Read compressed data from request body", logger.Fields{
		"request_id":       requestID,
		"compressed_bytes": len(compressed),
	})

	reqBuf, err := snappy.Decode(nil, compressed)
	if err != nil {
		logger.LogErrorWithFields("Failed to decode Snappy-compressed data", logger.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.LogDebugWithFields("Decompressed request data", logger.Fields{
		"request_id":         requestID,
		"decompressed_bytes": len(reqBuf),
	})

	var req prompb.WriteRequest
	if err := proto.Unmarshal(reqBuf, &req); err != nil {
		logger.LogErrorWithFields("Failed to unmarshal Prometheus write request", logger.Fields{
			"request_id": requestID,
			"error":      err.Error(),
		})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	timeseriesCount := len(req.Timeseries)
	logger.LogDebugWithFields("Unmarshalled Prometheus write request", logger.Fields{
		"request_id":       requestID,
		"timeseries_count": timeseriesCount,
	})

	// Process the timeseries data
	processedCount := 0
	sampleCount := 0
	metricNamesMap := make(map[string]bool)

	for _, ts := range req.Timeseries {
		metricName := ""
		labels := make(map[string]string)

		// Extract metric name and labels
		for _, l := range ts.Labels {
			if l.Name == "__name__" {
				metricName = l.Value
			} else {
				labels[l.Name] = l.Value
			}
		}

		// Skip if no metric name
		if metricName == "" {
			logger.LogDebugWithFields("Skipping timeseries without a metric name", logger.Fields{
				"request_id": requestID,
			})
			continue
		}

		metricNamesMap[metricName] = true
		sampleCount += len(ts.Samples)

		// Process each sample
		for _, s := range ts.Samples {
			// Convert to our internal metric sample format
			sample := &models.MetricSample{
				Name:      metricName,
				Value:     s.Value,
				Timestamp: time.Unix(0, s.Timestamp*int64(time.Millisecond)),
				Labels:    labels,
			}

			// Track metric usage for recommendation engine
			h.TrackMetric(sample.Name, sample.Labels, sample.Value)

			// Process the metric through the aggregation engine
			// This assumes we have a reference to the processor
			if h.processor != nil {
				h.processor.ProcessMetric(sample)
			}

			processedCount++
		}
	}

	processingDuration := time.Since(startTime)
	uniqueMetricsCount := len(metricNamesMap)

	logger.LogInfoWithFields("Processed remote write request", logger.Fields{
		"request_id":          requestID,
		"timeseries_count":    timeseriesCount,
		"unique_metrics":      uniqueMetricsCount,
		"samples_count":       sampleCount,
		"processed_count":     processedCount,
		"processing_duration": processingDuration.String(),
		"processing_ms":       processingDuration.Milliseconds(),
	})

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "success",
		"message":           "Remote write processed successfully",
		"metrics_processed": processedCount,
	})
}
