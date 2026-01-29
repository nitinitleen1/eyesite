// Package database provides database connection and query helpers
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"go.uber.org/zap"

	"github.com/agent-observability/platform/backend/pkg/config"
)

// DB wraps the SQL database with additional functionality
type DB struct {
	*sql.DB
	logger *zap.Logger
}

// New creates a new database connection
func New(cfg *config.DatabaseConfig, logger *zap.Logger) (*DB, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Database),
	)

	return &DB{
		DB:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	db.logger.Info("closing database connection")
	return db.DB.Close()
}

// HealthCheck checks if database is healthy
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// WithTransaction executes a function within a database transaction
func (db *DB) WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute function
	if err := fn(tx); err != nil {
		// Rollback on error
		if rbErr := tx.Rollback(); rbErr != nil {
			db.logger.Error("failed to rollback transaction",
				zap.Error(rbErr),
				zap.Error(err),
			)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// QueryRow wraps sql.DB.QueryRowContext with logging
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := db.QueryRowContext(ctx, query, args...)
	
	db.logger.Debug("query executed",
		zap.Duration("duration", time.Since(start)),
		zap.String("query", query),
	)
	
	return row
}

// Query wraps sql.DB.QueryContext with logging
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.QueryContext(ctx, query, args...)
	
	db.logger.Debug("query executed",
		zap.Duration("duration", time.Since(start)),
		zap.String("query", query),
		zap.Error(err),
	)
	
	return rows, err
}

// Exec wraps sql.DB.ExecContext with logging
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := db.ExecContext(ctx, query, args...)
	
	db.logger.Debug("query executed",
		zap.Duration("duration", time.Since(start)),
		zap.String("query", query),
		zap.Error(err),
	)
	
	return result, err
}
