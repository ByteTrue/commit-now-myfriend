package providers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	runtimex "github.com/ByteTrue/commit-now-myfriend/internal/runtime"
)

func TestProviderCapabilitiesReportNativeToolCalling(t *testing.T) {
	for _, provider := range config.ProviderTypes {
		capability, ok := CapabilityForProvider(provider)
		if !ok {
			t.Fatalf("missing capability for %s", provider)
		}
		if !capability.NativeToolCalls || !capability.StreamingProgress || capability.Protocol == "" {
			t.Fatalf("unexpected capability for %s: %+v", provider, capability)
		}
	}
}

func TestOpenAIResponsesToolProtocolAdapter(t *testing.T) {
	adapter, err := CreateToolProtocolAdapter(ProviderConfig{Provider: config.ProviderOpenAIResponses})
	if err != nil {
		t.Fatalf("CreateToolProtocolAdapter error: %v", err)
	}
	assertInitialToolRequestContains(t, adapter, "function", "inspect_commit_scope")
	turn, err := adapter.ParseTurn([]byte(`{"id":"resp_1","output":[{"type":"function_call","call_id":"call_1","name":"inspect_commit_scope","arguments":"{\"path\":\".\"}"}]}`))
	if err != nil {
		t.Fatalf("ParseTurn error: %v", err)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].ID != "call_1" || turn.ToolCalls[0].Name != runtimex.ToolInspectCommitScope || turn.ToolCalls[0].Arguments["path"] != "." {
		t.Fatalf("unexpected turn: %+v", turn)
	}
	payload, err := adapter.BuildToolResultPayload([]runtimex.ToolCallResult{{CallID: "call_1", Name: runtimex.ToolInspectCommitScope, OK: true, Result: map[string]any{"ok": true}}})
	if err != nil {
		t.Fatalf("BuildToolResultPayload error: %v", err)
	}
	var decoded struct {
		Input []struct {
			Type   string `json:"type"`
			CallID string `json:"call_id"`
			Output string `json:"output"`
		} `json:"input"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v\n%s", err, string(payload))
	}
	if len(decoded.Input) != 1 || decoded.Input[0].Type != "function_call_output" || decoded.Input[0].CallID != "call_1" || decoded.Input[0].Output == "" {
		t.Fatalf("unexpected payload: %+v", decoded)
	}
}

func TestOpenAICompatibleToolProtocolAdapter(t *testing.T) {
	adapter, err := CreateToolProtocolAdapter(ProviderConfig{Provider: config.ProviderOpenAICompatible})
	if err != nil {
		t.Fatalf("CreateToolProtocolAdapter error: %v", err)
	}
	assertInitialToolRequestContains(t, adapter, "function", "read_file")
	turn, err := adapter.ParseTurn([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_chat","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"README.md\"}"}}]}}]}`))
	if err != nil {
		t.Fatalf("ParseTurn error: %v", err)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].Name != runtimex.ToolReadFile || turn.ToolCalls[0].Arguments["path"] != "README.md" {
		t.Fatalf("unexpected turn: %+v", turn)
	}
	payload, err := adapter.BuildToolResultPayload([]runtimex.ToolCallResult{{CallID: "call_chat", Name: runtimex.ToolReadFile, OK: false, Error: &runtimex.ToolCallError{Code: "invalid_arguments", Message: "bad"}}})
	if err != nil {
		t.Fatalf("BuildToolResultPayload error: %v", err)
	}
	var decoded struct {
		Messages []struct {
			Role       string `json:"role"`
			ToolCallID string `json:"tool_call_id"`
			Content    string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v\n%s", err, string(payload))
	}
	if len(decoded.Messages) != 1 || decoded.Messages[0].Role != "tool" || decoded.Messages[0].ToolCallID != "call_chat" || decoded.Messages[0].Content == "" {
		t.Fatalf("unexpected chat tool result payload: %+v", decoded)
	}
}

