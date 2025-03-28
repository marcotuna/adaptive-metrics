package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/prometheus/prometheus/prompb"
)

// PrometheusRemoteWrite handles incoming remote write requests from Prometheus
func (h *Handler) PrometheusRemoteWrite(w http.ResponseWriter, r *http.Request) {
	compressed, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reqBuf, err := snappy.Decode(nil, compressed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req prompb.WriteRequest
	if err := proto.Unmarshal(reqBuf, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process the timeseries data
	processedCount := 0
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
			continue
		}
		
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
	
	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Remote write processed successfully",
		"metrics_processed": processedCount,
	})
}