package models

import (
	"errors"
)

var (
	XClientIdHeader	= "X-Client-Id"
	XServiceTokenHeader = "X-Service-Token"
	XRenewTokenHeader = "X-Renew-Token"
	XProxyByHeader = "X-Proxy-By"
	XServiceNameHeader = "X-Service-Name"
	XUserIdHeader = "X-User-Id"
	AuthorizationHeader = "Authorization"
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
)

const (
	AgentsPromptDirectory = "files/agents/prompts"
)

var (
	// ErrCacheMiss is returned by cache getters when the key does not exist or has expired
	ErrCacheMiss = errors.New("cache miss")

	// ErrMessageBrokerClosed is returned when publishing or subscribing on a message broker connection that has been closed
	ErrMessageBrokerClosed = errors.New("message broker connection closed")
)