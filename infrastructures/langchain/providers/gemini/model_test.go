package gemini_test

import (
	"context"
	"encoding/base64"
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
	"go-serviceboilerplate/infrastructures/langchain/providers/gemini"

	"github.com/tmc/langchaingo/llms"
	"google.golang.org/genai"
)

// fakeGemini answers streamGenerateContent with the next scripted SSE chunks and records the request bodies
type fakeGemini struct {
	mu       sync.Mutex
	replies  [][]string
	requests []map[string]any
}

func (f *fakeGemini) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var request map[string]any
	_ = json.Unmarshal(body, &request)

	f.mu.Lock()
	f.requests = append(f.requests, request)
	if len(f.replies) == 0 {
		f.mu.Unlock()
		http.Error(w, `{"error":{"code":500,"message":"no scripted reply"}}`, http.StatusInternalServerError)
		return
	}
	reply := f.replies[0]
	f.replies = f.replies[1:]
	f.mu.Unlock()

	if !strings.Contains(r.URL.Path, "streamGenerateContent") {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, reply[0])
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	for _, chunk := range reply {
		fmt.Fprintf(w, "data: %s\n\n", chunk)
	}
}

func newModel(t *testing.T, fake *fakeGemini) llms.Model {
	t.Helper()
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	model, err := gemini.NewGeminiLLM(models.LLMConfig{APIKey: "test", Model: "gemini-3.1-flash-lite"}, genai.HTTPOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewGeminiLLM: %v", err)
	}
	return model
}

var signature = base64.StdEncoding.EncodeToString([]byte("opaque-signature"))

type cityInput struct {
	City string `json:"city" validate:"required" description:"City name"`
}

func TestToolLoopSendsTheThoughtSignatureBack(t *testing.T) {
	fake := &fakeGemini{replies: [][]string{
		{
			`{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"get_weather","args":{"city":"Jakarta"}},"thoughtSignature":"` + signature + `"}]},"finishReason":"STOP"}]}`,
		},
		{
			`{"candidates":[{"content":{"role":"model","parts":[{"text":"<markdown>Jakarta is "}]}}]}`,
			`{"candidates":[{"content":{"role":"model","parts":[{"text":"31°C</markdown>"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`,
		},
	}}

	weather := agents.NewTool("get_weather", "Current weather", func(_ context.Context, input cityInput) (string, error) {
		return input.City + ": 31°C", nil
	})
	agent := &agents.Agent{
		Name:         "test",
		LLM:          &langchain.LangChainInstance{LLM: newModel(t, fake), Provider: models.LLMProviderGemini},
		SystemPrompt: "Answer in blocks.",
		Tools:        []agents.Tool{weather},
	}

	var streamed strings.Builder
	result, err := agent.Run(context.Background(), "Weather in Jakarta?", agents.WithHistory([]llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "<markdown>Hi</markdown>"),
		llms.TextParts(llms.ChatMessageTypeAI, "<markdown>Hello!</markdown>"),
	}), agents.WithOnToken(func(_ context.Context, chunk string) error { streamed.WriteString(chunk); return nil }))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Output != "<markdown>Jakarta is 31°C</markdown>" || streamed.String() != result.Output {
		t.Errorf("output %q, streamed %q", result.Output, streamed.String())
	}
	if result.ToolCalls[0].Output != "Jakarta: 31°C" {
		t.Errorf("tool output %q", result.ToolCalls[0].Output)
	}

	first := fake.requests[0]
	if instruction := first["systemInstruction"].(map[string]any)["parts"].([]any)[0].(map[string]any)["text"]; instruction != "Answer in blocks." {
		t.Errorf("system instruction %v", instruction)
	}
	declaration := first["tools"].([]any)[0].(map[string]any)["functionDeclarations"].([]any)[0].(map[string]any)
	if declaration["name"] != "get_weather" || declaration["parametersJsonSchema"].(map[string]any)["required"].([]any)[0] != "city" {
		t.Errorf("tool declaration %v", declaration)
	}

	// history (user, model), question (user), model turn with the tool call, function response (user)
	contents := fake.requests[1]["contents"].([]any)
	if len(contents) != 5 {
		t.Fatalf("second request has %d contents: %v", len(contents), contents)
	}
	modelTurn := contents[3].(map[string]any)["parts"].([]any)[0].(map[string]any)
	if modelTurn["thoughtSignature"] != signature {
		t.Errorf("the thought signature was not sent back: %v", modelTurn)
	}
	response := contents[4].(map[string]any)["parts"].([]any)[0].(map[string]any)["functionResponse"].(map[string]any)
	if response["name"] != "get_weather" || response["response"].(map[string]any)["output"] != "Jakarta: 31°C" {
		t.Errorf("function response %v", response)
	}
}

