package ollama_test

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
	"go-serviceboilerplate/infrastructures/langchain/providers/ollama"

	"github.com/tmc/langchaingo/llms"
)

// fakeOllama answers /api/chat with the next scripted NDJSON lines and records the request bodies
type fakeOllama struct {
	mu       sync.Mutex
	replies  [][]string
	requests []map[string]any
}

func (f *fakeOllama) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var request map[string]any
	_ = json.Unmarshal(body, &request)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, request)
	if r.URL.Path != "/api/chat" || len(f.replies) == 0 {
		http.Error(w, `{"error":"unexpected request"}`, http.StatusInternalServerError)
		return
	}
	reply := f.replies[0]
	f.replies = f.replies[1:]

	w.Header().Set("Content-Type", "application/x-ndjson")
	for _, line := range reply {
		fmt.Fprintln(w, line)
	}
}

type cityInput struct {
	City string `json:"city" validate:"required" description:"City name"`
}

func newAgent(t *testing.T, handler http.Handler, tools ...agents.Tool) *agents.Agent {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	model, err := ollama.NewOllamaLLM(models.LLMConfig{OllamaServerURL: server.URL + "/", Model: "qwen2.5"})
	if err != nil {
		t.Fatal(err)
	}
	return &agents.Agent{
		Name:         "test",
		LLM:          &langchain.LangChainInstance{LLM: model, Provider: models.LLMProviderOllama},
		SystemPrompt: "Answer in blocks.",
		Tools:        tools,
	}
}

func TestToolLoop(t *testing.T) {
	fake := &fakeOllama{replies: [][]string{
		{
			`{"message":{"role":"assistant","content":"","tool_calls":[{"function":{"index":0,"name":"get_weather","arguments":{"city":"Jakarta"}}}]},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true,"done_reason":"stop"}`,
		},
		{
			`{"message":{"role":"assistant","content":"<markdown>Jakarta is "},"done":false}`,
			`{"message":{"role":"assistant","content":"31°C</markdown>"},"done":false}`,
			`{"message":{"role":"assistant","content":""},"done":true,"done_reason":"stop","prompt_eval_count":20,"eval_count":8}`,
		},
	}}
	weather := agents.NewTool("get_weather", "Current weather", func(_ context.Context, input cityInput) (string, error) {
		return input.City + ": 31°C", nil
	})

	var streamed strings.Builder
	result, err := newAgent(t, fake, weather).Run(context.Background(), "Weather in Jakarta?",
		agents.WithOnToken(func(_ context.Context, chunk string) error { streamed.WriteString(chunk); return nil }))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Output != "<markdown>Jakarta is 31°C</markdown>" || streamed.String() != result.Output {
		t.Errorf("output %q, streamed %q", result.Output, streamed.String())
	}
	if result.ToolCalls[0].Output != "Jakarta: 31°C" {
		t.Errorf("tool output %q", result.ToolCalls[0].Output)
	}

	// The tools are sent: langchaingo's Ollama client never sends them
	tool := fake.requests[0]["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if tool["name"] != "get_weather" || tool["parameters"].(map[string]any)["required"].([]any)[0] != "city" {
		t.Errorf("tool %v", tool)
	}
	if fake.requests[0]["stream"] != true || fake.requests[0]["model"] != "qwen2.5" {
		t.Errorf("request %v", fake.requests[0])
	}

	// system, user, assistant with the tool call (arguments as an object), tool result
	messages := fake.requests[1]["messages"].([]any)
	if len(messages) != 4 {
		t.Fatalf("second request has %d messages: %v", len(messages), messages)
	}
	if messages[0].(map[string]any)["role"] != "system" || messages[1].(map[string]any)["role"] != "user" {
		t.Errorf("messages %v", messages[:2])
	}
	call := messages[2].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if call["name"] != "get_weather" || call["arguments"].(map[string]any)["city"] != "Jakarta" {
		t.Errorf("tool call %v", call)
	}
	toolResult := messages[3].(map[string]any)
	if toolResult["role"] != "tool" || toolResult["tool_name"] != "get_weather" || toolResult["content"] != "Jakarta: 31°C" {
		t.Errorf("tool result %v", toolResult)
	}
}

func TestErrorsKeepOllamasMessage(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"http error": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"registry.ollama.ai/library/llama3:latest does not support tools"}`, http.StatusBadRequest)
		},
		"error in the stream": func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintln(w, `{"error":"registry.ollama.ai/library/llama3:latest does not support tools"}`)
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := newAgent(t, handler).Run(context.Background(), "hi")
			if err == nil || !strings.Contains(err.Error(), "does not support tools") {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestImagesJSONModeAndStopWords(t *testing.T) {
	fake := &fakeOllama{replies: [][]string{{`{"message":{"role":"assistant","content":"{\"ok\":true}"},"done":true,"done_reason":"stop"}`}}}
	server := httptest.NewServer(fake)
	defer server.Close()

	model, err := ollama.NewOllamaLLM(models.LLMConfig{OllamaServerURL: server.URL, Model: "llava"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = model.GenerateContent(context.Background(), []llms.MessageContent{{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextContent{Text: "Describe"}, llms.BinaryContent{MIMEType: "image/png", Data: []byte("png")}},
	}}, llms.WithJSONMode(), llms.WithStopWords([]string{"END"}))
	if err != nil {
		t.Fatal(err)
	}

	request := fake.requests[0]
	message := request["messages"].([]any)[0].(map[string]any)
	if message["content"] != "Describe" || message["images"].([]any)[0] != "cG5n" {
		t.Errorf("message %v", message)
	}
	if request["format"] != "json" || request["options"].(map[string]any)["stop"].([]any)[0] != "END" {
		t.Errorf("request %v", request)
	}
}
