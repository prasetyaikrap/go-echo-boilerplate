package agents

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type addressInput struct {
	City string `json:"city" validate:"required"`
}

type pagingInput struct {
	Page int `json:"page" description:"1-based page number"`
}

type searchOrdersInput struct {
	pagingInput
	Status    string            `json:"status" validate:"required,oneof=paid pending cancelled" description:"Order status"`
	Customer  *string           `json:"customer,omitempty" description:"Customer email"`
	Tags      []string          `json:"tags"`
	Address   addressInput      `json:"address"`
	Since     time.Time         `json:"since"`
	Amount    float64           `json:"amount"`
	Urgent    bool              `json:"urgent"`
	Metadata  map[string]string `json:"metadata"`
	Internal  string            `json:"-"`
	unexported string
}

func TestSchemaFromInputStruct(t *testing.T) {
	got := schemaFor(reflect.TypeFor[searchOrdersInput]())

	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"page":     map[string]any{"type": "integer", "description": "1-based page number"},
			"status":   map[string]any{"type": "string", "description": "Order status", "enum": []string{"paid", "pending", "cancelled"}},
			"customer": map[string]any{"type": "string", "description": "Customer email"},
			"tags":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"address": map[string]any{
				"type":       "object",
				"properties": map[string]any{"city": map[string]any{"type": "string"}},
				"required":   []string{"city"},
			},
			"since":    map[string]any{"type": "string", "format": "date-time"},
			"amount":   map[string]any{"type": "number"},
			"urgent":   map[string]any{"type": "boolean"},
			"metadata": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
		},
		"required": []string{"status"},
	}

	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		t.Errorf("schema:\n%s", gotJSON)
	}
}

func TestToolWithoutInputHasNoSchema(t *testing.T) {
	tool := NewTool("ping", "", func(_ context.Context, _ struct{}) (string, error) { return "pong", nil })

	if tool.Parameters != nil {
		t.Errorf("parameters = %v, want nil", tool.Parameters)
	}
	if output, err := tool.Run(context.Background(), ""); output != "pong" || err != nil {
		t.Errorf("got (%q, %v)", output, err)
	}
}
