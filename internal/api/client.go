package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Provider string

const (
	OpenAIChat      Provider = "openai-chat"
	OpenAIResponse  Provider = "openai-response"
	AnthropicMessage Provider = "anthropic-message"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Provider   Provider
	APIKey     string
	APIURL     string
	Model      string
	System     string
	Prompt     string
	MaxTokens  int
	Timeout    time.Duration
}

func (r Request) Send() (string, error) {
	switch r.Provider {
	case OpenAIChat:
		return openAIChat(r)
	case OpenAIResponse:
		return openAIResponse(r)
	case AnthropicMessage:
		return anthropicMessage(r)
	default:
		return "", fmt.Errorf("unknown provider: %s", r.Provider)
	}
}

// ── OpenAI Chat Completions ────────────────────────────────────────────

type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func openAIChat(r Request) (string, error) {
	body := chatRequest{
		Model: r.Model,
		Messages: []Message{
			{Role: "system", Content: r.System},
			{Role: "user", Content: r.Prompt},
		},
		MaxTokens: r.MaxTokens,
	}
	resp, err := postJSON(r, "/v1/chat/completions", body)
	if err != nil {
		return "", err
	}
	var cr chatResponse
	if err := json.Unmarshal(resp, &cr); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if cr.Error != nil {
		return "", fmt.Errorf("api error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}

// ── OpenAI Responses ───────────────────────────────────────────────────

type responseRequest struct {
	Model     string    `json:"model"`
	Input     []Message `json:"input"`
	MaxTokens int       `json:"max_output_tokens,omitempty"`
}

type responseResponse struct {
	Output []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func openAIResponse(r Request) (string, error) {
	body := responseRequest{
		Model: r.Model,
		Input: []Message{
			{Role: "system", Content: r.System},
			{Role: "user", Content: r.Prompt},
		},
		MaxTokens: r.MaxTokens,
	}
	resp, err := postJSON(r, "/v1/responses", body)
	if err != nil {
		return "", err
	}
	var rr responseResponse
	if err := json.Unmarshal(resp, &rr); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if rr.Error != nil {
		return "", fmt.Errorf("api error: %s", rr.Error.Message)
	}
	var parts []string
	for _, o := range rr.Output {
		for _, c := range o.Content {
			if c.Text != "" {
				parts = append(parts, c.Text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}

// ── Anthropic Messages ─────────────────────────────────────────────────

type anthropicRequest struct {
	Model     string    `json:"model"`
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func anthropicMessage(r Request) (string, error) {
	body := anthropicRequest{
		Model:     r.Model,
		System:    r.System,
		Messages:  []Message{{Role: "user", Content: r.Prompt}},
		MaxTokens: r.MaxTokens,
	}
	url := strings.TrimRight(r.APIURL, "/") + "/v1/messages"
	resp, err := doPost(r.APIKey, url, body, r.Timeout)
	if err != nil {
		return "", err
	}
	var ar anthropicResponse
	if err := json.Unmarshal(resp, &ar); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if ar.Error != nil {
		return "", fmt.Errorf("api error: %s", ar.Error.Message)
	}
	var parts []string
	for _, c := range ar.Content {
		if c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}

// ── shared HTTP helper ─────────────────────────────────────────────────

func postJSON(r Request, path string, body interface{}) ([]byte, error) {
	url := strings.TrimRight(r.APIURL, "/") + path
	return doPost(r.APIKey, url, body, r.Timeout)
}

func doPost(apiKey, url string, body interface{}, timeout time.Duration) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("x-api-key", apiKey) // Anthropic uses this
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 500)]))
	}
	return respBody, nil
}
