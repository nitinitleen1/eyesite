// Package main implements the telemetry ingestion service
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/eyesite/platform/backend/internal/errors"
	"github.com/eyesite/platform/backend/pkg/config"
	"github.com/eyesite/platform/backend/pkg/database"
	"github.com/eyesite/platform/backend/pkg/models"
	"github.com/eyesite/platform/backend/pkg/providers"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	// Override port for ingest service
	cfg.Server.Port = 8001

	logger.Info("starting telemetry ingestion service",
		zap.String("environment", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
	)

	// Initialize database
	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Initialize provider registry
	providerRegistry := providers.NewProviderRegistry()
	// Providers will be initialized with API keys from workspace settings
	// For now, we'll create them on-demand

	// Create ingest handler
	ingestHandler := NewIngestHandler(db, providerRegistry, logger)

	// Setup router
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/health", healthCheckHandler(db)).Methods("GET")

	// OpenTelemetry endpoints
	router.HandleFunc("/v1/traces", ingestHandler.HandleTraces).Methods("POST")
	router.HandleFunc("/v1/spans", ingestHandler.HandleSpans).Methods("POST")

	// Custom endpoints for interactions
	router.HandleFunc("/v1/interactions", ingestHandler.RecordInteraction).Methods("POST")
	router.HandleFunc("/v1/interactions/batch", ingestHandler.RecordInteractionBatch).Methods("POST")

	// Metrics endpoint
	router.HandleFunc("/metrics", ingestHandler.GetMetrics).Methods("GET")

	// Configure HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      corsMiddleware(router),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("telemetry ingestion service listening",
			zap.String("address", server.Addr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}

// healthCheckHandler returns a handler for health checks
func healthCheckHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Check database connection
		if err := db.HealthCheck(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","database":"disconnected"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"ingest","database":"connected"}`))
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins for telemetry
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// IngestHandler handles telemetry ingestion
type IngestHandler struct {
	db               *database.DB
	providerRegistry *providers.ProviderRegistry
	logger           *zap.Logger
}

// NewIngestHandler creates a new ingest handler
func NewIngestHandler(db *database.DB, registry *providers.ProviderRegistry, logger *zap.Logger) *IngestHandler {
	return &IngestHandler{
		db:               db,
		providerRegistry: registry,
		logger:           logger,
	}
}

// InteractionRequest represents a request to record an interaction
type InteractionRequest struct {
	WorkspaceUUID  string                 `json:"workspace_uuid"`
	SessionUUID    *string                `json:"session_uuid,omitempty"`
	Provider       string                 `json:"provider"`
	Model          string                 `json:"model"`
	Prompt         string                 `json:"prompt"`
	Response       string                 `json:"response"`
	PromptTokens   int                    `json:"prompt_tokens"`
	ResponseTokens int                    `json:"response_tokens"`
	LatencyMs      int                    `json:"latency_ms"`
	Status         string                 `json:"status"` // success, error
	ErrorMessage   *string                `json:"error_message,omitempty"`
	TraceID        *string                `json:"trace_id,omitempty"`
	SpanID         *string                `json:"span_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// RecordInteraction records a single LLM interaction
func (h *IngestHandler) RecordInteraction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req InteractionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}

	// Validate required fields
	if req.WorkspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}
	if req.Provider == "" {
		h.respondError(w, errors.BadRequest("provider is required"))
		return
	}
	if req.Model == "" {
		h.respondError(w, errors.BadRequest("model is required"))
		return
	}

	// Calculate cost based on provider and model
	cost := 0.0
	if provider, err := h.getProvider(req.Provider); err == nil {
		cost, _ = provider.CalculateCost(req.Model, req.PromptTokens, req.ResponseTokens)
	}

	totalTokens := req.PromptTokens + req.ResponseTokens

	// Store interaction
	interaction := &models.Interaction{
		UUID:           models.NewUUID(),
		WorkspaceUUID:  models.ParseUUID(req.WorkspaceUUID),
		Provider:       req.Provider,
		Model:          req.Model,
		Prompt:         req.Prompt,
		Response:       req.Response,
		PromptTokens:   req.PromptTokens,
		ResponseTokens: req.ResponseTokens,
		TotalTokens:    totalTokens,
		Cost:           cost,
		LatencyMs:      req.LatencyMs,
		Status:         req.Status,
		ErrorMessage:   req.ErrorMessage,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
	}

	if req.SessionUUID != nil {
		sessionUUID := models.ParseUUID(*req.SessionUUID)
		interaction.SessionUUID = &sessionUUID
	}
	if req.TraceID != nil {
		interaction.TraceID = *req.TraceID
	}
	if req.SpanID != nil {
		interaction.SpanID = *req.SpanID
	}

	// Insert into database
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO spotlight.interactions (
			uuid, session_uuid, workspace_uuid, trace_id, span_id,
			provider, model, prompt, response,
			prompt_tokens, response_tokens, total_tokens, cost,
			latency_ms, status, error_message, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`,
		interaction.UUID, interaction.SessionUUID, interaction.WorkspaceUUID,
		interaction.TraceID, interaction.SpanID,
		interaction.Provider, interaction.Model, interaction.Prompt, interaction.Response,
		interaction.PromptTokens, interaction.ResponseTokens, interaction.TotalTokens, interaction.Cost,
		interaction.LatencyMs, interaction.Status, interaction.ErrorMessage,
		interaction.Metadata, interaction.CreatedAt,
	)

	if err != nil {
		h.logger.Error("failed to insert interaction", zap.Error(err))
		h.respondError(w, errors.Database(err, "insert interaction"))
		return
	}

	h.logger.Info("interaction recorded",
		zap.String("provider", req.Provider),
		zap.String("model", req.Model),
		zap.Int("total_tokens", totalTokens),
		zap.Float64("cost", cost),
	)

	h.respond(w, http.StatusCreated, map[string]interface{}{
		"uuid":         interaction.UUID,
		"cost":         cost,
		"total_tokens": totalTokens,
	})
}

// RecordInteractionBatch records multiple interactions at once
func (h *IngestHandler) RecordInteractionBatch(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement batch recording
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "Batch recording not implemented yet", http.StatusNotImplemented))
}

// HandleTraces handles OpenTelemetry trace data
func (h *IngestHandler) HandleTraces(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement OpenTelemetry trace ingestion
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "Trace ingestion not implemented yet", http.StatusNotImplemented))
}

// HandleSpans handles OpenTelemetry span data
func (h *IngestHandler) HandleSpans(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement OpenTelemetry span ingestion
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "Span ingestion not implemented yet", http.StatusNotImplemented))
}

// GetMetrics returns ingestion metrics
func (h *IngestHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement metrics
	h.respond(w, http.StatusOK, map[string]interface{}{
		"service": "ingest",
		"status":  "healthy",
	})
}

// Helper methods

func (h *IngestHandler) getProvider(name string) (providers.Provider, error) {
	return h.providerRegistry.Get(name)
}

func (h *IngestHandler) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *IngestHandler) respondError(w http.ResponseWriter, err error) {
	appErr := errors.GetAppError(err)
	if appErr == nil {
		appErr = errors.Internal(err, "An error occurred")
	}

	h.logger.Error("request error",
		zap.String("code", appErr.Code),
		zap.Error(err),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode)
	json.NewEncoder(w).Encode(appErr)
}
