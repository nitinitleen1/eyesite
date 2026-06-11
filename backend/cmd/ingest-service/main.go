// Package main implements the telemetry ingestion service
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
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

	// All ingestion endpoints require a valid workspace API key
	api := router.PathPrefix("/v1").Subrouter()
	api.Use(ingestHandler.APIKeyMiddleware)

	// OpenTelemetry endpoints
	api.HandleFunc("/traces", ingestHandler.HandleTraces).Methods("POST")
	api.HandleFunc("/spans", ingestHandler.HandleSpans).Methods("POST")

	// Custom endpoints for sessions and interactions
	api.HandleFunc("/sessions", ingestHandler.CreateSession).Methods("POST")
	api.HandleFunc("/interactions", ingestHandler.RecordInteraction).Methods("POST")
	api.HandleFunc("/interactions/batch", ingestHandler.RecordInteractionBatch).Methods("POST")

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

// buildInteraction validates a request against the API key and builds the interaction model
func (h *IngestHandler) buildInteraction(ctx context.Context, key *models.APIKey, req *InteractionRequest) (*models.Interaction, error) {
	// Validate required fields; the workspace always comes from the API key
	if req.WorkspaceUUID != "" && req.WorkspaceUUID != key.WorkspaceUUID.String() {
		return nil, errors.Forbidden("API key does not belong to this workspace")
	}
	if req.Provider == "" {
		return nil, errors.BadRequest("provider is required")
	}
	if req.Model == "" {
		return nil, errors.BadRequest("model is required")
	}
	if req.Status == "" {
		req.Status = "success"
	}

	// Calculate cost based on provider and model
	cost := 0.0
	if provider, err := h.getProvider(req.Provider); err == nil {
		cost, _ = provider.CalculateCost(req.Model, req.PromptTokens, req.ResponseTokens)
	}

	interaction := &models.Interaction{
		UUID:           models.NewUUID(),
		WorkspaceUUID:  key.WorkspaceUUID,
		Provider:       req.Provider,
		Model:          req.Model,
		Prompt:         req.Prompt,
		Response:       req.Response,
		PromptTokens:   req.PromptTokens,
		ResponseTokens: req.ResponseTokens,
		TotalTokens:    req.PromptTokens + req.ResponseTokens,
		Cost:           cost,
		LatencyMs:      req.LatencyMs,
		Status:         req.Status,
		ErrorMessage:   req.ErrorMessage,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
	}

	if req.SessionUUID != nil && *req.SessionUUID != "" {
		sessionUUID, err := uuid.Parse(*req.SessionUUID)
		if err != nil {
			return nil, errors.BadRequest("Invalid session_uuid")
		}
		// The session must belong to the API key's workspace
		var sessionWorkspace uuid.UUID
		err = h.db.QueryRowContext(ctx, `
			SELECT workspace_uuid FROM spotlight.sessions WHERE uuid = $1
		`, sessionUUID).Scan(&sessionWorkspace)
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("Session")
		}
		if err != nil {
			return nil, errors.Database(err, "check session")
		}
		if sessionWorkspace != key.WorkspaceUUID {
			return nil, errors.Forbidden("Session does not belong to this workspace")
		}
		interaction.SessionUUID = &sessionUUID
	}
	if req.TraceID != nil {
		interaction.TraceID = *req.TraceID
	}
	if req.SpanID != nil {
		interaction.SpanID = *req.SpanID
	}

	return interaction, nil
}

