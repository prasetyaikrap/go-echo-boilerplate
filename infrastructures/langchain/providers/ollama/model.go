package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

// Model is a langchaingo llms.Model on Ollama's /api/chat. langchaingo's Ollama client never sends tools, so with it
// an agent's model can't call any.
//
// It calls the HTTP API directly: the official Go client (github.com/ollama/ollama/api) types tool parameters as a
// fixed struct, not the JSON schemas agents.NewTool generates. For tools, the model must support tool calling
// (e.g. llama3.1, qwen2.5); Ollama rejects tools otherwise.
type Model struct {
	serverURL  string
	model      string
	httpClient *http.Client
}

var _ llms.Model = (*Model)(nil)

// Request and response bodies of POST /api/chat (fields as in github.com/ollama/ollama/api)
type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Format   string         `json:"format,omitempty"`
	Tools    []chatTool     `json:"tools,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Thinking   string     `json:"thinking,omitempty"`
	Images     [][]byte   `json:"images,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolName   string     `json:"tool_name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string           `json:"id,omitempty"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Index     int             `json:"index"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type chatTool struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
}

type chatResponse struct {
	Message         chatMessage `json:"message"`
	Done            bool        `json:"done"`
	DoneReason      string      `json:"done_reason,omitempty"`
	PromptEvalCount int         `json:"prompt_eval_count,omitempty"`
	EvalCount       int         `json:"eval_count,omitempty"`
	Error           string      `json:"error,omitempty"`
}

func (m *Model) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, m, prompt, options...)
}

func (m *Model) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	opts := llms.CallOptions{}
	for _, apply := range options {
		apply(&opts)
	}

	request := chatRequest{Model: m.model, Stream: true, Options: map[string]any{}}
	if opts.Model != "" {
		request.Model = opts.Model
	}
	if opts.Temperature > 0 {
		request.Options["temperature"] = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		request.Options["num_predict"] = opts.MaxTokens
	}
	if len(opts.StopWords) > 0 {
		request.Options["stop"] = opts.StopWords
	}
	if opts.JSONMode {
		request.Format = "json"
	}
	for _, tool := range opts.Tools {
		if tool.Function == nil {
			continue
		}
		request.Tools = append(request.Tools, chatTool{Type: "function", Function: chatToolFunction{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		}})
	}

	var err error
	if request.Messages, err = convertMessages(messages); err != nil {
		return nil, err
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, m.serverURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/x-ndjson")

	response, err := m.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, responseError(response)
	}

	var content strings.Builder
	var calls []toolCall
	var last chatResponse

	// One JSON object per line, until "done"
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var chunk chatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			return nil, fmt.Errorf("ollama: invalid stream line: %w", err)
		}
		if chunk.Error != "" {
			return nil, fmt.Errorf("ollama: %s", chunk.Error)
		}

		if text := chunk.Message.Content; text != "" {
			content.WriteString(text)
			if opts.StreamingFunc != nil {
				if err := opts.StreamingFunc(ctx, []byte(text)); err != nil {
					return nil, err
				}
			}
		}
		calls = append(calls, chunk.Message.ToolCalls...)

		if chunk.Done {
			last = chunk
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}

	choice := &llms.ContentChoice{
		Content:    content.String(),
		StopReason: last.DoneReason,
		GenerationInfo: map[string]any{
			"input_tokens":  last.PromptEvalCount,
			"output_tokens": last.EvalCount,
			"total_tokens":  last.PromptEvalCount + last.EvalCount,
		},
	}
	for _, call := range calls {
		// Ollama doesn't always give tool calls an ID; the agent needs one to match results to calls
		id := call.ID
		if id == "" {
			id = "call_" + uuid.NewString()
		}
		arguments := string(call.Function.Arguments)
		if arguments == "" || arguments == "null" {
			arguments = "{}"
		}
		choice.ToolCalls = append(choice.ToolCalls, llms.ToolCall{
			ID:           id,
			Type:         "function",
			FunctionCall: &llms.FunctionCall{Name: call.Function.Name, Arguments: arguments},
		})
	}

	return &llms.ContentResponse{Choices: []*llms.ContentChoice{choice}}, nil
}

func convertMessages(messages []llms.MessageContent) ([]chatMessage, error) {
	converted := make([]chatMessage, 0, len(messages))

	for _, message := range messages {
		switch message.Role {
		case llms.ChatMessageTypeSystem, llms.ChatMessageTypeHuman, llms.ChatMessageTypeGeneric:
			role := "user"
			if message.Role == llms.ChatMessageTypeSystem {
				role = "system"
			}
			converted = append(converted, chatMessage{Role: role})
			for _, part := range message.Parts {
				switch p := part.(type) {
				case llms.TextContent:
					converted[len(converted)-1].Content += p.Text
				case llms.BinaryContent:
					// Images for vision models (sent base64-encoded)
					converted[len(converted)-1].Images = append(converted[len(converted)-1].Images, p.Data)
				default:
					return nil, fmt.Errorf("ollama: unsupported part %T", part)
				}
			}

		case llms.ChatMessageTypeAI:
			assistant := chatMessage{Role: "assistant"}
			for _, part := range message.Parts {
				switch p := part.(type) {
				case llms.TextContent:
					assistant.Content += p.Text
				case llms.ToolCall:
					if p.FunctionCall == nil {
						continue
					}
					arguments := json.RawMessage(p.FunctionCall.Arguments)
					if !json.Valid(arguments) {
						arguments = json.RawMessage("{}")
					}
					assistant.ToolCalls = append(assistant.ToolCalls, toolCall{
						ID:       p.ID,
						Function: toolCallFunction{Index: len(assistant.ToolCalls), Name: p.FunctionCall.Name, Arguments: arguments},
					})
				default:
					return nil, fmt.Errorf("ollama: unsupported part %T in an assistant message", part)
				}
			}
			converted = append(converted, assistant)

		case llms.ChatMessageTypeTool:
			for _, part := range message.Parts {
				response, ok := part.(llms.ToolCallResponse)
				if !ok {
					return nil, fmt.Errorf("ollama: unsupported part %T in a tool message", part)
				}
				converted = append(converted, chatMessage{
					Role:       "tool",
					Content:    response.Content,
					ToolName:   response.Name,
					ToolCallID: response.ToolCallID,
				})
			}

		default:
			return nil, fmt.Errorf("ollama: role %q is not supported", message.Role)
		}
	}

	if len(converted) == 0 {
		return nil, errors.New("ollama: no message to send")
	}
	return converted, nil
}

// responseError keeps Ollama's message, e.g. "llama3 does not support tools"
func responseError(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 64*1024))

	var errorBody struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errorBody) == nil && errorBody.Error != "" {
		return fmt.Errorf("ollama: %s (%s)", errorBody.Error, response.Status)
	}
	return fmt.Errorf("ollama: %s: %s", response.Status, strings.TrimSpace(string(body)))
}
