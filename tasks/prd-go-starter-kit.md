# PRD: Go Starter Kit

## Introduction

A production-ready Go project starter kit that follows idiomatic Go practices, prioritizes the standard library, and includes essential features for building modern web applications. The kit provides a solid foundation with configuration management, HTML templating, REST API, hot-reload development, graceful shutdown, and observability - all using minimal external dependencies.

## Goals

- Provide a starter template that follows Go idioms and best practices
- Prioritize stdlib usage with justified external dependencies
- Enable rapid development with hot-reload and Makefile automation
- Support both HTML template rendering and JSON API endpoints
- Include production-ready features: graceful shutdown, health checks, structured logging
- Support Docker Swarm-style file-based configuration (_FILE suffix)
- Demonstrate proper project structure (routes, handlers, middlewares, models, templates)

## Environment Variable Naming Convention

**Principle:** All environment variables must be descriptive and include their domain context to prevent ambiguity and improve readability.

**Naming Rules:**
1. **Prefix with domain/category**: e.g., `HTTP_`, `SQLITE_`, `JWT_`, `SESSION_`, `APP_`, `RATE_LIMIT_`
2. **Use descriptive names**: `PORT` → `HTTP_PORT` (what port? HTTP server port)
3. **Separate words with underscores**: Standard convention for environment variables
4. **For secrets with file references**: Use `_FILE` suffix for Docker Swarm compatibility (e.g., `JWT_SIGNING_SECRET_FILE`)
5. **Avoid abbreviations**: Use full words for clarity (e.g., `AUTHENTICATION` not `AUTH`)

**Standard Categories:**
- **HTTP Server:** `HTTP_PORT`, `HTTP_SHUTDOWN_TIMEOUT`, `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT`
- **Database:** `SQLITE_DB_FILE`, `SQLITE_MAX_OPEN_CONNECTIONS`, `SQLITE_MAX_IDLE_CONNECTIONS`, `SQLITE_CONNECTION_MAX_LIFETIME_SECONDS`
- **Authentication (JWT):** `JWT_SIGNING_SECRET`/`JWT_SIGNING_SECRET_FILE`, `JWT_EXPIRATION_SECONDS`
- **Authentication (Session):** `SESSION_COOKIE_SECRET`, `SESSION_COOKIE_NAME`, `SESSION_MAX_AGE_SECONDS`, `SESSION_COOKIE_HTTP_ONLY`, `SESSION_COOKIE_SECURE`, `SESSION_COOKIE_SAMESITE`
- **Application:** `APP_ENV`, `APP_LOG_LEVEL`, `APP_LOG_FORMAT`, `APP_NAME`
- **Rate Limiting:** `RATE_LIMIT_REQUESTS_PER_WINDOW`, `RATE_LIMIT_WINDOW_SECONDS`
- **CORS:** `CORS_ALLOWED_ORIGINS`, `CORS_ALLOWED_METHODS`, `CORS_ALLOWED_HEADERS`, `CORS_MAX_AGE_SECONDS`

**Examples:**
- ❌ `PORT` → ✅ `HTTP_PORT`
- ❌ `DB_FILE` → ✅ `SQLITE_DB_FILE`
- ❌ `JWT_SECRET` → ✅ `JWT_SIGNING_SECRET`
- ❌ `LOG_LEVEL` → ✅ `APP_LOG_LEVEL`
- ❌ `RATE_LIMIT` → ✅ `RATE_LIMIT_REQUESTS_PER_WINDOW`

**Rationale:** Contextual naming makes configuration self-documenting, prevents naming collisions, and makes it immediately clear what each variable controls without looking up documentation.

## User Stories

### US-001: Initialize project structure and dependencies
**Description:** As a developer, I need a well-organized project structure with Go modules initialized so I can start building quickly.

**Acceptance Criteria:**
- [x] Create go.mod with appropriate Go version
- [x] Set up directory structure: cmd/app, internal/routes, internal/handlers, internal/middlewares, internal/models, templates/
- [x] Create Makefile with targets: init, tidy, build, run, watch, clean
- [x] Create .gitignore for Go projects
- [x] `make init` initializes Go modules successfully
- [x] `make tidy` downloads dependencies without errors

### US-002: Implement .env configuration with _FILE support
**Description:** As a developer, I need environment variable configuration with Docker Swarm file support so I can securely manage secrets.

