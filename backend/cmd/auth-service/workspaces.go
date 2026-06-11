package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/eyesite/platform/backend/internal/errors"
	"github.com/eyesite/platform/backend/pkg/models"
)

// WorkspaceRequest represents a create/update workspace request
type WorkspaceRequest struct {
	Name string `json:"name"`
}

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9-]+`)

// slugify converts a name to a URL-friendly slug
func slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugInvalidChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "workspace"
	}
	if len(slug) > 80 {
		slug = slug[:80]
	}
	return slug
}

// memberRole returns the user's role in a workspace, or an error if not a member
func (h *AuthHandler) memberRole(ctx context.Context, workspaceUUID, userUUID uuid.UUID) (string, error) {
	var role string
	err := h.db.QueryRowContext(ctx, `
		SELECT role FROM main.workspace_members
		WHERE workspace_uuid = $1 AND user_uuid = $2
	`, workspaceUUID, userUUID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", errors.NotFound("Workspace")
	}
	if err != nil {
		return "", errors.Database(err, "check membership")
	}
	return role, nil
}

// ListWorkspaces lists all workspaces for the current user
func (h *AuthHandler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	ctx := r.Context()

	rows, err := h.db.QueryContext(ctx, `
		SELECT w.uuid, w.name, w.owner_uuid, w.slug, w.created_at, w.updated_at
		FROM main.workspaces w
		JOIN main.workspace_members m ON m.workspace_uuid = w.uuid
		WHERE m.user_uuid = $1
		ORDER BY w.created_at ASC
	`, user.UUID)
	if err != nil {
		h.respondError(w, errors.Database(err, "list workspaces"))
		return
	}
	defer rows.Close()

	workspaces := []models.Workspace{}
	for rows.Next() {
		var ws models.Workspace
		if err := rows.Scan(&ws.UUID, &ws.Name, &ws.OwnerUUID, &ws.Slug, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
			h.respondError(w, errors.Database(err, "scan workspace"))
			return
		}
		workspaces = append(workspaces, ws)
	}

	h.respond(w, http.StatusOK, workspaces)
}

// CreateWorkspace creates a new workspace owned by the current user
func (h *AuthHandler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	ctx := r.Context()

	var req WorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		h.respondError(w, errors.Validation("Workspace name is required", nil))
		return
	}

	now := time.Now()
	ws := models.Workspace{
		UUID:      uuid.New(),
		Name:      strings.TrimSpace(req.Name),
		OwnerUUID: user.UUID,
		Slug:      fmt.Sprintf("%s-%s", slugify(req.Name), ws8(uuid.New())),
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := h.db.ExecContext(ctx, `
		INSERT INTO main.workspaces (uuid, name, owner_uuid, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, ws.UUID, ws.Name, ws.OwnerUUID, ws.Slug, ws.CreatedAt, ws.UpdatedAt)
	if err != nil {
		h.respondError(w, errors.Database(err, "create workspace"))
		return
	}

	h.respond(w, http.StatusCreated, ws)
}

// ws8 returns the first 8 hex chars of a UUID for slug uniqueness
func ws8(u uuid.UUID) string {
	return strings.ReplaceAll(u.String(), "-", "")[:8]
}

// GetWorkspace returns a specific workspace the user is a member of
func (h *AuthHandler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	ctx := r.Context()

	wsUUID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.respondError(w, errors.BadRequest("Invalid workspace ID"))
		return
	}

	if _, err := h.memberRole(ctx, wsUUID, user.UUID); err != nil {
		h.respondError(w, err)
		return
	}

	var ws models.Workspace
	err = h.db.QueryRowContext(ctx, `
		SELECT uuid, name, owner_uuid, slug, created_at, updated_at
		FROM main.workspaces WHERE uuid = $1
	`, wsUUID).Scan(&ws.UUID, &ws.Name, &ws.OwnerUUID, &ws.Slug, &ws.CreatedAt, &ws.UpdatedAt)
	if err == sql.ErrNoRows {
		h.respondError(w, errors.NotFound("Workspace"))
		return
	}
	if err != nil {
		h.respondError(w, errors.Database(err, "get workspace"))
		return
	}

	h.respond(w, http.StatusOK, ws)
}

// UpdateWorkspace renames a workspace (owner or admin only)
func (h *AuthHandler) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	ctx := r.Context()

	wsUUID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.respondError(w, errors.BadRequest("Invalid workspace ID"))
		return
	}

	role, err := h.memberRole(ctx, wsUUID, user.UUID)
	if err != nil {
		h.respondError(w, err)
		return
	}
	if role != "owner" && role != "admin" {
		h.respondError(w, errors.Forbidden("Only owners and admins can update a workspace"))
		return
	}

	var req WorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		h.respondError(w, errors.Validation("Workspace name is required", nil))
		return
	}

	var ws models.Workspace
	err = h.db.QueryRowContext(ctx, `
		UPDATE main.workspaces SET name = $1, updated_at = NOW()
		WHERE uuid = $2
		RETURNING uuid, name, owner_uuid, slug, created_at, updated_at
	`, strings.TrimSpace(req.Name), wsUUID).Scan(&ws.UUID, &ws.Name, &ws.OwnerUUID, &ws.Slug, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		h.respondError(w, errors.Database(err, "update workspace"))
		return
	}

	h.respond(w, http.StatusOK, ws)
}

// DeleteWorkspace deletes a workspace (owner only)
func (h *AuthHandler) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	ctx := r.Context()

	wsUUID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.respondError(w, errors.BadRequest("Invalid workspace ID"))
		return
	}

	role, err := h.memberRole(ctx, wsUUID, user.UUID)
	if err != nil {
		h.respondError(w, err)
		return
	}
	if role != "owner" {
		h.respondError(w, errors.Forbidden("Only the owner can delete a workspace"))
		return
	}

	if _, err := h.db.ExecContext(ctx, `DELETE FROM main.workspaces WHERE uuid = $1`, wsUUID); err != nil {
		h.respondError(w, errors.Database(err, "delete workspace"))
		return
	}

	h.respond(w, http.StatusOK, map[string]string{"status": "deleted"})
}
