package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/marcotuna/adaptive-metrics/internal/aggregator"
	"github.com/marcotuna/adaptive-metrics/internal/config"
	"github.com/marcotuna/adaptive-metrics/internal/metrics"
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/marcotuna/adaptive-metrics/internal/rules"
	"github.com/marcotuna/adaptive-metrics/pkg/kubernetes"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RuleStore interface for rule storage operations
type RuleStore interface {
	AddRule(rule models.Rule) error
	GetRule(id string) (models.Rule, error)
	GetRules() ([]models.Rule, error)
	UpdateRule(rule models.Rule) error
	DeleteRule(id string) error
}

// Handler handles HTTP API requests
type Handler struct {
	cfg                  *config.Config
	ruleEngine           *rules.Engine
	usageTracker         *metrics.UsageTracker
	recommendationEngine *metrics.RecommendationEngine
	recommendationStore  *RecommendationStore
	recommendationHandler *RecommendationHandler
	processor            *aggregator.Processor
}

// NewHandler creates a new API handler
func NewHandler(cfg *config.Config) (*Handler, error) {
	ruleEngine, err := rules.NewEngine(cfg)
	if err != nil {
		return nil, err
	}

	// Create usage tracker (90 days retention)
	usageTracker := metrics.NewUsageTracker(90 * 24 * time.Hour)

	// Create recommendation engine
	recommendationEngine := metrics.NewRecommendationEngine(
		usageTracker,
		1000,       // Minimum sample threshold
		100,        // Minimum cardinality threshold
		0.5,        // Minimum confidence
	)

	// Create recommendation store
	recommendationStore := NewRecommendationStore()

	// Create the handler
	h := &Handler{
		cfg:                  cfg,
		ruleEngine:           ruleEngine,
		usageTracker:         usageTracker,
		recommendationEngine: recommendationEngine,
		recommendationStore:  recommendationStore,
	}

	// Create and set recommendation handler
	h.recommendationHandler = NewRecommendationHandler(
		recommendationStore,
		usageTracker,
		recommendationEngine,
		h.ruleEngine,
	)

	return h, nil
}

// SetProcessor sets the metric processor for the handler
func (h *Handler) SetProcessor(processor *aggregator.Processor) {
	h.processor = processor
	// Also set the processor for the recommendation handler
	if h.recommendationHandler != nil {
		h.recommendationHandler.SetProcessor(processor)
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Include remote write status in health check
	remoteWriteStatus := "disabled"
	if h.cfg.RemoteWrite.Enabled {
		remoteWriteStatus = "enabled"
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
		"remote_write": remoteWriteStatus,
	})
}

// Metrics exposes Prometheus metrics
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// ListRules returns all aggregation rules
func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.ruleEngine.GetRules()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// GetRule returns a specific rule by ID
func (h *Handler) GetRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rule, err := h.ruleEngine.GetRule(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// CreateRule creates a new aggregation rule
func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var rule models.Rule

	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the rule
	if err := rule.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set creation time
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	// Save the rule
	if err := h.ruleEngine.SaveRule(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// UpdateRule updates an existing rule
func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var rule models.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure ID matches
	rule.ID = id

	// Validate the rule
	if err := rule.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update timestamp
	rule.UpdatedAt = time.Now()

	// Update the rule
	if err := h.ruleEngine.UpdateRule(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// DeleteRule deletes a rule
func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.ruleEngine.DeleteRule(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TrackMetric tracks a metric for usage analysis
func (h *Handler) TrackMetric(name string, labels map[string]string, value float64) {
	h.usageTracker.TrackMetric(name, labels, value)
}

// GetRuleEngine returns the rule engine instance
func (h *Handler) GetRuleEngine() *rules.Engine {
	return h.ruleEngine
}

// SetupRecommendationRoutes sets up the routes for the recommendation API
func (h *Handler) SetupRecommendationRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/recommendations", h.recommendationHandler.ListRecommendations).Methods("GET")
	router.HandleFunc("/api/v1/recommendations/{id}", h.recommendationHandler.GetRecommendation).Methods("GET")
	router.HandleFunc("/api/v1/recommendations/{id}/apply", h.recommendationHandler.ApplyRecommendation).Methods("POST")
	router.HandleFunc("/api/v1/recommendations/{id}/reject", h.recommendationHandler.RejectRecommendation).Methods("POST")
	router.HandleFunc("/api/v1/recommendations/generate", h.recommendationHandler.GenerateRecommendations).Methods("POST")
}

// KubernetesMonitor generates Kubernetes monitoring resources
func (h *Handler) KubernetesMonitor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rule, err := h.ruleEngine.GetRule(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check if the rule has Kubernetes output configured
	if rule.OutputKubernetes == nil || !rule.OutputKubernetes.Enabled {
		http.Error(w, "Rule does not have Kubernetes output configured", http.StatusBadRequest)
		return
	}

	// Generate the monitor resource
	monitorYAML, err := kubernetes.RenderMonitor(rule)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate Kubernetes monitor: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the generated YAML
	w.Header().Set("Content-Type", "application/yaml")
	w.Write([]byte(monitorYAML))
}

// SaveKubernetesMonitor generates and saves a Kubernetes monitoring resource to disk
func (h *Handler) SaveKubernetesMonitor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rule, err := h.ruleEngine.GetRule(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check if the rule has Kubernetes output configured
	if rule.OutputKubernetes == nil || !rule.OutputKubernetes.Enabled {
		http.Error(w, "Rule does not have Kubernetes output configured", http.StatusBadRequest)
		return
	}

	// Parse the output directory from the request
	var requestData struct {
		OutputDir string `json:"output_dir"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use default directory if not specified
	outputDir := requestData.OutputDir
	if outputDir == "" {
		outputDir = "kubernetes/monitors"
	}

	// Generate and save the monitor file
	filePath, err := kubernetes.WriteMonitorFile(rule, outputDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save Kubernetes monitor: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "Kubernetes monitor generated successfully",
		"file_path": filePath,
	})
}