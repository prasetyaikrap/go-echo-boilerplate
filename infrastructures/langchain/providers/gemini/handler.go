package gemini

import (
	"context"
	"fmt"
	"go-serviceboilerplate/commons/models"

	"github.com/tmc/langchaingo/llms"
	"google.golang.org/genai"
)

// NewGeminiLLM creates a Gemini-backed llms.Model on Google's Gen AI SDK.
// httpOptions are extra SDK options, e.g. a BaseURL in tests.
func NewGeminiLLM(cfg models.LLMConfig, httpOptions ...genai.HTTPOptions) (llms.Model, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model name is required for gemini provider")
	}

	clientConfig := &genai.ClientConfig{APIKey: cfg.APIKey, Backend: genai.BackendGeminiAPI}
	if len(httpOptions) > 0 {
		clientConfig.HTTPOptions = httpOptions[0]
	}

	client, err := genai.NewClient(context.Background(), clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini LLM: %w", err)
	}

	return &Model{client: client, model: cfg.Model, turns: newModelTurnCache(models.LLMAssistantTurnCacheSize)}, nil
}
