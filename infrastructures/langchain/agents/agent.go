package agents

import (
	"context"
	"errors"
	"fmt"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastructures/langchain"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

// Agent is a model with a role (system prompt) and tools. Run sends the input, executes the tools the model calls,
// sends the results back, and repeats until the model answers. An Agent holds no per-run state, so one value can
// serve concurrent requests.
//
//	researcher := &agents.Agent{
//		Name:         "researcher",
//		LLM:          mainLLM,
//		Model:        "claude-haiku-5-5", // optional: another model on the same provider
//		SystemPrompt: "You find facts using the tools. Answer briefly.",
//		Tools:        []agents.Tool{searchTool},
//	}
//	result, err := researcher.Run(ctx, "What changed in the last release?")
type Agent struct {
	// Name identifies the agent in errors, and is its tool name in AsTool (letters, digits, "_" and "-")
	Name string
	// LLM is the model instance from langchain.NewLangChainInstance. Different agents can use different instances
	LLM *langchain.LangChainInstance
	// Model overrides the instance's model for this agent only, e.g. a cheaper model for a sub-agent on the same
	// provider. Empty uses the instance's model. Use a separate LLM instance to switch provider
	Model string
	// SystemPrompt sets the agent's role and rules. Load it with utils.GetAgentPromptFileContent
	SystemPrompt string
	// Tools the model may call. Several calls in one turn run in parallel
	Tools []Tool
	// MaxIterations limits the model calls of one run. 0 uses models.AgentDefaultMaxIterations
	MaxIterations int
	// CallOptions are extra langchaingo options for every model call, e.g. llms.WithMaxTokens
	CallOptions []llms.CallOption
}

// ToolCallEvent describes one executed tool call, for logging or showing progress
type ToolCallEvent struct {
	Agent     string
	Tool      string
	Arguments string
	Output    string
	Err       error
}

// Result of a run
type Result struct {
	// Output is the model's final answer
	Output string
	// Messages is the conversation of this run (history, input, tool calls and results, answer) without the system
	// prompt. Pass it as WithHistory on the next run to continue the conversation
	Messages []llms.MessageContent
	// ToolCalls lists every tool call of the run, in order
	ToolCalls []ToolCallEvent
}

type runOptions struct {
	history    []llms.MessageContent
	onToken    func(ctx context.Context, chunk string) error
	onToolCall func(ctx context.Context, event ToolCallEvent)
}

// RunOption configures one run
type RunOption func(*runOptions)

// WithHistory continues a conversation, e.g. the Messages of the previous Result
func WithHistory(history []llms.MessageContent) RunOption {
	return func(o *runOptions) { o.history = history }
}

// WithOnToken streams the model's text as it is generated, including text it writes before calling tools
func WithOnToken(onToken func(ctx context.Context, chunk string) error) RunOption {
	return func(o *runOptions) { o.onToken = onToken }
}

// WithOnToolCall is called after each tool call, e.g. to log it or show "searching…" in the UI
func WithOnToolCall(onToolCall func(ctx context.Context, event ToolCallEvent)) RunOption {
	return func(o *runOptions) { o.onToolCall = onToolCall }
}

// Run sends input to the agent and returns its final answer once it stops calling tools
func (a *Agent) Run(ctx context.Context, input string, options ...RunOption) (*Result, error) {
	if a.LLM == nil {
		return nil, fmt.Errorf("agent %s: no LLM configured", a.Name)
	}

	run := runOptions{}
	for _, apply := range options {
		apply(&run)
	}

	conversation := append(append([]llms.MessageContent{}, run.history...), llms.TextParts(llms.ChatMessageTypeHuman, input))
	result := &Result{}

	toolsByName := make(map[string]Tool, len(a.Tools))
	for _, tool := range a.Tools {
		toolsByName[tool.Name] = tool
	}

	maxIterations := a.MaxIterations
	if maxIterations <= 0 {
		maxIterations = models.AgentDefaultMaxIterations
	}

	for range maxIterations {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		response, err := a.LLM.LLM.GenerateContent(ctx, a.withSystemPrompt(conversation), a.callOptions(run)...)
		if err != nil {
			return nil, fmt.Errorf("agent %s: %w", a.Name, err)
		}
		if len(response.Choices) == 0 {
			return nil, fmt.Errorf("agent %s: empty response from the model", a.Name)
		}
		choice := response.Choices[0]

		if len(choice.ToolCalls) == 0 {
			conversation = append(conversation, llms.TextParts(llms.ChatMessageTypeAI, choice.Content))
			result.Output = choice.Content
			result.Messages = conversation
			return result, nil
		}

		assistantParts := make([]llms.ContentPart, 0, len(choice.ToolCalls)+1)
		if choice.Content != "" {
			assistantParts = append(assistantParts, llms.TextContent{Text: choice.Content})
		}
		for _, call := range choice.ToolCalls {
			assistantParts = append(assistantParts, call)
		}
		conversation = append(conversation, llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: assistantParts})

		for _, event := range a.runTools(ctx, toolsByName, choice.ToolCalls, run) {
			result.ToolCalls = append(result.ToolCalls, event.ToolCallEvent)
			conversation = append(conversation, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: event.callID,
					Name:       event.Tool,
					Content:    event.modelContent(ctx),
				}},
			})
		}
	}

	return nil, fmt.Errorf("agent %s: %w (%d)", a.Name, models.ErrAgentMaxIterations, maxIterations)
}

