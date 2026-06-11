package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude
type AnthropicProvider struct {
	apiKey string
	client *http.Client
	apiURL string
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		client: &http.Client{},
		apiURL: "https://api.anthropic.com/v1",
	}
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// CountTokens estimates token count for Anthropic models
func (p *AnthropicProvider) CountTokens(text string, model string) (int, error) {
	// Rough estimation: ~4 characters per token
	// Anthropic uses similar tokenization to OpenAI
	return len(text) / 4, nil
}

// CalculateCost calculates cost for Anthropic models
func (p *AnthropicProvider) CalculateCost(model string, inputTokens, outputTokens int) (float64, error) {
	pricing, err := p.getModelPricing(model)
	if err != nil {
		return 0, err
	}

	inputCost := float64(inputTokens) * pricing.InputPricePerToken
	outputCost := float64(outputTokens) * pricing.OutputPricePerToken
	
	return inputCost + outputCost, nil
}

// Chat sends a chat completion request to Anthropic
func (p *AnthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert to Anthropic format
	// Anthropic requires system messages to be separate
	var systemMessage string
	var messages []map[string]string
	
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
		} else {
			messages = append(messages, map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	anthropicReq := map[string]interface{}{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": 4096, // Default max tokens
	}

	if systemMessage != "" {
		anthropicReq["system"] = systemMessage
	}

	if req.Temperature != nil {
		anthropicReq["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		anthropicReq["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		anthropicReq["top_p"] = *req.TopP
	}

	// Marshal request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, &ProviderError{
			Provider: "anthropic",
			Message:  "failed to marshal request",
			Cause:    err,
		}
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.apiURL+"/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, &ProviderError{
			Provider: "anthropic",
			Message:  "failed to create request",
			Cause:    err,
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{
			Provider: "anthropic",
			Message:  "request failed",
			Cause:    err,
		}
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{
			Provider: "anthropic",
			Message:  "failed to read response",
			Cause:    err,
		}
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return nil, &ProviderError{
			Provider: "anthropic",
			Message:  fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
		}
	}

	// Parse response
	var anthropicResp struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Content      []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, &ProviderError{
			Provider: "anthropic",
			Message:  "failed to parse response",
			Cause:    err,
		}
	}

	if len(anthropicResp.Content) == 0 {
		return nil, &ProviderError{
			Provider: "anthropic",
			Message:  "no content returned",
		}
	}

	// Extract text content
	var content string
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	// Calculate cost
	cost, _ := p.CalculateCost(req.Model, anthropicResp.Usage.InputTokens, anthropicResp.Usage.OutputTokens)
	totalTokens := anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens

	// Convert to standard format
	return &ChatResponse{
		ID:           anthropicResp.ID,
		Model:        anthropicResp.Model,
		Content:      content,
		InputTokens:  anthropicResp.Usage.InputTokens,
		OutputTokens: anthropicResp.Usage.OutputTokens,
		TotalTokens:  totalTokens,
		Cost:         cost,
		FinishReason: anthropicResp.StopReason,
	}, nil
}

// ValidateModel checks if the model is valid for Anthropic
func (p *AnthropicProvider) ValidateModel(model string) bool {
	validModels := map[string]bool{
		"claude-3-opus-20240229":   true,
		"claude-3-sonnet-20240229": true,
		"claude-3-haiku-20240307":  true,
		"claude-2.1":               true,
		"claude-2.0":               true,
	}
	
	return validModels[model]
}

// getModelPricing returns pricing for a model
func (p *AnthropicProvider) getModelPricing(model string) (*ModelPricing, error) {
	// Pricing as of Jan 2026 (prices per token)
	pricing := map[string]ModelPricing{
		"claude-3-opus-20240229": {
			InputPricePerToken:  0.000015,  // $15 per 1M tokens
			OutputPricePerToken: 0.000075,  // $75 per 1M tokens
		},
		"claude-3-sonnet-20240229": {
			InputPricePerToken:  0.000003,  // $3 per 1M tokens
			OutputPricePerToken: 0.000015,  // $15 per 1M tokens
		},
		"claude-3-haiku-20240307": {
			InputPricePerToken:  0.00000025, // $0.25 per 1M tokens
			OutputPricePerToken: 0.00000125, // $1.25 per 1M tokens
		},
		"claude-2.1": {
			InputPricePerToken:  0.000008,  // $8 per 1M tokens
			OutputPricePerToken: 0.000024,  // $24 per 1M tokens
		},
		"claude-2.0": {
			InputPricePerToken:  0.000008,  // $8 per 1M tokens
			OutputPricePerToken: 0.000024,  // $24 per 1M tokens
		},
	}

	if p, ok := pricing[model]; ok {
		return &p, nil
	}

	return nil, fmt.Errorf("unknown model: %s", model)
}