func TestAnthropicToolProtocolAdapter(t *testing.T) {
	adapter, err := CreateToolProtocolAdapter(ProviderConfig{Provider: config.ProviderAnthropic})
	if err != nil {
		t.Fatalf("CreateToolProtocolAdapter error: %v", err)
	}
	assertInitialToolRequestContains(t, adapter, "input_schema", "preview_commit")
	turn, err := adapter.ParseTurn([]byte(`{"content":[{"type":"tool_use","id":"toolu_1","name":"preview_commit","input":{"message":"docs: update guide"}}]}`))
	if err != nil {
		t.Fatalf("ParseTurn error: %v", err)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].ID != "toolu_1" || turn.ToolCalls[0].Name != runtimex.ToolPreviewCommit {
		t.Fatalf("unexpected anthropic turn: %+v", turn)
	}
	payload, err := adapter.BuildToolResultPayload([]runtimex.ToolCallResult{{CallID: "toolu_1", Name: runtimex.ToolPreviewCommit, OK: true, Result: map[string]any{"message": "docs: update guide"}}})
	if err != nil {
		t.Fatalf("BuildToolResultPayload error: %v", err)
	}
	if !json.Valid(payload) || !containsStringPayload(string(payload), "tool_result") || !containsStringPayload(string(payload), "toolu_1") {
		t.Fatalf("unexpected anthropic payload: %s", string(payload))
	}
}

func TestGeminiToolProtocolAdapter(t *testing.T) {
	adapter, err := CreateToolProtocolAdapter(ProviderConfig{Provider: config.ProviderGoogleGemini})
	if err != nil {
		t.Fatalf("CreateToolProtocolAdapter error: %v", err)
	}
	assertInitialToolRequestContains(t, adapter, "functionDeclarations", "get_diff")
	turn, err := adapter.ParseTurn([]byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_diff","args":{"scope":"selected"}}}]}}]}`))
	if err != nil {
		t.Fatalf("ParseTurn error: %v", err)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].Name != runtimex.ToolGetDiff || turn.ToolCalls[0].Arguments["scope"] != "selected" {
		t.Fatalf("unexpected gemini turn: %+v", turn)
	}
	payload, err := adapter.BuildToolResultPayload([]runtimex.ToolCallResult{{CallID: "get_diff", Name: runtimex.ToolGetDiff, OK: true, Result: map[string]any{"bytes": 10}}})
	if err != nil {
		t.Fatalf("BuildToolResultPayload error: %v", err)
	}
	if !json.Valid(payload) || !containsStringPayload(string(payload), "functionResponse") || !containsStringPayload(string(payload), "get_diff") {
		t.Fatalf("unexpected gemini payload: %s", string(payload))
	}
}

func TestToolProtocolAdapterRejectsMalformedArguments(t *testing.T) {
	adapter, err := CreateToolProtocolAdapter(ProviderConfig{Provider: config.ProviderOpenAICompatible})
	if err != nil {
		t.Fatalf("CreateToolProtocolAdapter error: %v", err)
	}
	_, err = adapter.ParseTurn([]byte(`{"choices":[{"message":{"tool_calls":[{"id":"call_bad","type":"function","function":{"name":"read_file","arguments":"not-json"}}]}}]}`))
	if err == nil {
		t.Fatal("expected malformed arguments error")
	}
}

func containsStringPayload(payload string, expected string) bool {
	return strings.Contains(payload, expected)
}

func assertInitialToolRequestContains(t *testing.T, adapter ToolProtocolAdapter, fragments ...string) {
	t.Helper()
	payload, err := adapter.BuildInitialRequest([]runtimex.ToolName{runtimex.ToolInspectCommitScope, runtimex.ToolReadFile, runtimex.ToolPreviewCommit, runtimex.ToolGetDiff})
	if err != nil {
		t.Fatalf("BuildInitialRequest error: %v", err)
	}
	if !json.Valid(payload) {
		t.Fatalf("initial request is not JSON: %s", string(payload))
	}
	for _, fragment := range fragments {
		if !strings.Contains(string(payload), fragment) {
			t.Fatalf("expected initial request for %s to contain %q: %s", adapter.Provider(), fragment, string(payload))
		}
	}
}
