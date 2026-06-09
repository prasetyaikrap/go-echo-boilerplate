package ollama

import (
	"fmt"
	"go-serviceboilerplate/commons/models"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// NewOllamaLLM creates a new Ollama-backed LLM client using the provided config.
func NewOllamaLLM(cfg models.LLMConfig) (llms.Model, error) {
	if cfg.OllamaServerURL == "" {
		return nil, fmt.Errorf("ollama server URL is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model name is required for ollama provider")
	}

	llm, err := ollama.New(
		ollama.WithModel(cfg.Model),
		ollama.WithServerURL(cfg.OllamaServerURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama LLM: %w", err)
	}

	return llm, nil
}
