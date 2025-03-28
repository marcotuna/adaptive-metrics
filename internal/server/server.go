// Server implementation for the adaptive metrics service
package server

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/marcotuna/adaptive-metrics/internal/api"
	"github.com/marcotuna/adaptive-metrics/internal/config"
	"github.com/marcotuna/adaptive-metrics/internal/types"
)

// FileServer is a convenient wrapper for http.FileServer
type FileServer struct {
	root    http.FileSystem
	indexes bool
}

// NewFileServer creates a new file server handler that can optionally fallback to index.html
func NewFileServer(root http.FileSystem, indexes bool) http.Handler {
	return &FileServer{root: root, indexes: indexes}
}

// ServeHTTP serves static files and falls back to index.html for SPA routing
func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Serve the file directly if it exists
	f, err := fs.root.Open(r.URL.Path)
	if err == nil {
		f.Close()
		http.FileServer(fs.root).ServeHTTP(w, r)
		return
	}

	// For SPA routing, serve index.html for non-existing paths
	if fs.indexes {
		index, err := fs.root.Open("index.html")
		if err == nil {
			index.Close()
			r.URL.Path = "index.html"
			http.FileServer(fs.root).ServeHTTP(w, r)
			return
		}
	}

	// If we got here, return 404
	http.NotFound(w, r)
}

// Server represents the adaptive metrics server
type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	router     *mux.Router
	apiHandler types.MetricTracker
	processor  types.MetricProcessor
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	router := mux.NewRouter()
	// Create API handler using our factory
	apiHandler, err := createMetricTracker(cfg)
	if err != nil {
		return nil, err
	}
	// Create processor using our factory
	processor, err := createMetricProcessor(cfg, apiHandler.GetRuleEngine(), apiHandler)
	if err != nil {
		return nil, err
	}
	// Connect the processor to the API handler
	apiHandler.SetProcessor(processor)

	// Construct the address with the configured port
	address := cfg.Server.Address
	// If Address doesn't contain a port (like ":8080") but we have a port set,
	// use the configured port
	if cfg.Server.Port != 0 && address == "" {
		address = fmt.Sprintf(":%d", cfg.Server.Port)
	} else if cfg.Server.Port != 0 && address != "" && !strings.Contains(address, ":") {
		// If Address is set (like "localhost") but doesn't include a port,
		// append the configured port
		address = fmt.Sprintf("%s:%d", address, cfg.Server.Port)
	}

	srv := &Server{
		cfg:        cfg,
		router:     router,
		apiHandler: apiHandler,
		processor:  processor,
		httpServer: &http.Server{
			Addr:         address,
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
	// Apply CORS middleware to all routes
	s.router.Use(api.CORSMiddleware)

	// API endpoints - match Grafana's API structure
	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()
	// Rules management
	apiRouter.HandleFunc("/rules", s.apiHandler.ListRules).Methods(http.MethodGet, http.MethodOptions)
	apiRouter.HandleFunc("/rules", s.apiHandler.CreateRule).Methods(http.MethodPost, http.MethodOptions)
	apiRouter.HandleFunc("/rules/{id}", s.apiHandler.GetRule).Methods(http.MethodGet, http.MethodOptions)
	apiRouter.HandleFunc("/rules/{id}", s.apiHandler.UpdateRule).Methods(http.MethodPut, http.MethodOptions)
	apiRouter.HandleFunc("/rules/{id}", s.apiHandler.DeleteRule).Methods(http.MethodDelete, http.MethodOptions)
	// Kubernetes monitor generation for rules
	apiRouter.HandleFunc("/rules/{id}/kubernetes-monitor", s.apiHandler.KubernetesMonitor).Methods(http.MethodGet, http.MethodOptions)
	apiRouter.HandleFunc("/rules/{id}/kubernetes-monitor", s.apiHandler.SaveKubernetesMonitor).Methods(http.MethodPost, http.MethodOptions)
	// Setup recommendation routes using the new handler
	s.apiHandler.SetupRecommendationRoutes(apiRouter)
	// Prometheus remote_write endpoint
	s.router.HandleFunc("/api/v1/write", s.apiHandler.PrometheusRemoteWrite).Methods(http.MethodPost, http.MethodOptions)
	// Metrics operations
	apiRouter.HandleFunc("/metrics/analyze", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"not implemented"}`))
	}).Methods(http.MethodPost, http.MethodOptions)
	// Plugin integration endpoints
	apiRouter.HandleFunc("/plugin/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"enabled":false}`))
	}).Methods(http.MethodGet, http.MethodOptions)

	// Health and metrics
	s.router.HandleFunc("/health", s.apiHandler.HealthCheck).Methods(http.MethodGet, http.MethodOptions)
	s.router.HandleFunc("/metrics", s.apiHandler.Metrics).Methods(http.MethodGet, http.MethodOptions)

	// Add a custom 404 handler for API routes
	s.router.PathPrefix("/api").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Not Found","status":404}`))
	})

	// File server for serving the web UI - only use index fallback for frontend routes
	webUIPath := filepath.Join(s.cfg.Server.WebUIPath)
	fileServer := NewFileServer(http.Dir(webUIPath), true)
	s.router.PathPrefix("/").Handler(fileServer)
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
