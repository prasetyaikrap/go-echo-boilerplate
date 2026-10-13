package configurations

import (
	"go-serviceboilerplate/commons/models"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func GetENVConfig() *models.ENVConfig {
	if _, err := os.Stat(".env"); err == nil {
		err = godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}

	envConfig := models.ENVConfig{
		Application: models.ApplicationConfig{
			Port: GetEnv("PORT", true, ""),
			ClientID: GetEnv("CLIENT_ID", true, ""),
			AllowedCleintIDs: GetAllowedClientIDs(),
			SecretToken: GetEnv("SERVICE_TOKEN", false, ""),
			JWTAccessSecret: GetEnv("JWT_ACCESS_SECRET", false, ""),
			JWTRefreshSecret: GetEnv("JWT_REFRESH_SECRET", false, ""),
			AccessTokenExpiration: time.Duration(GetEnvInt("ACCESS_TOKEN_EXPIRATION", false, 30)) * time.Minute,
			RefreshTokenExpiration: time.Duration(GetEnvInt("REFRESH_TOKEN_EXPIRATION", false, 60 * 24 * 7)) * time.Minute,
			RequestTimeout: time.Duration(GetEnvInt("REQUEST_TIMEOUT", false, 30)) * time.Second,
			ShutdownTimeout: time.Duration(GetEnvInt("SHUTDOWN_TIMEOUT", false, 10)) * time.Second,
			ServerReadHeaderTimeout: time.Duration(GetEnvInt("SERVER_READ_HEADER_TIMEOUT", false, 10)) * time.Second,
			ServerReadTimeout: time.Duration(GetEnvInt("SERVER_READ_TIMEOUT", false, 0)) * time.Second,
			ServerWriteTimeout: time.Duration(GetEnvInt("SERVER_WRITE_TIMEOUT", false, 0)) * time.Second,
			ServerIdleTimeout: time.Duration(GetEnvInt("SERVER_IDLE_TIMEOUT", false, 0)) * time.Second,
			RequestBodyLimit: GetEnv("REQUEST_BODY_LIMIT", false, "10M"),
			CORSConfig: middleware.CORSConfig{
				AllowOrigins:     GetAllowedOrigins(),
				AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
				AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization, models.XClientIdHeader, models.XClientVersionHeader, models.XServiceTokenHeader, models.XUserIdHeader, models.XRenewTokenHeader, echo.HeaderXRequestID},
				ExposeHeaders:    []string{echo.HeaderXRequestID},
				AllowCredentials: true,
			  },
		},
		MainDB: GetPostgresEnvConfig("main", "DB_"),
		MainMongo: GetMongoEnvConfig("main", "MONGO_"),
		MainRedis: GetRedisEnvConfig("main", "REDIS_"),
		MainStomp: GetStompEnvConfig("main", "STOMP_"),
		MainS3: GetS3EnvConfig("main", "S3_"),
		MainLocalStorage: GetLocalStorageEnvConfig("main", "LOCAL_STORAGE_"),
		LLM: models.LLMConfig{
			Provider:        models.LLMProvider(GetEnv("LLM_PROVIDER", false, "")),
			Model:           GetEnv("LLM_MODEL", false, "llama3"),
			OllamaServerURL: GetEnv("LLM_OLLAMA_SERVER_URL", false, ""),
			GeminiAPIKey:    GetEnv("LLM_GEMINI_API_KEY", false, ""),
			OpenAIAPIKey:    GetEnv("LLM_OPENAI_API_KEY", false, ""),
			ContextWindowSize: GetEnvInt("LLM_CONTEXT_WINDOW_SIZE", false, 20),
		},
	}

	return &envConfig
}

