# Implementation: Control Plane Foundation

## Agent: 07 - Foundation (BLOCKING)

> **Must complete before other Phase 5 agents can start.**

---

## 1. Overview

This agent sets up the Control Plane API project structure, database schema, and shared infrastructure code.

---

## 2. Tech Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| Language | Go 1.22+ | Match core library |
| Framework | Chi or Echo | Lightweight, fast |
| Database | PostgreSQL 15+ | Primary data store |
| Cache | Redis 7+ | Sessions, rate limiting |
| Migrations | golang-migrate | SQL migrations |
| Config | Viper | Environment + files |
| Validation | go-playground/validator | Request validation |

---

## 3. Project Structure

```
control-plane/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration loading
│   ├── database/
│   │   ├── postgres.go             # PostgreSQL connection
│   │   ├── redis.go                # Redis connection
│   │   └── migrations/
│   │       ├── 001_initial_schema.up.sql
│   │       ├── 001_initial_schema.down.sql
│   │       └── ...
│   ├── middleware/
│   │   ├── auth.go                 # Auth middleware
│   │   ├── ratelimit.go            # Rate limiting
│   │   ├── logging.go              # Request logging
│   │   └── cors.go                 # CORS handling
│   ├── models/
│   │   ├── user.go
│   │   ├── organization.go
│   │   ├── api_key.go
│   │   ├── key.go
│   │   └── audit_log.go
│   ├── repository/
│   │   ├── user_repo.go
│   │   ├── org_repo.go
│   │   └── ...
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── key_service.go
│   │   └── ...
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── keys_handler.go
│   │   └── ...
│   └── pkg/
│       ├── errors/
│       │   └── errors.go           # API error types
│       ├── response/
│       │   └── response.go         # JSON response helpers
│       └── ulid/
│           └── ulid.go             # ID generation
├── api/
│   └── openapi.yaml                # OpenAPI spec
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── go.mod
├── go.sum
└── Makefile
```

---

## 4. Database Schema

**File:** `internal/database/migrations/001_initial_schema.up.sql`

```sql
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Organizations (Tenants)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_stripe ON organizations(stripe_customer_id);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    name VARCHAR(255),
    avatar_url TEXT,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    oauth_provider VARCHAR(50),
    oauth_provider_id VARCHAR(255),
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oauth ON users(oauth_provider, oauth_provider_id);

-- Organization Members
CREATE TABLE org_members (
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    invited_by UUID REFERENCES users(id),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX idx_org_members_user ON org_members(user_id);

-- Namespaces (environments within org)
CREATE TABLE namespaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, name)
);

-- API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,  -- bbr_xxxx (for display)
    key_hash VARCHAR(255) NOT NULL,   -- Argon2 hash
    scopes TEXT[] NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_org ON api_keys(org_id);
CREATE INDEX idx_api_keys_prefix ON api_keys(key_prefix);

-- Keys (metadata - actual keys in OpenBao)
CREATE TABLE keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    namespace_id UUID NOT NULL REFERENCES namespaces(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    public_key BYTEA NOT NULL,
    address VARCHAR(100) NOT NULL,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'secp256k1',
    bao_key_path VARCHAR(500) NOT NULL,
    exportable BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, namespace_id, name)
);

CREATE INDEX idx_keys_org ON keys(org_id);
CREATE INDEX idx_keys_namespace ON keys(namespace_id);
CREATE INDEX idx_keys_address ON keys(address);
CREATE INDEX idx_keys_active ON keys(org_id) WHERE deleted_at IS NULL;

-- Audit Logs
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event VARCHAR(100) NOT NULL,
    actor_id UUID,
    actor_type VARCHAR(50) NOT NULL,  -- user, api_key, system
    resource_type VARCHAR(50),
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_org_time ON audit_logs(org_id, created_at DESC);
CREATE INDEX idx_audit_logs_event ON audit_logs(org_id, event);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);

-- Usage Metrics
CREATE TABLE usage_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    metric VARCHAR(100) NOT NULL,
    value BIGINT NOT NULL DEFAULT 0,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, metric, period_start)
);

CREATE INDEX idx_usage_metrics_org_period ON usage_metrics(org_id, period_start);

-- Webhooks
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    secret VARCHAR(255) NOT NULL,
    events TEXT[] NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_triggered_at TIMESTAMPTZ,
    failure_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_org ON webhooks(org_id);

-- Sessions (for web auth, API keys don't need this)
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data JSONB DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to tables with updated_at
CREATE TRIGGER update_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_keys_updated_at
    BEFORE UPDATE ON keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_usage_metrics_updated_at
    BEFORE UPDATE ON usage_metrics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_webhooks_updated_at
    BEFORE UPDATE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## 5. Core Components

### 5.1 Configuration

**File:** `internal/config/config.go`

```go
package config

import (
    "time"
    "github.com/spf13/viper"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    OpenBao  OpenBaoConfig
    Auth     AuthConfig
    Stripe   StripeConfig
}

