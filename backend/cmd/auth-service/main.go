// Package main implements the authentication service
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

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

	logger.Info("starting auth service",
		zap.String("environment", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
	)

	// Initialize database
	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Create auth handler
	authHandler := NewAuthHandler(db, cfg, logger)

	// Setup router
	router := mux.NewRouter()
	
	// Health check endpoint
	router.HandleFunc("/health", healthCheckHandler(db)).Methods("GET")
	
	// Auth endpoints
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", authHandler.Register).Methods("POST")
	authRouter.HandleFunc("/login", authHandler.Login).Methods("POST")
	authRouter.HandleFunc("/refresh", authHandler.RefreshToken).Methods("POST")
	authRouter.HandleFunc("/oauth/google", authHandler.GoogleOAuth).Methods("GET")
	authRouter.HandleFunc("/oauth/google/callback", authHandler.GoogleOAuthCallback).Methods("GET")
	authRouter.HandleFunc("/oauth/github", authHandler.GitHubOAuth).Methods("GET")
	authRouter.HandleFunc("/oauth/github/callback", authHandler.GitHubOAuthCallback).Methods("GET")
	
	// User endpoints (protected)
	userRouter := router.PathPrefix("/users").Subrouter()
	userRouter.Use(authHandler.AuthMiddleware)
	userRouter.HandleFunc("/me", authHandler.GetCurrentUser).Methods("GET")
	userRouter.HandleFunc("/me", authHandler.UpdateCurrentUser).Methods("PATCH")
	
	// Workspace endpoints (protected)
	workspaceRouter := router.PathPrefix("/workspaces").Subrouter()
	workspaceRouter.Use(authHandler.AuthMiddleware)
	workspaceRouter.HandleFunc("", authHandler.ListWorkspaces).Methods("GET")
	workspaceRouter.HandleFunc("", authHandler.CreateWorkspace).Methods("POST")
	workspaceRouter.HandleFunc("/{id}", authHandler.GetWorkspace).Methods("GET")
	workspaceRouter.HandleFunc("/{id}", authHandler.UpdateWorkspace).Methods("PATCH")
	workspaceRouter.HandleFunc("/{id}", authHandler.DeleteWorkspace).Methods("DELETE")
	
	// API Key endpoints (protected)
	apiKeyRouter := router.PathPrefix("/api-keys").Subrouter()
	apiKeyRouter.Use(authHandler.AuthMiddleware)
	apiKeyRouter.HandleFunc("", authHandler.ListAPIKeys).Methods("GET")
	apiKeyRouter.HandleFunc("", authHandler.CreateAPIKey).Methods("POST")
	apiKeyRouter.HandleFunc("/{id}", authHandler.RevokeAPIKey).Methods("DELETE")

	// Configure HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      corsMiddleware(router, cfg.Server.FrontendURL),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("auth service listening",
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
		w.Write([]byte(`{"status":"healthy","service":"auth","database":"connected"}`))
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler, frontendURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
