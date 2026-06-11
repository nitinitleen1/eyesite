// Package main implements the query service for analytics
package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	queryHandler := NewQueryHandler(db, cfg, logger)

	// Setup router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthCheckHandler(db)).Methods("GET")

	// All data endpoints require an authenticated user (JWT)
	api := router.PathPrefix("/v1").Subrouter()
	api.Use(queryHandler.AuthMiddleware)

	// Analytics endpoints
	api.HandleFunc("/analytics/overview", queryHandler.GetOverview).Methods("GET")
	api.HandleFunc("/analytics/costs", queryHandler.GetCosts).Methods("GET")
	api.HandleFunc("/analytics/usage", queryHandler.GetUsage).Methods("GET")
	api.HandleFunc("/analytics/providers", queryHandler.GetProviderStats).Methods("GET")

	// Session endpoints
	api.HandleFunc("/sessions", queryHandler.ListSessions).Methods("GET")
	api.HandleFunc("/sessions/{uuid}", queryHandler.GetSession).Methods("GET")
	api.HandleFunc("/sessions/{uuid}/interactions", queryHandler.GetSessionInteractions).Methods("GET")

	// Interaction endpoints
	api.HandleFunc("/interactions", queryHandler.ListInteractions).Methods("GET")
	api.HandleFunc("/interactions/search", queryHandler.SearchInteractions).Methods("GET")
	api.HandleFunc("/interactions/{uuid}", queryHandler.GetInteraction).Methods("GET")

	// Export endpoints
	api.HandleFunc("/export/interactions", queryHandler.ExportInteractions).Methods("GET")

	// Configure HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      corsMiddleware(router, cfg.Server.FrontendURL),
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

func corsMiddleware(next http.Handler, frontendURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendURL)
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
	cfg    *config.Config
	logger *zap.Logger
}

