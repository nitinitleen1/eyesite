package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/eyesite/platform/backend/internal/errors"
	"github.com/eyesite/platform/backend/pkg/config"
	"github.com/eyesite/platform/backend/pkg/database"
	"github.com/eyesite/platform/backend/pkg/models"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	db     *database.DB
	cfg    *config.Config
	logger *zap.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *database.DB, cfg *config.Config, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		db:     db,
		cfg:    cfg,
		logger: logger,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse represents an authentication token response
type TokenResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         *models.User `json:"user"`
}

//Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}

	// Validate input
	if err := h.validateRegisterRequest(&req); err != nil {
		h.respondError(w, err)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("failed to hash password", zap.Error(err))
		h.respondError(w, errors.Internal(err, "Failed to process password"))
		return
	}

	// Create user
	userUUID := uuid.New()
	nowTime := time.Now()
	
	user := &models.User{
		UUID:         userUUID,
		Email:        strings.ToLower(req.Email),
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		IsVerified:   false,
		CreatedAt:    nowTime,
		UpdatedAt:    nowTime,
	}

	// Insert user and create default workspace in a transaction
	err = h.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Insert user
		_, err := tx.ExecContext(ctx, `
			INSERT INTO main.users (uuid, email, password_hash, full_name, is_verified, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, user.UUID, user.Email, user.PasswordHash, user.FullName, user.IsVerified, user.CreatedAt, user.UpdatedAt)
		
		if err != nil {
			// Check for duplicate email
			if strings.Contains(err.Error(), "unique") {
				return errors.Conflict("Email already registered")
			}
			return errors.Database(err, "insert user")
		}

		// Create default workspace (random suffix keeps slugs unique across users)
		workspaceUUID := uuid.New()
		workspaceSlug := fmt.Sprintf("%s-workspace-%s", slugify(strings.Split(user.Email, "@")[0]), ws8(uuid.New()))
		
		_, err = tx.ExecContext(ctx, `
			INSERT INTO main.workspaces (uuid, name, owner_uuid, slug, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, workspaceUUID, "Default Workspace", user.UUID, workspaceSlug, nowTime, nowTime)
		
		if err != nil {
			return errors.Database(err, "create workspace")
		}

		return nil
	})

	if err != nil {
		h.respondError(w, err)
		return
	}

	// Generate tokens
	token, refreshToken, expiresAt, err := h.generateTokens(user)
	if err != nil {
		h.respondError(w, errors.Internal(err, "Failed to generate tokens"))
		return
	}

	// Clear password hash before sending
	user.PasswordHash = ""

	h.respond(w, http.StatusCreated, TokenResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         user,
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, errors.BadRequest("Invalid request body"))
		return
	}

	// Find user by email
	var user models.User
	err := h.db.QueryRowContext(ctx, `
		SELECT uuid, email, password_hash, full_name, is_verified, created_at, updated_at, last_login_at
		FROM main.users
		WHERE email = $1 AND oauth_provider IS NULL
	`, strings.ToLower(req.Email)).Scan(
		&user.UUID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err == sql.ErrNoRows {
		h.respondError(w, errors.Unauthorized("Invalid email or password"))
		return
	}
	if err != nil {
		h.logger.Error("database error during login", zap.Error(err))
		h.respondError(w, errors.Database(err, "login"))
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.respondError(w, errors.Unauthorized("Invalid email or password"))
		return
	}

	// Update last login time
	now := time.Now()
	_, err = h.db.ExecContext(ctx, `
		UPDATE main.users SET last_login_at = $1 WHERE uuid = $2
	`, now, user.UUID)
	if err != nil {
		h.logger.Warn("failed to update last login time", zap.Error(err))
	}

	// Generate tokens
	token, refreshToken, expiresAt, err := h.generateTokens(&user)
	if err != nil {
		h.respondError(w, errors.Internal(err, "Failed to generate tokens"))
		return
	}

	// Clear password hash before sending
	user.PasswordHash = ""
	user.LastLoginAt = &now

	h.respond(w, http.StatusOK, TokenResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         &user,
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement refresh token logic
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "Refresh token not implemented yet", http.StatusNotImplemented))
}

// GoogleOAuth handles Google OAuth login
func (h *AuthHandler) GoogleOAuth(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement Google OAuth
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "Google OAuth not implemented yet", http.StatusNotImplemented))
}

// GoogleOAuthCallback handles Google OAuth callback
func (h *AuthHandler) GoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement Google OAuth callback
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "Google OAuth callback not implemented yet", http.StatusNotImplemented))
}

// GitHubOAuth handles GitHub OAuth login
func (h *AuthHandler) GitHubOAuth(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement GitHub OAuth
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "GitHub OAuth not implemented yet", http.StatusNotImplemented))
}

// GitHubOAuthCallback handles GitHub OAuth callback
func (h *AuthHandler) GitHubOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement GitHub OAuth callback
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "GitHub OAuth callback not implemented yet", http.StatusNotImplemented))
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*models.User)
	user.PasswordHash = "" // Never expose password hash
	h.respond(w, http.StatusOK, user)
}

// UpdateCurrentUser updates the current user's profile
func (h *AuthHandler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement user update
	h.respondError(w, errors.New("NOT_IMPLEMENTED", "User update not implemented yet", http.StatusNotImplemented))
}

// AuthMiddleware validates JWT tokens and adds user to context
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
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

		if err != nil {
			h.respondError(w, errors.Unauthorized("Invalid or expired token"))
			return
		}

		if !token.Valid {
			h.respondError(w, errors.Unauthorized("Invalid token"))
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

		// Load user from database
		var user models.User
		err = h.db.QueryRowContext(r.Context(), `
			SELECT uuid, email, full_name, is_verified, created_at, updated_at, last_login_at
			FROM main.users
			WHERE uuid = $1
		`, userUUID).Scan(
			&user.UUID,
			&user.Email,
			&user.FullName,
			&user.IsVerified,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastLoginAt,
		)

		if err == sql.ErrNoRows {
			h.respondError(w, errors.Unauthorized("User not found"))
			return
		}
		if err != nil {
			h.logger.Error("failed to load user", zap.Error(err))
			h.respondError(w, errors.Internal(err, "Failed to authenticate"))
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), "user", &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper methods

func (h *AuthHandler) validateRegisterRequest(req *RegisterRequest) error {
	if req.Email == "" {
		return errors.Validation("Email is required", nil)
	}
	if !strings.Contains(req.Email, "@") {
		return errors.Validation("Invalid email format", nil)
	}
	if req.Password == "" {
		return errors.Validation("Password is required", nil)
	}
	if len(req.Password) < 8 {
		return errors.Validation("Password must be at least 8 characters", nil)
	}
	if req.FullName == "" {
		return errors.Validation("Full name is required", nil)
	}
	return nil
}

func (h *AuthHandler) generateTokens(user *models.User) (string, string, time.Time, error) {
	expiresAt := time.Now().Add(time.Hour * time.Duration(h.cfg.JWT.ExpirationHours))
	
	// Create JWT claims
	claims := jwt.MapClaims{
		"sub":   user.UUID.String(),
		"email": user.Email,
		"iss":   h.cfg.JWT.Issuer,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.cfg.JWT.Secret))
	if err != nil {
		return "", "", time.Time{}, err
	}

	// TODO: Generate proper refresh token
	refreshToken := tokenString // Temporary - use same token

	return tokenString, refreshToken, expiresAt, nil
}

func (h *AuthHandler) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *AuthHandler) respondError(w http.ResponseWriter, err error) {
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
