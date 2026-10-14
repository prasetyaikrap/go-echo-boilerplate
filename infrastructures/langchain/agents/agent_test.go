package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/langchain"

	"github.com/tmc/langchaingo/llms"
)

// scriptedLLM returns the next scripted response on every call and records what it received
type scriptedLLM struct {
	mu        sync.Mutex
	responses []*llms.ContentChoice
	calls     []scriptedCall
	stream    []string
}

type scriptedCall struct {
	messages []llms.MessageContent
	options  llms.CallOptions
}

func (s *scriptedLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	opts := llms.CallOptions{}
	for _, apply := range options {
		apply(&opts)
	}

	s.mu.Lock()
	s.calls = append(s.calls, scriptedCall{messages: messages, options: opts})
	if len(s.responses) == 0 {
		s.mu.Unlock()
		return nil, errors.New("no scripted response left")
	}
	choice := s.responses[0]
	s.responses = s.responses[1:]
	s.mu.Unlock()

	if opts.StreamingFunc != nil {
		for _, chunk := range s.stream {
			if err := opts.StreamingFunc(ctx, []byte(chunk)); err != nil {
				return nil, err
			}
		}
	}

	return &llms.ContentResponse{Choices: []*llms.ContentChoice{choice}}, nil
}

func (s *scriptedLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, s, prompt, options...)
}

func newInstance(llm llms.Model, provider models.LLMProvider, temperature *float64) *langchain.LangChainInstance {
	return &langchain.LangChainInstance{Name: "test", LLM: llm, Provider: provider, Temperature: temperature}
}

func answer(text string) *llms.ContentChoice {
	return &llms.ContentChoice{Content: text}
}

func toolCalls(text string, calls ...llms.ToolCall) *llms.ContentChoice {
	return &llms.ContentChoice{Content: text, ToolCalls: calls}
}

func call(id, name, arguments string) llms.ToolCall {
	return llms.ToolCall{ID: id, Type: "function", FunctionCall: &llms.FunctionCall{Name: name, Arguments: arguments}}
}

type cityInput struct {
	City string `json:"city" validate:"required" description:"City name"`
}

