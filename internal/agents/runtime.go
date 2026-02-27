package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Runner interface {
	Prompt(ctx context.Context, provider Provider, prompt string) (string, error)
}

type HTTPRunner struct {
	Client *http.Client
}

func (r HTTPRunner) Prompt(ctx context.Context, provider Provider, prompt string) (string, error) {
	switch provider.Type {
	case ProviderAnthropic:
		return r.promptAnthropic(ctx, provider, prompt)
	case ProviderGemini:
		return r.promptGemini(ctx, provider, prompt)
	case ProviderOpenAI, ProviderGrok, ProviderLLaMA:
		return r.promptOpenAICompat(ctx, provider, prompt)
	default:
		return "", fmt.Errorf("unsupported provider type %q", provider.Type)
	}
}

func (r HTTPRunner) client() *http.Client {
	if r.Client != nil {
		return r.Client
	}
	return &http.Client{Timeout: 45 * time.Second}
}

func apiKey(provider Provider) (string, error) {
	if provider.APIKey != "" {
		return provider.APIKey, nil
	}
	if provider.APIKeyEnv == "" {
		return "", fmt.Errorf("api key not set")
	}
	val := os.Getenv(provider.APIKeyEnv)
	if val == "" {
		return "", fmt.Errorf("missing api key in %s", provider.APIKeyEnv)
	}
	return val, nil
}

func (r HTTPRunner) promptOpenAICompat(ctx context.Context, provider Provider, prompt string) (string, error) {
	key, err := apiKey(provider)
	if err != nil {
		return "", err
	}
	base := provider.BaseURL
	if base == "" {
		switch provider.Type {
		case ProviderGrok:
			base = "https://api.x.ai"
		default:
			base = "https://api.openai.com"
		}
	}
	reqBody := map[string]any{
		"model": provider.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("provider error: %s", strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return parsed.Choices[0].Message.Content, nil
}

func (r HTTPRunner) promptAnthropic(ctx context.Context, provider Provider, prompt string) (string, error) {
	key, err := apiKey(provider)
	if err != nil {
		return "", err
	}
	base := provider.BaseURL
	if base == "" {
		base = "https://api.anthropic.com"
	}
	reqBody := map[string]any{
		"model":      provider.Model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	resp, err := r.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("provider error: %s", strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	var out strings.Builder
	for _, part := range parsed.Content {
		if part.Type == "text" {
			out.WriteString(part.Text)
		}
	}
	if out.Len() == 0 {
		return "", fmt.Errorf("empty response")
	}
	return out.String(), nil
}

func (r HTTPRunner) promptGemini(ctx context.Context, provider Provider, prompt string) (string, error) {
	key, err := apiKey(provider)
	if err != nil {
		return "", err
	}
	base := provider.BaseURL
	if base == "" {
		base = "https://generativelanguage.googleapis.com/v1beta"
	}
	endpoint := fmt.Sprintf("%s/models/%s:generateContent", strings.TrimRight(base, "/"), provider.Model)
	reqBody := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-goog-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("provider error: %s", strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}