**Acceptance Criteria:**
- [x] Create config package using stdlib os.Getenv
- [x] Support .env file parsing with lightweight lib or custom implementation
- [x] Implement _FILE suffix support (e.g., JWT_SIGNING_SECRET_FILE reads file content)
- [x] Provide typed config struct with validation
- [x] Configuration loads on startup and panics on critical errors
- [x] go build succeeds without external config lib if using stdlib
- [x] Support all environment variables with contextual naming: `HTTP_PORT`, `SQLITE_DB_FILE`, `JWT_SIGNING_SECRET`, `APP_ENV`, `APP_LOG_LEVEL`, etc.
- [x] Unit test for config loading from environment variables
- [x] Unit test for .env file parsing with various formats
- [x] Unit test for _FILE suffix reading secrets from files
- [x] Unit test for config struct validation with valid/invalid inputs
- [x] Test that panic occurs on critical missing config values
- [x] Edge case: Empty .env file
- [x] Edge case: Environment variables with whitespace
- [x] Edge case: Unicode/special characters in config values
- [x] Edge case: Very large config values
- [x] Negative case: File with wrong permissions for _FILE suffix
- [x] Edge case: Concurrent config access
- [x] Edge case: Malformed .env file syntax

### US-003: Set up HTML template rendering
**Description:** As a developer, I need HTML template support so I can render server-side pages.

**Acceptance Criteria:**
- [x] Create templates/ directory with base layout
- [x] Implement template parsing using stdlib html/template
- [x] Create example page (e.g., home page) with data binding
- [x] Template caching in production mode
- [x] Template hot-reload in development mode
- [x] Rendered HTML is accessible via browser
- [x] Unit test for template parsing without errors
- [x] Unit test for data binding to templates with various data types
- [x] Integration test for template caching in production mode
- [x] Integration test for template hot-reload in development mode
- [x] Negative case: Template with invalid syntax fails gracefully
- [x] Negative case: Template with circular references
- [x] Edge case: Empty data binding
- [x] Edge case: Nil/zero values in template data
- [x] Edge case: Very large template files
- [x] Edge case: Unicode/special characters in template data
- [x] Edge case: Nested template includes
- [x] Edge case: Template with missing defined variables

### US-004: Implement HTTP server with graceful shutdown
**Description:** As a developer, I need an HTTP server that shuts down gracefully so connections complete properly on termination.

**Acceptance Criteria:**
- [x] Create server using stdlib net/http
- [x] Implement graceful shutdown with context timeout (configured via `HTTP_SHUTDOWN_TIMEOUT`, default 30s)
- [x] Handle SIGTERM and SIGINT signals
- [x] Wait for in-flight requests up to timeout (default 30s)
- [x] Log shutdown process and completion
- [x] Server accepts connections on `HTTP_PORT` (default 8880)
- [x] Integration test for server startup on `HTTP_PORT`
- [x] Integration test for graceful shutdown on SIGTERM signal
- [x] Integration test for graceful shutdown on SIGINT signal
- [x] Integration test for in-flight request handling during shutdown
- [x] Test for timeout behavior when shutdown exceeds `HTTP_SHUTDOWN_TIMEOUT`
- [x] Negative case: Server startup failure when `HTTP_PORT` already in use
- [x] Edge case: Very large request body handling
- [x] Edge case: Concurrent shutdown signals (SIGTERM + SIGINT)
- [x] Edge case: Rapid start/stop cycles
- [x] Edge case: Shutdown during long-running request
- [x] Edge case: Zero connections during shutdown

### US-005: Create routing structure
**Description:** As a developer, I need a clear routing structure so I can organize endpoints logically.

**Acceptance Criteria:**
- [x] Create internal/routes/routes.go with route registration
- [x] Separate handlers by feature in internal/handlers/
- [x] Use stdlib http.ServeMux (Go 1.22+) with named routes
- [x] Example routes: GET /, GET /health, GET /livez, GET /readyz
- [x] Routes are testable via curl/browser
- [x] Unit test for route registration
- [x] Integration test for handler mapping to correct endpoints
- [x] Integration test for example routes via HTTP client
- [x] Negative case: Undefined routes return 404
- [x] Negative case: Wrong HTTP method returns 405
- [x] Edge case: Routes with trailing slashes vs without
- [x] Edge case: Case sensitivity in route paths
- [x] Edge case: Routes with special characters
- [x] Edge case: Very long path segments
- [x] Edge case: Routes with query parameters

### US-006: Implement structured logging
**Description:** As a developer, I need structured logging with correlation IDs so I can trace requests through logs.