type ServerConfig struct {
    Port         int           `mapstructure:"port"`
    Host         string        `mapstructure:"host"`
    ReadTimeout  time.Duration `mapstructure:"read_timeout"`
    WriteTimeout time.Duration `mapstructure:"write_timeout"`
    Environment  string        `mapstructure:"environment"` // dev, staging, prod
}

type DatabaseConfig struct {
    Host            string        `mapstructure:"host"`
    Port            int           `mapstructure:"port"`
    User            string        `mapstructure:"user"`
    Password        string        `mapstructure:"password"`
    Database        string        `mapstructure:"database"`
    SSLMode         string        `mapstructure:"ssl_mode"`
    MaxOpenConns    int           `mapstructure:"max_open_conns"`
    MaxIdleConns    int           `mapstructure:"max_idle_conns"`
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
}

type OpenBaoConfig struct {
    Address       string `mapstructure:"address"`
    Token         string `mapstructure:"token"`
    Namespace     string `mapstructure:"namespace"`
    Secp256k1Path string `mapstructure:"secp256k1_path"`
}

type AuthConfig struct {
    JWTSecret         string        `mapstructure:"jwt_secret"`
    JWTExpiry         time.Duration `mapstructure:"jwt_expiry"`
    SessionExpiry     time.Duration `mapstructure:"session_expiry"`
    BCryptCost        int           `mapstructure:"bcrypt_cost"`
    OAuthGitHubID     string        `mapstructure:"oauth_github_id"`
    OAuthGitHubSecret string        `mapstructure:"oauth_github_secret"`
    OAuthGoogleID     string        `mapstructure:"oauth_google_id"`
    OAuthGoogleSecret string        `mapstructure:"oauth_google_secret"`
}

type StripeConfig struct {
    SecretKey      string `mapstructure:"secret_key"`
    WebhookSecret  string `mapstructure:"webhook_secret"`
    PriceIDFree    string `mapstructure:"price_id_free"`
    PriceIDPro     string `mapstructure:"price_id_pro"`
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./config")
    viper.AutomaticEnv()

    // Defaults
    viper.SetDefault("server.port", 8080)
    viper.SetDefault("server.host", "0.0.0.0")
    viper.SetDefault("server.read_timeout", "30s")
    viper.SetDefault("server.write_timeout", "30s")
    viper.SetDefault("server.environment", "dev")
    viper.SetDefault("database.port", 5432)
    viper.SetDefault("database.ssl_mode", "disable")
    viper.SetDefault("database.max_open_conns", 25)
    viper.SetDefault("database.max_idle_conns", 5)
    viper.SetDefault("database.conn_max_lifetime", "5m")
    viper.SetDefault("redis.port", 6379)
    viper.SetDefault("redis.db", 0)
    viper.SetDefault("auth.bcrypt_cost", 12)
    viper.SetDefault("auth.jwt_expiry", "24h")
    viper.SetDefault("auth.session_expiry", "168h") // 7 days

    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, err
        }
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

### 5.2 API Error Types

**File:** `internal/pkg/errors/errors.go`

```go
package errors

import (
    "fmt"
    "net/http"
)

type APIError struct {
    Code       string `json:"code"`
    Message    string `json:"message"`
    StatusCode int    `json:"-"`
    Details    any    `json:"details,omitempty"`
}

func (e *APIError) Error() string {
    return e.Message
}

// Standard errors
var (
    ErrUnauthorized = &APIError{
        Code:       "unauthorized",
        Message:    "Authentication required",
        StatusCode: http.StatusUnauthorized,
    }
    ErrForbidden = &APIError{
        Code:       "forbidden",
        Message:    "You don't have permission to perform this action",
        StatusCode: http.StatusForbidden,
    }
    ErrNotFound = &APIError{
        Code:       "not_found",
        Message:    "Resource not found",
        StatusCode: http.StatusNotFound,
    }
    ErrBadRequest = &APIError{
        Code:       "bad_request",
        Message:    "Invalid request",
        StatusCode: http.StatusBadRequest,
    }
    ErrRateLimited = &APIError{
        Code:       "rate_limited",
        Message:    "Too many requests. Please try again later.",
        StatusCode: http.StatusTooManyRequests,
    }
    ErrQuotaExceeded = &APIError{
        Code:       "quota_exceeded",
        Message:    "You've exceeded your plan limits",
        StatusCode: http.StatusPaymentRequired,
    }
    ErrInternal = &APIError{
        Code:       "internal_error",
        Message:    "An internal error occurred",
        StatusCode: http.StatusInternalServerError,
    }
)

// Error constructors
func NewValidationError(field, message string) *APIError {
    return &APIError{
        Code:       "validation_error",
        Message:    fmt.Sprintf("Validation failed: %s", message),
        StatusCode: http.StatusBadRequest,
        Details:    map[string]string{"field": field, "error": message},
    }
}

func NewNotFoundError(resource string) *APIError {
    return &APIError{
        Code:       "not_found",
        Message:    fmt.Sprintf("%s not found", resource),
        StatusCode: http.StatusNotFound,
    }
}

func NewConflictError(message string) *APIError {
    return &APIError{
        Code:       "conflict",
        Message:    message,
        StatusCode: http.StatusConflict,
    }
}
```

