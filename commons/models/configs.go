package models

import (
	"time"

	"github.com/labstack/echo/v4/middleware"
)

// ApplicationConfig holds the service-level settings (server, client verification, auth tokens)
type ApplicationConfig struct {
	// Port the HTTP server listens on (env: PORT, required)
	Port 					string
	// ClientID identifies this service. Used as the JWT issuer (env: CLIENT_ID, required)
	ClientID				string
	// AllowedCleintIDs lists the X-Client-Id values allowed to call this service. "*" allows all (env: ALLOWED_CLIENT_IDS, comma separated)
	AllowedCleintIDs 		[]string
	// SecretToken is compared with the X-Service-Token header for service-to-service calls (env: SERVICE_TOKEN)
	SecretToken				string
	// JWTAccessSecret signs and verifies access tokens (env: JWT_ACCESS_SECRET)
	JWTAccessSecret			string
	// JWTRefreshSecret signs and verifies refresh tokens. Keep it different from JWTAccessSecret (env: JWT_REFRESH_SECRET)
	JWTRefreshSecret		string
	// AccessTokenExpiration is the access token lifetime (env: ACCESS_TOKEN_EXPIRATION in minutes, default 30)
	AccessTokenExpiration 	time.Duration
	// RefreshTokenExpiration is the refresh token lifetime (env: REFRESH_TOKEN_EXPIRATION in minutes, default 7 days)
	RefreshTokenExpiration 	time.Duration
	// CORSConfig is applied to every route. Origins come from ALLOWED_ORIGINS (comma separated, default "*")
	CORSConfig				middleware.CORSConfig
}

// PostgresConfig holds the configuration for a single Postgres connection. Create one per database the service needs.
// Loaded with GetPostgresEnvConfig(name, prefix), e.g. prefix "DB_" reads DB_DSN, DB_MAX_CONN, etc.
type PostgresConfig struct {
	// Name identifies the database in logs and errors (e.g. "main", "analytics")
	Name 	string
	// DSN connection string, e.g. "host=localhost user=postgres password=secret dbname=app port=5432 sslmode=disable TimeZone=Asia/Jakarta" (env: <prefix>DSN)
	DSN	  	 string
	// MaxConnIdle is the maximum number of idle connections kept in the pool (env: <prefix>MAX_CONN_IDLE, default 10)
	MaxConnIdle int
	// MaxConnIdleLifeTime is how long a connection may stay idle before being closed (env: <prefix>MAX_CONN_IDLE_LIFETIME in minutes, default 30)
	MaxConnIdleLifeTime time.Duration
	// MaxConn is the maximum number of open connections to the database (env: <prefix>MAX_CONN, default 25)
	MaxConn int
	// MaxConnLifeTime is how long a connection may be reused before being closed (env: <prefix>MAX_CONN_LIFETIME in minutes, default 60)
	MaxConnLifeTime time.Duration
	// LogLevel for GORM SQL logs: "silent", "error", "warn" or "info" (env: <prefix>LOG_LEVEL, default "info")
	LogLevel 	string

	// AutoMigrate runs GORM AutoMigrate for the models passed to NewPostgresInstance on startup (env: <prefix>AUTO_MIGRATE, "true" to enable)
	AutoMigrate bool
}

// StorageS3Config holds the configuration for any S3-compatible storage (AWS S3, Cloudflare R2, MinIO, etc.)
type StorageS3Config struct {
	// Endpoint is empty for AWS S3. Cloudflare R2: "https://<account_id>.r2.cloudflarestorage.com" (env: S3_ENDPOINT)
	Endpoint 			string
	// Region for AWS S3 (e.g. "ap-southeast-1"). Cloudflare R2: "auto" (env: S3_REGION, default "auto")
	Region 				string
	// AccessKeyID of the storage API credential (env: S3_ACCESS_KEY)
	AccessKeyID 		string
	// SecretAccessKey of the storage API credential (env: S3_SECRET_ACCESS_KEY)
	SecretAccessKey 	string
	// BucketName every object is stored in. Required when the storage is initialized (env: S3_BUCKET_NAME)
	BucketName 			string
	// PublicHost is the base URL for public object links, e.g. "https://cdn.example.com" or an R2 "https://pub-xxx.r2.dev" (env: S3_PUBLIC_HOST)
	PublicHost 	string
	// BaseFolder is prepended to every object key, e.g. "uploads" stores "a.png" as "uploads/a.png" (env: S3_BASE_FOLDER)
	BaseFolder		string
	// UsePathStyle is required by MinIO and some self-hosted storages. Keep false for AWS S3 and Cloudflare R2 (env: S3_USE_PATH_STYLE)
	UsePathStyle 	bool
	// DefaultPresignExpire is used when a presign method is called with expireTime <= 0. Zero falls back to the SDK default (15 minutes) (env: S3_PRESIGN_EXPIRE in minutes, default 15)
	DefaultPresignExpire 	time.Duration
}

