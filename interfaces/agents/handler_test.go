package agent_tools

import (
	"context"
	"testing"

	"go-serviceboilerplate/applications/usecases"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/langchain"
	"go-serviceboilerplate/infrastructures/langchain/agents"
	"go-serviceboilerplate/infrastructures/repositories"

	"github.com/tmc/langchaingo/llms"
)

// One scripted model call that asks for a tool, then the final answer
type toolThenAnswerLLM struct {
	calls       int
	toolResults []string
}

func (l *toolThenAnswerLLM) GenerateContent(_ context.Context, messages []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
	l.calls++
	if l.calls == 1 {
		return &llms.ContentResponse{Choices: []*llms.ContentChoice{{ToolCalls: []llms.ToolCall{{
			ID: "c1", Type: "function", FunctionCall: &llms.FunctionCall{Name: "get_system_info", Arguments: "{}"},
		}}}}}, nil
	}
	for _, message := range messages {
		for _, part := range message.Parts {
			if result, ok := part.(llms.ToolCallResponse); ok {
				l.toolResults = append(l.toolResults, result.Content)
			}
		}
	}
	return &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: "The service is running."}}}, nil
}

func (l *toolThenAnswerLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, l, prompt, options...)
}

// The whole clean-architecture path: agent → interfaces/agents tool → use case → repository
func TestAgentCallsUsecaseThroughDomainTool(t *testing.T) {
	systemUsecase := usecases.NewSystemUsecase(repositories.NewSystemRepositories(nil))
	agentTools := NewAgentTools(systemUsecase)

	llm := &toolThenAnswerLLM{}
	agent := &agents.Agent{
		Name:  "support",
		LLM:   &langchain.LangChainInstance{Name: "test", LLM: llm, Provider: models.LLMProviderAnthropic},
		Tools: agentTools.All(),
	}

	result, err := agent.Run(context.Background(), "Is the service up?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Output != "The service is running." || len(llm.toolResults) != 1 {
		t.Fatalf("output %q, tool results %v", result.Output, llm.toolResults)
	}
	if want := `{"app_name":"Go Service Boilerplate","status":"running","version":"1.0.0"}`; llm.toolResults[0] != want {
		t.Errorf("tool result = %s, want %s", llm.toolResults[0], want)
	}
}
