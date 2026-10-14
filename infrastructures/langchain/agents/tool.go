package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"go-serviceboilerplate/commons/utils"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// Tool is a function the model can call. The model sends arguments as JSON matching Parameters.
// Build tools with NewTool; set the fields directly only when the schema cannot come from a struct.
type Tool struct {
	// Name the model calls the tool by: letters, digits, "_" and "-" only (max 64)
	Name string
	// Description tells the model when and how to use the tool. Be specific: it is the tool's whole documentation
	Description string
	// Parameters is the JSON schema of the input object. nil means the tool takes no input
	Parameters map[string]any
	// Run executes the tool with the raw JSON arguments and returns the result shown to the model
	Run func(ctx context.Context, arguments string) (string, error)
}

// NewTool creates a tool that works like an HTTP handler: the model's JSON arguments are decoded into TInput,
// validated with its `validate` tags, and passed to run, which calls a use case. The JSON schema the model sees is
// generated from TInput's `json`, `validate` and `description` tags. Use struct{} for tools without input.
//
//	// commons/models/orders.go
//	type GetOrderToolInput struct {
//		OrderID string `json:"order_id" validate:"required" description:"The order ID, e.g. ord_123"`
//	}
//
//	// interfaces/agents/orders/handler.go
//	agents.NewTool("get_order", "Returns an order's status and items. Use it when the user asks about an order.",
//		func(ctx context.Context, input models.GetOrderToolInput) (string, error) {
//			order, err := h.ordersUsecase.GetOrder(ctx, input.OrderID) // ctx carries request ID and user claims
//			if err != nil {
//				return "", err // utils.New…Error messages reach the model; other errors are hidden and logged
//			}
//			return agents.JSON(order)
//		},
//	)
func NewTool[TInput any](name, description string, run func(ctx context.Context, input TInput) (string, error)) Tool {
	var parameters map[string]any
	if inputType := reflect.TypeFor[TInput](); inputType.Kind() == reflect.Struct && inputType.NumField() > 0 {
		parameters = schemaFor(inputType)
	}

	// Same validation rules as HTTP request models (c.Validate)
	inputValidator := validator.New()

	return Tool{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Run: func(ctx context.Context, arguments string) (string, error) {
			var input TInput
			if arguments != "" && arguments != "null" {
				if err := json.Unmarshal([]byte(arguments), &input); err != nil {
					return "", utils.NewClientError(fmt.Errorf("invalid arguments for %s: %w", name, err))
				}
			}

			if reflect.TypeFor[TInput]().Kind() == reflect.Struct {
				if err := inputValidator.Struct(input); err != nil {
					return "", utils.NewClientError(fmt.Errorf("invalid arguments for %s: %w", name, err))
				}
			}

			return run(ctx, input)
		},
	}
}

// JSON encodes a use case result as the tool output
func JSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// FromLangChainTool adapts a langchaingo tools.Tool (a single string input, as used with langchaingo's ReAct agents)
func FromLangChainTool(tool tools.Tool) Tool {
	type langChainToolInput struct {
		Input string `json:"input" validate:"required" description:"The tool input"`
	}

	return NewTool(tool.Name(), tool.Description(), func(ctx context.Context, input langChainToolInput) (string, error) {
		return tool.Call(ctx, input.Input)
	})
}

func (t Tool) definition() llms.Tool {
	parameters := t.Parameters
	if parameters == nil {
		parameters = map[string]any{"type": "object", "properties": map[string]any{}}
	}

	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  parameters,
		},
	}
}
