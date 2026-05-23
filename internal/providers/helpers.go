package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const defaultMaxOutputTokens = 8192
const responseSnippetLength = 500

const (
	defaultOpenAIBaseURL    = "https://api.openai.com/v1"
	defaultAnthropicBaseURL = "https://api.anthropic.com/v1"
	defaultGeminiBaseURL    = "https://generativelanguage.googleapis.com"
)

func resolveHTTPClient(cfg ProviderConfig) HTTPDoer {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return defaultHTTPClient()
}

func resolveMaxOutputTokens(cfg ProviderConfig) int {
	if cfg.MaxOutputTokens != nil && *cfg.MaxOutputTokens > 0 {
		return *cfg.MaxOutputTokens
	}
	return defaultMaxOutputTokens
}

func joinURL(base, path string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(base), "/")
	trimmedPath := "/" + strings.TrimLeft(path, "/")
	return trimmedBase + trimmedPath
}

func validateProviderConfig(cfg ProviderConfig, requireBaseURL bool) error {
	provider := string(cfg.Provider)
	if cfg.APIKey == nil || strings.TrimSpace(*cfg.APIKey) == "" {
		return MissingAPIKey(provider)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return MissingConfig(provider, "model")
	}
	if requireBaseURL && (cfg.BaseURL == nil || strings.TrimSpace(*cfg.BaseURL) == "") {
		return MissingConfig(provider, "baseURL")
	}
	return nil
}

func optionalStringOrDefault(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return strings.TrimSpace(*value)
}

func applyUserAgent(request *http.Request, cfg ProviderConfig) {
	if strings.TrimSpace(cfg.UserAgent) != "" {
		request.Header.Set("User-Agent", cfg.UserAgent)
	}
}

func responseSnippet(body []byte) string {
	serialized := strings.Join(strings.Fields(string(body)), " ")
	runes := []rune(serialized)
	if len(runes) > responseSnippetLength {
		return string(runes[:responseSnippetLength]) + "…"
	}
	if serialized == "" {
		return "<empty>"
	}
	return serialized
}

func executeJSONRequest(client HTTPDoer, request *http.Request, payload any, destination any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request.Body = ioNopCloserFromBytes(body)
	request.ContentLength = int64(len(body))
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	responseBody, err := readResponseBody(response)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", response.StatusCode, responseSnippet(responseBody))
	}
	if err := json.Unmarshal(responseBody, destination); err != nil {
		return fmt.Errorf("decode response: %w; snippet: %s", err, responseSnippet(responseBody))
	}
	return nil
}
