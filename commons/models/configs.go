package models

import (
	"time"

	"github.com/labstack/echo/v4/middleware"
)

type ApplicationConfig struct {
	Port 					string
	ClientID				string
	AllowedCleintIDs 		[]string
	SecretToken				string
	JWTAccessSecret			string
	JWTRefreshSecret		string
	AccessTokenExpiration 	time.Duration
	RefreshTokenExpiration 	time.Duration
	CORSConfig				middleware.CORSConfig
}

type DBConfig struct {
	DSN	  	 string
	MaxConnIdle int
	MaxConnIdleLifeTime time.Duration
	MaxConn int
	MaxConnLifeTime time.Duration

	AutoMigrate bool
}

type StorageR2Config struct {
	AccountID 			string
	AccessKeyID 		string
	SecretAccessKey 	string
	BucketName 			string
	PublicHost 	string
	BaseFolder		string
}

type ENVConfig struct {
	Application 	ApplicationConfig
	DB			DBConfig
	StorageR2	StorageR2Config
	LLM 		LLMConfig
}

// LLMProvider represents the supported LLM backend providers.
type LLMProvider string

// LLMConfig holds the configuration for the LLM provider.
type LLMConfig struct {
	// Provider selects which LLM backend to use ("ollama", "gemini", or "openai").
	Provider LLMProvider

	// Model is the model name/identifier (e.g. "llama3", "gemini-2.0-flash").
	Model string

	// OllamaServerURL is required when Provider is LLMProviderOllama.
	// Example: "http://localhost:11434"
	OllamaServerURL string

	// GeminiAPIKey is required when Provider is LLMProviderGemini.
	GeminiAPIKey string

	// OpenAIAPIKey is required when Provider is LLMProviderOpenAI.
	OpenAIAPIKey string

	// Optional: number of recent messages to keep in context for the LLM
	ContextWindowSize int 
}