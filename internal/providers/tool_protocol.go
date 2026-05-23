package providers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	runtimex "github.com/ByteTrue/commit-now-myfriend/internal/runtime"
)

type toolProtocolAdapter struct {
	provider     config.ProviderType
	capability   ProviderCapability
	buildInitial func([]runtimex.ToolName) ([]byte, error)
	parse        func([]byte) (runtimex.ProviderTurn, error)
	build        func([]runtimex.ToolCallResult) ([]byte, error)
}

func (a toolProtocolAdapter) Provider() config.ProviderType  { return a.provider }
func (a toolProtocolAdapter) Capability() ProviderCapability { return a.capability }
func (a toolProtocolAdapter) BuildInitialRequest(tools []runtimex.ToolName) ([]byte, error) {
	return a.buildInitial(tools)
}
func (a toolProtocolAdapter) ParseTurn(payload []byte) (runtimex.ProviderTurn, error) {
	return a.parse(payload)
}
func (a toolProtocolAdapter) BuildToolResultPayload(results []runtimex.ToolCallResult) ([]byte, error) {
	return a.build(results)
}

var providerCapabilities = map[config.ProviderType]ProviderCapability{
	config.ProviderOpenAIResponses:  {Provider: config.ProviderOpenAIResponses, Protocol: "openai_responses", NativeToolCalls: true, StreamingProgress: true, InteractiveRepair: true},
	config.ProviderOpenAICompatible: {Provider: config.ProviderOpenAICompatible, Protocol: "openai_compatible_chat", NativeToolCalls: true, StreamingProgress: true, InteractiveRepair: true},
	config.ProviderAnthropic:        {Provider: config.ProviderAnthropic, Protocol: "anthropic_messages", NativeToolCalls: true, StreamingProgress: true, InteractiveRepair: true},
	config.ProviderGoogleGemini:     {Provider: config.ProviderGoogleGemini, Protocol: "google_gemini", NativeToolCalls: true, StreamingProgress: true, InteractiveRepair: true},
}

func CapabilityForProvider(provider config.ProviderType) (ProviderCapability, bool) {
	capability, ok := providerCapabilities[provider]
	return capability, ok
}

func CreateToolProtocolAdapter(cfg ProviderConfig) (ToolProtocolAdapter, error) {
	capability, ok := CapabilityForProvider(cfg.Provider)
	if !ok {
		return nil, fmt.Errorf("unsupported provider protocol: %s", cfg.Provider)
	}
	switch cfg.Provider {
	case config.ProviderOpenAIResponses:
		return toolProtocolAdapter{provider: cfg.Provider, capability: capability, buildInitial: buildOpenAIResponsesInitialRequest, parse: parseOpenAIResponsesTurn, build: buildOpenAIResponsesToolResultPayload}, nil
	case config.ProviderOpenAICompatible:
		return toolProtocolAdapter{provider: cfg.Provider, capability: capability, buildInitial: buildOpenAICompatibleInitialRequest, parse: parseOpenAICompatibleTurn, build: buildOpenAICompatibleToolResultPayload}, nil
	case config.ProviderAnthropic:
		return toolProtocolAdapter{provider: cfg.Provider, capability: capability, buildInitial: buildAnthropicInitialRequest, parse: parseAnthropicTurn, build: buildAnthropicToolResultPayload}, nil
	case config.ProviderGoogleGemini:
		return toolProtocolAdapter{provider: cfg.Provider, capability: capability, buildInitial: buildGeminiInitialRequest, parse: parseGeminiTurn, build: buildGeminiToolResultPayload}, nil
	default:
		return nil, fmt.Errorf("unsupported provider protocol: %s", cfg.Provider)
	}
}

