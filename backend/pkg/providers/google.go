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

// GoogleProvider implements the Provider interface for Google Gemini
type GoogleProvider struct {
	apiKey string
	client *http.Client
	apiURL string
}

// NewGoogleProvider creates a new Google Gemini provider
func NewGoogleProvider(apiKey string) *GoogleProvider {
	return &GoogleProvider{
		apiKey: apiKey,
		client: &http.Client{},
		apiURL: "https://generativelanguage.googleapis.com/v1beta",
	}
}

// Name returns the provider name
func (p *GoogleProvider) Name() string {
	return "google"
}

// CountTokens estimates token count for Google models
func (p *GoogleProvider) CountTokens(text string, model string) (int, error) {
	// Rough estimation: ~4 characters per token
	return len(text) / 4, nil
}

// CalculateCost calculates cost for Google models
func (p *GoogleProvider) CalculateCost(model string, inputTokens, outputTokens int) (float64, error) {
	pricing, err := p.getModelPricing(model)
	if err != nil {
		return 0, err
	}

	inputCost := float64(inputTokens) * pricing.InputPricePerToken
	outputCost := float64(outputTokens) * pricing.OutputPricePerToken
	
	return inputCost + outputCost, nil
}

// Chat sends a chat completion request to Google Gemini
func (p *GoogleProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Convert to Google format
	var contents []map[string]interface{}
	
	for _, msg := range req.Messages {
		role := msg.Role
		// Google uses "user" and "model" roles, not "assistant"
		if role == "assistant" {
			role = "model"
		}
		// System messages are not directly supported, prepend to first user message
		if role == "system" {
			if len(contents) > 0 {
				// Prepend to existing first message
				if firstMsg, ok := contents[0]["parts"].([]map[string]string); ok && len(firstMsg) > 0 {
					firstMsg[0]["text"] = msg.Content + "\n\n" + firstMsg[0]["text"]
				}
			}
			continue
		}
		
		contents = append(contents, map[string]interface{}{
			"role": role,
			"parts": []map[string]string{
				{"text": msg.Content},
			},
		})
	}

	googleReq := map[string]interface{}{
		"contents": contents,
	}

	// Add generation config
	generationConfig := make(map[string]interface{})
	if req.Temperature != nil {
		generationConfig["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		generationConfig["maxOutputTokens"] = *req.MaxTokens
	}
	if req.TopP != nil {
		generationConfig["topP"] = *req.TopP
	}
	if len(generationConfig) > 0 {
		googleReq["generationConfig"] = generationConfig
	}

	// Marshal request
	reqBody, err := json.Marshal(googleReq)
	if err != nil {
		return nil, &ProviderError{
			Provider: "google",
			Message:  "failed to marshal request",
			Cause:    err,
		}
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.apiURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, &ProviderError{
			Provider: "google",
			Message:  "failed to create request",
			Cause:    err,
		}
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{
			Provider: "google",
			Message:  "request failed",
			Cause:    err,
		}
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{
			Provider: "google",
			Message:  "failed to read response",
			Cause:    err,
		}
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return nil, &ProviderError{
			Provider: "google",
			Message:  fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)),
		}
	}

	// Parse response
	var googleResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &googleResp); err != nil {
		return nil, &ProviderError{
			Provider: "google",
			Message:  "failed to parse response",
			Cause:    err,
		}
	}

	if len(googleResp.Candidates) == 0 {
		return nil, &ProviderError{
			Provider: "google",
			Message:  "no completion candidates returned",
		}
	}

	// Extract text content
	var content string
	for _, part := range googleResp.Candidates[0].Content.Parts {
		content += part.Text
	}

	// Calculate cost
	cost, _ := p.CalculateCost(
		req.Model,
		googleResp.UsageMetadata.PromptTokenCount,
		googleResp.UsageMetadata.CandidatesTokenCount,
	)

	// Convert to standard format
	return &ChatResponse{
		ID:           "", // Google doesn't provide an ID
		Model:        req.Model,
		Content:      content,
		InputTokens:  googleResp.UsageMetadata.PromptTokenCount,
		OutputTokens: googleResp.UsageMetadata.CandidatesTokenCount,
		TotalTokens:  googleResp.UsageMetadata.TotalTokenCount,
		Cost:         cost,
		FinishReason: googleResp.Candidates[0].FinishReason,
	}, nil
}

// ValidateModel checks if the model is valid for Google
func (p *GoogleProvider) ValidateModel(model string) bool {
	validModels := map[string]bool{
		"gemini-pro":        true,
		"gemini-pro-vision": true,
		"gemini-ultra":      true,
	}
	
	return validModels[model]
}

// getModelPricing returns pricing for a model
func (p *GoogleProvider) getModelPricing(model string) (*ModelPricing, error) {
	// Pricing as of Jan 2026 (prices per token)
	pricing := map[string]ModelPricing{
		"gemini-pro": {
			InputPricePerToken:  0.00000025, // $0.25 per 1M tokens
			OutputPricePerToken: 0.0000005,  // $0.50 per 1M tokens
		},
		"gemini-pro-vision": {
			InputPricePerToken:  0.00000025, // $0.25 per 1M tokens
			OutputPricePerToken: 0.0000005,  // $0.50 per 1M tokens
		},
		"gemini-ultra": {
			// Pricing TBD
			InputPricePerToken:  0.000001,   // Estimated
			OutputPricePerToken: 0.000002,   // Estimated
		},
	}

	if p, ok := pricing[model]; ok {
		return &p, nil
	}

	return nil, fmt.Errorf("unknown model: %s", model)
}
