package providers

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	runtimex "github.com/ByteTrue/commit-now-myfriend/internal/runtime"
)

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestToolCallProviderUsesOpenAIResponsesNativeToolLoop(t *testing.T) {
	apiKey := "test-key"
	requests := []string{}
	provider, err := CreateToolCallProvider(ToolCallProviderOptions{
		Config: ProviderConfig{
			Provider: config.ProviderOpenAIResponses,
			APIKey:   &apiKey,
			Model:    "responses-model",
			HTTPClient: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				body, _ := io.ReadAll(request.Body)
				requests = append(requests, string(body))
				switch len(requests) {
				case 1:
					if request.URL.String() != "https://api.openai.com/v1/responses" {
						t.Fatalf("unexpected URL: %s", request.URL.String())
					}
					if request.Header.Get("Authorization") != "Bearer test-key" {
						t.Fatalf("missing auth header")
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","output":[{"type":"function_call","call_id":"read","name":"read_file","arguments":"{\"path\":\"conflict.txt\"}"}]}`)), Header: make(http.Header)}, nil
				case 2:
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"resp_2","output":[{"type":"function_call","call_id":"finish","name":"finish","arguments":"{\"message\":\"repaired\"}"}]}`)), Header: make(http.Header)}, nil
				default:
					t.Fatalf("unexpected extra request: %s", string(body))
					return nil, nil
				}
			}),
		},
		Instructions: "Use read-before-write.",
		Input:        "Resolve conflict.txt.",
		Tools:        []runtimex.ToolName{runtimex.ToolReadFile, runtimex.ToolRepairFile, runtimex.ToolFinish},
	})
	if err != nil {
		t.Fatalf("CreateToolCallProvider error: %v", err)
	}

	turn, err := provider.NextToolCalls(nil)
	if err != nil {
		t.Fatalf("first NextToolCalls error: %v", err)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].Name != runtimex.ToolReadFile || turn.ToolCalls[0].Arguments["path"] != "conflict.txt" {
		t.Fatalf("unexpected first turn: %+v", turn)
	}
	if len(requests) != 1 || !strings.Contains(requests[0], "read_file") || !strings.Contains(requests[0], "repair_file") || !strings.Contains(requests[0], "Use read-before-write.") {
		t.Fatalf("initial request did not include native tools and instructions: %s", requests[0])
	}

	turn, err = provider.NextToolCalls([]runtimex.ToolCallResult{{CallID: "read", Name: runtimex.ToolReadFile, OK: true, Result: map[string]any{"content": "<<<<<<< HEAD\n"}}})
	if err != nil {
		t.Fatalf("second NextToolCalls error: %v", err)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].Name != runtimex.ToolFinish || turn.ToolCalls[0].Arguments["message"] != "repaired" {
		t.Fatalf("unexpected second turn: %+v", turn)
	}
	if len(requests) != 2 || !strings.Contains(requests[1], "function_call_output") || !strings.Contains(requests[1], `"call_id":"read"`) || !strings.Contains(requests[1], `"role":"user"`) {
		t.Fatalf("tool-result request did not include native tool output context: %s", requests[1])
	}
}