**Acceptance Criteria:**
- [x] Create logger package using stdlib log/slog (Go 1.21+)
- [x] Access logs go to stdout, error logs to stderr
- [x] Middleware generates and propagates correlation ID
- [x] Log format configured via `APP_LOG_FORMAT` ("json" or "text", default: "json" in production, "text" in development)
- [x] Log level configured via `APP_LOG_LEVEL` ("debug", "info", "warn", "error", default: "info")
- [x] Logs include correlation ID, timestamp, level, message
- [x] Unit test for logger initialization
- [x] Integration test for structured JSON format in production
- [x] Integration test for human-readable format in development
- [x] Integration test for correlation ID propagation in logs
- [x] Edge case: Very long log messages
- [x] Edge case: Concurrent log writes
- [x] Edge case: Logging nil/undefined values
- [x] Edge case: Unicode/emoji in log messages
- [x] Edge case: Invalid correlation ID format

### US-007: Implement request ID middleware
**Description:** As a developer, I need request ID tracking so I can correlate logs across services.

**Acceptance Criteria:**
- [x] Middleware generates UUID if not present in header
- [x] Request ID stored in request context
- [x] Request ID included in response headers (X-Request-ID)
- [x] Request ID available in all handlers via context
- [x] Request ID logged with all log entries
- [x] Middleware testable via integration tests
- [x] Negative case: Malformed UUID in header handled
- [x] Negative case: Empty/null request ID handled
- [x] Edge case: Very long request ID strings
- [x] Edge case: Request ID already present in header preserved
- [x] Edge case: Multiple concurrent requests with different IDs
- [x] Edge case: Request ID with special characters

### US-008: Build REST API helper functions
**Description:** As a developer, I need helper functions for building REST APIs so handler code is concise.

**Acceptance Criteria:**
- [x] JSONResponse helper for success responses
- [x] ErrorResponse helper for error responses
- [x] Helper handles Content-Type header
- [x] Helpers support status codes
- [x] Example API handler demonstrates usage
- [x] API returns proper HTTP status codes
- [x] Unit test for JSONResponse helper with various data types
- [x] Unit test for ErrorResponse helper with various error scenarios
- [x] Unit test for Content-Type header setting
- [x] Unit test for status code handling in helpers
- [x] Integration test for example API handler demonstrating usage
- [x] Negative case: JSON with circular references handled
- [x] Negative case: Invalid JSON structure handled
- [x] Edge case: Very large response bodies
- [x] Edge case: JSON with special characters
- [x] Edge case: Nil/empty data structures
- [x] Edge case: Multiple concurrent API responses
- [x] Edge case: Response with nested data structures

### US-009: Add input validation helpers
**Description:** As a developer, I need input validation helpers so I can ensure data integrity.

**Acceptance Criteria:**
- [x] Validation functions for common types (email, UUID, length)
- [x] Struct tag-based validation or fluent builder pattern
- [x] Helper function returns errors with field names
- [x] Validation middleware example
- [x] Validation errors are user-friendly
- [x] Unit tests for validation functions
- [x] Negative case: Empty string validation
- [x] Negative case: Invalid email format rejected
- [x] Negative case: Invalid UUID format rejected
- [x] Negative case: Length exceeding maximum rejected
- [x] Edge case: Validation at exact length boundary
- [x] Edge case: Validation with unicode/special characters
- [x] Edge case: Very long input strings
- [x] Edge case: Null/undefined values in validation
- [x] Edge case: Multiple validation errors aggregated

### US-010: Add rate limiting middleware
**Description:** As a developer, I need rate limiting so I can protect endpoints from abuse.

**Acceptance Criteria:**
- [ ] Rate limiter middleware using in-memory map
- [ ] Rate limit configured via `RATE_LIMIT_REQUESTS_PER_WINDOW` (default: 100)
- [ ] Time window configured via `RATE_LIMIT_WINDOW_SECONDS` (default: 60)
- [ ] Rate limits per IP address
- [ ] Returns 429 status when limit exceeded
- [ ] Includes Retry-After header
- [ ] Sliding window or token bucket algorithm
- [ ] Integration test for rate limiting enforcement
- [ ] Integration test for 429 status code on limit exceeded
- [ ] Integration test for Retry-After header presence
- [ ] Unit test for sliding window/token bucket algorithm
- [ ] Edge case: Boundary condition - exactly at limit
- [ ] Edge case: Boundary condition - limit + 1 request
- [ ] Edge case: Multiple IPs from same client (proxy)
- [ ] Edge case: Very high request volume
- [ ] Edge case: Rate limit reset timing
- [ ] Edge case: IPv6 address handling
- [ ] Edge case: Concurrent requests from same IP

### US-011: Set up SQLite database support
**Description:** As a developer, I need SQLite database support so I can persist data without external dependencies.

