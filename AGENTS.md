# AGENTS.md - AI Agent Development Guide

## Project Overview

**Project:** gostarterkit - Go Project Starter Kit

**Technology Stack:**
- Go 1.22+ with stdlib-first philosophy
- modernc.org/sqlite (CGO-free database driver)
- golang-jwt/jwt (JWT authentication)
- Air (hot-reload in development)

**Current Status:** Initial setup complete (project structure, configuration)

---

## Key Technical Decisions

### Go Version & Features
- Use Go 1.22+ features: improved http.ServeMux, log/slog
- Leverage modern stdlib capabilities

### Dependencies Philosophy
- **Prefer stdlib over external dependencies** unless strongly justified
- External libraries must be well-maintained and idiomatic
- Minimize dependency footprint for easier maintenance

### Database
- **Driver:** modernc.org/sqlite (pure Go, CGO-free)
- **Why:** Avoids CGO compilation issues, cross-platform friendly
- **Alternative rejected:** mattn/go-sqlite3 (requires CGO)

### Authentication
- **JWT:** HS256 signing algorithm
- **Session:** Secure cookie-based authentication
- **Session storage:** In-memory (simple) - upgradeable to SQLite table for production
- **Secret management:** Support both direct values and _FILE suffix for Docker Swarm

### Logging
- **Library:** stdlib log/slog (Go 1.21+)
- **Format:** JSON in production, text in development (configurable)
- **Features:** Structured logging with correlation IDs

### Template Rendering
- **Library:** stdlib html/template
- **Caching:** Enabled in production, hot-reload in development

---

## Coding Standards & Conventions

### Environment Variable Naming Convention

**Principle:** All environment variables must be descriptive and include their domain context to prevent ambiguity and improve readability.

**Naming Rules:**
1. **Prefix with domain/category:** e.g., `HTTP_`, `SQLITE_`, `JWT_`, `SESSION_`, `APP_`, `RATE_LIMIT_`
2. **Use descriptive names:** `PORT` → `HTTP_PORT` (what port? HTTP server port)
3. **Separate words with underscores:** Standard convention for environment variables
4. **For secrets with file references:** Use `_FILE` suffix for Docker Swarm compatibility (e.g., `JWT_SIGNING_SECRET_FILE`)
5. **Avoid abbreviations:** Use full words for clarity (e.g., `AUTHENTICATION` not `AUTH`)

**Standard Categories:**

- **HTTP Server:**
  - `HTTP_PORT` (default: 8880)
  - `HTTP_SHUTDOWN_TIMEOUT` (default: 30s)
  - `HTTP_READ_TIMEOUT`
  - `HTTP_WRITE_TIMEOUT`
  - `HTTP_IDLE_TIMEOUT`

- **Database:**
  - `SQLITE_DB_FILE` (default: "./app.db")
  - `SQLITE_MAX_OPEN_CONNECTIONS` (default: 25)
  - `SQLITE_MAX_IDLE_CONNECTIONS` (default: 25)
  - `SQLITE_CONNECTION_MAX_LIFETIME_SECONDS` (default: 300)

- **Authentication (JWT):**
  - `JWT_SIGNING_SECRET` or `JWT_SIGNING_SECRET_FILE`
  - `JWT_EXPIRATION_SECONDS` (default: 3600)

- **Authentication (Session):**
  - `SESSION_COOKIE_SECRET`
  - `SESSION_COOKIE_NAME` (default: "session")
  - `SESSION_MAX_AGE_SECONDS` (default: 3600)
  - `SESSION_COOKIE_HTTP_ONLY` (default: true)
  - `SESSION_COOKIE_SECURE` (default: true in production)
  - `SESSION_COOKIE_SAMESITE` (default: "Lax")

- **Application:**
  - `APP_ENV` ("development" or "production")
  - `APP_LOG_LEVEL` ("debug", "info", "warn", "error", default: "info")
  - `APP_LOG_FORMAT` ("json" or "text", default: "json" in production, "text" in development)
  - `APP_NAME`

- **Rate Limiting:**
  - `RATE_LIMIT_REQUESTS_PER_WINDOW` (default: 100)
  - `RATE_LIMIT_WINDOW_SECONDS` (default: 60)

