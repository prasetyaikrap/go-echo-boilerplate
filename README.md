# Go Echo Boilerplate

A production-ready starter template for building backend REST API services using **Go** and **Echo** framework, following Clean Architecture principles.

## Tech Stack

| Concern | Library |
|---|---|
| HTTP Framework | [Echo v4](https://echo.labstack.com/) |
| ORM | [GORM](https://gorm.io/) |
| Database | PostgreSQL |
| JWT | [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) |
| Validation | [go-playground/validator v10](https://github.com/go-playground/validator) |
| API Docs | [Swaggo](https://github.com/swaggo/swag) |
| Config | [godotenv](https://github.com/joho/godotenv) |

## Project Structure

```
.
├── main.go                          # Entry point — wires all layers together
├── go.mod
├── .env_example                     # Environment variable template
│
├── applications/
│   └── usecases/                    # Business logic layer (use cases)
│       ├── authentications.go       # Auth-related use cases (verify client, token)
│       └── system.go                # System info use case
│
├── commons/
│   ├── models/                      # Shared data models and types
│   │   ├── authentications.go       # JWT claims, auth request/response models
│   │   ├── configs.go               # Application and DB config structs
│   │   ├── constant.go              # Global constants (headers, filter operators)
│   │   └── queries.go               # Query/filter/sort models
│   └── utils/                       # Shared utility functions
│       ├── exceptions.go            # Custom error types (ClientError, NotFoundError, etc.)
│       ├── general.go               # General helpers
│       ├── http.go                  # HTTP response helpers (SuccessResponse, ErrorResponse)
│       └── queries.go               # GORM query builder helpers
│
├── docs/                            # Auto-generated Swagger docs (do not edit manually)
│
├── infrastructures/
│   ├── configurations/              # App config loading (env vars, logger)
│   │   ├── env.go                   # Reads and parses environment variables
│   │   ├── handler.go               # Configs struct (combines Envs + Logger)
│   │   └── logger.go                # Structured logger (slog)
│   ├── databases/
│   │   ├── handler.go               # DatabaseInstance — aggregates DB connections
│   │   └── postgres/maindb/         # Main PostgreSQL connection setup via GORM
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

- Go **1.24.3** or later
- PostgreSQL database

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

Edit `.env` with your values:

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
DB_AUTO_MIGRATE=true               # Set to true to run GORM AutoMigrate on startup
```

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

## Adding a New Domain

1. Add models to `commons/models/`
2. Add repository to `infrastructures/repositories/`
3. Add use case to `applications/usecases/`
4. Add handler + routes under `interfaces/http/api/<domain>/`
5. Register routes in `main.go`
