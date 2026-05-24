package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	runtimex "github.com/ByteTrue/commit-now-myfriend/internal/runtime"
)

type ToolCallProviderOptions struct {
	Config       ProviderConfig
	Instructions string
	Input        string
	Tools        []runtimex.ToolName
}

type httpToolCallProvider struct {
	cfg                ProviderConfig
	adapter            ToolProtocolAdapter
	client             HTTPDoer
	instructions       string
	input              string
	tools              []runtimex.ToolName
	previousResponseID string
	chatMessages       []map[string]any
	anthropicMessages  []map[string]any
	geminiContents     []map[string]any
	responsesInput     []map[string]any
	turnCount          int
}

func CreateToolCallProvider(options ToolCallProviderOptions) (runtimex.ToolCallProvider, error) {
	adapter, err := CreateToolProtocolAdapter(options.Config)
	if err != nil {
		return nil, err
	}
	if err := validateProviderConfig(options.Config, options.Config.Provider == config.ProviderOpenAICompatible); err != nil {
		return nil, err
	}
	return &httpToolCallProvider{
		cfg:          options.Config,
		adapter:      adapter,
		client:       resolveHTTPClient(options.Config),
		instructions: options.Instructions,
		input:        options.Input,
		tools:        append([]runtimex.ToolName{}, options.Tools...),
	}, nil
}