// GetPostgresEnvConfig loads a Postgres config from env vars sharing the same prefix,
// e.g. prefix "ANALYTICS_DB_" reads ANALYTICS_DB_DSN, ANALYTICS_DB_MAX_CONN, etc.
func GetPostgresEnvConfig(name string, prefix string) models.PostgresConfig {
	return models.PostgresConfig{
		Name: name,
		DSN: 	  GetEnv(prefix + "DSN", false, ""),
		MaxConnIdle: GetEnvInt(prefix + "MAX_CONN_IDLE", false, 10),
		MaxConnIdleLifeTime: time.Duration(GetEnvInt(prefix + "MAX_CONN_IDLE_LIFETIME", false, 30)) * time.Minute,
		MaxConn: GetEnvInt(prefix + "MAX_CONN", false, 25),
		MaxConnLifeTime: time.Duration(GetEnvInt(prefix + "MAX_CONN_LIFETIME", false, 60)) * time.Minute,
		LogLevel: GetEnv(prefix + "LOG_LEVEL", false, "info"),

		AutoMigrate: GetEnv(prefix + "AUTO_MIGRATE", false, "false") == "true",
	}
}

// GetMongoEnvConfig loads a MongoDB config from env vars sharing the same prefix,
// e.g. prefix "AUDIT_MONGO_" reads AUDIT_MONGO_URI, AUDIT_MONGO_DATABASE, etc.
func GetMongoEnvConfig(name string, prefix string) models.MongoConfig {
	return models.MongoConfig{
		Name: name,
		URI: GetEnv(prefix + "URI", false, "mongodb://localhost:27017"),
		Database: GetEnv(prefix + "DATABASE", false, ""),
		MaxPoolSize: uint64(GetEnvInt(prefix + "MAX_POOL_SIZE", false, 0)),
		MinPoolSize: uint64(GetEnvInt(prefix + "MIN_POOL_SIZE", false, 0)),
		MaxConnIdleTime: time.Duration(GetEnvInt(prefix + "MAX_CONN_IDLE_TIME", false, 0)) * time.Minute,
		ConnectTimeout: time.Duration(GetEnvInt(prefix + "CONNECT_TIMEOUT", false, 10)) * time.Second,
		AutoMigrate: GetEnv(prefix + "AUTO_MIGRATE", false, "false") == "true",
	}
}

// GetRedisEnvConfig loads a Redis config from env vars sharing the same prefix,
// e.g. prefix "SESSION_REDIS_" reads SESSION_REDIS_URL, SESSION_REDIS_KEY_PREFIX, etc.
func GetRedisEnvConfig(name string, prefix string) models.RedisConfig {
	return models.RedisConfig{
		Name: name,
		URL: GetEnv(prefix + "URL", false, "redis://localhost:6379/0"),
		KeyPrefix: GetEnv(prefix + "KEY_PREFIX", false, ""),
		DefaultTTL: time.Duration(GetEnvInt(prefix + "DEFAULT_TTL", false, 60)) * time.Minute,
		PoolSize: GetEnvInt(prefix + "POOL_SIZE", false, 0),
		PingTimeout: time.Duration(GetEnvInt(prefix + "PING_TIMEOUT", false, 5)) * time.Second,
	}
}

// GetStompEnvConfig loads a STOMP broker config from env vars sharing the same prefix,
// e.g. prefix "EVENTS_STOMP_" reads EVENTS_STOMP_ADDRESS, EVENTS_STOMP_LOGIN, etc.
func GetStompEnvConfig(name string, prefix string) models.StompConfig {
	return models.StompConfig{
		Name: name,
		Address: GetEnv(prefix + "ADDRESS", false, "localhost:61613"),
		Login: GetEnv(prefix + "LOGIN", false, ""),
		Passcode: GetEnv(prefix + "PASSCODE", false, ""),
		Host: GetEnv(prefix + "HOST", false, ""),
		UseTLS: GetEnv(prefix + "USE_TLS", false, "false") == "true",
		HeartBeat: time.Duration(GetEnvInt(prefix + "HEARTBEAT", false, 30)) * time.Second,
		ConnectTimeout: time.Duration(GetEnvInt(prefix + "CONNECT_TIMEOUT", false, 60)) * time.Second,
		ReconnectInterval: time.Duration(GetEnvInt(prefix + "RECONNECT_INTERVAL", false, 15)) * time.Second,
	}
}