// execer abstracts *database.DB and *sql.Tx for inserts
type execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// insertInteraction inserts an interaction using the given executor
func (h *IngestHandler) insertInteraction(ctx context.Context, ex execer, interaction *models.Interaction) error {
	metadata, err := marshalJSONB(interaction.Metadata)
	if err != nil {
		return err
	}

	_, err = ex.ExecContext(ctx, `
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
		metadata, interaction.CreatedAt,
	)
	return err
}

// RecordInteraction records a single LLM interaction
func (h *IngestHandler) RecordInteraction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := apiKeyFromContext(r)

	var req InteractionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}

	interaction, err := h.buildInteraction(ctx, key, &req)
	if err != nil {
		h.respondError(w, err)
		return
	}

	if err := h.insertInteraction(ctx, h.db, interaction); err != nil {
		h.logger.Error("failed to insert interaction", zap.Error(err))
		h.respondError(w, errors.Database(err, "insert interaction"))
		return
	}

	h.logger.Info("interaction recorded",
		zap.String("provider", interaction.Provider),
		zap.String("model", interaction.Model),
		zap.Int("total_tokens", interaction.TotalTokens),
		zap.Float64("cost", interaction.Cost),
	)

	h.respond(w, http.StatusCreated, map[string]interface{}{
		"uuid":         interaction.UUID,
		"cost":         interaction.Cost,
		"total_tokens": interaction.TotalTokens,
	})
}

// InteractionBatchRequest represents a batch of interactions
type InteractionBatchRequest struct {
	Interactions []InteractionRequest `json:"interactions"`
}

// RecordInteractionBatch records multiple interactions at once
func (h *IngestHandler) RecordInteractionBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := apiKeyFromContext(r)

	var req InteractionBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}
	if len(req.Interactions) == 0 {
		h.respondError(w, errors.BadRequest("interactions is required"))
		return
	}

	interactions := make([]*models.Interaction, 0, len(req.Interactions))
	for i := range req.Interactions {
		interaction, err := h.buildInteraction(ctx, key, &req.Interactions[i])
		if err != nil {
			h.respondError(w, err)
			return
		}
		interactions = append(interactions, interaction)
	}

	err := h.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, interaction := range interactions {
			if err := h.insertInteraction(ctx, tx, interaction); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		h.logger.Error("failed to insert interaction batch", zap.Error(err))
		h.respondError(w, errors.Database(err, "insert interaction batch"))
		return
	}

	uuids := make([]uuid.UUID, len(interactions))
	totalCost := 0.0
	for i, interaction := range interactions {
		uuids[i] = interaction.UUID
		totalCost += interaction.Cost
	}

	h.respond(w, http.StatusCreated, map[string]interface{}{
		"recorded":   len(interactions),
		"uuids":      uuids,
		"total_cost": totalCost,
	})
}

// SessionRequest represents a session creation request
type SessionRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateSession creates a new session bound to the API key's workspace
func (h *IngestHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := apiKeyFromContext(r)

	var req SessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Session " + time.Now().Format("2006-01-02 15:04:05")
	}

	metadata, err := marshalJSONB(req.Metadata)
	if err != nil {
		h.respondError(w, errors.BadRequest("Invalid metadata"))
		return
	}

	now := time.Now()
	session := models.Session{
		UUID:          models.NewUUID(),
		UserUUID:      key.UserUUID,
		WorkspaceUUID: key.WorkspaceUUID,
		APIKeyUUID:    &key.UUID,
		Name:          name,
		Description:   req.Description,
		Metadata:      req.Metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err = h.db.ExecContext(ctx, `
		INSERT INTO spotlight.sessions (uuid, user_uuid, workspace_uuid, api_key_uuid, name, description, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, session.UUID, session.UserUUID, session.WorkspaceUUID, session.APIKeyUUID,
		session.Name, session.Description, metadata, session.CreatedAt, session.UpdatedAt)
	if err != nil {
		h.respondError(w, errors.Database(err, "create session"))
		return
	}

	h.respond(w, http.StatusCreated, session)
}

// TraceRequest represents an OpenTelemetry trace payload
type TraceRequest struct {
	TraceID     string                 `json:"trace_id"`
	ServiceName string                 `json:"service_name"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	DurationMs  *int                   `json:"duration_ms,omitempty"`
	Status      string                 `json:"status"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// HandleTraces handles OpenTelemetry trace data
func (h *IngestHandler) HandleTraces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := apiKeyFromContext(r)

	var req TraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}
	if req.TraceID == "" {
		h.respondError(w, errors.BadRequest("trace_id is required"))
		return
	}
	if req.StartTime.IsZero() {
		h.respondError(w, errors.BadRequest("start_time is required"))
		return
	}
	if req.Status == "" {
		req.Status = "ok"
	}
	if req.Status != "ok" && req.Status != "error" {
		h.respondError(w, errors.BadRequest("status must be ok or error"))
		return
	}

	attributes, err := marshalJSONB(req.Attributes)
	if err != nil {
		h.respondError(w, errors.BadRequest("Invalid attributes"))
		return
	}

	// Upsert so a trace can be started and later finished; the WHERE clause
	// prevents overwriting a trace owned by another workspace
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO telemetry.traces (trace_id, user_uuid, workspace_uuid, service_name, start_time, end_time, duration_ms, status, attributes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (trace_id) DO UPDATE SET
			end_time = EXCLUDED.end_time,
			duration_ms = EXCLUDED.duration_ms,
			status = EXCLUDED.status,
			attributes = EXCLUDED.attributes
		WHERE telemetry.traces.workspace_uuid = EXCLUDED.workspace_uuid
	`, req.TraceID, key.UserUUID, key.WorkspaceUUID, req.ServiceName,
		req.StartTime, req.EndTime, req.DurationMs, req.Status, attributes)
	if err != nil {
		h.logger.Error("failed to insert trace", zap.Error(err))
		h.respondError(w, errors.Database(err, "insert trace"))
		return
	}

	h.respond(w, http.StatusCreated, map[string]interface{}{
		"trace_id": req.TraceID,
	})
}

// SpanRequest represents an OpenTelemetry span payload
type SpanRequest struct {
	SpanID       string                 `json:"span_id"`
	TraceID      string                 `json:"trace_id"`
	ParentSpanID *string                `json:"parent_span_id,omitempty"`
	Name         string                 `json:"name"`
	Kind         string                 `json:"kind"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	DurationMs   *int                   `json:"duration_ms,omitempty"`
	Status       string                 `json:"status"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	Events       []models.SpanEvent     `json:"events,omitempty"`
}

var validSpanKinds = map[string]bool{
	"internal": true, "server": true, "client": true, "producer": true, "consumer": true,
}

// HandleSpans handles OpenTelemetry span data
func (h *IngestHandler) HandleSpans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := apiKeyFromContext(r)

	var req SpanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}
	if req.SpanID == "" {
		h.respondError(w, errors.BadRequest("span_id is required"))
		return
	}
	if req.TraceID == "" {
		h.respondError(w, errors.BadRequest("trace_id is required"))
		return
	}
	if req.Name == "" {
		h.respondError(w, errors.BadRequest("name is required"))
		return
	}
	if req.StartTime.IsZero() {
		h.respondError(w, errors.BadRequest("start_time is required"))
		return
	}
	if req.Kind == "" {
		req.Kind = "internal"
	}
	if !validSpanKinds[req.Kind] {
		h.respondError(w, errors.BadRequest("invalid span kind"))
		return
	}
	if req.Status == "" {
		req.Status = "ok"
	}
	if req.Status != "ok" && req.Status != "error" {
		h.respondError(w, errors.BadRequest("status must be ok or error"))
		return
	}

	// The parent trace must exist and belong to the API key's workspace
	var traceWorkspace uuid.UUID
	err := h.db.QueryRowContext(ctx, `
		SELECT workspace_uuid FROM telemetry.traces WHERE trace_id = $1
	`, req.TraceID).Scan(&traceWorkspace)
	if err == sql.ErrNoRows {
		h.respondError(w, errors.NotFound("Trace"))
		return
	}
	if err != nil {
		h.respondError(w, errors.Database(err, "check trace"))
		return
	}
	if traceWorkspace != key.WorkspaceUUID {
		h.respondError(w, errors.Forbidden("Trace does not belong to this workspace"))
		return
	}

	attributes, err := marshalJSONB(req.Attributes)
	if err != nil {
		h.respondError(w, errors.BadRequest("Invalid attributes"))
		return
	}
	events := []byte("[]")
	if req.Events != nil {
		if events, err = json.Marshal(req.Events); err != nil {
			h.respondError(w, errors.BadRequest("Invalid events"))
			return
		}
	}

	_, err = h.db.ExecContext(ctx, `
		INSERT INTO telemetry.spans (span_id, trace_id, parent_span_id, name, kind, start_time, end_time, duration_ms, status, attributes, events)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (span_id) DO UPDATE SET
			end_time = EXCLUDED.end_time,
			duration_ms = EXCLUDED.duration_ms,
			status = EXCLUDED.status,
			attributes = EXCLUDED.attributes,
			events = EXCLUDED.events
		WHERE telemetry.spans.trace_id = EXCLUDED.trace_id
	`, req.SpanID, req.TraceID, req.ParentSpanID, req.Name, req.Kind,
		req.StartTime, req.EndTime, req.DurationMs, req.Status, attributes, events)
	if err != nil {
		h.logger.Error("failed to insert span", zap.Error(err))
		h.respondError(w, errors.Database(err, "insert span"))
		return
	}

	h.respond(w, http.StatusCreated, map[string]interface{}{
		"span_id":  req.SpanID,
		"trace_id": req.TraceID,
	})
}

// GetMetrics returns ingestion metrics
func (h *IngestHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var interactions, traces, spans int
	err := h.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM spotlight.interactions),
			(SELECT COUNT(*) FROM telemetry.traces),
			(SELECT COUNT(*) FROM telemetry.spans)
	`).Scan(&interactions, &traces, &spans)
	if err != nil {
		h.respondError(w, errors.Database(err, "query metrics"))
		return
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"service":            "ingest",
		"status":             "healthy",
		"interactions_total": interactions,
		"traces_total":       traces,
		"spans_total":        spans,
	})
}

// marshalJSONB converts a metadata map to JSONB bytes, defaulting to an empty object
func marshalJSONB(m map[string]interface{}) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
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