**Acceptance Criteria:**
- [ ] Use stdlib database/sql with modernc.org/sqlite driver
- [ ] Create database connection pool with config
- [ ] Database file path configured via `SQLITE_DB_FILE` (default: "./app.db")
- [ ] Connection pool configured via: `SQLITE_MAX_OPEN_CONNECTIONS` (default: 25), `SQLITE_MAX_IDLE_CONNECTIONS` (default: 25), `SQLITE_CONNECTION_MAX_LIFETIME_SECONDS` (default: 300)
- [ ] Implement database migration system
- [ ] Example model with CRUD operations
- [ ] Database connection tests pass
- [ ] Connection handles graceful shutdown
- [ ] Negative case: Connection pool exhaustion handled
- [ ] Negative case: Invalid `SQLITE_DB_FILE` path handled
- [ ] Negative case: Corrupted database file handled
- [ ] Edge case: Very large query results
- [ ] Edge case: Concurrent database operations
- [ ] Edge case: Migration with very long SQL statements
- [ ] Edge case: Database file permissions
- [ ] Edge case: Database connection during graceful shutdown

### US-012: Implement health check endpoints
**Description:** As a developer, I need Kubernetes-style health endpoints so my deployment is monitorable.

**Acceptance Criteria:**
- [ ] GET /healthz returns 200 with basic health info
- [ ] GET /livez returns 200 if app is running
- [ ] GET /readyz returns 200 if dependencies are ready
- [ ] Health checks include database connectivity
- [ ] Endpoints return JSON with status field
- [ ] Endpoints accessible without authentication
- [ ] Integration test for /healthz endpoint (200 status, JSON response)
- [ ] Integration test for /livez endpoint (200 status)
- [ ] Integration test for /readyz endpoint with database connectivity
- [ ] Unit test for JSON response structure validation
- [ ] Integration test for endpoint accessibility without auth
- [ ] Negative case: /readyz returns 503 when database unavailable
- [ ] Negative case: Service unavailable when app not responding
- [ ] Edge case: Health checks under high load
- [ ] Edge case: Database reconnect attempts in health checks
- [ ] Edge case: Multiple simultaneous health check requests
- [ ] Edge case: Health check during graceful shutdown

### US-013: Add session-based authentication for HTML
**Description:** As a developer, I need session authentication for HTML pages so users can log in securely.

**Acceptance Criteria:**
- [ ] Implement session middleware using secure cookie
- [ ] Session storage using in-memory (or SQLite table for production)
- [ ] Login handler validates credentials
- [ ] Session middleware protects HTML routes
- [ ] Logout handler clears session
- [ ] Session is secure (HTTP-only, Secure, SameSite)
- [ ] Configurable via environment: `SESSION_COOKIE_SECRET`, `SESSION_COOKIE_NAME` (default: "session"), `SESSION_MAX_AGE_SECONDS` (default: 3600), `SESSION_COOKIE_HTTP_ONLY` (default: true), `SESSION_COOKIE_SECURE` (default: true in production), `SESSION_COOKIE_SAMESITE` (default: "Lax")
- [ ] Integration test for session creation on login
- [ ] Integration test for session validation in middleware
- [ ] Integration test for session clearing on logout
- [ ] Unit test for secure cookie attributes (HTTP-only, Secure, SameSite)
- [ ] Negative case: Login with invalid credentials fails
- [ ] Negative case: Session with invalid token rejected
- [ ] Negative case: Corrupted session data handled gracefully
- [ ] Negative case: Session cookie tampering detected
- [ ] Edge case: Very long session cookie values
- [ ] Edge case: Multiple concurrent sessions for same user
- [ ] Edge case: Session expiry exactly at request time
- [ ] Edge case: Session storage with maximum capacity

### US-014: Add JWT authentication for API
**Description:** As a developer, I need JWT authentication for API endpoints so stateless auth is available.

**Acceptance Criteria:**
- [ ] Generate JWT tokens with HS256 signing
- [ ] JWT middleware validates token and sets user context
- [ ] Token generation endpoint (/api/login)
- [ ] Protected API routes require valid JWT
- [ ] Token expiration configured via `JWT_EXPIRATION_SECONDS` (default: 3600)
- [ ] JWT signing secret configured via `JWT_SIGNING_SECRET` or `JWT_SIGNING_SECRET_FILE` (Docker Swarm)
- [ ] Use idiomatic JWT library (golang-jwt/jwt) or minimal stdlib implementation
- [ ] Unit test for JWT token generation with HS256
- [ ] Integration test for JWT token validation in middleware
- [ ] Integration test for middleware rejection of invalid/expired tokens
- [ ] Integration test for protected API route requiring valid JWT
- [ ] Negative case: Expired token rejected
- [ ] Negative case: Malformed token rejected
- [ ] Negative case: Token with wrong signature rejected
- [ ] Edge case: JWT with 0 seconds expiration
- [ ] Edge case: JWT with very large payload
- [ ] Edge case: Multiple concurrent token validations
- [ ] Edge case: Token with unusual character encoding

