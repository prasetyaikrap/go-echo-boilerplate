package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"go-serviceboilerplate/commons/models"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
	"github.com/tmc/langchaingo/llms"
)

// Model implements langchaingo's llms.Model on the official Anthropic SDK.
//
// Differences from other providers:
//   - Temperature / TopP are not sent: current Claude models reject sampling parameters. Use the effort setting instead.
//   - Forced tool choice ("required" or a named tool) is sent as "auto": current Claude models reject forced tool use.
//   - A declined request returns models.ErrLLMRefusal instead of a response.
type Model struct {
	client          anthropic.Client
	model           string
	effort          string
	refusalFallback bool
	assistantTurns  *assistantTurnCache
}

var _ llms.Model = (*Model)(nil)

// Call generates a reply to a single prompt
func (m *Model) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, m, prompt, options...)
}

// GenerateContent sends the conversation to Claude. It streams when a StreamingFunc is set
func (m *Model) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	opts := llms.CallOptions{}
	for _, apply := range options {
		apply(&opts)
	}

	params, err := m.buildParams(messages, opts)
	if err != nil {
		return nil, err
	}

	var message anthropic.BetaMessage
	if isStreaming(opts) {
		message, err = m.stream(ctx, params, opts)
	} else {
		var response *anthropic.BetaMessage
		response, err = m.client.Beta.Messages.New(ctx, params)
		if response != nil {
			message = *response
		}
	}
	if err != nil {
		return nil, fmt.Errorf("anthropic: %w", err)
	}

	return m.toContentResponse(message)
}

func isStreaming(opts llms.CallOptions) bool {
	return opts.StreamingFunc != nil || opts.StreamingReasoningFunc != nil
}

func (m *Model) stream(ctx context.Context, params anthropic.BetaMessageNewParams, opts llms.CallOptions) (anthropic.BetaMessage, error) {
	stream := m.client.Beta.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	message := anthropic.BetaMessage{}
	for stream.Next() {
		event := stream.Current()
		if err := message.Accumulate(event); err != nil {
			return message, err
		}

		delta, ok := event.AsAny().(anthropic.BetaRawContentBlockDeltaEvent)
		if !ok {
			continue
		}

		switch chunk := delta.Delta.AsAny().(type) {
		case anthropic.BetaTextDelta:
			if opts.StreamingFunc != nil {
				if err := opts.StreamingFunc(ctx, []byte(chunk.Text)); err != nil {
					return message, err
				}
			}
		case anthropic.BetaThinkingDelta:
			if opts.StreamingReasoningFunc != nil {
				if err := opts.StreamingReasoningFunc(ctx, []byte(chunk.Thinking), nil); err != nil {
					return message, err
				}
			}
		}
	}

	return message, stream.Err()
}

func (m *Model) buildParams(messages []llms.MessageContent, opts llms.CallOptions) (anthropic.BetaMessageNewParams, error) {
	model := m.model
	if opts.Model != "" {
		model = opts.Model
	}

	params := anthropic.BetaMessageNewParams{
		Model:         model,
		MaxTokens:     maxTokens(opts),
		StopSequences: opts.StopWords,
	}

	system, conversation, err := m.convertMessages(messages)
	if err != nil {
		return params, err
	}
	params.System = system
	params.Messages = conversation

	tools, err := convertTools(opts.Tools)
	if err != nil {
		return params, err
	}
	params.Tools = tools
	if opts.ToolChoice == "none" {
		params.ToolChoice = anthropic.BetaToolChoiceUnionParam{OfNone: &anthropic.BetaToolChoiceNoneParam{}}
	}

	if m.effort != "" {
		params.OutputConfig = anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffort(m.effort)}
	}

	// Server-side fallback: a declined request is re-run on the model Anthropic recommends for that refusal category
	if m.refusalFallback && supportsServerSideFallback(model) {
		params.Fallbacks = anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()}
		params.Betas = append(params.Betas, anthropic.AnthropicBetaServerSideFallback2026_07_01)
	}

	return params, nil
}

func maxTokens(opts llms.CallOptions) int64 {
	if opts.MaxTokens > 0 {
		return int64(opts.MaxTokens)
	}
	if isStreaming(opts) {
		return models.LLMDefaultStreamingMaxTokens
	}
	return models.LLMDefaultMaxTokens
}

