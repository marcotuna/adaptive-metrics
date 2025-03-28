package api

import (
	"encoding/json"
	"net/http"
	"time"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/marcotuna/adaptive-metrics/internal/metrics"
	"github.com/marcotuna/adaptive-metrics/internal/models"
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
	store              *RecommendationStore
	usageTracker       *metrics.UsageTracker
	recommendationEngine *metrics.RecommendationEngine
	ruleStore          RuleStore
	processor          ProcessorInterface // For registering recommendation rules
}

// ProcessorInterface defines the interface required for the processor
type ProcessorInterface interface {
	RegisterRecommendationRule(ruleID string)
}

// NewRecommendationHandler creates a new recommendation handler
func NewRecommendationHandler(store *RecommendationStore, usageTracker *metrics.UsageTracker, 
	recommendationEngine *metrics.RecommendationEngine, ruleStore RuleStore) *RecommendationHandler {
	return &RecommendationHandler{
		store:               store,
		usageTracker:        usageTracker,
		recommendationEngine: recommendationEngine,
		ruleStore:           ruleStore,
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
	rule.Enabled = true  // Enable the rule when applying a recommendation
	
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
		"status":          "success",
		"message":         "Recommendation applied successfully",
		"recommendation":  recommendation,
		"rule":            rule,
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
		"status":          "success",
		"message":         "Recommendation rejected",
		"recommendation":  recommendation,
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