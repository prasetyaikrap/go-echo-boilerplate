# Go Echo Boilerplate

A production-ready starter template for building backend REST API services using **Go** and **Echo** framework, following Clean Architecture principles.

## Tech Stack

| Concern | Library |
|---|---|
| HTTP Framework | [Echo v4](https://echo.labstack.com/) |
| ORM | [GORM](https://gorm.io/) |
| Database | PostgreSQL (one or more databases) |
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
├── instruction.md                   # Conventions and architecture guide for AI agents
│
├── applications/
│   └── usecases/                    # Business logic layer (use cases)
│       ├── authentications.go       # Auth-related use cases (verify client, token)
│       └── system.go                # System info use case
│
├── commons/
│   ├── models/                      # Shared data models and types
│   │   ├── app.go                   # Base request/payload types
│   │   ├── authentications.go       # JWT claims, auth request/response models
│   │   ├── configs.go               # All config structs (app, Postgres, Redis, STOMP, S3, LLM)
│   │   ├── constant.go              # Global constants and shared errors
│   │   ├── message_broker.go        # BrokerMessage and BrokerMessageHandler (protocol-agnostic)
│   │   └── queries.go               # Query/filter/sort models
│   └── utils/                       # Shared utility functions
│       ├── agents.go                # LLM prompt file loading and templating
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
│   │   └── redis/                   # Generic Redis connection with cache helpers, reusable per instance
│   ├── message_broker/
│   │   ├── handler.go               # MessageBroker interface + MessageBrokerInstance
│   │   └── stomp/                   # STOMP client with auto-reconnect, reusable per connection
│   ├── storages/
│   │   ├── handler.go               # StorageInstance — aggregates storage clients
│   │   └── s3/                      # S3-compatible storage client (AWS S3, R2, MinIO)
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
        │   └── logger/              # Request/response logging middleware
        └── validator/               # Custom Echo request validator
```

## Prerequisites

- Go **1.26** or later
- PostgreSQL database
- Optional: Redis, a STOMP message broker, an S3-compatible bucket, and an LLM provider — only needed when you use those features

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
| GET | `/system` | Service health / info | Client ID only |

## Request Headers

| Header | Description |
|---|---|
| `X-Client-Id` | Required on every request — identifies the calling client |
| `Authorization` | `Bearer <token>` — required for protected routes |
| `X-Service-Token` | Service-to-service secret token |
| `X-Renew-Token` | Set to `"true"` to use a refresh token instead of access token |

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

`Close()` waits for in-flight message handlers before disconnecting, so call it on shutdown. `e.Logger.Fatal(e.Start(...))` exits without running deferred calls, so use a graceful shutdown instead:

```go
go func() {
    if err := e.Start(fmt.Sprintf(":%s", configs.Envs.Application.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
        e.Logger.Fatal(err)
    }
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = e.Shutdown(ctx)    // stop accepting HTTP requests
_ = mainStomp.Close()  // finish in-flight messages, then disconnect
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
storage := storages.NewStorageInstance(configs)
s3 := storage.MainS3Client

s3.UploadObject(ctx, "avatars/1.png", file, "image/png")
s3.GetPublicObjectURL("avatars/1.png")
s3.GetPresignedURL(ctx, "invoices/1.pdf", 0)                       // temporary download link
s3.GetPresignedUploadURL(ctx, "avatars/2.png", "image/png", 0)     // let the client upload directly
```

Pass `0` as the expiry to use `S3_PRESIGN_EXPIRE`.

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