func TestRunWithoutTools(t *testing.T) {
	llm := &scriptedLLM{responses: []*llms.ContentChoice{answer("Hello!")}}
	agent := &Agent{Name: "greeter", LLM: newInstance(llm, models.LLMProviderAnthropic, nil), SystemPrompt: "Be friendly."}

	result, err := agent.Run(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Output != "Hello!" || len(result.Messages) != 2 {
		t.Errorf("output %q, %d messages", result.Output, len(result.Messages))
	}
	sent := llm.calls[0].messages
	if sent[0].Role != llms.ChatMessageTypeSystem || sent[0].Parts[0].(llms.TextContent).Text != "Be friendly." {
		t.Errorf("first message must be the system prompt, got %+v", sent[0])
	}
	if result.Messages[0].Role != llms.ChatMessageTypeHuman {
		t.Error("Result.Messages must not include the system prompt")
	}
	if len(llm.calls[0].options.Tools) != 0 {
		t.Error("no tools must be sent when the agent has none")
	}
}

func TestToolLoopRunsToolsInParallel(t *testing.T) {
	llm := &scriptedLLM{responses: []*llms.ContentChoice{
		toolCalls("Checking both.", call("c1", "get_weather", `{"city":"Jakarta"}`), call("c2", "get_weather", `{"city":"Bandung"}`)),
		answer("Jakarta 31°C, Bandung 24°C"),
	}}

	// Each call waits until both have started: this only finishes if the calls run in parallel
	var started sync.WaitGroup
	started.Add(2)
	weather := NewTool("get_weather", "Current weather", func(ctx context.Context, input cityInput) (string, error) {
		started.Done()
		done := make(chan struct{})
		go func() { started.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			return "", errors.New("tool calls did not run in parallel")
		}
		return map[string]string{"Jakarta": "31°C", "Bandung": "24°C"}[input.City], nil
	})

	var events []ToolCallEvent
	agent := &Agent{Name: "weather", LLM: newInstance(llm, models.LLMProviderAnthropic, nil), Tools: []Tool{weather}}
	result, err := agent.Run(context.Background(), "Weather in Jakarta and Bandung?",
		WithOnToolCall(func(_ context.Context, event ToolCallEvent) { events = append(events, event) }))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Output != "Jakarta 31°C, Bandung 24°C" || len(result.ToolCalls) != 2 || len(events) != 2 {
		t.Fatalf("output %q, %d tool calls, %d events", result.Output, len(result.ToolCalls), len(events))
	}

	tool := llm.calls[0].options.Tools[0].Function
	if tool.Name != "get_weather" || tool.Parameters.(map[string]any)["required"].([]string)[0] != "city" ||
		tool.Parameters.(map[string]any)["properties"].(map[string]any)["city"].(map[string]any)["description"] != "City name" {
		t.Errorf("tool definition = %+v", tool)
	}

	second := llm.calls[1].messages
	if len(second) != 4 {
		t.Fatalf("want human, assistant, tool, tool; got %d messages", len(second))
	}
	if second[1].Role != llms.ChatMessageTypeAI || len(second[1].Parts) != 3 {
		t.Errorf("assistant turn must keep its text and both tool calls: %+v", second[1])
	}
	for i, want := range []struct{ id, content string }{{"c1", "31°C"}, {"c2", "24°C"}} {
		response := second[2+i].Parts[0].(llms.ToolCallResponse)
		if response.ToolCallID != want.id || response.Content != want.content {
			t.Errorf("tool result %d = %+v, want %+v", i, response, want)
		}
	}
}

func TestToolErrorsFollowErrorResponseRules(t *testing.T) {
	llm := &scriptedLLM{responses: []*llms.ContentChoice{
		toolCalls("",
			call("c1", "find_order", `{"order_id":"ord_404"}`),
			call("c2", "query_db", `{}`),
			call("c3", "missing", `{}`),
			call("c4", "boom", `{}`),
			call("c5", "find_order", `not json`),
			call("c6", "find_order", `{}`),
		),
		answer("Recovered"),
	}}

	type orderInput struct {
		OrderID string `json:"order_id" validate:"required"`
	}
	agent := &Agent{Name: "a", LLM: newInstance(llm, models.LLMProviderAnthropic, nil), Tools: []Tool{
		NewTool("find_order", "", func(_ context.Context, input orderInput) (string, error) {
			return "", utils.NewNotFoundError(fmt.Errorf("order %s not found", input.OrderID))
		}),
		NewTool("query_db", "", func(context.Context, struct{}) (string, error) {
			return "", errors.New(`pq: relation "secret_table" does not exist at db-prod-01`)
		}),
		NewTool("boom", "", func(context.Context, struct{}) (string, error) { panic("bug") }),
	}}

	result, err := agent.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Output != "Recovered" {
		t.Errorf("output = %q", result.Output)
	}

	content := func(i int) string {
		return llm.calls[1].messages[2+i].Parts[0].(llms.ToolCallResponse).Content
	}

	// Application errors (utils.New…Error): the model sees the message and can correct itself
	if content(0) != "Error: order ord_404 not found" {
		t.Errorf("not-found error = %q", content(0))
	}
	if content(2) != `Error: unknown tool "missing"` {
		t.Errorf("unknown tool = %q", content(2))
	}
	if !strings.HasPrefix(content(4), "Error: invalid arguments for find_order") {
		t.Errorf("invalid JSON = %q", content(4))
	}
	if !strings.Contains(content(5), "OrderID") || !strings.Contains(content(5), "required") {
		t.Errorf("validation error = %q", content(5))
	}

	// Internal errors and panics: hidden behind an error id, never shown to the model
	for _, i := range []int{1, 3} {
		if !strings.HasPrefix(content(i), "Error: the tool failed with an internal error (error id ") {
			t.Errorf("internal error %d = %q", i, content(i))
		}
		if strings.Contains(content(i), "secret_table") || strings.Contains(content(i), "bug") {
			t.Errorf("internal details leaked to the model: %q", content(i))
		}
	}

	// The caller still gets the original error, e.g. for its own logging
	if !strings.Contains(result.ToolCalls[1].Err.Error(), "secret_table") {
		t.Errorf("ToolCallEvent.Err must keep the original error, got %v", result.ToolCalls[1].Err)
	}
}

func TestModelOverrideAndTemperature(t *testing.T) {
	temperature := 0.3

	for name, tt := range map[string]struct {
		provider        models.LLMProvider
		wantTemperature float64
	}{
		"openai keeps the temperature": {provider: models.LLMProviderOpenAI, wantTemperature: 0.3},
		"anthropic never receives it":  {provider: models.LLMProviderAnthropic, wantTemperature: 0},
	} {
		t.Run(name, func(t *testing.T) {
			llm := &scriptedLLM{responses: []*llms.ContentChoice{answer("ok")}}
			agent := &Agent{Name: "a", LLM: newInstance(llm, tt.provider, &temperature), Model: "cheaper-model"}

			if _, err := agent.Run(context.Background(), "hi"); err != nil {
				t.Fatalf("Run: %v", err)
			}
			options := llm.calls[0].options
			if options.Model != "cheaper-model" || options.Temperature != tt.wantTemperature {
				t.Errorf("model %q, temperature %v", options.Model, options.Temperature)
			}
		})
	}
}

func TestMaxIterations(t *testing.T) {
	loop := toolCalls("", call("c", "noop", `{}`))
	llm := &scriptedLLM{responses: []*llms.ContentChoice{loop, loop, loop}}
	agent := &Agent{Name: "looper", LLM: newInstance(llm, models.LLMProviderAnthropic, nil), MaxIterations: 3,
		Tools: []Tool{NewTool("noop", "", func(context.Context, struct{}) (string, error) { return "", nil })}}

	_, err := agent.Run(context.Background(), "go")
	if !errors.Is(err, models.ErrAgentMaxIterations) || len(llm.calls) != 3 {
		t.Errorf("err = %v after %d calls", err, len(llm.calls))
	}
}

func TestStreamingAndHistory(t *testing.T) {
	llm := &scriptedLLM{responses: []*llms.ContentChoice{answer("Second answer")}, stream: []string{"Second ", "answer"}}
	agent := &Agent{Name: "a", LLM: newInstance(llm, models.LLMProviderAnthropic, nil)}
	history := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "First question"),
		llms.TextParts(llms.ChatMessageTypeAI, "First answer"),
	}

	var chunks []string
	result, err := agent.Run(context.Background(), "Second question", WithHistory(history),
		WithOnToken(func(_ context.Context, chunk string) error { chunks = append(chunks, chunk); return nil }))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if strings.Join(chunks, "") != "Second answer" {
		t.Errorf("streamed %q", chunks)
	}
	if len(llm.calls[0].messages) != 3 || len(result.Messages) != 4 {
		t.Errorf("sent %d messages, result has %d", len(llm.calls[0].messages), len(result.Messages))
	}
	if len(history) != 2 {
		t.Error("the caller's history slice must not be modified")
	}
}

