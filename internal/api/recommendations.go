package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/marcotuna/adaptive-metrics/internal/metrics"
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/marcotuna/adaptive-metrics/pkg/logger"
)

// RecommendationStore provides storage for recommendations
type RecommendationStore struct {
	mu              sync.RWMutex
	recommendations map[string]models.Recommendation
}

// NewRecommendationStore creates a new recommendation store
func NewRecommendationStore() *RecommendationStore {
	return &RecommendationStore{
		recommendations: make(map[string]models.Recommendation),
	}
}

// AddRecommendation adds a recommendation to the store
func (rs *RecommendationStore) AddRecommendation(rec models.Recommendation) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.recommendations[rec.ID] = rec
}

// GetRecommendation retrieves a recommendation by ID
func (rs *RecommendationStore) GetRecommendation(id string) (models.Recommendation, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	rec, exists := rs.recommendations[id]
	return rec, exists
}

// GetAllRecommendations retrieves all recommendations
func (rs *RecommendationStore) GetAllRecommendations() []models.Recommendation {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	recs := make([]models.Recommendation, 0, len(rs.recommendations))
	for _, rec := range rs.recommendations {
		recs = append(recs, rec)
	}
	return recs
}

// UpdateRecommendation updates an existing recommendation
func (rs *RecommendationStore) UpdateRecommendation(rec models.Recommendation) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, exists := rs.recommendations[rec.ID]; !exists {
		return false
	}

	rs.recommendations[rec.ID] = rec
	return true
}

// DeleteRecommendation removes a recommendation from the store
func (rs *RecommendationStore) DeleteRecommendation(id string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, exists := rs.recommendations[id]; !exists {
		return false
	}

	delete(rs.recommendations, id)
	return true
}

// RecommendationHandler handles recommendation-related endpoints
type RecommendationHandler struct {
	store                *RecommendationStore
	usageTracker         *metrics.UsageTracker
	recommendationEngine *metrics.RecommendationEngine
	ruleStore            RuleStore
	processor            ProcessorInterface // For registering recommendation rules
}

// ProcessorInterface defines the interface required for the processor
type ProcessorInterface interface {
	RegisterRecommendationRule(ruleID string)
}

// NewRecommendationHandler creates a new recommendation handler
func NewRecommendationHandler(store *RecommendationStore, usageTracker *metrics.UsageTracker,
	recommendationEngine *metrics.RecommendationEngine, ruleStore RuleStore) *RecommendationHandler {
	return &RecommendationHandler{
		store:                store,
		usageTracker:         usageTracker,
		recommendationEngine: recommendationEngine,
		ruleStore:            ruleStore,
	}
}

// SetProcessor sets the processor for the recommendation handler
func (h *RecommendationHandler) SetProcessor(processor ProcessorInterface) {
	h.processor = processor
}

// ListRecommendations returns all metric aggregation recommendations
func (h *RecommendationHandler) ListRecommendations(w http.ResponseWriter, r *http.Request) {
	recommendations := h.store.GetAllRecommendations()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": recommendations,
		"total":           len(recommendations),
	})
}

// GetRecommendation returns a specific recommendation by ID
func (h *RecommendationHandler) GetRecommendation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	recommendation, exists := h.store.GetRecommendation(id)
	if !exists {
		http.Error(w, "Recommendation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recommendation)
}

// ApplyRecommendation creates a rule from a recommendation
func (h *RecommendationHandler) ApplyRecommendation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	recommendation, exists := h.store.GetRecommendation(id)
	if !exists {
		http.Error(w, "Recommendation not found", http.StatusNotFound)
		return
	}

	// Update recommendation status
	recommendation.Status = "applied"
	h.store.UpdateRecommendation(recommendation)

	// Create rule from recommendation
	rule := recommendation.Rule
	rule.RecommendationID = recommendation.ID
	rule.Enabled = true // Enable the rule when applying a recommendation

	// Add the rule to the rule store
	err := h.ruleStore.AddRule(rule)
	if err != nil {
		http.Error(w, "Failed to create rule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Register rule as coming from a recommendation for remote write filtering
	if h.processor != nil {
		h.processor.RegisterRecommendationRule(rule.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"message":        "Recommendation applied successfully",
		"recommendation": recommendation,
		"rule":           rule,
	})
}

// RejectRecommendation marks a recommendation as rejected
func (h *RecommendationHandler) RejectRecommendation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	recommendation, exists := h.store.GetRecommendation(id)
	if !exists {
		http.Error(w, "Recommendation not found", http.StatusNotFound)
		return
	}

	// Update recommendation status
	recommendation.Status = "rejected"
	h.store.UpdateRecommendation(recommendation)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "success",
		"message":        "Recommendation rejected",
		"recommendation": recommendation,
	})
}