// AsTool exposes the agent as a tool, so another agent can delegate tasks to it (multi-agent). The sub-agent starts
// fresh for every task: give it everything it needs in the task text.
//
//	coordinator := &agents.Agent{
//		Name:  "coordinator",
//		LLM:   mainLLM,
//		Tools: []agents.Tool{researcher.AsTool("Finds facts. Give it one precise question.")},
//	}
func (a *Agent) AsTool(description string) Tool {
	type subAgentInput struct {
		Task string `json:"task" validate:"required" description:"The complete task, with all context needed"`
	}

	return NewTool(a.Name, description, func(ctx context.Context, input subAgentInput) (string, error) {
		result, err := a.Run(ctx, input.Task)
		if err != nil {
			return "", err
		}
		return result.Output, nil
	})
}

func (a *Agent) withSystemPrompt(conversation []llms.MessageContent) []llms.MessageContent {
	if a.SystemPrompt == "" {
		return conversation
	}
	return append([]llms.MessageContent{llms.TextParts(llms.ChatMessageTypeSystem, a.SystemPrompt)}, conversation...)
}

func (a *Agent) callOptions(run runOptions) []llms.CallOption {
	options := append(a.LLM.CallOptions(), a.CallOptions...)

	if a.Model != "" {
		options = append(options, llms.WithModel(a.Model))
	}

	if len(a.Tools) > 0 {
		definitions := make([]llms.Tool, 0, len(a.Tools))
		for _, tool := range a.Tools {
			definitions = append(definitions, tool.definition())
		}
		options = append(options, llms.WithTools(definitions))
	}

	if run.onToken != nil {
		options = append(options, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			return run.onToken(ctx, string(chunk))
		}))
	}

	return options
}

type toolCallOutcome struct {
	ToolCallEvent
	callID string
}

// modelContent is what the model sees, with the same rules as utils.ErrorResponse: application errors
// (utils.New…Error) show their message so the model can correct itself; any other error is hidden behind an error id
// and logged, so internal details (SQL errors, hostnames) never reach the model or the user
func (o toolCallOutcome) modelContent(ctx context.Context) string {
	if o.Err == nil {
		return o.Output
	}

	var exception *utils.Exceptions
	if errors.As(o.Err, &exception) {
		return "Error: " + exception.Error()
	}

	errorID := uuid.New().String()
	slog.ErrorContext(ctx, "Agent tool failed",
		"error_id", errorID,
		"agent", o.Agent,
		"tool", o.Tool,
		"error", o.Err.Error(),
	)
	return fmt.Sprintf("Error: the tool failed with an internal error (error id %s)", errorID)
}

// runTools executes the tool calls of one turn in parallel and returns the outcomes in the order of the calls
func (a *Agent) runTools(ctx context.Context, toolsByName map[string]Tool, calls []llms.ToolCall, run runOptions) []toolCallOutcome {
	outcomes := make([]toolCallOutcome, len(calls))

	var wg sync.WaitGroup
	for i, call := range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			outcomes[i] = a.runTool(ctx, toolsByName, call)
		}()
	}
	wg.Wait()

	if run.onToolCall != nil {
		for _, outcome := range outcomes {
			run.onToolCall(ctx, outcome.ToolCallEvent)
		}
	}

	return outcomes
}

func (a *Agent) runTool(ctx context.Context, toolsByName map[string]Tool, call llms.ToolCall) (outcome toolCallOutcome) {
	outcome.callID = call.ID
	outcome.Agent = a.Name
	if call.FunctionCall != nil {
		outcome.Tool = call.FunctionCall.Name
		outcome.Arguments = call.FunctionCall.Arguments
	}

	// A panicking tool must not crash the service: report it to the model like any other tool error
	defer func() {
		if recovered := recover(); recovered != nil {
			outcome.Err = fmt.Errorf("tool %s panicked: %v", outcome.Tool, recovered)
		}
	}()

	tool, ok := toolsByName[outcome.Tool]
	if !ok {
		outcome.Err = utils.NewClientError(fmt.Errorf("unknown tool %q", outcome.Tool))
		return outcome
	}

	outcome.Output, outcome.Err = tool.Run(ctx, outcome.Arguments)
	return outcome
}