// RedisConfig holds the configuration for a single Redis connection. Create one per Redis instance the service needs.
// Loaded with GetRedisEnvConfig(name, prefix), e.g. prefix "REDIS_" reads REDIS_URL, REDIS_KEY_PREFIX, etc.
type RedisConfig struct {
	// Name identifies the instance in logs and errors (e.g. "main", "session")
	Name 		string
	// URL format: "redis://<user>:<password>@<host>:<port>/<db>". Use "rediss://" for TLS (env: <prefix>URL, default "redis://localhost:6379/0")
	URL 		string
	// KeyPrefix is prepended to every key (e.g. "boilerplate:"), useful when the Redis is shared between services (env: <prefix>KEY_PREFIX)
	KeyPrefix 	string
	// DefaultTTL is used when SaveCache is called with ttl <= 0. Zero means no expiration (env: <prefix>DEFAULT_TTL in minutes, default 60)
	DefaultTTL 	time.Duration
	// PoolSize overrides the go-redis default pool size (10 * GOMAXPROCS) when > 0 (env: <prefix>POOL_SIZE)
	PoolSize 	int
	// PingTimeout bounds the connection check on startup. Zero relies on the go-redis dial timeout (5 seconds) (env: <prefix>PING_TIMEOUT in seconds, default 5)
	PingTimeout 	time.Duration
}

// StompConfig holds the configuration for a single STOMP message broker connection (ActiveMQ, RabbitMQ STOMP plugin, Artemis, etc.).
// Create one per connection the service needs. Loaded with GetStompEnvConfig(name, prefix), e.g. prefix "STOMP_" reads STOMP_ADDRESS, STOMP_LOGIN, etc.
type StompConfig struct {
	// Name identifies the connection in logs and errors (e.g. "main", "events")
	Name 		string
	// Address of the broker as "<host>:<port>" (env: <prefix>ADDRESS, default "localhost:61613")
	Address 	string
	// Login username. Empty sends no credentials (env: <prefix>LOGIN)
	Login 		string
	// Passcode for Login (env: <prefix>PASSCODE)
	Passcode 	string
	// Host is the virtual host sent on CONNECT. Empty uses the host part of Address. RabbitMQ default vhost: "/" (env: <prefix>HOST)
	Host 		string
	// UseTLS connects over TLS (e.g. Amazon MQ "stomp+ssl" endpoints) (env: <prefix>USE_TLS, "true" to enable)
	UseTLS 		bool
	// HeartBeat is the send/receive heart-beat interval used to detect dead connections. Zero uses the go-stomp default (1 minute) (env: <prefix>HEARTBEAT in seconds, default 10)
	HeartBeat 	time.Duration
	// ConnectTimeout bounds dialing and the CONNECT handshake. Zero means no timeout (env: <prefix>CONNECT_TIMEOUT in seconds, default 10)
	ConnectTimeout 	time.Duration
	// ReconnectInterval is the wait between reconnect attempts after the connection drops. Zero disables auto-reconnect (env: <prefix>RECONNECT_INTERVAL in seconds, default 5)
	ReconnectInterval 	time.Duration
}

// ENVConfig is the root configuration loaded from environment variables (and .env if present) on startup
type ENVConfig struct {
	// Application holds the service-level settings
	Application 	ApplicationConfig
	// MainDB is the primary Postgres database (env prefix: DB_)
	MainDB		PostgresConfig
	// MainRedis is the primary Redis instance (env prefix: REDIS_)
	MainRedis	RedisConfig
	// MainStomp is the primary STOMP message broker connection (env prefix: STOMP_)
	MainStomp	StompConfig
	// StorageS3 is the S3-compatible object storage (env prefix: S3_)
	StorageS3	StorageS3Config
	// LLM selects and configures the LLM provider (env prefix: LLM_)
	LLM 		LLMConfig
}

// LLMProvider represents the supported LLM backend providers.
type LLMProvider string

// LLMConfig holds the configuration for the LLM provider.
type LLMConfig struct {
	// Provider selects which LLM backend to use ("ollama", "gemini", or "openai") (env: LLM_PROVIDER).
	Provider LLMProvider

	// Model is the model name/identifier (e.g. "llama3", "gemini-2.0-flash") (env: LLM_MODEL, default "llama3").
	Model string

	// OllamaServerURL is required when Provider is LLMProviderOllama.
	// Example: "http://localhost:11434" (env: LLM_OLLAMA_SERVER_URL)
	OllamaServerURL string

	// GeminiAPIKey is required when Provider is LLMProviderGemini (env: LLM_GEMINI_API_KEY).
	GeminiAPIKey string

	// OpenAIAPIKey is required when Provider is LLMProviderOpenAI (env: LLM_OPENAI_API_KEY).
	OpenAIAPIKey string

	// ContextWindowSize is the number of recent messages to keep in context for the LLM (env: LLM_CONTEXT_WINDOW_SIZE, default 20)
	ContextWindowSize int 
}