### US-015: Configure Air for hot-reload
**Description:** As a developer, I need hot-reload in development so I can see changes immediately.

**Acceptance Criteria:**
- [ ] Install Air as dev dependency
- [ ] Create .air.toml configuration file
- [ ] Configure watch directories and build command
- [ ] Makefile `make watch` runs Air correctly
- [ ] Air restarts on file changes within 2 seconds
- [ ] Air logs build output clearly
- [ ] Manual verification: Air restarts on file changes within 2 seconds
- [ ] Manual verification: Air logs build output clearly

## Functional Requirements

- FR-1: Project follows idiomatic Go conventions (package naming, error handling, interfaces)
- FR-2: Configuration loads from .env files with _FILE suffix support for Docker Swarm
- FR-3: HTML templates use stdlib html/template with caching support
- FR-4: HTTP server uses stdlib net/http with graceful shutdown on SIGTERM/SIGINT
- FR-5: Routes organized in internal/routes/ with handlers in internal/handlers/
- FR-6: Health endpoints: /healthz, /livez, /readyz return JSON status
- FR-7: Session auth uses secure cookies for HTML routes
- FR-8: JWT auth provides stateless authentication for API routes
- FR-9: SQLite database uses stdlib database/sql with modernc.org/sqlite driver
- FR-10: Structured logging uses stdlib log/slog with correlation IDs
- FR-11: Request ID middleware generates and tracks UUID per request
- FR-12: Rate limiting middleware protects endpoints with configurable limits
- FR-13: Input validation helpers provide common validation patterns
- FR-14: Air hot-reload configured for development workflow
- FR-15: REST API helpers simplify JSON response/error handling

## Non-Goals (Out of Scope)

- No ORM/database layer abstraction (use database/sql directly)
- No OpenAPI/Swagger code generation
- No metrics collection (Prometheus, etc.) beyond logging
- No distributed tracing (Jaeger, etc.)
- No WebSocket support
- No database GUI/migration tools beyond basic migration system
- No comprehensive test suite (only critical path tests)
- No CI/CD configuration
- No Dockerfile/containerization setup
- No GraphQL support
- No caching layer beyond HTTP headers

## Design Considerations

- Use Go 1.22+ features (improved http.ServeMux, log/slog, etc.)
- Prefer stdlib over external dependencies unless strongly justified
- External libraries must be well-maintained and idiomatic
- Project structure follows "standard Go project layout" patterns
- **Environment Variable Naming:** All configuration must use contextual, domain-prefixed naming (see Environment Variable Naming Convention section above)
- Session storage: in-memory for simplicity, upgradeable to SQLite
- JWT signing: HS256 with secret from `JWT_SIGNING_SECRET` or `JWT_SIGNING_SECRET_FILE`
- Database migrations: simple versioned SQL files
- Makefile targets should work cross-platform (Unix-like systems)

## Technical Considerations

- **Database:** modernc.org/sqlite (pure Go, CGO-free) vs mattn/go-sqlite3
- **Session:** github.com/gorilla/sessions or custom implementation
- **JWT:** github.com/golang-jwt/jwt or minimal stdlib implementation
- **Template caching:** based on `APP_ENV` ("development" or "production")
- **Graceful shutdown timeout:** configurable via `HTTP_SHUTDOWN_TIMEOUT`, default 30s
- **Rate limiting:** memory-based, configured via `RATE_LIMIT_REQUESTS_PER_WINDOW` and `RATE_LIMIT_WINDOW_SECONDS`, consider production upgrade path
- **Logging:** `APP_LOG_FORMAT` controls JSON vs text format, `APP_LOG_LEVEL` controls verbosity
- **Config validation:** panic on startup for critical missing values (e.g., `HTTP_PORT`, `SQLITE_DB_FILE`, `JWT_SIGNING_SECRET`)

## Success Metrics

- Developer can `make watch` and start coding in <5 minutes
- All Makefile targets execute without errors
- Health endpoints respond in <10ms
- Graceful shutdown completes within timeout on all platforms
- Project builds without CGO dependency issues
- Hot-reload detects changes and rebuilds within 2 seconds
- Template rendering works with and without caching

## Open Questions

- Should JWT and session auth be optional via build tags or always included?
- Should rate limiting be configurable per-route?
- Should database migrations run automatically on startup or require manual command?
- Should we include example API handlers beyond authentication?