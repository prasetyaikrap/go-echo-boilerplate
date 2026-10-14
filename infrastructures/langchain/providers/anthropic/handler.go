package anthropic

import (
	"fmt"
	"go-serviceboilerplate/commons/models"
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/tmc/langchaingo/llms"
)

// NewAnthropicLLM creates a Claude-backed llms.Model on the official Anthropic SDK.
// requestOptions are extra SDK options, e.g. option.WithBaseURL in tests.
func NewAnthropicLLM(cfg models.LLMConfig, requestOptions ...option.RequestOption) (llms.Model, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model name is required for anthropic provider")
	}

	validEfforts := []string{"low", "medium", "high", "xhigh", "max"}
	effort := strings.ToLower(strings.TrimSpace(cfg.Effort))
	if effort != "" && !slices.Contains(validEfforts, effort) {
		return nil, fmt.Errorf("invalid effort %q, expected one of %v", cfg.Effort, validEfforts)
	}

	clientOptions := append([]option.RequestOption{option.WithAPIKey(cfg.APIKey)}, requestOptions...)

	return &Model{
		client:          anthropic.NewClient(clientOptions...),
		model:           cfg.Model,
		effort:          effort,
		refusalFallback: cfg.RefusalFallback,
		assistantTurns:  newAssistantTurnCache(models.LLMAssistantTurnCacheSize),
	}, nil
}