- **CORS:**
  - `CORS_ALLOWED_ORIGINS`
  - `CORS_ALLOWED_METHODS`
  - `CORS_ALLOWED_HEADERS`
  - `CORS_MAX_AGE_SECONDS`

**Examples:**
- ❌ `PORT` → ✅ `HTTP_PORT`
- ❌ `DB_FILE` → ✅ `SQLITE_DB_FILE`
- ❌ `JWT_SECRET` → ✅ `JWT_SIGNING_SECRET`
- ❌ `LOG_LEVEL` → ✅ `APP_LOG_LEVEL`
- ❌ `RATE_LIMIT` → ✅ `RATE_LIMIT_REQUESTS_PER_WINDOW`

**Rationale:** Contextual naming makes configuration self-documenting, prevents naming collisions, and makes it immediately clear what each variable controls without looking up documentation.

### Go Code Conventions
- Follow idiomatic Go conventions (package naming, error handling, interfaces)
- Use standard project layout: cmd/app, internal/routes, internal/handlers, internal/middlewares, internal/models, templates/
- Prefer explicit error handling over silent failures
- Use context.Context for request-scoped values

---

## Architecture Guidelines

### Project Structure
```
gostarterkit/
├── cmd/app/           # Application entry point
├── internal/
│   ├── config/        # Configuration management
│   ├── handlers/      # HTTP handlers
│   ├── middlewares/   # HTTP middlewares
│   ├── models/        # Data models
│   └── routes/        # Route registration
├── templates/         # HTML templates
├── Makefile           # Build automation
├── .env.example       # Environment variable template
└── go.mod            # Go module definition
```

### HTTP Server
- Use stdlib net/http with Go 1.22+ http.ServeMux
- Implement graceful shutdown with context timeout (configurable via `HTTP_SHUTDOWN_TIMEOUT`)
- Handle SIGTERM and SIGINT signals
- Wait for in-flight requests up to timeout

### Routing
- Use stdlib http.ServeMux (Go 1.22+) with named routes
- Separate handlers by feature in internal/handlers/
- Routes registered in internal/routes/routes.go

### Health Endpoints
- `GET /healthz` - Basic health info
- `GET /livez` - Liveness check (app is running)
- `GET /readyz` - Readiness check (dependencies ready)
- All endpoints return JSON with status field
- Accessible without authentication

### Logging & Observability
- Structured logging with stdlib log/slog
- Access logs go to stdout, error logs to stderr
- Middleware generates and propagates correlation ID
- Request ID middleware generates UUID if not present in header
- Request ID stored in context and logged with all entries

### Development Workflow
- Hot-reload via Air for rapid development
- Template hot-reload in development mode
- Template caching in production mode

---

## Testing Patterns

### General Testing Guidelines
- Write unit tests for individual functions
- Write integration tests for middleware, handlers, endpoints
- Use test table patterns for multiple test cases
- Always include negative test cases for error handling
- Test edge cases (boundary conditions, special characters, concurrent access)

### Common Test Categories
- **Unit tests:** Individual functions and methods
- **Integration tests:** Middleware, handlers, endpoints with HTTP client
- **Negative cases:** Invalid inputs, error conditions, failures
- **Edge cases:** Boundary values, special characters, concurrent access, large inputs
- **Performance tests:** Rate limiting, high load scenarios (where applicable)

### Test Examples from PRD
- Template parsing and data binding
- Server startup and graceful shutdown
- Route registration and handler mapping
- Health endpoint responses
- Session creation, validation, and clearing
- JWT token generation and validation
- Database connection and CRUD operations
- Structured logging format and correlation ID propagation
- Request ID generation and propagation
- Rate limiting enforcement
- Input validation functions

---

## Environment Variables Reference