func NewQueryHandler(db *database.DB, cfg *config.Config, logger *zap.Logger) *QueryHandler {
	return &QueryHandler{
		db:     db,
		cfg:    cfg,
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
	if err := h.requireWorkspaceMember(r, workspaceUUID); err != nil {
		h.respondError(w, err)
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
	if err := h.requireWorkspaceMember(r, workspaceUUID); err != nil {
		h.respondError(w, err)
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

// GetUsage returns token usage statistics over the last 30 days
func (h *QueryHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}
	if err := h.requireWorkspaceMember(r, workspaceUUID); err != nil {
		h.respondError(w, err)
		return
	}

	ctx := r.Context()

	rows, err := h.db.QueryContext(ctx, `
		SELECT
			DATE(created_at) as date,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(response_tokens), 0) as response_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM spotlight.interactions
		WHERE workspace_uuid = $1
		AND created_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, workspaceUUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "query usage"))
		return
	}
	defer rows.Close()

	type usageData struct {
		Date           string `json:"date"`
		PromptTokens   int    `json:"prompt_tokens"`
		ResponseTokens int    `json:"response_tokens"`
		TotalTokens    int    `json:"total_tokens"`
	}

	usage := []usageData{}
	for rows.Next() {
		var u usageData
		if err := rows.Scan(&u.Date, &u.PromptTokens, &u.ResponseTokens, &u.TotalTokens); err != nil {
			h.respondError(w, errors.Database(err, "scan usage"))
			return
		}
		usage = append(usage, u)
	}

	h.respond(w, http.StatusOK, usage)
}

// GetProviderStats returns statistics by provider
func (h *QueryHandler) GetProviderStats(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}
	if err := h.requireWorkspaceMember(r, workspaceUUID); err != nil {
		h.respondError(w, err)
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

// sessionSummary holds session data with aggregated interaction stats
type sessionSummary struct {
	UUID             string     `json:"uuid"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	WorkspaceUUID    string     `json:"workspace_uuid"`
	InteractionCount int        `json:"interaction_count"`
	TotalTokens      int        `json:"total_tokens"`
	TotalCost        float64    `json:"total_cost"`
	CreatedAt        time.Time  `json:"created_at"`
	FirstInteraction *time.Time `json:"first_interaction,omitempty"`
	LastInteraction  *time.Time `json:"last_interaction,omitempty"`
}

const sessionSummaryQuery = `
	SELECT
		s.uuid, s.name, COALESCE(s.description, ''), s.workspace_uuid, s.created_at,
		COUNT(i.uuid) as interaction_count,
		COALESCE(SUM(i.total_tokens), 0) as total_tokens,
		COALESCE(SUM(i.cost), 0) as total_cost,
		MIN(i.created_at) as first_interaction,
		MAX(i.created_at) as last_interaction
	FROM spotlight.sessions s
	LEFT JOIN spotlight.interactions i ON i.session_uuid = s.uuid
`

func scanSessionSummary(scan func(dest ...interface{}) error) (sessionSummary, error) {
	var s sessionSummary
	err := scan(&s.UUID, &s.Name, &s.Description, &s.WorkspaceUUID, &s.CreatedAt,
		&s.InteractionCount, &s.TotalTokens, &s.TotalCost, &s.FirstInteraction, &s.LastInteraction)
	return s, err
}

// ListSessions lists sessions for a workspace with aggregated stats
func (h *QueryHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}
	if err := h.requireWorkspaceMember(r, workspaceUUID); err != nil {
		h.respondError(w, err)
		return
	}

	rows, err := h.db.QueryContext(r.Context(), sessionSummaryQuery+`
		WHERE s.workspace_uuid = $1
		GROUP BY s.uuid, s.name, s.description, s.workspace_uuid, s.created_at
		ORDER BY s.created_at DESC
		LIMIT 100
	`, workspaceUUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "list sessions"))
		return
	}
	defer rows.Close()

	sessions := []sessionSummary{}
	for rows.Next() {
		s, err := scanSessionSummary(rows.Scan)
		if err != nil {
			h.respondError(w, errors.Database(err, "scan session"))
			return
		}
		sessions = append(sessions, s)
	}

	h.respond(w, http.StatusOK, sessions)
}

// GetSession returns a single session with aggregated stats
func (h *QueryHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionUUID := mux.Vars(r)["uuid"]

	row := h.db.QueryRowContext(r.Context(), sessionSummaryQuery+`
		WHERE s.uuid = $1
		GROUP BY s.uuid, s.name, s.description, s.workspace_uuid, s.created_at
	`, sessionUUID)

	s, err := scanSessionSummary(row.Scan)
	if err == sql.ErrNoRows {
		h.respondError(w, errors.NotFound("Session"))
		return
	}
	if err != nil {
		h.respondError(w, errors.Database(err, "get session"))
		return
	}

	if err := h.requireWorkspaceMember(r, s.WorkspaceUUID); err != nil {
		h.respondError(w, err)
		return
	}

	h.respond(w, http.StatusOK, s)
}

// GetSessionInteractions returns all interactions for a session
func (h *QueryHandler) GetSessionInteractions(w http.ResponseWriter, r *http.Request) {
	sessionUUID := mux.Vars(r)["uuid"]

	var sessionWorkspaceUUID string
	err := h.db.QueryRowContext(r.Context(), `
		SELECT workspace_uuid FROM spotlight.sessions WHERE uuid = $1
	`, sessionUUID).Scan(&sessionWorkspaceUUID)
	if err == sql.ErrNoRows {
		h.respondError(w, errors.NotFound("Session"))
		return
	}
	if err != nil {
		h.respondError(w, errors.Database(err, "get session workspace"))
		return
	}
	if err := h.requireWorkspaceMember(r, sessionWorkspaceUUID); err != nil {
		h.respondError(w, err)
		return
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT uuid, provider, model, prompt_tokens, response_tokens, cost, created_at, status
		FROM spotlight.interactions
		WHERE session_uuid = $1
		ORDER BY created_at DESC
		LIMIT 500
	`, sessionUUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "query session interactions"))
		return
	}
	defer rows.Close()

	interactions := []interactionRow{}
	for rows.Next() {
		var i interactionRow
		if err := rows.Scan(&i.UUID, &i.Provider, &i.Model, &i.PromptTokens, &i.ResponseTokens, &i.Cost, &i.CreatedAt, &i.Status); err != nil {
			h.respondError(w, errors.Database(err, "scan interaction"))
			return
		}
		interactions = append(interactions, i)
	}

	h.respond(w, http.StatusOK, interactions)
}

// ListInteractions lists all interactions
func (h *QueryHandler) ListInteractions(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}
	if err := h.requireWorkspaceMember(r, workspaceUUID); err != nil {
		h.respondError(w, err)
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

	interactions := []interactionRow{}
	for rows.Next() {
		var i interactionRow
		if err := rows.Scan(&i.UUID, &i.Provider, &i.Model, &i.PromptTokens, &i.ResponseTokens, &i.Cost, &i.CreatedAt, &i.Status); err != nil {
			h.respondError(w, errors.Database(err, "scan interaction"))
			return
		}
		interactions = append(interactions, i)
	}

	h.respond(w, http.StatusOK, interactions)
}

// interactionRow is the list/search representation of an interaction
type interactionRow struct {
	UUID           string    `json:"uuid"`
	Provider       string    `json:"provider"`
	Model          string    `json:"model"`
	PromptTokens   int       `json:"prompt_tokens"`
	ResponseTokens int       `json:"response_tokens"`
	Cost           float64   `json:"cost"`
	CreatedAt      time.Time `json:"created_at"`
	Status         string    `json:"status"`
}

// GetInteraction returns a single interaction with full prompt/response
func (h *QueryHandler) GetInteraction(w http.ResponseWriter, r *http.Request) {
	interactionUUID := mux.Vars(r)["uuid"]

	type interactionDetail struct {
		interactionRow
		WorkspaceUUID string  `json:"workspace_uuid"`
		SessionUUID   *string `json:"session_uuid,omitempty"`
		Prompt        string  `json:"prompt"`
		Response      string  `json:"response"`
		TotalTokens   int     `json:"total_tokens"`
		LatencyMs     *int    `json:"latency_ms,omitempty"`
		ErrorMessage  *string `json:"error_message,omitempty"`
	}

	var d interactionDetail
	err := h.db.QueryRowContext(r.Context(), `
		SELECT uuid, workspace_uuid, session_uuid, provider, model, prompt, COALESCE(response, ''),
		       prompt_tokens, response_tokens, total_tokens, cost, latency_ms,
		       status, error_message, created_at
		FROM spotlight.interactions
		WHERE uuid = $1
	`, interactionUUID).Scan(
		&d.UUID, &d.WorkspaceUUID, &d.SessionUUID, &d.Provider, &d.Model, &d.Prompt, &d.Response,
		&d.PromptTokens, &d.ResponseTokens, &d.TotalTokens, &d.Cost, &d.LatencyMs,
		&d.Status, &d.ErrorMessage, &d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		h.respondError(w, errors.NotFound("Interaction"))
		return
	}
	if err != nil {
		h.respondError(w, errors.Database(err, "get interaction"))
		return
	}

	if err := h.requireWorkspaceMember(r, d.WorkspaceUUID); err != nil {
		h.respondError(w, err)
		return
	}

	h.respond(w, http.StatusOK, d)
}

// SearchInteractions searches prompt/response text within a workspace
func (h *QueryHandler) SearchInteractions(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}
	if err := h.requireWorkspaceMember(r, workspaceUUID); err != nil {
		h.respondError(w, err)
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		h.respondError(w, errors.BadRequest("q is required"))
		return
	}

	pattern := "%" + query + "%"
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT uuid, provider, model, prompt_tokens, response_tokens, cost, created_at, status
		FROM spotlight.interactions
		WHERE workspace_uuid = $1
		AND (prompt ILIKE $2 OR response ILIKE $2)
		ORDER BY created_at DESC
		LIMIT 100
	`, workspaceUUID, pattern)
	if err != nil {
		h.respondError(w, errors.Database(err, "search interactions"))
		return
	}
	defer rows.Close()

	results := []interactionRow{}
	for rows.Next() {
		var i interactionRow
		if err := rows.Scan(&i.UUID, &i.Provider, &i.Model, &i.PromptTokens, &i.ResponseTokens, &i.Cost, &i.CreatedAt, &i.Status); err != nil {
			h.respondError(w, errors.Database(err, "scan interaction"))
			return
		}
		results = append(results, i)
	}

	h.respond(w, http.StatusOK, results)
}

// ExportInteractions exports workspace interactions as CSV or JSON
func (h *QueryHandler) ExportInteractions(w http.ResponseWriter, r *http.Request) {
	workspaceUUID := r.URL.Query().Get("workspace_uuid")
	if workspaceUUID == "" {
		h.respondError(w, errors.BadRequest("workspace_uuid is required"))
		return
	}
	if err := h.requireWorkspaceMember(r, workspaceUUID); err != nil {
		h.respondError(w, err)
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		h.respondError(w, errors.BadRequest("format must be json or csv"))
		return
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT uuid, COALESCE(session_uuid::text, ''), provider, model,
		       prompt_tokens, response_tokens, total_tokens, cost,
		       COALESCE(latency_ms, 0), status, created_at
		FROM spotlight.interactions
		WHERE workspace_uuid = $1
		ORDER BY created_at DESC
		LIMIT 10000
	`, workspaceUUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "export interactions"))
		return
	}
	defer rows.Close()

	type exportRow struct {
		UUID           string    `json:"uuid"`
		SessionUUID    string    `json:"session_uuid,omitempty"`
		Provider       string    `json:"provider"`
		Model          string    `json:"model"`
		PromptTokens   int       `json:"prompt_tokens"`
		ResponseTokens int       `json:"response_tokens"`
		TotalTokens    int       `json:"total_tokens"`
		Cost           float64   `json:"cost"`
		LatencyMs      int       `json:"latency_ms"`
		Status         string    `json:"status"`
		CreatedAt      time.Time `json:"created_at"`
	}

	records := []exportRow{}
	for rows.Next() {
		var e exportRow
		if err := rows.Scan(&e.UUID, &e.SessionUUID, &e.Provider, &e.Model,
			&e.PromptTokens, &e.ResponseTokens, &e.TotalTokens, &e.Cost,
			&e.LatencyMs, &e.Status, &e.CreatedAt); err != nil {
			h.respondError(w, errors.Database(err, "scan export row"))
			return
		}
		records = append(records, e)
	}

	if format == "json" {
		w.Header().Set("Content-Disposition", `attachment; filename="interactions.json"`)
		h.respond(w, http.StatusOK, records)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="interactions.csv"`)
	cw := csv.NewWriter(w)
	cw.Write([]string{"uuid", "session_uuid", "provider", "model", "prompt_tokens",
		"response_tokens", "total_tokens", "cost", "latency_ms", "status", "created_at"})
	for _, e := range records {
		cw.Write([]string{
			e.UUID, e.SessionUUID, e.Provider, e.Model,
			strconv.Itoa(e.PromptTokens), strconv.Itoa(e.ResponseTokens), strconv.Itoa(e.TotalTokens),
			strconv.FormatFloat(e.Cost, 'f', -1, 64), strconv.Itoa(e.LatencyMs),
			e.Status, e.CreatedAt.Format(time.RFC3339),
		})
	}
	cw.Flush()
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
