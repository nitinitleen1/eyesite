package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"

	"go.uber.org/zap"

	"github.com/eyesite/platform/backend/internal/errors"
	"github.com/eyesite/platform/backend/pkg/models"
)

// apiKeyContextKey is the context key under which the authenticated API key is stored
const apiKeyContextKey = "api_key"

// APIKeyMiddleware validates the X-API-Key header against main.api_keys.
// Keys are stored as deterministic SHA-256 hashes so they can be looked up directly.
func (h *IngestHandler) APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawKey := r.Header.Get("X-API-Key")
		if rawKey == "" {
			h.respondError(w, errors.Unauthorized("Missing X-API-Key header"))
			return
		}

		sum := sha256.Sum256([]byte(rawKey))
		keyHash := hex.EncodeToString(sum[:])

		var key models.APIKey
		err := h.db.QueryRowContext(r.Context(), `
			SELECT uuid, user_uuid, workspace_uuid, name, key_preview,
			       last_used_at, expires_at, created_at, revoked_at
			FROM main.api_keys
			WHERE key_hash = $1
		`, keyHash).Scan(
			&key.UUID, &key.UserUUID, &key.WorkspaceUUID, &key.Name, &key.KeyPreview,
			&key.LastUsedAt, &key.ExpiresAt, &key.CreatedAt, &key.RevokedAt,
		)
		if err == sql.ErrNoRows {
			h.respondError(w, errors.Unauthorized("Invalid API key"))
			return
		}
		if err != nil {
			h.logger.Error("failed to look up api key", zap.Error(err))
			h.respondError(w, errors.Internal(err, "Failed to authenticate"))
			return
		}

		if !key.IsValid() {
			h.respondError(w, errors.Unauthorized("API key is revoked or expired"))
			return
		}

		// Track key usage (best effort)
		if _, err := h.db.ExecContext(r.Context(), `
			UPDATE main.api_keys SET last_used_at = NOW() WHERE uuid = $1
		`, key.UUID); err != nil {
			h.logger.Warn("failed to update api key last_used_at", zap.Error(err))
		}

		ctx := context.WithValue(r.Context(), apiKeyContextKey, &key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// apiKeyFromContext returns the authenticated API key for the request
func apiKeyFromContext(r *http.Request) *models.APIKey {
	key, _ := r.Context().Value(apiKeyContextKey).(*models.APIKey)
	return key
}