### HTTP Server Configuration
| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | HTTP server port | 8880 |
| `HTTP_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout (seconds) | 30 |
| `HTTP_READ_TIMEOUT` | Read timeout (seconds) | - |
| `HTTP_WRITE_TIMEOUT` | Write timeout (seconds) | - |
| `HTTP_IDLE_TIMEOUT` | Idle timeout (seconds) | - |

### Database Configuration
| Variable | Description | Default |
|----------|-------------|---------|
| `SQLITE_DB_FILE` | SQLite database file path | ./app.db |
| `SQLITE_MAX_OPEN_CONNECTIONS` | Maximum open connections | 25 |
| `SQLITE_MAX_IDLE_CONNECTIONS` | Maximum idle connections | 25 |
| `SQLITE_CONNECTION_MAX_LIFETIME_SECONDS` | Connection lifetime (seconds) | 300 |

### JWT Authentication
| Variable | Description | Default |
|----------|-------------|---------|
| `JWT_SIGNING_SECRET` | JWT signing secret | - |
| `JWT_SIGNING_SECRET_FILE` | Path to file containing JWT secret (Docker Swarm) | - |
| `JWT_EXPIRATION_SECONDS` | Token expiration time (seconds) | 3600 |

### Session Authentication
| Variable | Description | Default |
|----------|-------------|---------|
| `SESSION_COOKIE_SECRET` | Session cookie secret | - |
| `SESSION_COOKIE_NAME` | Session cookie name | "session" |
| `SESSION_MAX_AGE_SECONDS` | Session max age (seconds) | 3600 |
| `SESSION_COOKIE_HTTP_ONLY` | HTTP-only cookie flag | true |
| `SESSION_COOKIE_SECURE` | Secure cookie flag (true in production) | - |
| `SESSION_COOKIE_SAMESITE` | SameSite attribute | "Lax" |

### Application Configuration
| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | Environment ("development" or "production") | - |
| `APP_LOG_LEVEL` | Log level ("debug", "info", "warn", "error") | "info" |
| `APP_LOG_FORMAT` | Log format ("json" or "text") | "json" in prod, "text" in dev |
| `APP_NAME` | Application name | - |

### Rate Limiting
| Variable | Description | Default |
|----------|-------------|---------|
| `RATE_LIMIT_REQUESTS_PER_WINDOW` | Max requests per window | 100 |
| `RATE_LIMIT_WINDOW_SECONDS` | Time window (seconds) | 60 |

### CORS Configuration
| Variable | Description | Default |
|----------|-------------|---------|
| `CORS_ALLOWED_ORIGINS` | Allowed origins (comma-separated) | - |
| `CORS_ALLOWED_METHODS` | Allowed methods (comma-separated) | - |
| `CORS_ALLOWED_HEADERS` | Allowed headers (comma-separated) | - |
| `CORS_MAX_AGE_SECONDS` | CORS max age (seconds) | - |

---

## What to Avoid (Non-Goals)

These features are intentionally out of scope for this project:

- **ORM/database layer abstraction** - Use database/sql directly
- **OpenAPI/Swagger code generation** - Manual API documentation
- **Metrics collection** - No Prometheus, etc. beyond logging
- **Distributed tracing** - No Jaeger, etc.
- **WebSocket support** - Not included
- **Database GUI/migration tools** - Basic migration system only
- **Comprehensive test suite** - Critical path tests only
- **CI/CD configuration** - Not included
- **Dockerfile/containerization setup** - Not included
- **GraphQL support** - REST only
- **Caching layer** - Beyond HTTP headers only

---

## Development Workflow

### Makefile Targets

```bash
make init       # Initialize Go modules
make tidy       # Download dependencies
make build      # Build application
make run        # Run application
make watch      # Run with hot-reload (Air)
make clean      # Clean build artifacts
```

### Development Mode
- Set `APP_ENV=development`
- Use `make watch` for hot-reload
- Templates auto-reload on file changes
- Log format defaults to text

### Production Mode
- Set `APP_ENV=production`
- Use `make build` and `./bin/app` to run
- Templates cached for performance
- Log format defaults to JSON
- Secure cookies enabled by default

---

## Feature Status

**Completed:**
- US-001: Project structure and dependencies
- US-002: Configuration with _FILE support

**Pending:**
- US-003 through US-015: Templates, server, routing, auth, database, logging, middleware, validation, hot-reload, REST helpers

See PRD for detailed requirements and acceptance criteria.

---

## Quick Reference for AI Agents

When working on this project, always:

1. **Use contextual environment variable names** (e.g., `HTTP_PORT` not `PORT`)
2. **Prefer stdlib** unless there's a strong justification
3. **Follow the existing project structure** in internal/ directory
4. **Write tests** for critical paths (unit + integration + negative + edge cases)
5. **Use structured logging** with correlation IDs
6. **Implement graceful shutdown** for all long-running processes
7. **Use Go 1.22+ features** where applicable
8. **Keep it simple** - avoid adding features from the "What to Avoid" section