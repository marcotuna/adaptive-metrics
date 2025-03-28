package core

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/marcotuna/adaptive-metrics/internal/rules"
)

// MetricProcessor defines the interface for processing metrics
type MetricProcessor interface {
	Start()
	Stop()
	ProcessMetric(sample *models.MetricSample)
}

// MetricTracker defines the interface for tracking metrics and API operations
type MetricTracker interface {
	// Metric tracking
	TrackMetric(name string, labels map[string]string, value float64)

	// Rule management
	GetRuleEngine() *rules.Engine
	ListRules(w http.ResponseWriter, r *http.Request)
	CreateRule(w http.ResponseWriter, r *http.Request)
	GetRule(w http.ResponseWriter, r *http.Request)
	UpdateRule(w http.ResponseWriter, r *http.Request)
	DeleteRule(w http.ResponseWriter, r *http.Request)

	// Health and metrics
	HealthCheck(w http.ResponseWriter, r *http.Request)
	Metrics(w http.ResponseWriter, r *http.Request)

	// Kubernetes monitors
	KubernetesMonitor(w http.ResponseWriter, r *http.Request)
	SaveKubernetesMonitor(w http.ResponseWriter, r *http.Request)

	// Remote write
	PrometheusRemoteWrite(w http.ResponseWriter, r *http.Request)

	// Recommendations
	SetupRecommendationRoutes(router *mux.Router)

	// Processor management
	SetProcessor(processor MetricProcessor)
}