func parseOpenAIResponsesTurn(payload []byte) (runtimex.ProviderTurn, error) {
	var response struct {
		Output []struct {
			Type      string `json:"type"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"output"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return runtimex.ProviderTurn{}, err
	}
	calls := make([]runtimex.ToolCallRequest, 0)
	for _, item := range response.Output {
		if item.Type != "function_call" {
			continue
		}
		arguments, err := parseToolArgumentsString(item.Arguments)
		if err != nil {
			return runtimex.ProviderTurn{}, err
		}
		calls = append(calls, runtimex.ToolCallRequest{ID: item.CallID, Name: runtimex.ToolName(item.Name), Arguments: arguments})
	}
	return runtimex.ProviderTurn{ToolCalls: calls}, nil
}

func buildOpenAIResponsesInitialRequest(tools []runtimex.ToolName) ([]byte, error) {
	return json.Marshal(map[string]any{"tools": openAIResponsesFunctionTools(tools)})
}

func buildOpenAIResponsesToolResultPayload(results []runtimex.ToolCallResult) ([]byte, error) {
	items := make([]map[string]any, 0, len(results))
	for _, result := range results {
		output, err := encodeToolResult(result)
		if err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"type": "function_call_output", "call_id": result.CallID, "output": output})
	}
	return json.Marshal(map[string]any{"input": items})
}

func parseOpenAICompatibleTurn(payload []byte) (runtimex.ProviderTurn, error) {
	var response struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return runtimex.ProviderTurn{}, err
	}
	calls := make([]runtimex.ToolCallRequest, 0)
	for _, choice := range response.Choices {
		for _, call := range choice.Message.ToolCalls {
			if call.Type != "" && call.Type != "function" {
				continue
			}
			arguments, err := parseToolArgumentsString(call.Function.Arguments)
			if err != nil {
				return runtimex.ProviderTurn{}, err
			}
			calls = append(calls, runtimex.ToolCallRequest{ID: call.ID, Name: runtimex.ToolName(call.Function.Name), Arguments: arguments})
		}
	}
	return runtimex.ProviderTurn{ToolCalls: calls}, nil
}

func buildOpenAICompatibleToolResultPayload(results []runtimex.ToolCallResult) ([]byte, error) {
	messages := make([]map[string]any, 0, len(results))
	for _, result := range results {
		content, err := encodeToolResult(result)
		if err != nil {
			return nil, err
		}
		messages = append(messages, map[string]any{"role": "tool", "tool_call_id": result.CallID, "content": content})
	}
	return json.Marshal(map[string]any{"messages": messages})
}

func buildOpenAICompatibleInitialRequest(tools []runtimex.ToolName) ([]byte, error) {
	return json.Marshal(map[string]any{"tools": openAIFunctionTools(tools)})
}

func parseAnthropicTurn(payload []byte) (runtimex.ProviderTurn, error) {
	var response struct {
		Content []struct {
			Type  string         `json:"type"`
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"content"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return runtimex.ProviderTurn{}, err
	}
	calls := make([]runtimex.ToolCallRequest, 0)
	for _, block := range response.Content {
		if block.Type != "tool_use" {
			continue
		}
		calls = append(calls, runtimex.ToolCallRequest{ID: block.ID, Name: runtimex.ToolName(block.Name), Arguments: cloneArguments(block.Input)})
	}
	return runtimex.ProviderTurn{ToolCalls: calls}, nil
}

func buildAnthropicToolResultPayload(results []runtimex.ToolCallResult) ([]byte, error) {
	content := make([]map[string]any, 0, len(results))
	for _, result := range results {
		toolResult, err := toolResultObject(result)
		if err != nil {
			return nil, err
		}
		content = append(content, map[string]any{"type": "tool_result", "tool_use_id": result.CallID, "content": []map[string]any{{"type": "text", "text": toolResult}}})
	}
	return json.Marshal(map[string]any{"messages": []map[string]any{{"role": "user", "content": content}}})
}

func buildAnthropicInitialRequest(tools []runtimex.ToolName) ([]byte, error) {
	items := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		items = append(items, map[string]any{
			"name":         string(tool),
			"description":  toolDescription(tool),
			"input_schema": toolSchema(tool),
		})
	}
	return json.Marshal(map[string]any{"tools": items})
}

func parseGeminiTurn(payload []byte) (runtimex.ProviderTurn, error) {
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					FunctionCall *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return runtimex.ProviderTurn{}, err
	}
	calls := make([]runtimex.ToolCallRequest, 0)
	for _, candidate := range response.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall == nil {
				continue
			}
			name := part.FunctionCall.Name
			calls = append(calls, runtimex.ToolCallRequest{ID: name, Name: runtimex.ToolName(name), Arguments: cloneArguments(part.FunctionCall.Args)})
		}
	}
	return runtimex.ProviderTurn{ToolCalls: calls}, nil
}

func buildGeminiToolResultPayload(results []runtimex.ToolCallResult) ([]byte, error) {
	parts := make([]map[string]any, 0, len(results))
	for _, result := range results {
		object, err := toolResultMap(result)
		if err != nil {
			return nil, err
		}
		parts = append(parts, map[string]any{"functionResponse": map[string]any{"name": string(result.Name), "response": object}})
	}
	return json.Marshal(map[string]any{"contents": []map[string]any{{"role": "function", "parts": parts}}})
}

func buildGeminiInitialRequest(tools []runtimex.ToolName) ([]byte, error) {
	declarations := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		declarations = append(declarations, map[string]any{
			"name":        string(tool),
			"description": toolDescription(tool),
			"parameters":  toolSchema(tool),
		})
	}
	return json.Marshal(map[string]any{"tools": []map[string]any{{"functionDeclarations": declarations}}})
}

func openAIFunctionTools(tools []runtimex.ToolName) []map[string]any {
	items := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		items = append(items, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        string(tool),
				"description": toolDescription(tool),
				"parameters":  toolSchema(tool),
			},
		})
	}
	return items
}

func openAIResponsesFunctionTools(tools []runtimex.ToolName) []map[string]any {
	items := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		items = append(items, map[string]any{
			"type":        "function",
			"name":        string(tool),
			"description": toolDescription(tool),
			"parameters":  toolSchema(tool),
		})
	}
	return items
}

func parseToolArgumentsString(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var arguments map[string]any
	if err := json.Unmarshal([]byte(raw), &arguments); err != nil {
		return nil, fmt.Errorf("provider returned malformed tool arguments: %w", err)
	}
	return arguments, nil
}

func encodeToolResult(result runtimex.ToolCallResult) (string, error) {
	object, err := toolResultMap(result)
	if err != nil {
		return "", err
	}
	buffer, err := json.Marshal(object)
	if err != nil {
		return "", err
	}
	return string(buffer), nil
}

func toolResultObject(result runtimex.ToolCallResult) (string, error) {
	return encodeToolResult(result)
}

func toolResultMap(result runtimex.ToolCallResult) (map[string]any, error) {
	object := map[string]any{"ok": result.OK}
	if result.OK {
		object["result"] = result.Result
	} else if result.Error != nil {
		object["error"] = result.Error
	} else {
		object["error"] = map[string]any{"code": "tool_failed", "message": "tool call failed"}
	}
	return object, nil
}

func cloneArguments(arguments map[string]any) map[string]any {
	cloned := make(map[string]any, len(arguments))
	for key, value := range arguments {
		cloned[key] = value
	}
	return cloned
}