### 5.3 Response Helpers

**File:** `internal/pkg/response/response.go`

```go
package response

import (
    "encoding/json"
    "net/http"

    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type Response struct {
    Data  any    `json:"data,omitempty"`
    Error any    `json:"error,omitempty"`
    Meta  *Meta  `json:"meta,omitempty"`
}

type Meta struct {
    Page       int    `json:"page,omitempty"`
    PerPage    int    `json:"per_page,omitempty"`
    Total      int64  `json:"total,omitempty"`
    NextCursor string `json:"next_cursor,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(Response{Data: data})
}

func JSONWithMeta(w http.ResponseWriter, status int, data any, meta *Meta) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(Response{Data: data, Meta: meta})
}

func Error(w http.ResponseWriter, err error) {
    apiErr, ok := err.(*apierrors.APIError)
    if !ok {
        apiErr = apierrors.ErrInternal
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(apiErr.StatusCode)
    json.NewEncoder(w).Encode(Response{Error: apiErr})
}

func Created(w http.ResponseWriter, data any) {
    JSON(w, http.StatusCreated, data)
}

func OK(w http.ResponseWriter, data any) {
    JSON(w, http.StatusOK, data)
}

func NoContent(w http.ResponseWriter) {
    w.WriteHeader(http.StatusNoContent)
}
```

---

## 6. Entry Point

**File:** `cmd/server/main.go`

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/go-chi/chi/v5"
    chimiddleware "github.com/go-chi/chi/v5/middleware"

    "github.com/Bidon15/banhbaoring/control-plane/internal/config"
    "github.com/Bidon15/banhbaoring/control-plane/internal/database"
    "github.com/Bidon15/banhbaoring/control-plane/internal/middleware"
)

func main() {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Connect to PostgreSQL
    db, err := database.NewPostgres(cfg.Database)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Connect to Redis
    redis, err := database.NewRedis(cfg.Redis)
    if err != nil {
        log.Fatalf("Failed to connect to Redis: %v", err)
    }
    defer redis.Close()

    // Setup router
    r := chi.NewRouter()

    // Global middleware
    r.Use(chimiddleware.RequestID)
    r.Use(chimiddleware.RealIP)
    r.Use(chimiddleware.Logger)
    r.Use(chimiddleware.Recoverer)
    r.Use(middleware.CORS())
    r.Use(chimiddleware.Timeout(30 * time.Second))

    // Health check
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })

    // API v1 routes (mounted by other agents)
    r.Route("/v1", func(r chi.Router) {
        // Auth routes (Agent 07A, 07B, 07C)
        // r.Mount("/auth", authHandler.Routes())
        
        // Key routes (Agent 08B)
        // r.Mount("/keys", keysHandler.Routes())
        
        // Billing routes (Agent 10A, 10B)
        // r.Mount("/billing", billingHandler.Routes())
        
        // Audit routes (Agent 09A)
        // r.Mount("/audit", auditHandler.Routes())
        
        // Webhooks routes (Agent 09B)
        // r.Mount("/webhooks", webhooksHandler.Routes())
    })

    // Server
    srv := &http.Server{
        Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
        Handler:      r,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
    }

    // Graceful shutdown
    go func() {
        log.Printf("Starting server on %s", srv.Addr)
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server shutdown error: %v", err)
    }
    log.Println("Server stopped")
}
```

---

## 7. Deliverables

| File | Description |
|------|-------------|
| `control-plane/go.mod` | Module definition |
| `control-plane/cmd/server/main.go` | Entry point |
| `control-plane/internal/config/config.go` | Configuration |
| `control-plane/internal/database/postgres.go` | DB connection |
| `control-plane/internal/database/redis.go` | Redis connection |
| `control-plane/internal/database/migrations/*.sql` | All migrations |
| `control-plane/internal/pkg/errors/errors.go` | API errors |
| `control-plane/internal/pkg/response/response.go` | Response helpers |
| `control-plane/internal/middleware/*.go` | Common middleware |
| `control-plane/docker/docker-compose.yml` | Local dev |
| `control-plane/Makefile` | Build commands |

---

## 8. Success Criteria

- [ ] `go build ./...` succeeds
- [ ] Database migrations run successfully
- [ ] Health endpoint returns 200
- [ ] docker-compose up starts all services
- [ ] PostgreSQL and Redis connections established

---

## 9. Agent Prompt

```
You are Agent 07 - Control Plane Foundation. Your task is to set up the Control Plane API project structure.

Read the spec: doc/implementation/IMPL_07_CONTROL_PLANE_FOUNDATION.md

Deliverables:
1. Create control-plane/ directory with Go module
2. Database schema (all tables from PRD)
3. Configuration system with Viper
4. Database/Redis connection code
5. API error types and response helpers
6. Server entry point with Chi router
7. Docker Compose for local dev

Tech stack: Go 1.22+, Chi router, PostgreSQL, Redis, golang-migrate

Test: go build ./... && docker-compose up -d && curl localhost:8080/health
```

---

## 10. Dependencies

This agent blocks ALL other Phase 5 agents.