// GenerateRecommendations triggers the recommendation engine to generate new recommendations
func (h *RecommendationHandler) GenerateRecommendations(w http.ResponseWriter, r *http.Request) {
	// Generate recommendations using the engine
	recommendations := h.recommendationEngine.GenerateRecommendations()

	// Store the generated recommendations
	for _, rec := range recommendations {
		h.store.AddRecommendation(rec)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"message":         "Recommendation generation completed",
		"recommendations": recommendations,
		"total":           len(recommendations),
	})
}

// ListMetricsUsage returns usage information for all tracked metrics
func (h *RecommendationHandler) ListMetricsUsage(w http.ResponseWriter, r *http.Request) {
	// Get metrics usage information from the tracker
	metricsInfo := h.usageTracker.GetAllMetricsInfo()

	// Log the metrics info count for debugging
	infoCount := len(metricsInfo)

	// Convert to a slice for better JSON serialization
	metricsInfoSlice := make([]MetricUsageInfoResponse, 0, infoCount)
	for _, info := range metricsInfo {
		metricsInfoSlice = append(metricsInfoSlice, convertToMetricUsageInfoResponse(info))
	}

	// Include a debug message in the response when empty
	response := map[string]interface{}{
		"metrics": metricsInfoSlice,
		"total":   infoCount,
	}

	// Add debug information if no metrics are found
	if infoCount == 0 {
		response["debug_info"] = map[string]interface{}{
			"tracker_initialized": h.usageTracker != nil,
			"timestamp":           time.Now(),
			"message":             "No metrics found in usage tracker. This could indicate that metrics are not being properly tracked or that the tracker instance is not shared correctly.",
		}

		// Log this situation with proper structured logging
		logger.LogWarnWithFields("No metrics found in usage tracker", logger.Fields{
			"tracker_initialized": h.usageTracker != nil,
			"endpoint":            "ListMetricsUsage",
			"method":              r.Method,
			"path":                r.URL.Path,
		})
	} else {
		// Log successful metrics retrieval with count
		logger.LogDebugWithFields("Metrics usage data retrieved", logger.Fields{
			"count":    infoCount,
			"endpoint": "ListMetricsUsage",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetMetricUsage returns usage information for a specific metric
func (h *RecommendationHandler) GetMetricUsage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	metricInfo := h.usageTracker.GetMetricInfo(name)
	if metricInfo == nil {
		http.Error(w, "Metric not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convertToMetricUsageInfoResponse(metricInfo))
}

// MetricUsageInfoResponse is a serializable version of MetricUsageInfo
type MetricUsageInfoResponse struct {
	MetricName       string         `json:"metric_name"`
	SampleCount      int64          `json:"sample_count"`
	FirstSeen        time.Time      `json:"first_seen"`
	LastSeen         time.Time      `json:"last_seen"`
	Cardinality      int            `json:"cardinality"`
	LabelCardinality map[string]int `json:"label_cardinality"`
	MinValue         float64        `json:"min_value"`
	MaxValue         float64        `json:"max_value"`
	SumValue         float64        `json:"sum_value"`
	AvgValue         float64        `json:"avg_value"`
}

// Convert internal MetricUsageInfo to response format
func convertToMetricUsageInfoResponse(info *metrics.MetricUsageInfo) MetricUsageInfoResponse {
	avgValue := 0.0
	if info.SampleCount > 0 {
		avgValue = info.SumValue / float64(info.SampleCount)
	}

	return MetricUsageInfoResponse{
		MetricName:       info.MetricName,
		SampleCount:      info.SampleCount,
		FirstSeen:        info.FirstSeen,
		LastSeen:         info.LastSeen,
		Cardinality:      info.Cardinality,
		LabelCardinality: info.LabelCardinality,
		MinValue:         info.MinValue,
		MaxValue:         info.MaxValue,
		SumValue:         info.SumValue,
		AvgValue:         avgValue,
	}
}
