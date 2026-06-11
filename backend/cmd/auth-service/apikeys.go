package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"

	"github.com/eyesite/platform/backend/internal/errors"
	"github.com/eyesite/platform/backend/pkg/models"
)

const apiKeyPrefix = "eys_"

// CreateAPIKeyRequest represents an API key creation request
type CreateAPIKeyRequest struct {
	WorkspaceUUID string   `json:"workspace_uuid"`
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes,omitempty"`
	ExpiresInDays int      `json:"expires_in_days,omitempty"`
}

// CreateAPIKeyResponse includes the plaintext key, shown only once at creation
type CreateAPIKeyResponse struct {
	models.APIKey
	Key string `json:"key"`
}

// generateAPIKey returns a new plaintext key and its SHA-256 hash.
// SHA-256 (not bcrypt) so the ingest path can look keys up by deterministic hash.
func generateAPIKey() (key, hash string, err error) {
	raw := make([]byte, 24)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	key = apiKeyPrefix + hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(key))
	return key, hex.EncodeToString(sum[:]), nil
}

// ListAPIKeys lists all API keys for a workspace
func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	ctx := r.Context()

	wsUUID, err := uuid.Parse(r.URL.Query().Get("workspace_uuid"))
	if err != nil {
		h.respondError(w, errors.BadRequest("workspace_uuid query parameter is required"))
		return
	}

	if _, err := h.memberRole(ctx, wsUUID, user.UUID); err != nil {
		h.respondError(w, err)
		return
	}

	rows, err := h.db.QueryContext(ctx, `
		SELECT uuid, user_uuid, workspace_uuid, name, key_preview, scopes,
		       last_used_at, expires_at, created_at, revoked_at
		FROM main.api_keys
		WHERE workspace_uuid = $1
		ORDER BY created_at DESC
	`, wsUUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "list api keys"))
		return
	}
	defer rows.Close()

	keys := []models.APIKey{}
	for rows.Next() {
		var k models.APIKey
		if err := rows.Scan(&k.UUID, &k.UserUUID, &k.WorkspaceUUID, &k.Name, &k.KeyPreview,
			pq.Array(&k.Scopes), &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt, &k.RevokedAt); err != nil {
			h.respondError(w, errors.Database(err, "scan api key"))
			return
		}
		keys = append(keys, k)
	}

	h.respond(w, http.StatusOK, keys)
}

// CreateAPIKey creates a new API key; the plaintext key is returned only once
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	ctx := r.Context()

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}

	wsUUID, err := uuid.Parse(req.WorkspaceUUID)
	if err != nil {
		h.respondError(w, errors.Validation("workspace_uuid is required", nil))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		h.respondError(w, errors.Validation("Key name is required", nil))
		return
	}

	if _, err := h.memberRole(ctx, wsUUID, user.UUID); err != nil {
		h.respondError(w, err)
		return
	}

	key, hash, err := generateAPIKey()
	if err != nil {
		h.respondError(w, errors.Internal(err, "Failed to generate API key"))
		return
	}

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read", "write"}
	}

	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		t := time.Now().AddDate(0, 0, req.ExpiresInDays)
		expiresAt = &t
	}

	apiKey := models.APIKey{
		UUID:          uuid.New(),
		UserUUID:      user.UUID,
		WorkspaceUUID: wsUUID,
		Name:          strings.TrimSpace(req.Name),
		KeyHash:       hash,
		KeyPreview:    key[:12],
		Scopes:        scopes,
		ExpiresAt:     expiresAt,
		CreatedAt:     time.Now(),
	}

	_, err = h.db.ExecContext(ctx, `
		INSERT INTO main.api_keys (uuid, user_uuid, workspace_uuid, name, key_hash, key_preview, scopes, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, apiKey.UUID, apiKey.UserUUID, apiKey.WorkspaceUUID, apiKey.Name, apiKey.KeyHash,
		apiKey.KeyPreview, pq.Array(apiKey.Scopes), apiKey.ExpiresAt, apiKey.CreatedAt)
	if err != nil {
		h.respondError(w, errors.Database(err, "create api key"))
		return
	}

	h.respond(w, http.StatusCreated, CreateAPIKeyResponse{APIKey: apiKey, Key: key})
}

// RevokeAPIKey revokes an API key (key creator, or workspace owner/admin)
func (h *AuthHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	ctx := r.Context()

	keyUUID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.respondError(w, errors.BadRequest("Invalid API key ID"))
		return
	}

	var ownerUUID, wsUUID uuid.UUID
	var revokedAt *time.Time
	err = h.db.QueryRowContext(ctx, `
		SELECT user_uuid, workspace_uuid, revoked_at FROM main.api_keys WHERE uuid = $1
	`, keyUUID).Scan(&ownerUUID, &wsUUID, &revokedAt)
	if err == sql.ErrNoRows {
		h.respondError(w, errors.NotFound("API key"))
		return
	}
	if err != nil {
		h.respondError(w, errors.Database(err, "get api key"))
		return
	}

	if ownerUUID != user.UUID {
		role, err := h.memberRole(ctx, wsUUID, user.UUID)
		if err != nil {
			h.respondError(w, err)
			return
		}
		if role != "owner" && role != "admin" {
			h.respondError(w, errors.Forbidden("Not allowed to revoke this key"))
			return
		}
	}

	if revokedAt != nil {
		h.respondError(w, errors.Conflict("API key is already revoked"))
		return
	}

	if _, err := h.db.ExecContext(ctx, `
		UPDATE main.api_keys SET revoked_at = NOW() WHERE uuid = $1
	`, keyUUID); err != nil {
		h.respondError(w, errors.Database(err, "revoke api key"))
		return
	}

	h.respond(w, http.StatusOK, map[string]string{"status": "revoked"})
}
