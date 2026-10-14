package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-serviceboilerplate/commons/models"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/tmc/langchaingo/llms"
)

type capturedRequest struct {
	body    map[string]any
	headers http.Header
}

// mockAnthropic answers each request with the next canned response and records what it received
type mockAnthropic struct {
	server    *httptest.Server
	responses []string
	requests  []capturedRequest
}

func newMockAnthropic(t *testing.T, responses ...string) *mockAnthropic {
	t.Helper()
	mock := &mockAnthropic{responses: responses}
	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mock.requests = append(mock.requests, capturedRequest{body: body, headers: r.Header.Clone()})

		response := mock.responses[0]
		mock.responses = mock.responses[1:]
		if strings.HasPrefix(response, "event:") {
			w.Header().Set("Content-Type", "text/event-stream")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(mock.server.Close)
	return mock
}

func (m *mockAnthropic) lastRequest() capturedRequest {
	return m.requests[len(m.requests)-1]
}

func newTestModel(t *testing.T, mock *mockAnthropic, cfg models.LLMConfig) llms.Model {
	t.Helper()
	if cfg.APIKey == "" {
		cfg.APIKey = "test-key"
	}
	if cfg.Model == "" {
		cfg.Model = "claude-opus-5-5"
	}
	llm, err := NewAnthropicLLM(cfg, option.WithBaseURL(mock.server.URL), option.WithMaxRetries(0))
	if err != nil {
		t.Fatalf("NewAnthropicLLM: %v", err)
	}
	return llm
}

const textResponse = `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5",
	"content":[{"type":"text","text":"Hello from Claude"}],
	"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":12,"output_tokens":4}}`

func TestRequestShape(t *testing.T) {
	mock := newMockAnthropic(t, textResponse)
	llm := newTestModel(t, mock, models.LLMConfig{Effort: "high", RefusalFallback: true})

	response, err := llm.GenerateContent(context.Background(), []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are concise."),
		llms.TextParts(llms.ChatMessageTypeHuman, "Hi"),
	}, llms.WithTemperature(0.7))
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if got := response.Choices[0].Content; got != "Hello from Claude" {
		t.Errorf("content = %q", got)
	}

	request := mock.lastRequest()
	if request.headers.Get("X-Api-Key") != "test-key" {
		t.Errorf("api key header = %q", request.headers.Get("X-Api-Key"))
	}
	if _, sent := request.body["temperature"]; sent {
		t.Error("temperature must not be sent: current Claude models reject sampling parameters")
	}
	if request.body["model"] != "claude-opus-5-5" || request.body["max_tokens"] != float64(16000) {
		t.Errorf("model/max_tokens = %v / %v", request.body["model"], request.body["max_tokens"])
	}
	if effort := request.body["output_config"].(map[string]any)["effort"]; effort != "high" {
		t.Errorf("effort = %v", effort)
	}
	if request.body["fallbacks"] != "default" {
		t.Errorf("fallbacks = %v, want \"default\"", request.body["fallbacks"])
	}
	if !strings.Contains(request.headers.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01") {
		t.Errorf("beta header = %q", request.headers.Get("Anthropic-Beta"))
	}
	system := request.body["system"].([]any)[0].(map[string]any)
	if system["text"] != "You are concise." {
		t.Errorf("system = %v", system)
	}
	if messages := request.body["messages"].([]any); len(messages) != 1 {
		t.Errorf("system message must not be sent as a turn, got %d messages", len(messages))
	}
}

func TestNoFallbackForUnsupportedModelsOrWhenDisabled(t *testing.T) {
	for name, cfg := range map[string]models.LLMConfig{
		"haiku":    {Model: "claude-haiku-5-5", RefusalFallback: true},
		"disabled": {Model: "claude-opus-5-5", RefusalFallback: false},
	} {
		t.Run(name, func(t *testing.T) {
			mock := newMockAnthropic(t, textResponse)
			llm := newTestModel(t, mock, cfg)
			if _, err := llm.Call(context.Background(), "Hi"); err != nil {
				t.Fatalf("Call: %v", err)
			}
			request := mock.lastRequest()
			if _, sent := request.body["fallbacks"]; sent {
				t.Error("fallbacks must not be sent")
			}
			if strings.Contains(request.headers.Get("Anthropic-Beta"), "server-side-fallback") {
				t.Error("fallback beta header must not be sent")
			}
		})
	}
}

const toolUseResponse = `{"id":"msg_2","type":"message","role":"assistant","model":"claude-opus-5-5",
	"content":[
		{"type":"thinking","thinking":"","signature":"sig-abc"},
		{"type":"text","text":"Checking both cities."},
		{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"city":"Jakarta"}},
		{"type":"tool_use","id":"toolu_2","name":"get_weather","input":{"city":"Bandung"}}
	],
	"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":30,"output_tokens":20}}`

