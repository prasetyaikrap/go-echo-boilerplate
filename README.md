# Go Echo Boilerplate

A production-ready starter template for building backend REST API services using **Go** and **Echo** framework, following Clean Architecture principles.

## Tech Stack

| Concern | Library |
|---|---|
| HTTP Framework | [Echo v4](https://echo.labstack.com/) |
| ORM | [GORM](https://gorm.io/) |
| Database | PostgreSQL (one or more databases) |
| Document Database | MongoDB via the [official Go driver v2](https://github.com/mongodb/mongo-go-driver) |
| Cache | Redis via [go-redis v9](https://github.com/redis/go-redis) |
| Message Broker | STOMP (ActiveMQ, RabbitMQ, Artemis) via [go-stomp v3](https://github.com/go-stomp/stomp) |
| Object Storage | Any S3-compatible storage (AWS S3, Cloudflare R2, MinIO) via [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) |
| LLM | [LangChainGo](https://github.com/tmc/langchaingo) — Ollama, Gemini, or OpenAI |
| JWT | [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) |
| Validation | [go-playground/validator v10](https://github.com/go-playground/validator) |
| Logging | `log/slog` (JSON) |
| API Docs | [Swaggo](https://github.com/swaggo/swag) |
| Config | [godotenv](https://github.com/joho/godotenv) |

## Project Structure

```
.
├── main.go                          # Entry point — wires all layers together
├── go.mod
├── .env_example                     # Environment variable template
├── Dockerfile                       # Multi-stage build → distroless, non-root image
├── instruction.md                   # Conventions and architecture guide for AI agents
│
├── applications/
│   └── usecases/                    # Business logic layer (use cases)
│       ├── authentications.go       # Auth-related use cases (verify client, token)
│       └── system.go                # System info use case
│
├── commons/
│   ├── models/                      # Shared data models and types
│   │   ├── authentications.go       # JWT claims, auth request/response models
│   │   ├── configs.go               # All config structs (app, Postgres, Redis, STOMP, S3, LLM)
│   │   ├── constant.go              # Global constants and shared errors
│   │   ├── message_broker.go        # BrokerMessage and BrokerMessageHandler (protocol-agnostic)
│   │   └── queries.go               # Query/filter/sort models
│   └── utils/                       # Shared utility functions
│       ├── agents.go                # LLM prompt file loading and templating
│       ├── context.go               # context.Context helpers (request ID, token claims)
│       ├── exceptions.go            # Custom error types (ClientError, NotFoundError, etc.)
│       ├── general.go               # General helpers
│       ├── http.go                  # HTTP response helpers (SuccessResponse, ErrorResponse)
│       ├── message_broker.go        # Message payload encoding
│       └── queries.go               # GORM query builder helpers
│
├── docs/                            # Auto-generated Swagger docs (do not edit manually)
├── files/
│   └── agents/prompts/              # LLM prompt templates
│
├── infrastructures/
│   ├── configurations/              # App config loading (env vars, logger)
│   │   ├── env.go                   # Reads and parses environment variables
│   │   ├── handler.go               # Configs struct (combines Envs + Logger)
│   │   └── logger.go                # Structured logger (slog)
│   ├── databases/
│   │   ├── handler.go               # DatabaseInstance — aggregates DB connections
│   │   ├── postgres/                # Generic PostgreSQL connection (GORM), reusable per database
│   │   ├── mongodb/                 # Generic MongoDB connection with index setup, reusable per database
│   │   └── redis/                   # Generic Redis connection with cache helpers, reusable per instance
│   ├── message_broker/
│   │   ├── handler.go               # MessageBroker interface + MessageBrokerInstance
│   │   └── stomp/                   # STOMP client with auto-reconnect, reusable per connection
│   ├── storages/
│   │   ├── handler.go               # StorageInstance — aggregates storage clients
│   │   ├── s3/                      # S3-compatible storage client (AWS S3, R2, MinIO), reusable per bucket
│   │   └── localstorage/            # Local filesystem storage, reusable per directory
│   ├── langchain/
│   │   ├── handler.go               # LLM initialization, selects provider from env
│   │   └── providers/               # ollama, gemini, openai
│   ├── repositories/                # Data access layer
│   │   ├── authentications.go       # Client ID / token verification against config
│   │   └── system.go                # System info queries
│   └── security/
│       ├── password_hash.go         # bcrypt password hashing
│       └── token_manager.go         # JWT access/refresh token sign & verify
│
└── interfaces/
    └── http/
        ├── api/
        │   └── system/              # Example domain: system info endpoint
        │       ├── handler.go       # HTTP handler methods
        │       └── route.go         # Route registration (RegisterRoutes)
        ├── middlewares/
        │   ├── authentications/     # Client ID and JWT auth middlewares
        │   ├── logger/              # Request/response logging middleware
        │   └── request_context/     # Request ID and per-request timeout
        └── validator/               # Custom Echo request validator
```

## Prerequisites

- Go **1.26** or later
- PostgreSQL database
- Optional: MongoDB, Redis, a STOMP message broker, an S3-compatible bucket, a writable directory for local storage, and an LLM provider — only needed when you use those features

## Getting Started

### 1. Clone the repository

```bash
git clone <repo-url>
cd go-echo-boilerplate
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Configure environment variables

```bash
cp .env_example .env
```

Edit `.env` with your values. The minimum to start the server:

```env
# Application
PORT=3005
CLIENT_ID=GO_BOILERPLATE           # This service's own client ID
ALLOWED_CLIENT_IDS=*               # Comma-separated; use * to allow all
ALLOWED_ORIGINS=http://localhost:3000

# Security
SERVICE_TOKEN=your-secret-token
JWT_ACCESS_SECRET=your-access-secret
JWT_REFRESH_SECRET=your-refresh-secret
ACCESS_TOKEN_EXPIRATION=15         # Minutes
REFRESH_TOKEN_EXPIRATION=43200     # Minutes (30 days)
REQUEST_TIMEOUT=30                 # Seconds; deadline for each request (0 disables)
SHUTDOWN_TIMEOUT=10                # Seconds to wait for in-flight requests on shutdown
SERVER_READ_HEADER_TIMEOUT=10      # Seconds to receive request headers (slowloris protection)
REQUEST_BODY_LIMIT=10M             # Max request body size (413 when exceeded); 0 disables it
# Optional connection timeouts, disabled unless set: SERVER_READ_TIMEOUT, SERVER_WRITE_TIMEOUT, SERVER_IDLE_TIMEOUT

# PostgreSQL
DB_DSN="host=localhost user=postgres password=postgres dbname=mydb port=5432 sslmode=disable TimeZone=UTC"
DB_MAX_CONN_IDLE=10
DB_MAX_CONN_IDLE_LIFETIME=30       # Minutes
DB_MAX_CONN=25
DB_MAX_CONN_LIFETIME=60            # Minutes
DB_LOG_LEVEL=info                  # silent | error | warn | info
DB_AUTO_MIGRATE=true               # Set to true to run GORM AutoMigrate on startup
```

Optional services:

```env
# MongoDB
MONGO_URI=mongodb://app:app@localhost:27017/?authSource=admin
MONGO_DATABASE=mydb
MONGO_AUTO_MIGRATE=true           # Create registered indexes on startup

# Redis
REDIS_URL=redis://localhost:6379/0 # rediss:// for TLS
REDIS_KEY_PREFIX=myservice:
REDIS_DEFAULT_TTL=60               # Minutes

# STOMP message broker (example: RabbitMQ with the STOMP plugin)
STOMP_ADDRESS=localhost:61613
STOMP_LOGIN=guest
STOMP_PASSCODE=guest
STOMP_HOST=/                       # Virtual host

# S3-compatible storage (example: Cloudflare R2)
S3_ENDPOINT=https://<account_id>.r2.cloudflarestorage.com
S3_REGION=auto
S3_ACCESS_KEY=...
S3_SECRET_ACCESS_KEY=...
S3_BUCKET_NAME=my-bucket
S3_PUBLIC_HOST=https://cdn.example.com

# LLM (leave LLM_PROVIDER empty to disable)
LLM_PROVIDER=ollama                # ollama | gemini | openai
LLM_MODEL=llama3
LLM_OLLAMA_SERVER_URL=http://localhost:11434
```

`.env_example` lists every supported variable, and each field in `commons/models/configs.go` documents its env var, unit and default.

### 4. Run the server

```bash
go run main.go
```

The server starts on the port defined by `PORT` (default: `3005`).

### 5. Run the tests

```bash
go test ./...
```

Tests use only the standard library (`testing`, `httptest`) and need no database or broker.

### 6. Run with Docker

```bash
docker build -t go-echo-boilerplate .
docker run --rm -p 3005:3005 --env-file .env go-echo-boilerplate
```

The image is a static binary on `gcr.io/distroless/static` (no shell, ~47 MB) running as a non-root user (UID 65532). `.env` is excluded from the image by `.dockerignore` — pass configuration as environment variables at runtime. For multi-arch images use `docker buildx build --platform linux/amd64,linux/arm64 ...`.

## API Documentation

Swagger UI is available at:

```
http://localhost:3005/swagger/index.html
```

To regenerate Swagger docs after adding/modifying annotations:

```bash
swag init
```

## Key Endpoints

| Method | Path | Description | Auth Required |
|---|---|---|---|
| GET | `/healthz` | Liveness probe — process is up, checks no dependencies | None |
| GET | `/readyz` | Readiness probe — checks the database (and any registered dependency); `503` when one is down | None |
| GET | `/system` | Service info | Client ID only |

## Context and Request Tracing

`echo.Context` (`c`) stays in handlers and middlewares. Everything below them receives Go's `context.Context` as the first parameter:

```go
// Handler — the only layer that sees echo.Context
func (h *OrdersHandler) GetOrders(c echo.Context) error {
    ctx := c.Request().Context()
    orders, err := h.ordersUsecase.GetOrders(ctx, c.QueryParam("status"))
    ...
}

// Repository — gets ctx, passes it to every I/O call
func (r *OrdersRepositories) GetOrders(ctx context.Context, status string) ([]models.Order, error) {
    claims, _ := utils.AccessTokenClaimsFromContext(ctx) // set by the auth middleware

    var orders []models.Order
    err := r.dbInstances.MainDB.WithContext(ctx).
        Where("user_id = ? AND status = ?", claims.Subject, status).
        Find(&orders).Error

    r.configs.Logger.InfoContext(ctx, "Orders loaded", "count", len(orders)) // includes request_id
    return orders, err
}
```

What the request `ctx` gives you:

- **Cancellation** — database, cache and broker calls stop when the client disconnects or `REQUEST_TIMEOUT` passes (the request returns 503).
- **Request ID** — taken from the incoming `X-Request-Id` header or generated, returned in the response, added to every log line, and forwarded to STOMP consumers. One ID follows a request across services.
- **User claims** — `utils.AccessTokenClaimsFromContext(ctx)` on authenticated routes.

Connections that don't send their request headers within `SERVER_READ_HEADER_TIMEOUT` (default 10s) are closed. `SERVER_READ_TIMEOUT`, `SERVER_WRITE_TIMEOUT` and `SERVER_IDLE_TIMEOUT` are available but disabled by default — they are mostly needed when the service is exposed directly rather than behind a reverse proxy, and `SERVER_WRITE_TIMEOUT` cuts off streaming (SSE) responses.

On SIGINT/SIGTERM the server stops accepting requests and waits up to `SHUTDOWN_TIMEOUT` for in-flight ones before closing the database. See `instruction.md` for the full rules.

## Kubernetes

Probes, shutdown and image are set up for Kubernetes:

```yaml
spec:
  terminationGracePeriodSeconds: 30        # must be longer than SHUTDOWN_TIMEOUT (10s)
  containers:
    - name: api
      image: go-echo-boilerplate:latest
      ports:
        - containerPort: 3005
      envFrom:
        - secretRef: { name: api-env }       # PORT, CLIENT_ID, DB_DSN, ...
      securityContext:
        runAsNonRoot: true
        readOnlyRootFilesystem: true         # mount a volume at /app/storage if local storage is used
        allowPrivilegeEscalation: false
      livenessProbe:
        httpGet: { path: /healthz, port: 3005 }
        periodSeconds: 10
      readinessProbe:
        httpGet: { path: /readyz, port: 3005 }
        periodSeconds: 5
        timeoutSeconds: 2                    # the readiness checks are cancelled when this passes
      lifecycle:
        preStop:
          sleep: { seconds: 5 }              # let the Service stop routing before SIGTERM (Kubernetes 1.30+)
```

- **Liveness** never checks dependencies — a database outage should take pods out of rotation (readiness), not restart them.
- **Readiness** checks are registered in `main.go` (`health.NewHealthHandler`); add Redis, MongoDB, etc. when you wire them in.
- Probe paths skip `X-Client-Id` verification and are not written to the request log.
- Exit code is `0` after a graceful shutdown and `1` when the server fails, so Kubernetes restarts failed pods.

## Request Headers

| Header | Description |
|---|---|
| `X-Client-Id` | Required on every request — identifies the calling client |
| `Authorization` | `Bearer <token>` — required for protected routes |
| `X-Service-Token` | Service-to-service secret token |
| `X-Renew-Token` | Set to `"true"` to use a refresh token instead of access token |
| `X-Request-Id` | Optional — reused as the request ID for tracing; generated when absent and always returned in the response |

## Error Response Format

```json
{
  "code": 400,
  "message": "error description",
  "error": {
    "type": "CLIENT_ERROR",
    "code": 400,
    "data": null
  }
}
```

Error types: `CLIENT_ERROR`, `AUTHENTICATION_ERROR`, `AUTHORIZATION_ERROR`, `NOTFOUND_ERROR`, `INVARIANT_ERROR`, `INTERNALSERVER_ERROR`.

## Infrastructure Usage

### Multiple Postgres databases

Each database is one config plus one constructor call — no new package needed. For an extra `analytics` database:

```go
// commons/models/configs.go → ENVConfig
AnalyticsDB PostgresConfig

// infrastructures/configurations/env.go → reads ANALYTICS_DB_DSN, ANALYTICS_DB_MAX_CONN, ...
AnalyticsDB: GetPostgresEnvConfig("analytics", "ANALYTICS_DB_"),

// infrastructures/databases/handler.go → models are auto-migrated when ANALYTICS_DB_AUTO_MIGRATE=true
analyticsDB := postgres.NewPostgresInstance(configs, configs.Envs.AnalyticsDB, &models.Event{})
```

### MongoDB

MongoDB is not wired by default. Initialize one or more databases in `main.go`, registering the indexes each collection needs — they are created on startup when `MONGO_AUTO_MIGRATE=true`, like `DB_AUTO_MIGRATE` for Postgres:

```go
import (
    "go-serviceboilerplate/infrastructures/databases/mongodb"
    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

mainMongo := mongodb.NewMongoInstance(configs, configs.Envs.MainMongo,
    mongodb.CollectionIndexes{
        Collection: "users",
        Indexes: []mongo.IndexModel{
            {Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
        },
    },
)
defer mainMongo.Close(context.Background())
```

Query with the official driver through `Collection`. Map documents with `bson` tags:

```go
type User struct {
    ID    bson.ObjectID `bson:"_id,omitempty"`
    Email string        `bson:"email"`
    Name  string        `bson:"name"`
}

users := mainMongo.Collection("users")

_, err := users.InsertOne(ctx, User{Email: "ana@example.com", Name: "Ana"})
if mongo.IsDuplicateKeyError(err) {
    // email already exists (unique index)
}

var user User
err = users.FindOne(ctx, bson.M{"email": "ana@example.com"}).Decode(&user)
if errors.Is(err, mongo.ErrNoDocuments) {
    // not found
}
```

Add another database with `GetMongoEnvConfig("audit", "AUDIT_MONGO_")`, the same way as Postgres and Redis. Each instance has its own connection pool, so two databases on the same server can also share one instance via `mainMongo.Client.Database("other")`.

### Redis cache

Redis is not wired by default, so the server starts without it. Initialize one or more instances in `main.go`:

```go
mainRedis := redis.NewRedisInstance(configs, configs.Envs.MainRedis)
defer mainRedis.Close()
```

```go
err := mainRedis.SaveCache(ctx, "user:1", user, 10*time.Minute) // ttl 0 → REDIS_DEFAULT_TTL

var cached models.User
err = mainRedis.GetCache(ctx, "user:1", &cached)
if errors.Is(err, models.ErrCacheMiss) {
    // not cached — load from the database
}

err = mainRedis.RemoveCache(ctx, "user:1", "user:2")
```

Add another instance with `GetRedisEnvConfig("session", "SESSION_REDIS_")`, the same way as Postgres.

### Message broker (STOMP)

Works with any STOMP broker — ActiveMQ, RabbitMQ (STOMP plugin), Artemis, Amazon MQ. The broker is not wired by default, so the server starts without one.

#### 1. Run a broker locally (optional)

RabbitMQ with the STOMP plugin enabled:

```bash
docker run -d --name rabbitmq \
  -e RABBITMQ_DEFAULT_USER=app -e RABBITMQ_DEFAULT_PASS=app \
  -p 61613:61613 -p 15672:15672 \
  rabbitmq:3.13-management \
  sh -c "rabbitmq-plugins enable --offline rabbitmq_stomp && exec docker-entrypoint.sh rabbitmq-server"
```

Management UI: http://localhost:15672 (app / app).

#### 2. Configure

```env
STOMP_ADDRESS=localhost:61613
STOMP_LOGIN=app
STOMP_PASSCODE=app
STOMP_HOST=/                       # RabbitMQ virtual host — required for RabbitMQ
STOMP_HEARTBEAT=10                 # Seconds
STOMP_RECONNECT_INTERVAL=5         # Seconds; 0 disables auto-reconnect
```

#### 3. Connect in `main.go`

```go
import "go-serviceboilerplate/infrastructures/message_broker/stomp"

mainStomp := stomp.NewStompInstance(configs, configs.Envs.MainStomp)
```

`Close()` waits for in-flight message handlers before disconnecting. Add it to the shutdown sequence at the end of `main.go`, after `e.Shutdown`:

```go
if err := e.Shutdown(shutdownCtx); err != nil { ... }
if err := mainStomp.Close(); err != nil {
    configs.Logger.Error("STOMP close failed", "error", err)
}
```

#### 4. Publish from a repository

Depend on the `message_broker.MessageBroker` interface, not the STOMP type, so the protocol can change without touching this code.

```go
// infrastructures/repositories/orders.go
package repositories

import (
    "context"
    "go-serviceboilerplate/commons/models"
    "go-serviceboilerplate/infrastructures/message_broker"
)

type OrdersRepositories struct {
    broker message_broker.MessageBroker
}

func NewOrdersRepositories(broker message_broker.MessageBroker) *OrdersRepositories {
    return &OrdersRepositories{broker}
}

func (r *OrdersRepositories) PublishOrderCreated(ctx context.Context, order models.Order) error {
    // Structs are sent as JSON, string as text/plain, []byte as-is. Headers are optional
    return r.broker.Publish(ctx, "/queue/orders.created", order, map[string]string{
        "x-order-id": order.ID,
    })
}
```

#### 5. Consume in the interface layer

Consumers are an entry point like HTTP handlers, so they live under `interfaces/`, with the same `handler.go` + registration file split as `interfaces/http/api/<domain>/`.

```go
// interfaces/message_broker/orders/handler.go
package orders

import (
    "context"
    "go-serviceboilerplate/applications/usecases"
    "go-serviceboilerplate/commons/models"
)

type OrdersConsumer struct {
    ordersUsecase *usecases.OrdersUsecase
}

func NewOrdersConsumer(ordersUsecase *usecases.OrdersUsecase) *OrdersConsumer {
    return &OrdersConsumer{ordersUsecase}
}

// HandleOrderCreated returns nil to acknowledge the message, or an error to reject it so the broker redelivers it
func (h *OrdersConsumer) HandleOrderCreated(ctx context.Context, msg models.BrokerMessage) error {
    var order models.Order
    if err := msg.Bind(&order); err != nil {
        return nil // malformed payload will never succeed: ack it instead of redelivering forever
    }

    return h.ordersUsecase.ProcessOrder(ctx, order)
}
```

```go
// interfaces/message_broker/orders/subscription.go
package orders

import "go-serviceboilerplate/infrastructures/message_broker"

func (h *OrdersConsumer) RegisterSubscriptions(broker message_broker.MessageBroker) error {
    // The last argument is optional SUBSCRIBE headers, e.g. a prefetch limit
    return broker.Subscribe("/queue/orders.created", h.HandleOrderCreated, map[string]string{
        "prefetch-count": "10",
    })
}
```

```go
// main.go
ordersRepositories := repositories.NewOrdersRepositories(mainStomp)
ordersUsecase := usecases.NewOrdersUsecase(ordersRepositories)

ordersConsumer := orders.NewOrdersConsumer(ordersUsecase)
if err := ordersConsumer.RegisterSubscriptions(mainStomp); err != nil {
    configs.Logger.Fatal("Failed to register order subscriptions", "error", err)
}
```

#### Behaviour

- **Reconnect** — if the connection drops, it reconnects every `STOMP_RECONNECT_INTERVAL` seconds and resubscribes every active subscription. `Publish` reconnects and retries once.
- **Ack / nack** — each message is acknowledged individually after the handler returns `nil`. An error or a panic sends a NACK and the broker redelivers it (or dead-letters it, depending on broker setup).
- **At-least-once** — a message can be delivered more than once (e.g. after a reconnect), so make handlers idempotent.
- **One message at a time per subscription** — set a prefetch limit (`prefetch-count` for RabbitMQ, `activemq.prefetchSize` for ActiveMQ) so a slow handler doesn't stall the connection.
- **Destinations** — `/queue/<name>` for point-to-point, `/topic/<name>` for publish/subscribe. Brokers may support more (e.g. RabbitMQ `/exchange/<name>/<routing-key>`).
- **More connections** — add a config with `GetStompEnvConfig("events", "EVENTS_STOMP_")` and call `stomp.NewStompInstance` again, the same way as Postgres and Redis.

### S3-compatible storage

The same client works for AWS S3, Cloudflare R2 and MinIO — only the env vars change:

| Provider | `S3_ENDPOINT` | `S3_REGION` | `S3_USE_PATH_STYLE` |
|---|---|---|---|
| AWS S3 | *(empty)* | e.g. `ap-southeast-1` | `false` |
| Cloudflare R2 | `https://<account_id>.r2.cloudflarestorage.com` | `auto` | `false` |
| MinIO | `http://localhost:9000` | `us-east-1` | `true` |

```go
mainS3 := s3.NewS3Instance(configs, configs.Envs.MainS3)

mainS3.UploadObject(ctx, "avatars/1.png", file, "image/png")
mainS3.GetPublicObjectURL("avatars/1.png")
mainS3.GetPresignedURL(ctx, "invoices/1.pdf", 0)                       // temporary download link
mainS3.GetPresignedUploadURL(ctx, "avatars/2.png", "image/png", 0)     // let the client upload directly
```

Pass `0` as the expiry to use `S3_PRESIGN_EXPIRE`. Add another bucket with `GetS3EnvConfig("media", "MEDIA_S3_")`, the same way as Postgres and Redis.

### Local storage

Stores files in a directory on the server — useful for file caches, generated files, or local config. Every key is confined to `LOCAL_STORAGE_BASE_DIR`: keys like `../etc/passwd` and symlinks pointing outside it are rejected.

```go
localStorage := localstorage.NewLocalStorageInstance(configs, configs.Envs.MainLocalStorage)
defer localStorage.Close()

// Write — parent directories are created; the write is atomic (no half-written files)
err := localStorage.WriteFile("config/settings.json", data)
err = localStorage.SaveFile("exports/report.csv", reader)

// Read
data, err := localStorage.ReadFile("config/settings.json")
if errors.Is(err, fs.ErrNotExist) {
    // not found
}
file, err := localStorage.GetFile("exports/report.csv") // stream; close it when done
defer file.Close()

// Inspect and clean up
exists, err := localStorage.FileExists("config/settings.json")
keys, err := localStorage.ListFiles("cache")        // recursive; "" lists everything
err = localStorage.DeleteFile("exports/report.csv") // missing file is not an error
err = localStorage.DeleteDir("cache")               // remove a folder and its contents
```

**Docker** — files in the container are lost when it is recreated. Mount a volume at the base directory to keep them:

```yaml
# docker-compose.yml
services:
  api:
    environment:
      LOCAL_STORAGE_BASE_DIR: /app/storage
    volumes:
      - app-storage:/app/storage
volumes:
  app-storage:
```

If the container runs as a non-root user, make sure that user can write to the mounted directory. Add another directory with `GetLocalStorageEnvConfig("cache", "CACHE_STORAGE_")`.

### LLM

`langchain.NewLangChainInstance(configs)` returns the model for the provider set in `LLM_PROVIDER`, or `nil` when it is empty. Keep prompts as files in `files/agents/prompts/` and load them with `utils.GetAgentPromptFileContent`.

## Adding a New Domain

1. Add models to `commons/models/`
2. Add repository to `infrastructures/repositories/` (query via `dbInstances.MainDB`)
3. If the model needs a table, register it for auto-migration in `infrastructures/databases/handler.go`
4. Add use case to `applications/usecases/`
5. Add handler + routes under `interfaces/http/api/<domain>/`
6. Register routes in `main.go`

See `instruction.md` for the full conventions.

## Author

**Prasetya Ikra Priyadi** — [mail@prasetyapriyadi.my.id](mailto:mail@prasetyapriyadi.my.id)
