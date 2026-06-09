package openai

import (
	"fmt"
	"go-serviceboilerplate/commons/models"

	"github.com/tmc/langchaingo/llms"
	openaiLLM "github.com/tmc/langchaingo/llms/openai"
)

// NewOpenAILLM creates a new OpenAI-backed LLM client using the provided config.
func NewOpenAILLM(cfg models.LLMConfig) (llms.Model, error) {
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("openai API key is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model name is required for openai provider")
	}

	llm, err := openaiLLM.New(
		openaiLLM.WithToken(cfg.OpenAIAPIKey),
		openaiLLM.WithModel(cfg.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create openai LLM: %w", err)
	}

	return llm, nil
}
