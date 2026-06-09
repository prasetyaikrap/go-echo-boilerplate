package models

var (
	XClientIdHeader	= "X-Client-Id"
	XServiceTokenHeader = "X-Service-Token"
	XRenewTokenHeader = "X-Renew-Token"
	XProxyByHeader = "X-Proxy-By"
	XServiceNameHeader = "X-Service-Name"
	XUserIdHeader = "X-User-Id"
	AuthorizationHeader = "Authorization"

	ContextUserClaim = "user_claim"
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