func TestMultiAgentWithDifferentModels(t *testing.T) {
	researcherLLM := &scriptedLLM{responses: []*llms.ContentChoice{answer("Go 1.26 was released in February 2026")}}
	researcher := &Agent{
		Name:         "researcher",
		LLM:          newInstance(researcherLLM, models.LLMProviderOpenAI, nil),
		SystemPrompt: "You research facts.",
	}

	coordinatorLLM := &scriptedLLM{responses: []*llms.ContentChoice{
		toolCalls("", call("c1", "researcher", `{"task":"When was Go 1.26 released?"}`)),
		answer("It was released in February 2026."),
	}}
	coordinator := &Agent{
		Name:  "coordinator",
		LLM:   newInstance(coordinatorLLM, models.LLMProviderAnthropic, nil),
		Tools: []Tool{researcher.AsTool("Finds facts. Give it one precise question.")},
	}

	result, err := coordinator.Run(context.Background(), "When did Go 1.26 come out?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Output != "It was released in February 2026." {
		t.Errorf("output = %q", result.Output)
	}
	task := researcherLLM.calls[0].messages[1].Parts[0].(llms.TextContent).Text
	if task != "When was Go 1.26 released?" {
		t.Errorf("researcher received %q", task)
	}
	delegated := coordinatorLLM.calls[1].messages[2].Parts[0].(llms.ToolCallResponse).Content
	if delegated != "Go 1.26 was released in February 2026" {
		t.Errorf("coordinator received %q", delegated)
	}
}

type echoLangChainTool struct{}

func (echoLangChainTool) Name() string        { return "echo" }
func (echoLangChainTool) Description() string { return "Echoes the input" }
func (echoLangChainTool) Call(_ context.Context, input string) (string, error) {
	return fmt.Sprintf("echo: %s", input), nil
}

func TestFromLangChainTool(t *testing.T) {
	tool := FromLangChainTool(echoLangChainTool{})

	output, err := tool.Run(context.Background(), `{"input":"hello"}`)
	if err != nil || output != "echo: hello" || tool.Name != "echo" {
		t.Errorf("got (%q, %v), name %q", output, err, tool.Name)
	}
}

func TestCancelledContextStopsTheRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	agent := &Agent{Name: "a", LLM: newInstance(&scriptedLLM{}, models.LLMProviderAnthropic, nil)}
	if _, err := agent.Run(ctx, "hi"); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
