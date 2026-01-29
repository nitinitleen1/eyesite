// Package models defines core data structures
package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account
type User struct {
	UUID          uuid.UUID  `json:"uuid" db:"uuid"`
	Email         string     `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"` // Never expose in JSON
	OAuthProvider *string    `json:"oauth_provider,omitempty" db:"oauth_provider"`
	OAuthID       *string    `json:"oauth_id,omitempty" db:"oauth_id"`
	FullName      string     `json:"full_name" db:"full_name"`
	IsVerified    bool       `json:"is_verified" db:"is_verified"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// Workspace represents a multi-tenant workspace
type Workspace struct {
	UUID      uuid.UUID `json:"uuid" db:"uuid"`
	Name      string    `json:"name" db:"name"`
	OwnerUUID uuid.UUID `json:"owner_uuid" db:"owner_uuid"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// WorkspaceMember represents a user's membership in a workspace
type WorkspaceMember struct {
	WorkspaceUUID uuid.UUID `json:"workspace_uuid" db:"workspace_uuid"`
	UserUUID      uuid.UUID `json:"user_uuid" db:"user_uuid"`
	Role          string    `json:"role" db:"role"` // owner, admin, developer, viewer
	JoinedAt      time.Time `json:"joined_at" db:"joined_at"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	UUID          uuid.UUID  `json:"uuid" db:"uuid"`
	UserUUID      uuid.UUID  `json:"user_uuid" db:"user_uuid"`
	WorkspaceUUID uuid.UUID  `json:"workspace_uuid" db:"workspace_uuid"`
	Name          string     `json:"name" db:"name"`
	KeyHash       string     `json:"-" db:"key_hash"` // Never expose the hash
	KeyPreview    string     `json:"key_preview" db:"key_preview"` // First 8 chars for display
	Scopes        []string   `json:"scopes" db:"scopes"` // read, write, admin
	LastUsedAt    *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// IsRevoked checks if the API key has been revoked
func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid checks if the API key is valid (not revoked and not expired)
func (k *APIKey) IsValid() bool {
	return !k.IsRevoked() && !k.IsExpired()
}

// Session represents a user session for tracking interactions
type Session struct {
	UUID          uuid.UUID              `json:"uuid" db:"uuid"`
	UserUUID      uuid.UUID              `json:"user_uuid" db:"user_uuid"`
	WorkspaceUUID uuid.UUID              `json:"workspace_uuid" db:"workspace_uuid"`
	APIKeyUUID    *uuid.UUID             `json:"api_key_uuid,omitempty" db:"api_key_uuid"`
	Name          string                 `json:"name" db:"name"`
	Description   string                 `json:"description" db:"description"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// Provider represents an AI provider (OpenAI, Anthropic, etc.)
type Provider struct {
	UUID        uuid.UUID              `json:"uuid" db:"uuid"`
	Name        string                 `json:"name" db:"name"` // openai, anthropic, google
	DisplayName string                 `json:"display_name" db:"display_name"`
	IsActive    bool                   `json:"is_active" db:"is_active"`
	Pricing     map[string]interface{} `json:"pricing" db:"pricing"` // Model-specific pricing
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// Interaction represents a single LLM interaction
type Interaction struct {
	UUID           uuid.UUID              `json:"uuid" db:"uuid"`
	SessionUUID    *uuid.UUID             `json:"session_uuid,omitempty" db:"session_uuid"`
	WorkspaceUUID  uuid.UUID              `json:"workspace_uuid" db:"workspace_uuid"`
	TraceID        string                 `json:"trace_id" db:"trace_id"`
	SpanID         string                 `json:"span_id" db:"span_id"`
	Provider       string                 `json:"provider" db:"provider"`
	Model          string                 `json:"model" db:"model"`
	Prompt         string                 `json:"prompt" db:"prompt"`
	Response       string                 `json:"response" db:"response"`
	PromptTokens   int                    `json:"prompt_tokens" db:"prompt_tokens"`
	ResponseTokens int                    `json:"response_tokens" db:"response_tokens"`
	TotalTokens    int                    `json:"total_tokens" db:"total_tokens"`
	Cost           float64                `json:"cost" db:"cost"` // In USD
	LatencyMs      int                    `json:"latency_ms" db:"latency_ms"`
	Status         string                 `json:"status" db:"status"` // success, error
	ErrorMessage   *string                `json:"error_message,omitempty" db:"error_message"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

// Trace represents an OpenTelemetry trace
type Trace struct {
	TraceID       string                 `json:"trace_id" db:"trace_id"`
	UserUUID      uuid.UUID              `json:"user_uuid" db:"user_uuid"`
	WorkspaceUUID uuid.UUID              `json:"workspace_uuid" db:"workspace_uuid"`
	ServiceName   string                 `json:"service_name" db:"service_name"`
	StartTime     time.Time              `json:"start_time" db:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty" db:"end_time"`
	DurationMs    *int                   `json:"duration_ms,omitempty" db:"duration_ms"`
	Status        string                 `json:"status" db:"status"` // ok, error
	Attributes    map[string]interface{} `json:"attributes" db:"attributes"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// Span represents an OpenTelemetry span
type Span struct {
	SpanID       string                 `json:"span_id" db:"span_id"`
	TraceID      string                 `json:"trace_id" db:"trace_id"`
	ParentSpanID *string                `json:"parent_span_id,omitempty" db:"parent_span_id"`
	Name         string                 `json:"name" db:"name"`
	Kind         string                 `json:"kind" db:"kind"` // internal, server, client
	StartTime    time.Time              `json:"start_time" db:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty" db:"end_time"`
	DurationMs   *int                   `json:"duration_ms,omitempty" db:"duration_ms"`
	Status       string                 `json:"status" db:"status"`
	Attributes   map[string]interface{} `json:"attributes" db:"attributes"`
	Events       []SpanEvent            `json:"events,omitempty" db:"events"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// TokenCount represents token usage
type TokenCount struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

// Calculate computes the total
func (tc *TokenCount) Calculate() {
	tc.Total = tc.Input + tc.Output
}

// Helper functions for UUID handling

// NewUUID generates a new UUID
func NewUUID() uuid.UUID {
	return uuid.New()
}

// ParseUUID parses a UUID string
func ParseUUID(s string) uuid.UUID {
	u, _ := uuid.Parse(s)
	return u
}