func TestPartsWithoutDataAreNeverSentBack(t *testing.T) {
	other := base64.StdEncoding.EncodeToString([]byte("other-signature"))
	call := `{"functionCall":{"name":"get_weather","args":{"city":"Jakarta"}}`

	for name, tt := range map[string]struct {
		stream        []string
		wantSignature string
	}{
		"signature only in a trailing empty part": {
			stream: []string{
				`{"candidates":[{"content":{"role":"model","parts":[` + call + `}]}}]}`,
				`{"candidates":[{"content":{"role":"model","parts":[{"text":"","thoughtSignature":"` + signature + `"}]},"finishReason":"STOP"}]}`,
			},
			wantSignature: signature,
		},
		"trailing empty part after a signed call": {
			stream: []string{
				`{"candidates":[{"content":{"role":"model","parts":[` + call + `,"thoughtSignature":"` + signature + `"}]}}]}`,
				`{"candidates":[{"content":{"role":"model","parts":[{"text":"","thoughtSignature":"` + other + `"}]},"finishReason":"STOP"}]}`,
			},
			wantSignature: signature,
		},
		"signature only in a leading empty part": {
			stream: []string{
				`{"candidates":[{"content":{"role":"model","parts":[{"text":"","thoughtSignature":"` + signature + `"}]}}]}`,
				`{"candidates":[{"content":{"role":"model","parts":[` + call + `}]},"finishReason":"STOP"}]}`,
			},
			wantSignature: signature,
		},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &fakeGemini{replies: [][]string{
				tt.stream,
				{`{"candidates":[{"content":{"role":"model","parts":[{"text":"<markdown>31°C</markdown>"}]},"finishReason":"STOP"}]}`},
			}}
			weather := agents.NewTool("get_weather", "", func(_ context.Context, input cityInput) (string, error) { return "31°C", nil })
			agent := &agents.Agent{
				Name:  "test",
				LLM:   &langchain.LangChainInstance{LLM: newModel(t, fake), Provider: models.LLMProviderGemini},
				Tools: []agents.Tool{weather},
			}

			if _, err := agent.Run(context.Background(), "Weather?",
				agents.WithOnToken(func(context.Context, string) error { return nil })); err != nil {
				t.Fatalf("Run: %v", err)
			}

			parts := fake.requests[1]["contents"].([]any)[1].(map[string]any)["parts"].([]any)
			if len(parts) != 1 {
				t.Fatalf("model turn sent back with %d parts: %v", len(parts), parts)
			}
			part := parts[0].(map[string]any)
			if part["functionCall"] == nil || part["thoughtSignature"] != tt.wantSignature {
				t.Errorf("part %v", part)
			}
		})
	}
}

func TestUnknownToolCallSkipsSignatureValidation(t *testing.T) {
	fake := &fakeGemini{replies: [][]string{{`{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}]}`}}}
	model := newModel(t, fake)

	// A tool call this process never saw, e.g. a loop continued on another instance
	_, err := model.GenerateContent(context.Background(), []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "go"),
		{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{llms.ToolCall{
			ID: "call_unknown", Type: "function", FunctionCall: &llms.FunctionCall{Name: "ping", Arguments: "{}"},
		}}},
		{Role: llms.ChatMessageTypeTool, Parts: []llms.ContentPart{llms.ToolCallResponse{ToolCallID: "call_unknown", Name: "ping", Content: "pong"}}},
	})
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}

	call := fake.requests[0]["contents"].([]any)[1].(map[string]any)["parts"].([]any)[0].(map[string]any)
	if call["thoughtSignature"] != base64.StdEncoding.EncodeToString([]byte(models.GeminiSkipThoughtSignature)) {
		t.Errorf("call %v", call)
	}
}

func TestAPIErrorsKeepGeminisMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"code":400,"message":"Function call is missing a thought_signature","status":"INVALID_ARGUMENT"}}`)
	}))
	defer server.Close()

	model, err := gemini.NewGeminiLLM(models.LLMConfig{APIKey: "test", Model: "m"}, genai.HTTPOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = model.GenerateContent(context.Background(), []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "hi")},
		llms.WithStreamingFunc(func(context.Context, []byte) error { return nil }))
	if err == nil || !strings.Contains(err.Error(), "missing a thought_signature") {
		t.Errorf("err = %v", err)
	}
}

func TestJSONModeAndStopWords(t *testing.T) {
	fake := &fakeGemini{replies: [][]string{{`{"candidates":[{"content":{"role":"model","parts":[{"text":"{}"}]},"finishReason":"STOP"}]}`}}}
	model := newModel(t, fake)

	_, err := model.GenerateContent(context.Background(), []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "hi")},
		llms.WithJSONMode(), llms.WithStopWords([]string{"END"}))
	if err != nil {
		t.Fatal(err)
	}

	config := fake.requests[0]["generationConfig"].(map[string]any)
	if config["responseMimeType"] != "application/json" || config["stopSequences"].([]any)[0] != "END" {
		t.Errorf("generationConfig %v", config)
	}
}
