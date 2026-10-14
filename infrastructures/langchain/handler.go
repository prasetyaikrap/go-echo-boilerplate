package langchain

import (
	"fmt"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
	anthropicProvider "go-serviceboilerplate/infrastructures/langchain/providers/anthropic"
	geminiProvider "go-serviceboilerplate/infrastructures/langchain/providers/gemini"
	ollamaProvider "go-serviceboilerplate/infrastructures/langchain/providers/ollama"
	openaiProvider "go-serviceboilerplate/infrastructures/langchain/providers/openai"

	"github.com/tmc/langchaingo/llms"
)

// LangChainInstance holds one initialized LLM and its provider. Create one per LLM config, e.g.
//
//	mainLLM := langchain.NewLangChainInstance(configs, configs.Envs.MainLLM)
//	fastLLM := langchain.NewLangChainInstance(configs, configs.Envs.FastLLM)
type LangChainInstance struct {
	Name     string
	LLM      llms.Model
	Provider models.LLMProvider
	Model    string
	// Temperature is the configured sampling temperature, or nil to keep the provider default. Pass CallOptions()
	// to each LLM call rather than reading it directly, so providers that reject it never receive it
	Temperature *float64
}

// NewLangChainInstance initialises the LLM backend selected by cfg.Provider, or returns nil when it is empty.
// Supported providers: "ollama", "gemini", "openai", "anthropic".
func NewLangChainInstance(configs *configurations.Configs, cfg models.LLMConfig) *LangChainInstance {
	var (
		llm llms.Model
		err error
	)

	if cfg.Provider == "" {
		configs.Logger.Info("No LLM provider configured. LangChain features will be disabled.", "name", cfg.Name)
		return nil
	}

	switch cfg.Provider {
	case models.LLMProviderOllama:
		llm, err = ollamaProvider.NewOllamaLLM(cfg)
	case models.LLMProviderGemini:
		llm, err = geminiProvider.NewGeminiLLM(cfg)
	case models.LLMProviderOpenAI:
		llm, err = openaiProvider.NewOpenAILLM(cfg)
	case models.LLMProviderAnthropic:
		llm, err = anthropicProvider.NewAnthropicLLM(cfg)
		if err == nil && cfg.Temperature != nil {
			configs.Logger.Warn("Temperature is ignored for anthropic: current Claude models reject sampling parameters; use effort instead", "name", cfg.Name)
		}
	default:
		err = fmt.Errorf(
			"unsupported LLM provider: %q — valid options are %q, %q, %q, and %q",
			cfg.Provider,
			models.LLMProviderOllama,
			models.LLMProviderGemini,
			models.LLMProviderOpenAI,
			models.LLMProviderAnthropic,
		)
	}

	if err != nil {
		configs.Logger.Fatal("LangChain initialization failed", "name", cfg.Name, "provider", cfg.Provider, "error", err)
		return nil
	}

	configs.Logger.Info("LangChain initialized successfully", "name", cfg.Name, "provider", cfg.Provider, "model", cfg.Model)

	return &LangChainInstance{
		Name:        cfg.Name,
		LLM:         llm,
		Provider:    cfg.Provider,
		Model:       cfg.Model,
		Temperature: cfg.Temperature,
	}
}

// CallOptions returns the configured call options (currently the temperature) for this provider, e.g.
//
//	l.LLM.GenerateContent(ctx, messages, append(l.CallOptions(), llms.WithStreamingFunc(fn))...)
func (l *LangChainInstance) CallOptions() []llms.CallOption {
	var options []llms.CallOption
	if l.Temperature != nil && !l.IsAnthropicProvider() {
		options = append(options, llms.WithTemperature(*l.Temperature))
	}
	return options
}

func (l *LangChainInstance) IsOllamaProvider() bool {
	return l.Provider == models.LLMProviderOllama
}

func (l *LangChainInstance) IsGeminiProvider() bool {
	return l.Provider == models.LLMProviderGemini
}

func (l *LangChainInstance) IsOpenAIProvider() bool {
	return l.Provider == models.LLMProviderOpenAI
}

func (l *LangChainInstance) IsAnthropicProvider() bool {
	return l.Provider == models.LLMProviderAnthropic
}