func supportsServerSideFallback(model string) bool {
	for _, prefix := range models.LLMServerSideFallbackModels {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

// convertMessages maps langchaingo messages to Claude's system prompt and alternating user/assistant turns
func (m *Model) convertMessages(messages []llms.MessageContent) ([]anthropic.BetaTextBlockParam, []anthropic.BetaMessageParam, error) {
	var system []anthropic.BetaTextBlockParam
	var conversation []anthropic.BetaMessageParam

	for _, message := range messages {
		if message.Role == llms.ChatMessageTypeSystem {
			for _, part := range message.Parts {
				text, ok := part.(llms.TextContent)
				if !ok {
					return nil, nil, fmt.Errorf("anthropic: system messages only support text, got %T", part)
				}
				system = append(system, anthropic.BetaTextBlockParam{Text: text.Text})
			}
			continue
		}

		turn, err := m.convertTurn(message)
		if err != nil {
			return nil, nil, err
		}
		if len(turn.Content) == 0 {
			continue
		}

		// Consecutive turns with the same role are merged, e.g. one tool message per tool result becomes a single
		// user turn with every tool_result, as Claude expects after parallel tool calls
		if last := len(conversation) - 1; last >= 0 && conversation[last].Role == turn.Role {
			conversation[last].Content = append(conversation[last].Content, turn.Content...)
			continue
		}
		conversation = append(conversation, turn)
	}

	return system, conversation, nil
}

func (m *Model) convertTurn(message llms.MessageContent) (anthropic.BetaMessageParam, error) {
	role := anthropic.BetaMessageParamRoleUser
	if message.Role == llms.ChatMessageTypeAI {
		role = anthropic.BetaMessageParamRoleAssistant
	}

	var toolCallIDs []string
	content := make([]anthropic.BetaContentBlockParamUnion, 0, len(message.Parts))

	for _, part := range message.Parts {
		switch p := part.(type) {
		case llms.TextContent:
			if p.Text != "" {
				content = append(content, anthropic.NewBetaTextBlock(p.Text))
			}
		case llms.ToolCall:
			if p.FunctionCall == nil {
				continue
			}
			input := json.RawMessage(p.FunctionCall.Arguments)
			if len(input) == 0 {
				input = json.RawMessage("{}")
			}
			toolCallIDs = append(toolCallIDs, p.ID)
			content = append(content, anthropic.NewBetaToolUseBlock(p.ID, input, p.FunctionCall.Name))
		case llms.ToolCallResponse:
			content = append(content, anthropic.NewBetaToolResultBlock(p.ToolCallID, p.Content, false))
		default:
			return anthropic.BetaMessageParam{}, fmt.Errorf("anthropic: unsupported message part %T", part)
		}
	}

	// Restore Claude's original turn (with its thinking blocks) when this is a turn it produced
	if role == anthropic.BetaMessageParamRoleAssistant {
		if original, ok := m.assistantTurns.find(toolCallIDs); ok {
			return original, nil
		}
	}

	return anthropic.BetaMessageParam{Role: role, Content: content}, nil
}

func convertTools(tools []llms.Tool) ([]anthropic.BetaToolUnionParam, error) {
	converted := make([]anthropic.BetaToolUnionParam, 0, len(tools))

	for _, tool := range tools {
		if tool.Function == nil {
			continue
		}

		schema, err := toInputSchema(tool.Function.Parameters)
		if err != nil {
			return nil, fmt.Errorf("anthropic: tool %q: %w", tool.Function.Name, err)
		}

		param := anthropic.BetaToolParam{
			Name:        tool.Function.Name,
			Description: anthropic.String(tool.Function.Description),
			InputSchema: schema,
		}
		if tool.Function.Strict {
			param.Strict = anthropic.Bool(true)
		}
		converted = append(converted, anthropic.BetaToolUnionParam{OfTool: &param})
	}

	return converted, nil
}

// toInputSchema converts a JSON-schema value (map or struct) into Claude's input_schema
func toInputSchema(parameters any) (anthropic.BetaToolInputSchemaParam, error) {
	schema := anthropic.BetaToolInputSchemaParam{}
	if parameters == nil {
		return schema, nil
	}

	raw, err := json.Marshal(parameters)
	if err != nil {
		return schema, err
	}

	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return schema, fmt.Errorf("parameters must be a JSON object schema: %w", err)
	}

	schema.Properties = fields["properties"]
	if required, ok := fields["required"].([]any); ok {
		for _, name := range required {
			if s, ok := name.(string); ok {
				schema.Required = append(schema.Required, s)
			}
		}
	}

	delete(fields, "type")
	delete(fields, "properties")
	delete(fields, "required")
	if len(fields) > 0 {
		schema.ExtraFields = fields
	}

	return schema, nil
}

func (m *Model) toContentResponse(message anthropic.BetaMessage) (*llms.ContentResponse, error) {
	if message.StopReason == anthropic.BetaStopReasonRefusal {
		return nil, fmt.Errorf("anthropic: %w (category %q): %s", models.ErrLLMRefusal, message.StopDetails.Category, message.StopDetails.Explanation)
	}

	choice := &llms.ContentChoice{
		StopReason: string(message.StopReason),
		GenerationInfo: map[string]any{
			"Model":                    message.Model,
			"InputTokens":              message.Usage.InputTokens,
			"OutputTokens":             message.Usage.OutputTokens,
			"CacheReadInputTokens":     message.Usage.CacheReadInputTokens,
			"CacheCreationInputTokens": message.Usage.CacheCreationInputTokens,
		},
	}

	var text strings.Builder
	var toolCallIDs []string
	for _, block := range message.Content {
		switch b := block.AsAny().(type) {
		case anthropic.BetaTextBlock:
			text.WriteString(b.Text)
		case anthropic.BetaToolUseBlock:
			toolCallIDs = append(toolCallIDs, b.ID)
			choice.ToolCalls = append(choice.ToolCalls, llms.ToolCall{
				ID:   b.ID,
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      b.Name,
					Arguments: b.JSON.Input.Raw(),
				},
			})
		}
	}
	choice.Content = text.String()

	if len(choice.ToolCalls) > 0 {
		choice.FuncCall = choice.ToolCalls[0].FunctionCall
		m.assistantTurns.store(toolCallIDs, message.ToParam())
	}

	return &llms.ContentResponse{Choices: []*llms.ContentChoice{choice}}, nil
}
