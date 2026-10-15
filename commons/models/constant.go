package models

import (
	"errors"
)

var (
	XClientIdHeader	= "X-Client-Id"
	XClientVersionHeader = "X-Client-Version"
	XServiceTokenHeader = "X-Service-Token"
	XRenewTokenHeader = "X-Renew-Token"
	XProxyByHeader = "X-Proxy-By"
	XServiceNameHeader = "X-Service-Name"
	XUserIdHeader = "X-User-Id"
	AuthorizationHeader = "Authorization"
	// HealthLivenessPath reports whether the process is alive (Kubernetes livenessProbe). It checks no dependencies
	HealthLivenessPath = "/healthz"
	// HealthReadinessPath reports whether the service can serve traffic (Kubernetes readinessProbe). It checks dependencies
	HealthReadinessPath = "/readyz"

	// ClientVerificationSkipPaths are route paths that do not require the X-Client-Id header (e.g. probes from Kubernetes).
	// Add "/swagger/*" here to open the Swagger UI in a browser
	ClientVerificationSkipPaths = []string{HealthLivenessPath, HealthReadinessPath}

	// RequestTimeoutSkipPaths are route paths without the REQUEST_TIMEOUT deadline, e.g. SSE streams of LLM responses
	// that can run longer than the timeout. Use the route pattern, such as "/v1/agents/:id/chat"
	RequestTimeoutSkipPaths = []string{}

	// XRequestIDHeader carries the request ID across services (HTTP) and through message brokers (message header)
	XRequestIDHeader = "X-Request-Id"

	// ContextUserClaim is the echo.Context key for token claims (c.Get). Below the HTTP layer use utils.AccessTokenClaimsFromContext instead
	ContextUserClaim = "user_claim"
)

// ContextKey is the type for values stored in a context.Context. A dedicated type prevents collisions with keys from other packages
type ContextKey string

const (
	// ContextKeyRequestID holds the request ID (X-Request-Id), or the message ID for broker messages. Added to every log line
	ContextKeyRequestID ContextKey = "request_id"
	// ContextKeyAccessTokenClaims holds the verified *AccessTokenClaims of the caller
	ContextKeyAccessTokenClaims ContextKey = "access_token_claims"
	// ContextKeyRefreshTokenClaims holds the verified *RefreshTokenClaims when the request renews a token
	ContextKeyRefreshTokenClaims ContextKey = "refresh_token_claims"
)

const (
	OpEqual               = "equal"
	OpNotEqual			  = "not-equal"
	OpIlike               = "ilike"
	OpLike                = "like"
	OpStartsWith          = "starts-with"
	OpEndsWith            = "ends-with"
	OpSliceIn             = "slice-in"
	OpSliceNotIn          = "slice-not-in"
	OpGreaterThan          = "greater-than"
	OpGreaterThanOrEqualTo = "greater-than-or-equal-to"
	OpLessThan            = "less-than"
	OpLessThanOrEqualTo   = "less-than-or-equal-to"
	OpBetween             = "between"
	OpNotBetween          = "not-between"
	OpIsNull			  = "is-null"
	OpIsNotNull			  = "is-not-null"
	OpNullConditional	  = "null-conditional"
	OpBoolean			  = "boolean"
	OpOr				  = "or"
)

const (
	LLMProviderOllama LLMProvider = "ollama"
	LLMProviderGemini LLMProvider = "gemini"
	LLMProviderOpenAI LLMProvider = "openai"
	LLMProviderAnthropic LLMProvider = "anthropic"
)

const (
	AgentsPromptDirectory = "files/agents/prompts"
)

const (
	// LLMAssistantTurnCacheSize is how many assistant turns with tool calls are kept in memory, so what langchaingo
	// messages can't carry (Claude's thinking blocks, Gemini's thought signatures) is sent back unchanged on the next
	// request of a tool-use loop
	LLMAssistantTurnCacheSize = 1000

	// GeminiSkipThoughtSignature tells Gemini to skip signature validation for a tool call the adapter has no
	// signature for. Google documents it as a last resort: it lowers the model's quality for that turn
	GeminiSkipThoughtSignature = "skip_thought_signature_validator"

	// LLMDefaultMaxTokens / LLMDefaultStreamingMaxTokens are Claude's output limits when the caller sets none.
	// Generous, so answers are not cut off mid-thought; streaming allows more since it has no HTTP timeout risk
	LLMDefaultMaxTokens          = 16000
	LLMDefaultStreamingMaxTokens = 64000

	// AgentDefaultMaxIterations is how many model calls an agent run may make (each tool round is one) when the
	// agent sets none. Stops a model that keeps calling tools from looping forever
	AgentDefaultMaxIterations = 10
)

var (
	// LLMServerSideFallbackModels are Claude model ID prefixes that accept server-side refusal fallbacks
	LLMServerSideFallbackModels = []string{"claude-fable-5-1", "claude-opus-5", "claude-sonnet-5-5"}
)

var (
	// ErrCacheMiss is returned by cache getters when the key does not exist or has expired
	ErrCacheMiss = errors.New("cache miss")

	// ErrMessageBrokerClosed is returned when publishing or subscribing on a message broker connection that has been closed
	ErrMessageBrokerClosed = errors.New("message broker connection closed")

	// ErrLLMRefusal is returned when the LLM (and every fallback model) declines the request. Check it with errors.Is
	ErrLLMRefusal = errors.New("llm request declined")

	// ErrAgentMaxIterations is returned when an agent is still calling tools after its maximum number of iterations
	ErrAgentMaxIterations = errors.New("agent reached its maximum number of iterations")
)