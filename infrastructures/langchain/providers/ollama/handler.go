package ollama

import (
	"fmt"
	"go-serviceboilerplate/commons/models"
	"net/http"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// NewOllamaLLM creates an Ollama-backed llms.Model on Ollama's /api/chat.
func NewOllamaLLM(cfg models.LLMConfig) (llms.Model, error) {
	if cfg.OllamaServerURL == "" {
		return nil, fmt.Errorf("ollama server URL is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("model name is required for ollama provider")
	}

	// No client timeout: requests end with their context, and streams can run long
	return &Model{serverURL: strings.TrimRight(cfg.OllamaServerURL, "/"), model: cfg.Model, httpClient: &http.Client{}}, nil
}