func TestToolLoopRestoresThinkingBlocks(t *testing.T) {
	mock := newMockAnthropic(t, toolUseResponse, textResponse)
	llm := newTestModel(t, mock, models.LLMConfig{})

	weatherTool := llms.Tool{Type: "function", Function: &llms.FunctionDefinition{
		Name:        "get_weather",
		Description: "Current weather for a city",
		Parameters: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{"city": map[string]any{"type": "string"}},
			"required":             []string{"city"},
			"additionalProperties": false,
		},
		Strict: true,
	}}
	history := []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "Weather in Jakarta and Bandung?")}

	first, err := llm.GenerateContent(context.Background(), history, llms.WithTools([]llms.Tool{weatherTool}))
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	tool := mock.lastRequest().body["tools"].([]any)[0].(map[string]any)
	schema := tool["input_schema"].(map[string]any)
	if tool["strict"] != true || schema["additionalProperties"] != false || schema["required"].([]any)[0] != "city" {
		t.Errorf("tool definition = %v", tool)
	}

	choice := first.Choices[0]
	if len(choice.ToolCalls) != 2 || choice.ToolCalls[0].FunctionCall.Arguments != `{"city":"Jakarta"}` {
		t.Fatalf("tool calls = %+v", choice.ToolCalls)
	}

	// How a langchaingo agent sends the next request: the assistant turn without thinking, then one message per result
	assistant := llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{
		llms.TextContent{Text: choice.Content}, choice.ToolCalls[0], choice.ToolCalls[1],
	}}
	history = append(history, assistant,
		llms.MessageContent{Role: llms.ChatMessageTypeTool, Parts: []llms.ContentPart{llms.ToolCallResponse{ToolCallID: "toolu_1", Name: "get_weather", Content: "31°C"}}},
		llms.MessageContent{Role: llms.ChatMessageTypeTool, Parts: []llms.ContentPart{llms.ToolCallResponse{ToolCallID: "toolu_2", Name: "get_weather", Content: "24°C"}}},
	)
	if _, err := llm.GenerateContent(context.Background(), history, llms.WithTools([]llms.Tool{weatherTool})); err != nil {
		t.Fatalf("second call: %v", err)
	}

	messages := mock.lastRequest().body["messages"].([]any)
	if len(messages) != 3 {
		t.Fatalf("want user, assistant, user turns, got %d", len(messages))
	}

	assistantBlocks := messages[1].(map[string]any)["content"].([]any)
	thinking := assistantBlocks[0].(map[string]any)
	if thinking["type"] != "thinking" || thinking["signature"] != "sig-abc" {
		t.Errorf("assistant turn must start with the original thinking block, got %v", thinking)
	}
	if len(assistantBlocks) != 4 {
		t.Errorf("assistant turn must be sent back unchanged (4 blocks), got %d", len(assistantBlocks))
	}

	results := messages[2].(map[string]any)["content"].([]any)
	if len(results) != 2 || results[1].(map[string]any)["tool_use_id"] != "toolu_2" {
		t.Errorf("both tool results must be in one user turn, got %v", results)
	}
}

func TestStreaming(t *testing.T) {
	stream := strings.Join([]string{
		`event: message_start
data: {"type":"message_start","message":{"id":"msg_3","type":"message","role":"assistant","model":"claude-opus-5-5","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":5,"output_tokens":0}}}`,
		`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hel"}}`,
		`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"lo\nworld"}}`,
		`event: content_block_stop
data: {"type":"content_block_stop","index":0}`,
		`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":3}}`,
		`event: message_stop
data: {"type":"message_stop"}`,
	}, "\n\n") + "\n\n"

	mock := newMockAnthropic(t, stream)
	llm := newTestModel(t, mock, models.LLMConfig{})

	var chunks []string
	response, err := llm.GenerateContent(context.Background(),
		[]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "Hi")},
		llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
			chunks = append(chunks, string(chunk))
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}

	if strings.Join(chunks, "|") != "Hel|lo\nworld" {
		t.Errorf("chunks = %q", chunks)
	}
	if response.Choices[0].Content != "Hello\nworld" || response.Choices[0].StopReason != "end_turn" {
		t.Errorf("final = %+v", response.Choices[0])
	}
	if mock.lastRequest().body["stream"] != true || mock.lastRequest().body["max_tokens"] != float64(64000) {
		t.Errorf("stream/max_tokens = %v / %v", mock.lastRequest().body["stream"], mock.lastRequest().body["max_tokens"])
	}
}

func TestRefusalReturnsError(t *testing.T) {
	mock := newMockAnthropic(t, `{"id":"msg_4","type":"message","role":"assistant","model":"claude-opus-5-5",
		"content":[],"stop_reason":"refusal","stop_sequence":null,
		"stop_details":{"type":"refusal","category":"cyber","explanation":"declined"},
		"usage":{"input_tokens":5,"output_tokens":0}}`)
	llm := newTestModel(t, mock, models.LLMConfig{})

	_, err := llm.Call(context.Background(), "...")
	if !errors.Is(err, models.ErrLLMRefusal) || !strings.Contains(err.Error(), "cyber") {
		t.Errorf("err = %v, want ErrLLMRefusal with the category", err)
	}
}

func TestToolChoiceNone(t *testing.T) {
	mock := newMockAnthropic(t, textResponse)
	llm := newTestModel(t, mock, models.LLMConfig{})

	_, err := llm.GenerateContent(context.Background(),
		[]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "Hi")},
		llms.WithToolChoice("none"),
	)
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}
	if choice := mock.lastRequest().body["tool_choice"].(map[string]any); choice["type"] != "none" {
		t.Errorf("tool_choice = %v", choice)
	}
}

func TestConfigValidation(t *testing.T) {
	if _, err := NewAnthropicLLM(models.LLMConfig{Model: "claude-opus-5-5"}); err == nil {
		t.Error("missing API key must fail")
	}
	if _, err := NewAnthropicLLM(models.LLMConfig{APIKey: "k", Model: "claude-opus-5-5", Effort: "extreme"}); err == nil {
		t.Error("invalid effort must fail")
	}
}
