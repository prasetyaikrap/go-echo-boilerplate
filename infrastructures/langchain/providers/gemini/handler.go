package gemini

import (
	"context"
	"fmt"
	"go-serviceboilerplate/commons/models"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
)

// NewGeminiLLM creates a new Google Gemini-backed LLM client using the provided config.
func NewGeminiLLM(cfg models.LLMConfig) (llms.Model, error) {
	if cfg.GeminiAPIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model name is required for gemini provider")
	}

	llm, err := googleai.New(
		context.Background(),
		googleai.WithAPIKey(cfg.GeminiAPIKey),
		googleai.WithDefaultModel(cfg.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini LLM: %w", err)
	}

	return llm, nil
}