func (p *httpToolCallProvider) NextToolCalls(results []runtimex.ToolCallResult) (runtimex.ProviderTurn, error) {
	payload, err := p.requestPayload(results)
	if err != nil {
		return runtimex.ProviderTurn{}, err
	}
	request, err := p.newRequest(payload)
	if err != nil {
		return runtimex.ProviderTurn{}, err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return runtimex.ProviderTurn{}, err
	}
	body, err := readResponseBody(response)
	if err != nil {
		return runtimex.ProviderTurn{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return runtimex.ProviderTurn{}, fmt.Errorf("provider request failed with http %d", response.StatusCode)
	}
	p.captureProviderState(body)
	p.turnCount++
	turn, err := p.adapter.ParseTurn(body)
	if err != nil {
		return runtimex.ProviderTurn{}, fmt.Errorf("provider response parse error: %w", err)
	}
	return turn, nil
}

func (p *httpToolCallProvider) requestPayload(results []runtimex.ToolCallResult) (map[string]any, error) {
	if p.turnCount == 0 && len(results) == 0 {
		return p.initialPayload()
	}
	if len(results) == 1 && results[0].Name == runtimex.ToolName("__reminder__") {
		return p.reminderPayload(results[0])
	}
	fragment, err := p.adapter.BuildToolResultPayload(results)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(fragment, &payload); err != nil {
		return nil, err
	}
	p.applyContinuationPayload(payload)
	return payload, nil
}

func (p *httpToolCallProvider) reminderPayload(reminder runtimex.ToolCallResult) (map[string]any, error) {
	message := "Reminder: please call one of the available tools to continue."
	if reminder.Error != nil && reminder.Error.Message != "" {
		message = reminder.Error.Message
	}
	payload := map[string]any{}
	fragment, err := p.adapter.BuildInitialRequest(p.tools)
	if err == nil {
		var toolsPayload map[string]any
		if err := json.Unmarshal(fragment, &toolsPayload); err == nil {
			if t, ok := toolsPayload["tools"]; ok {
				payload["tools"] = t
			}
		}
	}
	switch p.cfg.Provider {
	case config.ProviderOpenAIResponses:
		payload["model"] = p.cfg.Model
		payload["instructions"] = p.instructions
		p.responsesInput = append(p.responsesInput, map[string]any{"role": "user", "content": message})
		payload["input"] = p.responsesInput
		payload["max_output_tokens"] = resolveMaxOutputTokens(p.cfg)
		payload["tool_choice"] = "auto"
	case config.ProviderOpenAICompatible:
		payload["model"] = p.cfg.Model
		p.chatMessages = append(p.chatMessages, map[string]any{"role": "user", "content": message})
		payload["messages"] = p.chatMessages
		payload["temperature"] = 0.2
		payload["max_tokens"] = resolveMaxOutputTokens(p.cfg)
		payload["tool_choice"] = "required"
	case config.ProviderAnthropic:
		payload["model"] = p.cfg.Model
		payload["system"] = p.instructions
		payload["max_tokens"] = resolveMaxOutputTokens(p.cfg)
		p.anthropicMessages = append(p.anthropicMessages, map[string]any{"role": "user", "content": message})
		payload["messages"] = p.anthropicMessages
		payload["tool_choice"] = map[string]any{"type": "any"}
	case config.ProviderGoogleGemini:
		p.geminiContents = append(p.geminiContents, map[string]any{"role": "user", "parts": []map[string]string{{"text": message}}})
		payload["contents"] = p.geminiContents
		payload["generationConfig"] = map[string]any{"temperature": 0.2, "maxOutputTokens": resolveMaxOutputTokens(p.cfg)}
	}
	return payload, nil
}

func (p *httpToolCallProvider) initialPayload() (map[string]any, error) {
	fragment, err := p.adapter.BuildInitialRequest(p.tools)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(fragment, &payload); err != nil {
		return nil, err
	}
	switch p.cfg.Provider {
	case config.ProviderOpenAIResponses:
		payload["model"] = p.cfg.Model
		payload["instructions"] = p.instructions
		p.responsesInput = []map[string]any{{"role": "user", "content": p.input}}
		payload["input"] = p.responsesInput
		payload["temperature"] = 0.2
		payload["max_output_tokens"] = resolveMaxOutputTokens(p.cfg)
		payload["tool_choice"] = "auto"
	case config.ProviderOpenAICompatible:
		payload["model"] = p.cfg.Model
		p.chatMessages = []map[string]any{{"role": "system", "content": p.instructions}, {"role": "user", "content": p.input}}
		payload["messages"] = p.chatMessages
		payload["temperature"] = 0.2
		payload["max_tokens"] = resolveMaxOutputTokens(p.cfg)
		payload["tool_choice"] = "required"
	case config.ProviderAnthropic:
		payload["model"] = p.cfg.Model
		payload["system"] = p.instructions
		payload["max_tokens"] = resolveMaxOutputTokens(p.cfg)
		p.anthropicMessages = []map[string]any{{"role": "user", "content": p.input}}
		payload["messages"] = p.anthropicMessages
		payload["tool_choice"] = map[string]any{"type": "any"}
	case config.ProviderGoogleGemini:
		payload["systemInstruction"] = map[string]any{"parts": []map[string]string{{"text": p.instructions}}}
		p.geminiContents = []map[string]any{{"role": "user", "parts": []map[string]string{{"text": p.input}}}}
		payload["contents"] = p.geminiContents
		payload["generationConfig"] = map[string]any{"temperature": 0.2, "maxOutputTokens": resolveMaxOutputTokens(p.cfg)}
	}
	return payload, nil
}

func (p *httpToolCallProvider) applyContinuationPayload(payload map[string]any) {
	if payload["tools"] == nil {
		fragment, err := p.adapter.BuildInitialRequest(p.tools)
		if err == nil {
			var toolsPayload map[string]any
			if err := json.Unmarshal(fragment, &toolsPayload); err == nil {
				if t, ok := toolsPayload["tools"]; ok {
					payload["tools"] = t
				}
			}
		}
	}
	switch p.cfg.Provider {
	case config.ProviderOpenAIResponses:
		payload["model"] = p.cfg.Model
		payload["instructions"] = p.instructions
		newInputs, _ := payload["input"].([]any)
		for _, item := range newInputs {
			if m, ok := item.(map[string]any); ok {
				p.responsesInput = append(p.responsesInput, m)
			}
		}
		payload["input"] = p.responsesInput
		payload["max_output_tokens"] = resolveMaxOutputTokens(p.cfg)
		payload["tool_choice"] = "auto"
	case config.ProviderOpenAICompatible:
		payload["model"] = p.cfg.Model
		toolMessages, _ := payload["messages"].([]any)
		p.chatMessages = append(p.chatMessages, normalizeToolMessages(toolMessages)...)
		payload["messages"] = p.chatMessages
		payload["tool_choice"] = "required"
	case config.ProviderAnthropic:
		payload["model"] = p.cfg.Model
		payload["system"] = p.instructions
		payload["max_tokens"] = resolveMaxOutputTokens(p.cfg)
		toolMessages, _ := payload["messages"].([]any)
		p.anthropicMessages = append(p.anthropicMessages, normalizeToolMessages(toolMessages)...)
		payload["messages"] = p.anthropicMessages
		payload["tool_choice"] = map[string]any{"type": "any"}
	case config.ProviderGoogleGemini:
		contents, _ := payload["contents"].([]any)
		p.geminiContents = append(p.geminiContents, normalizeToolMessages(contents)...)
		payload["contents"] = p.geminiContents
		payload["generationConfig"] = map[string]any{"temperature": 0.2, "maxOutputTokens": resolveMaxOutputTokens(p.cfg)}
	}
}

func (p *httpToolCallProvider) newRequest(payload map[string]any) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint := p.endpoint()
	request, err := newJSONRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	switch p.cfg.Provider {
	case config.ProviderOpenAIResponses, config.ProviderOpenAICompatible:
		request.Header.Set("Authorization", "Bearer "+*p.cfg.APIKey)
	case config.ProviderAnthropic:
		request.Header.Set("x-api-key", *p.cfg.APIKey)
		request.Header.Set("anthropic-version", "2023-06-01")
	}
	applyUserAgent(request, p.cfg)
	return request, nil
}

