# AI Instruction — Go Echo Boilerplate

This document provides context for AI agents to understand the structure and conventions of this codebase when generating or modifying code.

---

## Project Identity

- **Module name**: `go-serviceboilerplate` (used in all import paths)
- **Language**: Go 1.24.3+
- **HTTP framework**: Echo v4
- **ORM**: GORM with PostgreSQL
- **Auth**: JWT (golang-jwt/jwt v5) + bcrypt password hashing

---

## Architecture

This project follows **Clean Architecture** with four layers. Dependencies flow inward — outer layers depend on inner, never the reverse.

```
interfaces → applications → infrastructures → commons
```

| Layer | Location | Responsibility |
|---|---|---|
| Interface | `interfaces/http/` | HTTP handlers, routes, middlewares, validator |
| Application | `applications/usecases/` | Business logic, orchestrates repositories and security |
| Infrastructure | `infrastructures/` | DB connections, repositories, config loading, security |
| Commons | `commons/` | Shared models, utilities, constants — no external deps |

---

## Layer Conventions

### `commons/models/`
- Pure data structs and constants shared across all layers.
- No business logic. No external package dependencies (except stdlib and JWT types for claims).
- Key files:
  - `configs.go` — `ENVConfig`, `ApplicationConfig`, `DBConfig`
  - `authentications.go` — `AccessTokenClaims`, `RefreshTokenClaims`, auth request/response types
  - `queries.go` — `Queries`, `FilterQueries`, `QueriesOptions`, `BaseQueriesRequest`
  - `constant.go` — custom HTTP headers (`X-Client-Id`, etc.) and filter operator constants

### `commons/utils/`
- Stateless helper functions used across layers.
- Key utilities:
  - `exceptions.go` — Custom error types implementing `error` interface. Always return these from use cases and repositories. Types: `ClientError (400)`, `AuthenticationError (401)`, `AuthorizationError (403)`, `NotFoundError (404)`, `InvariantError (400)`.
  - `http.go` — `SuccessResponse(c, config)` and `ErrorResponse(c, err)`. All handlers must use these. `HttpErrorHandler` is registered as Echo's global error handler.
  - `queries.go` — `GenerateQueries(db, queries, defaultLimit)` for building GORM WHERE clauses from `models.Queries`. `GenerateFilterQueries(request, options)` for parsing query string filters/sort from a bound request struct.

### `infrastructures/configurations/`
- `Configs` struct holds `*models.ENVConfig` and `*SlogLogger`.
- Always passed as `*configurations.Configs` to repositories and security.
- Never instantiate configs inline — always use `configurations.NewConfigurations()`.

### `infrastructures/databases/`
- `DatabaseInstance` wraps one or more `*gorm.DB` connections.
- Currently one connection: `postgres/maindb` (main PostgreSQL DB).
- To add a new DB connection: create a new sub-package under `databases/`, initialize it in `databases/handler.go`, and add the field to `DatabaseInstance`.

### `infrastructures/repositories/`
- Repositories receive `*databases.DatabaseInstance` and `*configurations.Configs`.
- Repositories do NOT contain business logic — only data access.
- Return custom errors from `utils/exceptions.go` on failure.

### `infrastructures/security/`
- `PasswordHashSecurity` — bcrypt hash and verify.
- `TokenManagerSecurity` — sign and verify JWT access/refresh tokens using secrets from config.

### `applications/usecases/`
- One file per domain (e.g., `authentications.go`, `system.go`).
- Struct holds repository and security dependencies injected via constructor.
- All use case methods return `error` (using `utils/exceptions.go` types) or `(value, error)`.

### `interfaces/http/api/<domain>/`
- Each domain has two files:
  - `handler.go` — struct with usecase dependency + handler methods
  - `route.go` — `RegisterRoutes(g *echo.Group)` method
- Handlers always call `utils.SuccessResponse` or `utils.ErrorResponse` — never `c.JSON` directly.
- Bind request structs using `c.Bind(&req)`, then validate with `c.Validate(&req)`.
- Routes are registered in `main.go` as: `e.Group("/domain")` → `handler.RegisterRoutes(group)`.

### `interfaces/http/middlewares/`
- **`authentications`**: Provides `VerifyClient` (checks `X-Client-Id` header — applied globally), `VerifyAuthentication` (checks Bearer JWT), and `VerifyAuthenticationWithServiceToken` (checks both `X-Service-Token` and Bearer JWT).
- Apply `VerifyAuthentication` or `VerifyAuthenticationWithServiceToken` to route groups that require auth, not globally.
- Authenticated user claims are stored in Echo context with key `models.ContextUserClaim`. Retrieve via `c.Get(models.ContextUserClaim).(*models.AccessTokenClaims)`.

---

## Adding a New Domain (Step-by-Step)

1. **Model** — add request/response structs and DB model to `commons/models/<domain>.go`
2. **Repository** — create `infrastructures/repositories/<domain>.go`, implement DB queries using GORM via `dbInstances.db`
3. **Use case** — create `applications/usecases/<domain>.go`, inject repository, implement business logic
4. **Handler** — create `interfaces/http/api/<domain>/handler.go` + `route.go`
5. **Wire** — in `main.go`, instantiate repository → usecase → handler, then call `handler.RegisterRoutes(group)`

---

## Response Conventions

**Success:**
```go
utils.SuccessResponse(c, utils.SuccessResponseConfig{
    Code:    http.StatusOK,
    Message: "...",
    Data:    data,
})
```

**Error (from handler):**
```go
return utils.ErrorResponse(c, err) // err must be *utils.Exceptions or error
```

**Returning errors from use cases / repositories:**
```go
return nil, utils.NewNotFoundError(errors.New("user not found"))
return nil, utils.NewClientError(fmt.Errorf("invalid input: %w", err))
```

---

## Query Filtering Pattern

Use `models.BaseQueriesRequest` (or embed it) for list endpoints:

```go
type ListUsersRequest struct {
    models.BaseQueriesRequest        // provides Queries (JSON string), SortBy, Limit, Page
    Status string `query:"status"`
}
```

Parse filters with `utils.GenerateFilterQueries(request, options)`.
Build GORM query with `utils.GenerateQueries(db, queries, defaultLimit)`.

Filter operators are constants in `models/constant.go`: `OpEqual`, `OpIlike`, `OpSliceIn`, `OpBetween`, `OpOr`, etc.

---

## Environment Variables Reference

| Key | Required | Description |
|---|---|---|
| `PORT` | Yes | HTTP server port |
| `CLIENT_ID` | Yes | This service's own client ID |
| `ALLOWED_CLIENT_IDS` | No | Comma-separated allowed client IDs; `*` allows all |
| `ALLOWED_ORIGINS` | No | CORS allowed origins |
| `SERVICE_TOKEN` | No | Secret for service-to-service calls |
| `JWT_ACCESS_SECRET` | No | Secret for signing access tokens |
| `JWT_REFRESH_SECRET` | No | Secret for signing refresh tokens |
| `ACCESS_TOKEN_EXPIRATION` | No | Minutes; default 30 |
| `REFRESH_TOKEN_EXPIRATION` | No | Minutes; default 10080 (7 days) |
| `DB_DSN` | No | PostgreSQL DSN string |
| `DB_AUTO_MIGRATE` | No | `true` to run GORM AutoMigrate on startup |

---

## Swagger Docs

Swagger annotations go on handler methods. Regenerate with:
```bash
swag init
```
Output goes to `docs/` — do not edit those files manually.