// GetS3EnvConfig loads an S3-compatible storage config from env vars sharing the same prefix,
// e.g. prefix "MEDIA_S3_" reads MEDIA_S3_ENDPOINT, MEDIA_S3_BUCKET_NAME, etc.
func GetS3EnvConfig(name string, prefix string) models.S3Config {
	return models.S3Config{
		Name: name,
		Endpoint: GetEnv(prefix + "ENDPOINT", false, ""),
		Region: GetEnv(prefix + "REGION", false, "auto"),
		AccessKeyID: GetEnv(prefix + "ACCESS_KEY", false, ""),
		SecretAccessKey: GetEnv(prefix + "SECRET_ACCESS_KEY", false, ""),
		BucketName: GetEnv(prefix + "BUCKET_NAME", false, ""),
		PublicHost: GetEnv(prefix + "PUBLIC_HOST", false, ""),
		BaseFolder: GetEnv(prefix + "BASE_FOLDER", false, ""),
		UsePathStyle: GetEnv(prefix + "USE_PATH_STYLE", false, "false") == "true",
		DefaultPresignExpire: time.Duration(GetEnvInt(prefix + "PRESIGN_EXPIRE", false, 15)) * time.Minute,
	}
}

// GetLocalStorageEnvConfig loads a local filesystem storage config from env vars sharing the same prefix,
// e.g. prefix "CACHE_STORAGE_" reads CACHE_STORAGE_BASE_DIR, CACHE_STORAGE_FILE_MODE, etc.
func GetLocalStorageEnvConfig(name string, prefix string) models.LocalStorageConfig {
	return models.LocalStorageConfig{
		Name: name,
		BaseDir: GetEnv(prefix + "BASE_DIR", false, "storage"),
		DirMode: GetEnvFileMode(prefix + "DIR_MODE", 0755),
		FileMode: GetEnvFileMode(prefix + "FILE_MODE", 0644),
	}
}

// GetEnvFileMode reads an octal file permission such as "0644" or "644"
func GetEnvFileMode(key string, defaultValue os.FileMode) os.FileMode {
	value := GetEnv(key, false, "")
	if value == "" {
		return defaultValue
	}

	mode, err := strconv.ParseUint(value, 8, 32)
	if err != nil || mode > 0777 {
		log.Fatalf("Environment variable %v must be an octal file permission such as 0644 (got %q)", key, value)
	}

	return os.FileMode(mode)
}

func GetEnv(key string, required bool, defaultValue string) string {
	value, exists := os.LookupEnv(key); 
	if(value == "") {
		value = defaultValue
	}

	if(!required) {
		return value
	}

	if(!exists || value == "") {
		log.Fatalf("Environment variable %v is required", key)
	}

	return value
}

// GetEnvInt reads a non-negative integer. A missing or empty variable uses defaultValue,
// while an explicit "0" is kept as 0 (used by many settings to mean "disabled" or "no limit")
func GetEnvInt(key string, required bool, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		if required {
			log.Fatalf("Environment variable %v is required", key)
		}

		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("Environment variable %v must be a valid integer (got %q)", key, value)
	}

	if intValue < 0 {
		log.Fatalf("Environment variable %v must not be negative (got %d)", key, intValue)
	}

	return intValue
}

func GetAllowedOrigins() []string {
	allowedOrigins := strings.Split(GetEnv("ALLOWED_ORIGINS", false, "*"), ",")
	var cleaned []string
	for _, s := range allowedOrigins {
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}

	if(len(cleaned) <= 0) {
		cleaned = append(cleaned, "*")
	}

	return cleaned
}

func GetAllowedClientIDs() []string {
	allowedClientIDs := strings.Split(GetEnv("ALLOWED_CLIENT_IDS", false, "*"), ",")
	var cleaned []string
	for _, s := range allowedClientIDs {
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}

	if(len(cleaned) <= 0) {
		cleaned = append(cleaned, "*")
	}

	return cleaned
}