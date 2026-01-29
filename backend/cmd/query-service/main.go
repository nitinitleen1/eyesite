// Package main implements the query service for analytics
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

	// Override port for query service
	cfg.Server.Port = 8002

	logger.Info("starting query service",
		zap.String("environment", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
	)

	// Initialize database
	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Create query handler
	queryHandler := NewQueryHandler(db, logger)

	// Setup router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthCheckHandler(db)).Methods("GET")

	// Analytics endpoints
	router.HandleFunc("/v1/analytics/overview", queryHandler.GetOverview).Methods("GET")
	router.HandleFunc("/v1/analytics/costs", queryHandler.GetCosts).Methods("GET")
	router.HandleFunc("/v1/analytics/usage", queryHandler.GetUsage).Methods("GET")
	router.HandleFunc("/v1/analytics/providers", queryHandler.GetProviderStats).Methods("GET")

	// Session endpoints
	router.HandleFunc("/v1/sessions", queryHandler.ListSessions).Methods("GET")
	router.HandleFunc("/v1/sessions/{uuid}", queryHandler.GetSession).Methods("GET")
	router.HandleFunc("/v1/sessions/{uuid}/interactions", queryHandler.GetSessionInteractions).Methods("GET")

	// Interaction endpoints
	router.HandleFunc("/v1/interactions", queryHandler.ListInteractions).Methods("GET")
	router.HandleFunc("/v1/interactions/{uuid}", queryHandler.GetInteraction).Methods("GET")
	router.HandleFunc("/v1/interactions/search", queryHandler.SearchInteractions).Methods("GET")

	// Export endpoints
	router.HandleFunc("/v1/export/interactions", queryHandler.ExportInteractions).Methods("GET")

	// Configure HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      corsMiddleware(router),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("query service listening",
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

func healthCheckHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.HealthCheck(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","database":"disconnected"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"query","database":"connected"}`))
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// QueryHandler handles analytics and query requests
type QueryHandler struct {
	db     *database.DB
	logger *zap.Logger
}

func NewQueryHandler(db *database.DB, logger *zap.Logger) *QueryHandler {
	return &QueryHandler{
		db:     db,
		logger: logger,
	}
}

// GetOverview returns high-level analytics
func (h *QueryHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}

	ctx := r.Context()

	// Get total interactions
	var totalInteractions int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM spotlight.interactions WHERE workspace_uuid = $1
	`, workspaceUUID).Scan(&totalInteractions)
	if err != nil {
		h.respondError(w, errors.Database(err, "count interactions"))
		return
	}

	// Get total cost
	var totalCost float64
	err = h.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(cost), 0) FROM spotlight.interactions WHERE workspace_uuid = $1
	`, workspaceUUID).Scan(&totalCost)
	if err != nil {
		h.respondError(w, errors.Database(err, "sum cost"))
		return
	}

	// Get total tokens
	var totalTokens int
	err = h.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_tokens), 0) FROM spotlight.interactions WHERE workspace_uuid = $1
	`, workspaceUUID).Scan(&totalTokens)
	if err != nil {
		h.respondError(w, errors.Database(err, "sum tokens"))
		return
	}

	// Get active sessions count
	var activeSessions int
	err = h.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT session_uuid) FROM spotlight.interactions 
		WHERE workspace_uuid = $1 AND session_uuid IS NOT NULL
	`, workspaceUUID).Scan(&activeSessions)
	if err != nil {
		h.respondError(w, errors.Database(err, "count sessions"))
		return
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"total_interactions": totalInteractions,
		"total_cost":         totalCost,
		"total_tokens":       totalTokens,
		"active_sessions":    activeSessions,
	})
}

