package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/eyesite/platform/backend/internal/errors"
)

// userUUIDContextKey is the context key under which the authenticated user UUID is stored
const userUUIDContextKey = "user_uuid"

// AuthMiddleware validates JWT tokens and adds the user UUID to the request context
func (h *QueryHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.respondError(w, errors.Unauthorized("Missing authorization header"))
			return
		}

		// Parse "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			h.respondError(w, errors.Unauthorized("Invalid authorization header format"))
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(h.cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			h.respondError(w, errors.Unauthorized("Invalid or expired token"))
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			h.respondError(w, errors.Unauthorized("Invalid token claims"))
			return
		}

		// Get user UUID from claims
		userUUIDStr, ok := claims["sub"].(string)
		if !ok {
			h.respondError(w, errors.Unauthorized("Invalid user ID in token"))
			return
		}

		userUUID, err := uuid.Parse(userUUIDStr)
		if err != nil {
			h.respondError(w, errors.Unauthorized("Invalid user ID format"))
			return
		}

		// Add user UUID to request context
		ctx := context.WithValue(r.Context(), userUUIDContextKey, userUUID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireWorkspaceMember verifies the authenticated user is a member of the workspace
func (h *QueryHandler) requireWorkspaceMember(r *http.Request, workspaceUUID string) error {
	userUUID, ok := r.Context().Value(userUUIDContextKey).(uuid.UUID)
	if !ok {
		return errors.Unauthorized("Not authenticated")
	}

	wsUUID, err := uuid.Parse(workspaceUUID)
	if err != nil {
		return errors.BadRequest("Invalid workspace_uuid")
	}

	var isMember bool
	err = h.db.QueryRowContext(r.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM main.workspace_members
			WHERE workspace_uuid = $1 AND user_uuid = $2
		)
	`, wsUUID, userUUID).Scan(&isMember)
	if err != nil {
		return errors.Database(err, "check workspace membership")
	}
	if !isMember {
		return errors.Forbidden("Not a member of this workspace")
	}
	return nil
}
