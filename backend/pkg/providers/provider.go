// Package providers implements the abstraction layer for AI providers
package providers

import (
	"context"
	"fmt"
)

// Provider represents an AI provider interface
type Provider interface {
	// Name returns the provider name (e.g., "openai", "anthropic")
	Name() string
	
	// CountTokens estimates token count for the given text and model
	CountTokens(text string, model string) (int, error)
	
	// CalculateCost calculates the cost for the given token usage
	CalculateCost(model string, inputTokens, outputTokens int) (float64, error)
	
	// Chat sends a chat completion request
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	
	// ValidateModel checks if the model name is valid for this provider
	ValidateModel(model string) bool
}

// ChatRequest represents a standardized chat request
type ChatRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// ChatResponse represents a standardized chat response
type ChatResponse struct {
	ID           string          `json:"id"`
	Model        string          `json:"model"`
	Content      string          `json:"content"`
	InputTokens  int             `json:"input_tokens"`
	OutputTokens int             `json:"output_tokens"`
	TotalTokens  int             `json:"total_tokens"`
	Cost         float64         `json:"cost"`
	FinishReason string          `json:"finish_reason"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ModelPricing represents pricing for a model
type ModelPricing struct {
	InputPricePerToken  float64 `json:"input_price_per_token"`  // Price per input token
	OutputPricePerToken float64 `json:"output_price_per_token"` // Price per output token
}

// ProviderRegistry manages all available providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider
func (r *ProviderRegistry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// List returns all registered provider names
func (r *ProviderRegistry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// ProviderError represents a provider-specific error
type ProviderError struct {
	Provider string
	Message  string
	Cause    error
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s provider error: %s: %v", e.Provider, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s provider error: %s", e.Provider, e.Message)
}

// Unwrap implements error unwrapping
func (e *ProviderError) Unwrap() error {
	return e.Cause
}