// GetCosts returns cost analytics over time
func (h *QueryHandler) GetCosts(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}

	ctx := r.Context()

	rows, err := h.db.QueryContext(ctx, `
		SELECT 
			DATE(created_at) as date,
			SUM(cost) as total_cost,
			COUNT(*) as interaction_count
		FROM spotlight.interactions
		WHERE workspace_uuid = $1
		AND created_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, workspaceUUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "query costs"))
		return
	}
	defer rows.Close()

	type costData struct {
		Date             string  `json:"date"`
		TotalCost        float64 `json:"total_cost"`
		InteractionCount int     `json:"interaction_count"`
	}

	var costs []costData
	for rows.Next() {
		var c costData
		if err := rows.Scan(&c.Date, &c.TotalCost, &c.InteractionCount); err != nil {
			h.respondError(w, errors.Database(err, "scan cost"))
			return
		}
		costs = append(costs, c)
	}

	h.respond(w, http.StatusOK, costs)
}

// GetUsage returns token usage statistics
func (h *QueryHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	h.respond(w, http.StatusOK, map[string]interface{}{
		"message": "Usage analytics coming soon",
	})
}

// GetProviderStats returns statistics by provider
func (h *QueryHandler) GetProviderStats(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}

	ctx := r.Context()

	rows, err := h.db.QueryContext(ctx, `
		SELECT 
			provider,
			model,
			COUNT(*) as count,
			SUM(cost) as total_cost,
			SUM(total_tokens) as total_tokens
		FROM spotlight.interactions
		WHERE workspace_uuid = $1
		GROUP BY provider, model
		ORDER BY count DESC
	`, workspaceUUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "query provider stats"))
		return
	}
	defer rows.Close()

	type providerStat struct {
		Provider    string  `json:"provider"`
		Model       string  `json:"model"`
		Count       int     `json:"count"`
		TotalCost   float64 `json:"total_cost"`
		TotalTokens int     `json:"total_tokens"`
	}

	var stats []providerStat
	for rows.Next() {
		var s providerStat
		if err := rows.Scan(&s.Provider, &s.Model, &s.Count, &s.TotalCost, &s.TotalTokens); err != nil {
			h.respondError(w, errors.Database(err, "scan provider stat"))
			return
		}
		stats = append(stats, s)
	}

	h.respond(w, http.StatusOK, stats)
}

// List sessions
func (h *QueryHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	h.respond(w, http.StatusOK, []interface{}{})
}

// Get session
func (h *QueryHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	h.respond(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// Get session interactions
func (h *QueryHandler) GetSessionInteractions(w http.ResponseWriter, r *http.Request) {
	h.respond(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// ListInteractions lists all interactions
func (h *QueryHandler) ListInteractions(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}

	ctx := r.Context()

	rows, err := h.db.QueryContext(ctx, `
		SELECT uuid, provider, model, prompt_tokens, response_tokens, cost, created_at, status
		FROM spotlight.interactions
		WHERE workspace_uuid = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, workspaceUUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "query interactions"))
		return
	}
	defer rows.Close()

	type interaction struct {
		UUID           string    `json:"uuid"`
		Provider       string    `json:"provider"`
		Model          string    `json:"model"`
		PromptTokens   int       `json:"prompt_tokens"`
		ResponseTokens int       `json:"response_tokens"`
		Cost           float64   `json:"cost"`
		CreatedAt      time.Time `json:"created_at"`
		Status         string    `json:"status"`
	}

	var interactions []interaction
	for rows.Next() {
		var i interaction
		if err := rows.Scan(&i.UUID, &i.Provider, &i.Model, &i.PromptTokens, &i.ResponseTokens, &i.Cost, &i.CreatedAt, &i.Status); err != nil {
			h.respondError(w, errors.Database(err, "scan interaction"))
			return
		}
		interactions = append(interactions, i)
	}

	h.respond(w, http.StatusOK, interactions)
}

// Get single interaction
func (h *QueryHandler) GetInteraction(w http.ResponseWriter, r *http.Request) {
	h.respond(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// Search interactions
func (h *QueryHandler) SearchInteractions(w http.ResponseWriter, r *http.Request) {
	h.respond(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// Export interactions
func (h *QueryHandler) ExportInteractions(w http.ResponseWriter, r *http.Request) {
	h.respond(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// Helper methods
func (h *QueryHandler) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *QueryHandler) respondError(w http.ResponseWriter, err error) {
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
