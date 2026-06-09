package langchain

import (
	"fmt"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
	geminiProvider "go-serviceboilerplate/infrastructures/langchain/providers/gemini"
	ollamaProvider "go-serviceboilerplate/infrastructures/langchain/providers/ollama"
	openaiProvider "go-serviceboilerplate/infrastructures/langchain/providers/openai"

	"github.com/tmc/langchaingo/llms"
)

// LangChainInstance holds the initialized LLM model and its active provider.
// Use NewLangChainInstance to construct it. Swap providers by changing LLM_PROVIDER env.
type LangChainInstance struct {
	LLM      llms.Model
	Provider models.LLMProvider
	Model    string
}

// NewLangChainInstance initialises the LLM backend selected by configs.Envs.LLM.Provider.
// Supported providers: "ollama", "gemini", "openai".
func NewLangChainInstance(configs *configurations.Configs) (*LangChainInstance) {
	cfg := configs.Envs.LLM

	var (
		llm llms.Model
		err error
	)

	if cfg.Provider == "" {
		configs.Logger.Info("No LLM provider configured. LangChain features will be disabled.")
		return nil
	}

	switch cfg.Provider {
	case models.LLMProviderOllama:
		llm, err = ollamaProvider.NewOllamaLLM(cfg)
	case models.LLMProviderGemini:
		llm, err = geminiProvider.NewGeminiLLM(cfg)
	case models.LLMProviderOpenAI:
		llm, err = openaiProvider.NewOpenAILLM(cfg)
	default:
		err = fmt.Errorf(
			"unsupported LLM provider: %q — valid options are %q, %q, and %q",
			cfg.Provider,
			models.LLMProviderOllama,
			models.LLMProviderGemini,
			models.LLMProviderOpenAI,
		)
	}

	if err != nil {
		configs.Logger.Fatal("LangChain initialization failed [provider=%s]: %v", cfg.Provider, err)
		return nil
	}

	configs.Logger.Info("LangChain initialized successfully with provider: %s", cfg.Provider)

	return &LangChainInstance{
		LLM:      llm,
		Provider: cfg.Provider,
		Model:    cfg.Model,
	}
}