func (p *httpToolCallProvider) endpoint() string {
	switch p.cfg.Provider {
	case config.ProviderOpenAICompatible:
		return joinURL(*p.cfg.BaseURL, "/chat/completions")
	case config.ProviderOpenAIResponses:
		return joinURL(optionalStringOrDefault(p.cfg.BaseURL, defaultOpenAIBaseURL), "/responses")
	case config.ProviderAnthropic:
		return joinURL(optionalStringOrDefault(p.cfg.BaseURL, defaultAnthropicBaseURL), "/messages")
	case config.ProviderGoogleGemini:
		baseURL := optionalStringOrDefault(p.cfg.BaseURL, defaultGeminiBaseURL)
		return joinURL(baseURL, "/v1beta/models/"+url.PathEscape(p.cfg.Model)+":generateContent") + "?key=" + url.QueryEscape(*p.cfg.APIKey)
	default:
		return ""
	}
}

func (p *httpToolCallProvider) captureProviderState(body []byte) {
	switch p.cfg.Provider {
	case config.ProviderOpenAIResponses:
		p.captureOpenAIResponsesID(body)
	case config.ProviderOpenAICompatible:
		p.captureOpenAICompatibleAssistant(body)
	case config.ProviderAnthropic:
		p.captureAnthropicAssistant(body)
	case config.ProviderGoogleGemini:
		p.captureGeminiModelContent(body)
	}
}

func (p *httpToolCallProvider) captureOpenAIResponsesID(body []byte) {
	var response struct {
		ID     string           `json:"id"`
		Output []map[string]any `json:"output"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return
	}
	if strings.TrimSpace(response.ID) != "" {
		p.previousResponseID = response.ID
	}
	for _, item := range response.Output {
		t, _ := item["type"].(string)
		if t == "function_call" || t == "message" || t == "reasoning" {
			p.responsesInput = append(p.responsesInput, item)
		}
	}
}

func (p *httpToolCallProvider) captureOpenAICompatibleAssistant(body []byte) {
	var response struct {
		Choices []struct {
			Message map[string]any `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &response); err != nil || len(response.Choices) == 0 || len(response.Choices[0].Message) == 0 {
		return
	}
	p.chatMessages = append(p.chatMessages, response.Choices[0].Message)
}

func (p *httpToolCallProvider) captureAnthropicAssistant(body []byte) {
	var response struct {
		Content []map[string]any `json:"content"`
	}
	if err := json.Unmarshal(body, &response); err != nil || len(response.Content) == 0 {
		return
	}
	p.anthropicMessages = append(p.anthropicMessages, map[string]any{"role": "assistant", "content": response.Content})
}

func (p *httpToolCallProvider) captureGeminiModelContent(body []byte) {
	var response struct {
		Candidates []struct {
			Content map[string]any `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &response); err != nil || len(response.Candidates) == 0 || len(response.Candidates[0].Content) == 0 {
		return
	}
	content := response.Candidates[0].Content
	if _, ok := content["role"]; !ok {
		content["role"] = "model"
	}
	p.geminiContents = append(p.geminiContents, content)
}

func normalizeToolMessages(values []any) []map[string]any {
	messages := make([]map[string]any, 0, len(values))
	for _, value := range values {
		message, ok := value.(map[string]any)
		if ok {
			messages = append(messages, message)
		}
	}
	return messages
}
