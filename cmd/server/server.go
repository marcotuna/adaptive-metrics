package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/marcotuna/adaptive-metrics/internal/aggregator"
	"github.com/marcotuna/adaptive-metrics/internal/api"
	"github.com/marcotuna/adaptive-metrics/internal/config"
)

// Server represents the adaptive metrics server
type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	router     *mux.Router
	apiHandler *api.Handler
	processor  *aggregator.Processor
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	router := mux.NewRouter()

	apiHandler, err := api.NewHandler(cfg)
	if err != nil {
		return nil, err
	}

	// Create the processor with reference to the API handler for usage tracking
	processor, err := aggregator.NewProcessor(cfg, apiHandler.GetRuleEngine(), apiHandler)
	if err != nil {
		return nil, err
	}

	// Set the processor in the API handler for use in remote write endpoint
	apiHandler.SetProcessor(processor)

	srv := &Server{
		cfg:        cfg,
		router:     router,
		apiHandler: apiHandler,
		processor:  processor,
		httpServer: &http.Server{
			Addr:         cfg.Server.Address,
			Handler:      router,
			ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
			WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
		},
	}

	srv.setupRoutes()
	return srv, nil
}

// setupRoutes configures the server routes
func (s *Server) setupRoutes() {
	// API endpoints - match Grafana's API structure
	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()

	// Rules management
	apiRouter.HandleFunc("/rules", s.apiHandler.ListRules).Methods(http.MethodGet)
	apiRouter.HandleFunc("/rules", s.apiHandler.CreateRule).Methods(http.MethodPost)
	apiRouter.HandleFunc("/rules/{id}", s.apiHandler.GetRule).Methods(http.MethodGet)
	apiRouter.HandleFunc("/rules/{id}", s.apiHandler.UpdateRule).Methods(http.MethodPut)
	apiRouter.HandleFunc("/rules/{id}", s.apiHandler.DeleteRule).Methods(http.MethodDelete)

	// Kubernetes monitor generation for rules
	apiRouter.HandleFunc("/rules/{id}/kubernetes-monitor", s.apiHandler.KubernetesMonitor).Methods(http.MethodGet)
	apiRouter.HandleFunc("/rules/{id}/kubernetes-monitor", s.apiHandler.SaveKubernetesMonitor).Methods(http.MethodPost)

	// Setup recommendation routes using the new handler
	s.apiHandler.SetupRecommendationRoutes(apiRouter)

	// Prometheus remote_write endpoint
	s.router.HandleFunc("/api/v1/write", s.apiHandler.PrometheusRemoteWrite).Methods(http.MethodPost)

	// Metrics operations
	apiRouter.HandleFunc("/metrics/analyze", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"not implemented"}`))
	}).Methods(http.MethodPost)

	// Plugin integration endpoints
	apiRouter.HandleFunc("/plugin/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"enabled":false}`))
	}).Methods(http.MethodGet)

	// Health and metrics
	s.router.HandleFunc("/health", s.apiHandler.HealthCheck).Methods(http.MethodGet)
	s.router.HandleFunc("/metrics", s.apiHandler.Metrics).Methods(http.MethodGet)
}

// Start starts the server and processors
func (s *Server) Start() error {
	// Start the metric processor
	s.processor.Start()

	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	// Stop the processor first
	s.processor.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}