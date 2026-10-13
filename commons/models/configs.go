package models

import (
	"os"
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
	// RequestTimeout is the deadline set on every HTTP request's context. Database, cache and broker calls made with that ctx
	// are cancelled when it passes, and the request returns 503. Zero disables it (env: REQUEST_TIMEOUT in seconds, default 30)
	RequestTimeout 			time.Duration
	// ShutdownTimeout is how long the server waits for in-flight requests to finish on SIGINT/SIGTERM before forcing exit.
	// Zero waits indefinitely (env: SHUTDOWN_TIMEOUT in seconds, default 10)
	ShutdownTimeout 		time.Duration
	// ServerReadHeaderTimeout closes connections that do not send their request headers in time (slowloris protection).
	// Enabled by default as recommended by gosec G112; 0 disables it (env: SERVER_READ_HEADER_TIMEOUT in seconds, default 10)
	ServerReadHeaderTimeout time.Duration
	// ServerReadTimeout limits reading the whole request including the body. Large uploads on slow networks can hit it.
	// Zero disables it (env: SERVER_READ_TIMEOUT in seconds, default disabled)
	ServerReadTimeout 		time.Duration
	// ServerWriteTimeout limits the time from reading the request headers until the response is fully written.
	// It cuts off streaming responses (SSE), so leave it disabled when streaming. Zero disables it (env: SERVER_WRITE_TIMEOUT in seconds, default disabled)
	ServerWriteTimeout 		time.Duration
	// ServerIdleTimeout closes keep-alive connections idle between requests. Zero falls back to ServerReadTimeout,
	// and no limit when that is also zero (env: SERVER_IDLE_TIMEOUT in seconds, default disabled)
	ServerIdleTimeout 		time.Duration
	// RequestBodyLimit rejects request bodies larger than this with 413, e.g. "10M", "512K", "1G". "0" disables it
	// (env: REQUEST_BODY_LIMIT, default "10M")
	RequestBodyLimit 		string
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

// MongoConfig holds the configuration for a single MongoDB database. Create one per database the service needs.
// Loaded with GetMongoEnvConfig(name, prefix), e.g. prefix "MONGO_" reads MONGO_URI, MONGO_DATABASE, etc.
type MongoConfig struct {
	// Name identifies the database in logs and errors (e.g. "main", "audit")
	Name 		string
	// URI format: "mongodb://<user>:<password>@<host>:<port>/?authSource=admin". Use "mongodb+srv://" for Atlas / DNS seed lists (env: <prefix>URI, default "mongodb://localhost:27017")
	URI 		string
	// Database name every collection is read from and written to. Required when connecting (env: <prefix>DATABASE)
	Database 	string
	// MaxPoolSize is the maximum number of connections in the pool. Zero uses the driver default (100) (env: <prefix>MAX_POOL_SIZE)
	MaxPoolSize 	uint64
	// MinPoolSize is the number of connections kept open even when idle (env: <prefix>MIN_POOL_SIZE, default 0)
	MinPoolSize 	uint64
	// MaxConnIdleTime closes connections idle for longer than this. Zero means no limit (env: <prefix>MAX_CONN_IDLE_TIME in minutes)
	MaxConnIdleTime 	time.Duration
	// ConnectTimeout bounds connecting, server selection and the startup ping. Zero uses the driver defaults (30 seconds) (env: <prefix>CONNECT_TIMEOUT in seconds, default 10)
	ConnectTimeout 	time.Duration
	// AutoMigrate creates the indexes passed to NewMongoInstance on startup (env: <prefix>AUTO_MIGRATE, "true" to enable)
	AutoMigrate 	bool
}

// S3Config holds the configuration for a single S3-compatible storage bucket (AWS S3, Cloudflare R2, MinIO, etc.).
// Create one per bucket the service needs. Loaded with GetS3EnvConfig(name, prefix), e.g. prefix "S3_" reads S3_ENDPOINT, S3_BUCKET_NAME, etc.
type S3Config struct {
	// Name identifies the storage in logs and errors (e.g. "main", "media")
	Name 				string
	// Endpoint is empty for AWS S3. Cloudflare R2: "https://<account_id>.r2.cloudflarestorage.com" (env: <prefix>ENDPOINT)
	Endpoint 			string
	// Region for AWS S3 (e.g. "ap-southeast-1"). Cloudflare R2: "auto" (env: <prefix>REGION, default "auto")
	Region 				string
	// AccessKeyID of the storage API credential (env: <prefix>ACCESS_KEY)
	AccessKeyID 		string
	// SecretAccessKey of the storage API credential (env: <prefix>SECRET_ACCESS_KEY)
	SecretAccessKey 	string
	// BucketName every object is stored in. Required when the storage is initialized (env: <prefix>BUCKET_NAME)
	BucketName 			string
	// PublicHost is the base URL for public object links, e.g. "https://cdn.example.com" or an R2 "https://pub-xxx.r2.dev" (env: <prefix>PUBLIC_HOST)
	PublicHost 	string
	// BaseFolder is prepended to every object key, e.g. "uploads" stores "a.png" as "uploads/a.png" (env: <prefix>BASE_FOLDER)
	BaseFolder		string
	// UsePathStyle is required by MinIO and some self-hosted storages. Keep false for AWS S3 and Cloudflare R2 (env: <prefix>USE_PATH_STYLE)
	UsePathStyle 	bool
	// DefaultPresignExpire is used when a presign method is called with expireTime <= 0. Zero falls back to the SDK default (15 minutes) (env: <prefix>PRESIGN_EXPIRE in minutes, default 15)
	DefaultPresignExpire 	time.Duration
}

// LocalStorageConfig holds the configuration for a single directory on the local filesystem (file cache, generated files, local config, etc.).
// Create one per directory the service needs. Loaded with GetLocalStorageEnvConfig(name, prefix), e.g. prefix "LOCAL_STORAGE_" reads LOCAL_STORAGE_BASE_DIR, etc.
// In Docker, mount a volume at BaseDir so the files survive container restarts.
type LocalStorageConfig struct {
	// Name identifies the storage in logs and errors (e.g. "main", "cache")
	Name 		string
	// BaseDir is the root directory. Every key is resolved inside it and can never escape it. Created on startup if missing (env: <prefix>BASE_DIR, default "storage")
	BaseDir 	string
	// DirMode is the permission for directories created by the storage (env: <prefix>DIR_MODE as octal, default "0755")
	DirMode 	os.FileMode
	// FileMode is the permission for files written by the storage (env: <prefix>FILE_MODE as octal, default "0644")
	FileMode 	os.FileMode
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
	// MainMongo is the primary MongoDB database (env prefix: MONGO_)
	MainMongo	MongoConfig
	// MainRedis is the primary Redis instance (env prefix: REDIS_)
	MainRedis	RedisConfig
	// MainStomp is the primary STOMP message broker connection (env prefix: STOMP_)
	MainStomp	StompConfig
	// MainS3 is the primary S3-compatible object storage (env prefix: S3_)
	MainS3		S3Config
	// MainLocalStorage is the primary local filesystem storage (env prefix: LOCAL_STORAGE_)
	MainLocalStorage	LocalStorageConfig
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