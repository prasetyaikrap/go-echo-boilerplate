package agents

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-serviceboilerplate/commons/models"
	anthropicProvider "go-serviceboilerplate/infrastructures/langchain/providers/anthropic"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// The agent loop with the real Claude adapter (only the HTTP API is mocked): Claude's assistant turn, including its
// thinking block, must reach the API unchanged when the agent sends the tool result back
func TestAgentWithClaudeAdapterKeepsThinkingBlocks(t *testing.T) {
	responses := []string{
		`{"id":"m1","type":"message","role":"assistant","model":"claude-opus-5-5","content":[
			{"type":"thinking","thinking":"","signature":"sig-xyz"},
			{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"city":"Jakarta"}}
		],"stop_reason":"tool_use","stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":5}}`,
		`{"id":"m2","type":"message","role":"assistant","model":"claude-opus-5-5","content":[
			{"type":"text","text":"It is 31°C in Jakarta."}
		],"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":20,"output_tokens":8}}`,
	}
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		requests = append(requests, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, responses[0])
		responses = responses[1:]
	}))
	defer server.Close()

	claude, err := anthropicProvider.NewAnthropicLLM(
		models.LLMConfig{APIKey: "test", Model: "claude-opus-5-5"},
		option.WithBaseURL(server.URL), option.WithMaxRetries(0),
	)
	if err != nil {
		t.Fatal(err)
	}

	agent := &Agent{
		Name: "weather",
		LLM:  newInstance(claude, models.LLMProviderAnthropic, nil),
		Tools: []Tool{NewTool("get_weather", "Current weather",
			func(context.Context, cityInput) (string, error) { return "31°C", nil })},
	}

	result, err := agent.Run(context.Background(), "Weather in Jakarta?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Output != "It is 31°C in Jakarta." {
		t.Errorf("output = %q", result.Output)
	}

	messages := requests[1]["messages"].([]any)
	assistant := messages[1].(map[string]any)["content"].([]any)
	if thinking := assistant[0].(map[string]any); thinking["type"] != "thinking" || thinking["signature"] != "sig-xyz" {
		t.Errorf("thinking block not sent back unchanged: %v", assistant)
	}
	// The SDK sends tool result content as text blocks
	toolResult := messages[2].(map[string]any)["content"].([]any)[0].(map[string]any)
	resultText := toolResult["content"].([]any)[0].(map[string]any)["text"]
	if toolResult["tool_use_id"] != "toolu_1" || resultText != "31°C" {
		t.Errorf("tool result = %v", toolResult)
	}
}
