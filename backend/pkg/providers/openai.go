package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	apiKey string
	client *http.Client
	apiURL string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: apiKey,
		client: &http.Client{},
		apiURL: "https://api.openai.com/v1",
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// CountTokens estimates token count for OpenAI models
// This is a rough estimation. For production, use tiktoken library
func (p *OpenAIProvider) CountTokens(text string, model string) (int, error) {
	// Rough estimation: ~4 characters per token for English
	// For production, integrate with tiktoken
	return len(text) / 4, nil
}

// CalculateCost calculates cost for OpenAI models
func (p *OpenAIProvider) CalculateCost(model string, inputTokens, outputTokens int) (float64, error) {
	pricing, err := p.getModelPricing(model)
	if err != nil {
		return 0, err
	}

	inputCost := float64(inputTokens) * pricing.InputPricePerToken
	outputCost := float64(outputTokens) * pricing.OutputPricePerToken
	
	return inputCost + outputCost, nil
}

// Chat sends a chat completion request to OpenAI
func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert to OpenAI format
	openAIReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
	}

	if req.Temperature != nil {
		openAIReq["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		openAIReq["max_tokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		openAIReq["top_p"] = *req.TopP
	}

	// Marshal request
	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, &ProviderError{
			Provider: "openai",
			Message:  "failed to marshal request",
			Cause:    err,
		}
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.apiURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, &ProviderError{
			Provider: "openai",
			Message:  "failed to create request",
			Cause:    err,
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{
			Provider: "openai",
			Message:  "request failed",
			Cause:    err,
		}
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{
			Provider: "openai",
			Message:  "failed to read response",
			Cause:    err,
		}
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return nil, &ProviderError{
			Provider: "openai",
			Message:  fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
		}
	}

	// Parse response
	var openAIResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, &ProviderError{
			Provider: "openai",
			Message:  "failed to parse response",
			Cause:    err,
		}
	}

	if len(openAIResp.Choices) == 0 {
		return nil, &ProviderError{
			Provider: "openai",
			Message:  "no completion choices returned",
		}
	}

	// Calculate cost
	cost, _ := p.CalculateCost(req.Model, openAIResp.Usage.PromptTokens, openAIResp.Usage.CompletionTokens)

	// Convert to standard format
	return &ChatResponse{
		ID:           openAIResp.ID,
		Model:        openAIResp.Model,
		Content:      openAIResp.Choices[0].Message.Content,
		InputTokens:  openAIResp.Usage.PromptTokens,
		OutputTokens: openAIResp.Usage.CompletionTokens,
		TotalTokens:  openAIResp.Usage.TotalTokens,
		Cost:         cost,
		FinishReason: openAIResp.Choices[0].FinishReason,
	}, nil
}

// ValidateModel checks if the model is valid for OpenAI
func (p *OpenAIProvider) ValidateModel(model string) bool {
	validModels := map[string]bool{
		"gpt-4":              true,
		"gpt-4-turbo":        true,
		"gpt-4-turbo-preview": true,
		"gpt-3.5-turbo":      true,
		"gpt-3.5-turbo-16k":  true,
	}
	
	// Also accept versioned models (e.g., gpt-4-0613)
	for validModel := range validModels {
		if strings.HasPrefix(model, validModel) {
			return true
		}
	}
	
	return validModels[model]
}

// getModelPricing returns pricing for a model
func (p *OpenAIProvider) getModelPricing(model string) (*ModelPricing, error) {
	// Pricing as of Jan 2026 (prices per token)
	pricing := map[string]ModelPricing{
		"gpt-4": {
			InputPricePerToken:  0.00003,  // $0.03 per 1K tokens
			OutputPricePerToken: 0.00006,  // $0.06 per 1K tokens
		},
		"gpt-4-turbo": {
			InputPricePerToken:  0.00001,  // $0.01 per 1K tokens
			OutputPricePerToken: 0.00003,  // $0.03 per 1K tokens
		},
		"gpt-3.5-turbo": {
			InputPricePerToken:  0.0000015, // $0.0015 per 1K tokens
			OutputPricePerToken: 0.000002,  // $0.002 per 1K tokens
		},
	}

	// Try exact match first
	if p, ok := pricing[model]; ok {
		return &p, nil
	}

	// Try prefix match for versioned models
	for modelPrefix, p := range pricing {
		if strings.HasPrefix(model, modelPrefix) {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("unknown model: %s", model)
}
