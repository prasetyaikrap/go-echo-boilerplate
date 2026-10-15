package openai_test

// Agents use langchaingo's OpenAI client as it is (no adapter): this checks a full tool loop through it

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/langchain"
	"go-serviceboilerplate/infrastructures/langchain/agents"

	openaiLLM "github.com/tmc/langchaingo/llms/openai"
)

type fakeOpenAI struct {
	mu       sync.Mutex
	replies  [][]string
	requests []map[string]any
}

func (f *fakeOpenAI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var request map[string]any
	_ = json.Unmarshal(body, &request)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, request)
	if len(f.replies) == 0 {
		http.Error(w, `{"error":{"message":"no scripted reply"}}`, http.StatusInternalServerError)
		return
	}
	reply := f.replies[0]
	f.replies = f.replies[1:]

	w.Header().Set("Content-Type", "text/event-stream")
	for _, chunk := range reply {
		fmt.Fprintf(w, "data: %s\n\n", chunk)
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
}

type cityInput struct {
	City string `json:"city" validate:"required" description:"City name"`
}

func TestToolLoopThroughLangChainGoClient(t *testing.T) {
	fake := &fakeOpenAI{replies: [][]string{
		{
			`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]}}]}`,
			`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":\"Jakarta\"}"}}]}}]}`,
			`{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		},
		{
			`{"id":"2","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"<markdown>Jakarta is "}}]}`,
			`{"id":"2","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"31°C</markdown>"},"finish_reason":"stop"}]}`,
		},
	}}
	server := httptest.NewServer(fake)
	defer server.Close()

	llm, err := openaiLLM.New(openaiLLM.WithToken("test"), openaiLLM.WithModel("gpt-test"), openaiLLM.WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	weather := agents.NewTool("get_weather", "Current weather", func(_ context.Context, input cityInput) (string, error) {
		return input.City + ": 31°C", nil
	})
	agent := &agents.Agent{
		Name:  "test",
		LLM:   &langchain.LangChainInstance{LLM: llm, Provider: models.LLMProviderOpenAI},
		Tools: []agents.Tool{weather},
	}

	result, err := agent.Run(context.Background(), "Weather in Jakarta?",
		agents.WithOnToken(func(context.Context, string) error { return nil }))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Output != "<markdown>Jakarta is 31°C</markdown>" || result.ToolCalls[0].Output != "Jakarta: 31°C" {
		t.Errorf("output %q, tool calls %+v", result.Output, result.ToolCalls)
	}

	tool := fake.requests[0]["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if tool["name"] != "get_weather" {
		t.Errorf("tool %v", tool)
	}

	messages := fake.requests[1]["messages"].([]any)
	last := messages[len(messages)-1].(map[string]any)
	if last["role"] != "tool" || last["tool_call_id"] != "call_1" || !strings.Contains(fmt.Sprint(last["content"]), "Jakarta: 31°C") {
		t.Errorf("tool result message %v", last)
	}
}