func TestToolCallProviderPreservesConversationStateForToolResults(t *testing.T) {
	apiKey := "test-key"
	baseURL := "https://example.invalid/v1"
	tests := []struct {
		name             string
		provider         config.ProviderType
		baseURL          *string
		firstResponse    string
		secondResponse   string
		assistantMarker  string
		toolResultMarker string
	}{
		{
			name:             "openai-compatible",
			provider:         config.ProviderOpenAICompatible,
			baseURL:          &baseURL,
			firstResponse:    `{"id":"chat_1","choices":[{"message":{"role":"assistant","tool_calls":[{"id":"read","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"conflict.txt\"}"}}]}}]}`,
			secondResponse:   `{"id":"chat_2","choices":[{"message":{"role":"assistant","tool_calls":[{"id":"finish","type":"function","function":{"name":"finish","arguments":"{\"message\":\"done\"}"}}]}}]}`,
			assistantMarker:  `"tool_calls"`,
			toolResultMarker: `"tool_call_id":"read"`,
		},
		{
			name:             "anthropic",
			provider:         config.ProviderAnthropic,
			firstResponse:    `{"id":"msg_1","content":[{"type":"tool_use","id":"read","name":"read_file","input":{"path":"conflict.txt"}}]}`,
			secondResponse:   `{"id":"msg_2","content":[{"type":"tool_use","id":"finish","name":"finish","input":{"message":"done"}}]}`,
			assistantMarker:  `"tool_use"`,
			toolResultMarker: `"tool_use_id":"read"`,
		},
		{
			name:             "gemini",
			provider:         config.ProviderGoogleGemini,
			firstResponse:    `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"read_file","args":{"path":"conflict.txt"}}}]}}]}`,
			secondResponse:   `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"finish","args":{"message":"done"}}}]}}]}`,
			assistantMarker:  `"functionCall"`,
			toolResultMarker: `"functionResponse"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := []string{}
			provider, err := CreateToolCallProvider(ToolCallProviderOptions{
				Config: ProviderConfig{
					Provider: tt.provider,
					APIKey:   &apiKey,
					BaseURL:  tt.baseURL,
					Model:    "tool-model",
					HTTPClient: roundTripFunc(func(request *http.Request) (*http.Response, error) {
						body, _ := io.ReadAll(request.Body)
						requests = append(requests, string(body))
						if len(requests) == 1 {
							return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tt.firstResponse)), Header: make(http.Header)}, nil
						}
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tt.secondResponse)), Header: make(http.Header)}, nil
					}),
				},
				Instructions: "Repair carefully.",
				Input:        "Resolve conflict.txt.",
				Tools:        []runtimex.ToolName{runtimex.ToolReadFile, runtimex.ToolRepairFile, runtimex.ToolFinish},
			})
			if err != nil {
				t.Fatalf("CreateToolCallProvider error: %v", err)
			}
			turn, err := provider.NextToolCalls(nil)
			if err != nil {
				t.Fatalf("first turn error: %v", err)
			}
			if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].Name != runtimex.ToolReadFile {
				t.Fatalf("unexpected first turn: %+v", turn)
			}
			_, err = provider.NextToolCalls([]runtimex.ToolCallResult{{CallID: "read", Name: runtimex.ToolReadFile, OK: true, Result: map[string]any{"content": "<<<<<<< HEAD\n"}}})
			if err != nil {
				t.Fatalf("second turn error: %v", err)
			}
			if len(requests) != 2 {
				t.Fatalf("expected two requests, got %d", len(requests))
			}
			if !strings.Contains(requests[1], tt.assistantMarker) || !strings.Contains(requests[1], tt.toolResultMarker) {
				t.Fatalf("second request lost tool-call conversation state:\n%s", requests[1])
			}
		})
	}
}

func TestToolCallProviderDoesNotLeakProviderResponseBodyInHTTPError(t *testing.T) {
	apiKey := "test-key"
	provider, err := CreateToolCallProvider(ToolCallProviderOptions{
		Config: ProviderConfig{
			Provider: config.ProviderOpenAIResponses,
			APIKey:   &apiKey,
			Model:    "responses-model",
			HTTPClient: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(strings.NewReader(`{"error":"provider exploded","secret":"sk-live-123"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		},
		Instructions: "Use read-before-write.",
		Input:        "Resolve conflict.txt.",
		Tools:        []runtimex.ToolName{runtimex.ToolReadFile, runtimex.ToolRepairFile, runtimex.ToolFinish},
	})
	if err != nil {
		t.Fatalf("CreateToolCallProvider error: %v", err)
	}
	_, err = provider.NextToolCalls(nil)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	message := err.Error()
	if !strings.Contains(message, "provider request failed with http 500") {
		t.Fatalf("unexpected error message: %s", message)
	}
	if strings.Contains(message, "provider exploded") || strings.Contains(message, "sk-live-123") {
		t.Fatalf("error leaked provider response body: %s", message)
	}
}

func TestToolCallProviderDoesNotLeakProviderResponseBodyInParseError(t *testing.T) {
	apiKey := "test-key"
	provider, err := CreateToolCallProvider(ToolCallProviderOptions{
		Config: ProviderConfig{
			Provider: config.ProviderOpenAIResponses,
			APIKey:   &apiKey,
			Model:    "responses-model",
			HTTPClient: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"output":"not-a-valid-tool-response","secret":"sk-live-456"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		},
		Instructions: "Use read-before-write.",
		Input:        "Resolve conflict.txt.",
		Tools:        []runtimex.ToolName{runtimex.ToolReadFile, runtimex.ToolRepairFile, runtimex.ToolFinish},
	})
	if err != nil {
		t.Fatalf("CreateToolCallProvider error: %v", err)
	}
	_, err = provider.NextToolCalls(nil)
	if err == nil {
		t.Fatal("expected parse error")
	}
	message := err.Error()
	if !strings.Contains(message, "provider response parse error:") {
		t.Fatalf("unexpected error message: %s", message)
	}
	if strings.Contains(message, "not-a-valid-tool-response") || strings.Contains(message, "sk-live-456") {
		t.Fatalf("error leaked provider response body: %s", message)
	}
}
