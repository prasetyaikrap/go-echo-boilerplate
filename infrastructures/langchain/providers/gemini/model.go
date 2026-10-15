package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go-serviceboilerplate/commons/models"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
	"google.golang.org/genai"
)

// Model is a langchaingo llms.Model on Google's current Gen AI SDK (google.golang.org/genai). langchaingo's googleai
// client uses the deprecated SDK, which drops the thought signatures Gemini 3 attaches to tool calls, so the second
// call of an agent's tool loop fails with a 400. Model keeps those signatures (modelTurnCache) and sends them back.
type Model struct {
	client *genai.Client
	model  string
	turns  *modelTurnCache
}

var _ llms.Model = (*Model)(nil)

func (m *Model) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, m, prompt, options...)
}

func (m *Model) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	opts := llms.CallOptions{}
	for _, apply := range options {
		apply(&opts)
	}

	model := m.model
	if opts.Model != "" {
		model = opts.Model
	}

	contents, systemInstruction, err := m.convertMessages(messages)
	if err != nil {
		return nil, err
	}

	config := &genai.GenerateContentConfig{SystemInstruction: systemInstruction}
	if opts.Temperature > 0 {
		temperature := float32(opts.Temperature)
		config.Temperature = &temperature
	}
	if opts.MaxTokens > 0 {
		config.MaxOutputTokens = int32(opts.MaxTokens)
	}
	if len(opts.StopWords) > 0 {
		config.StopSequences = opts.StopWords
	}
	if opts.JSONMode {
		config.ResponseMIMEType = "application/json"
	}
	if len(opts.Tools) > 0 {
		config.Tools = convertTools(opts.Tools)
	}

	turn := &genai.Content{Role: genai.RoleModel}
	var finishReason genai.FinishReason
	var usage *genai.GenerateContentResponseUsageMetadata
	var pendingSignature []byte

	collect := func(response *genai.GenerateContentResponse) error {
		if response.UsageMetadata != nil {
			usage = response.UsageMetadata
		}
		if response.PromptFeedback != nil && response.PromptFeedback.BlockReason != "" {
			return fmt.Errorf("gemini blocked the prompt: %s", response.PromptFeedback.BlockReason)
		}
		if len(response.Candidates) == 0 {
			return nil
		}

		candidate := response.Candidates[0]
		if candidate.FinishReason != "" {
			finishReason = candidate.FinishReason
		}
		if candidate.Content == nil {
			return nil
		}

		for _, part := range candidate.Content.Parts {
			if part == nil {
				continue
			}

			// A stream can send a part with no data (empty text) that only carries a thought signature. Gemini
			// rejects such a part when the turn is sent back, so its signature moves to a neighbouring part
			if !hasPartData(part) {
				if len(part.ThoughtSignature) > 0 {
					if last := len(turn.Parts) - 1; last >= 0 && len(turn.Parts[last].ThoughtSignature) == 0 {
						signed := *turn.Parts[last]
						signed.ThoughtSignature = part.ThoughtSignature
						turn.Parts[last] = &signed
					} else if last < 0 {
						pendingSignature = part.ThoughtSignature
					}
				}
				continue
			}
			if len(pendingSignature) > 0 && len(part.ThoughtSignature) == 0 {
				signed := *part
				signed.ThoughtSignature = pendingSignature
				part = &signed
				pendingSignature = nil
			}

			// Otherwise parts are kept as received: their thought signatures belong to their position in the turn
			turn.Parts = append(turn.Parts, part)
			if opts.StreamingFunc != nil && part.Text != "" && !part.Thought {
				if err := opts.StreamingFunc(ctx, []byte(part.Text)); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if opts.StreamingFunc == nil {
		response, err := m.client.Models.GenerateContent(ctx, model, contents, config)
		if err != nil {
			return nil, err
		}
		if err := collect(response); err != nil {
			return nil, err
		}
	} else {
		for response, err := range m.client.Models.GenerateContentStream(ctx, model, contents, config) {
			if err != nil {
				return nil, err
			}
			if err := collect(response); err != nil {
				return nil, err
			}
		}
	}

	return m.toContentResponse(turn, finishReason, usage)
}

func (m *Model) toContentResponse(turn *genai.Content, finishReason genai.FinishReason, usage *genai.GenerateContentResponseUsageMetadata) (*llms.ContentResponse, error) {
	choice := &llms.ContentChoice{StopReason: string(finishReason), GenerationInfo: map[string]any{}}
	if usage != nil {
		choice.GenerationInfo["input_tokens"] = int(usage.PromptTokenCount)
		choice.GenerationInfo["output_tokens"] = int(usage.CandidatesTokenCount)
		choice.GenerationInfo["thinking_tokens"] = int(usage.ThoughtsTokenCount)
		choice.GenerationInfo["total_tokens"] = int(usage.TotalTokenCount)
	}

	var text strings.Builder
	entry := cachedModelTurn{turn: turn, geminiIDs: map[string]string{}}

	for _, part := range turn.Parts {
		switch {
		case part.FunctionCall != nil:
			arguments := []byte("{}")
			if part.FunctionCall.Args != nil {
				encoded, err := json.Marshal(part.FunctionCall.Args)
				if err != nil {
					return nil, fmt.Errorf("gemini tool call %s: %w", part.FunctionCall.Name, err)
				}
				arguments = encoded
			}

			// Gemini doesn't always give function calls an ID; the agent needs one to match results to calls
			id := "call_" + uuid.NewString()
			entry.toolCallIDs = append(entry.toolCallIDs, id)
			entry.geminiIDs[id] = part.FunctionCall.ID

			choice.ToolCalls = append(choice.ToolCalls, llms.ToolCall{
				ID:           id,
				Type:         "function",
				FunctionCall: &llms.FunctionCall{Name: part.FunctionCall.Name, Arguments: string(arguments)},
			})
		case part.Text != "" && !part.Thought:
			text.WriteString(part.Text)
		}
	}

	choice.Content = text.String()
	if len(entry.toolCallIDs) > 0 {
		m.turns.store(entry)
	}

	if choice.Content == "" && len(choice.ToolCalls) == 0 && finishReason != "" && finishReason != genai.FinishReasonStop {
		return nil, fmt.Errorf("gemini returned no content (finish reason %s)", finishReason)
	}

	return &llms.ContentResponse{Choices: []*llms.ContentChoice{choice}}, nil
}

// convertMessages turns langchaingo messages into Gemini contents. System messages become the system instruction,
// and consecutive messages of the same role are merged (Gemini expects one user turn with all function responses)
func (m *Model) convertMessages(messages []llms.MessageContent) ([]*genai.Content, *genai.Content, error) {
	var contents []*genai.Content
	var systemParts []*genai.Part

	for _, message := range messages {
		var content *genai.Content

		switch message.Role {
		case llms.ChatMessageTypeSystem:
			for _, part := range message.Parts {
				if textPart, ok := part.(llms.TextContent); ok && textPart.Text != "" {
					systemParts = append(systemParts, genai.NewPartFromText(textPart.Text))
				}
			}
			continue
		case llms.ChatMessageTypeHuman, llms.ChatMessageTypeGeneric:
			parts, err := convertUserParts(message.Parts)
			if err != nil {
				return nil, nil, err
			}
			content = &genai.Content{Role: genai.RoleUser, Parts: parts}
		case llms.ChatMessageTypeAI:
			content = m.convertModelTurn(message.Parts)
		case llms.ChatMessageTypeTool:
			content = &genai.Content{Role: genai.RoleUser}
			for _, part := range message.Parts {
				response, ok := part.(llms.ToolCallResponse)
				if !ok {
					return nil, nil, fmt.Errorf("gemini: unsupported part %T in a tool message", part)
				}
				content.Parts = append(content.Parts, &genai.Part{FunctionResponse: &genai.FunctionResponse{
					ID:       m.turns.geminiID(response.ToolCallID),
					Name:     response.Name,
					Response: map[string]any{"output": response.Content},
				}})
			}
		default:
			return nil, nil, fmt.Errorf("gemini: role %q is not supported", message.Role)
		}

		// Gemini rejects empty contents, e.g. a history message without text
		if len(content.Parts) == 0 {
			continue
		}

		if last := len(contents) - 1; last >= 0 && contents[last].Role == content.Role {
			// Copy: a cached turn must never be modified
			contents[last] = &genai.Content{
				Role:  content.Role,
				Parts: append(slices.Clone(contents[last].Parts), content.Parts...),
			}
			continue
		}
		contents = append(contents, content)
	}

	var systemInstruction *genai.Content
	if len(systemParts) > 0 {
		systemInstruction = &genai.Content{Parts: systemParts}
	}

	if len(contents) == 0 {
		return nil, nil, errors.New("gemini: no message to send")
	}
	return contents, systemInstruction, nil
}

// convertModelTurn restores a turn with tool calls exactly as Gemini returned it (with its thought signatures).
// Without it in the cache, the turn is rebuilt and its first tool call skips signature validation
func (m *Model) convertModelTurn(parts []llms.ContentPart) *genai.Content {
	var toolCallIDs []string
	for _, part := range parts {
		if call, ok := part.(llms.ToolCall); ok {
			toolCallIDs = append(toolCallIDs, call.ID)
		}
	}

	if cached, ok := m.turns.find(toolCallIDs); ok {
		return &genai.Content{Role: genai.RoleModel, Parts: slices.Clone(cached.turn.Parts)}
	}

	content := &genai.Content{Role: genai.RoleModel}
	signed := false
	for _, part := range parts {
		switch p := part.(type) {
		case llms.TextContent:
			if p.Text != "" {
				content.Parts = append(content.Parts, genai.NewPartFromText(p.Text))
			}
		case llms.ToolCall:
			if p.FunctionCall == nil {
				continue
			}
			var args map[string]any
			if p.FunctionCall.Arguments != "" {
				_ = json.Unmarshal([]byte(p.FunctionCall.Arguments), &args)
			}

			functionCall := &genai.Part{FunctionCall: &genai.FunctionCall{Name: p.FunctionCall.Name, Args: args}}
			if !signed {
				functionCall.ThoughtSignature = []byte(models.GeminiSkipThoughtSignature)
				signed = true
			}
			content.Parts = append(content.Parts, functionCall)
		}
	}
	return content
}

// hasPartData reports whether the part has content Gemini accepts (its "data" oneof)
func hasPartData(part *genai.Part) bool {
	return part.Text != "" || part.FunctionCall != nil || part.FunctionResponse != nil || part.InlineData != nil ||
		part.FileData != nil || part.ExecutableCode != nil || part.CodeExecutionResult != nil ||
		part.ToolCall != nil || part.ToolResponse != nil
}

func convertUserParts(parts []llms.ContentPart) ([]*genai.Part, error) {
	converted := make([]*genai.Part, 0, len(parts))
	for _, part := range parts {
		switch p := part.(type) {
		case llms.TextContent:
			if p.Text != "" {
				converted = append(converted, genai.NewPartFromText(p.Text))
			}
		case llms.BinaryContent:
			converted = append(converted, genai.NewPartFromBytes(p.Data, p.MIMEType))
		default:
			return nil, fmt.Errorf("gemini: unsupported part %T in a user message", part)
		}
	}
	return converted, nil
}

// convertTools sends the tools' JSON schemas as they are (enum, nested objects, additionalProperties)
func convertTools(tools []llms.Tool) []*genai.Tool {
	declarations := make([]*genai.FunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		if tool.Function == nil {
			continue
		}
		declarations = append(declarations, &genai.FunctionDeclaration{
			Name:                 tool.Function.Name,
			Description:          tool.Function.Description,
			ParametersJsonSchema: tool.Function.Parameters,
		})
	}
	return []*genai.Tool{{FunctionDeclarations: declarations}}
}
