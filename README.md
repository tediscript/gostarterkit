# Go Starter Kit

A production-ready Go web application starter kit following idiomatic Go conventions with a stdlib-first philosophy.

## Features

- 🚀 **Modern Go 1.22+** - Leverages latest Go features including improved http.ServeMux and log/slog
- 📦 **Stdlib-First** - Minimal external dependencies for easier maintenance
- 🗄️ **CGO-Free SQLite** - Uses modernc.org/sqlite for cross-platform compatibility
- 🔐 **Dual Authentication** - JWT tokens and secure session cookies
- 📝 **Structured Logging** - JSON production logs with correlation IDs
- 🛡️ **Comprehensive Middleware** - Request ID, rate limiting, CORS, recovery
- 🔄 **Hot-Reload Development** - Air integration for rapid development
- 🐳 **Docker Swarm Ready** - _FILE suffix support for secrets
- 🧪 **Test Coverage** - Unit and integration tests for critical paths
- ⚡ **Graceful Shutdown** - Handles SIGTERM/SIGINT with configurable timeout

## Technology Stack

- **Language:** Go 1.22+
- **Web Server:** stdlib net/http with Go 1.22+ http.ServeMux
- **Database:** modernc.org/sqlite (CGO-free SQLite driver)
- **Authentication:** golang-jwt/jwt (JWT), stdlib cookie-based sessions
- **Logging:** stdlib log/slog
- **Templating:** stdlib html/template
- **Hot-Reload:** Air (development only)

## Requirements

- Go 1.22 or higher
- (Optional) Air for hot-reload: `go install github.com/cosmtrek/air@latest`

## Quick Start

```bash
# Clone the repository
git clone https://github.com/tediscript/gostarterkit.git
cd gostarterkit

# Initialize dependencies
make init

# Create configuration file
cp .env.example .env

# Build and run
make run

# Or run with hot-reload (requires Air)
make watch
```

The application will start on `http://localhost:8880` by default.

## Configuration

Configuration is managed through environment variables. Copy `.env.example` to `.env` and customize as needed.

### Key Environment Variables

| Category | Variable | Description | Default |
|----------|----------|-------------|---------|
| **HTTP Server** | `HTTP_PORT` | HTTP server port | 8880 |
| | `HTTP_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | 30s |
| | `HTTP_READ_TIMEOUT` | Read timeout | 15s |
| | `HTTP_WRITE_TIMEOUT` | Write timeout | 15s |
| | `HTTP_IDLE_TIMEOUT` | Idle timeout | 60s |
| **Database** | `SQLITE_DB_FILE` | SQLite database file path | ./app.db |
| | `SQLITE_MAX_OPEN_CONNECTIONS` | Maximum open connections | 25 |
| | `SQLITE_MAX_IDLE_CONNECTIONS` | Maximum idle connections | 25 |
| **JWT Auth** | `JWT_SIGNING_SECRET` | JWT signing secret | - |
| | `JWT_EXPIRATION_SECONDS` | Token expiration time | 3600 |
| **Session** | `SESSION_COOKIE_SECRET` | Session cookie secret | - |
| | `SESSION_COOKIE_NAME` | Session cookie name | session |
| | `SESSION_MAX_AGE_SECONDS` | Session max age | 3600 |
| | `SESSION_COOKIE_HTTP_ONLY` | HTTP-only cookie flag | true |
| | `SESSION_COOKIE_SECURE` | Secure cookie flag | true (production) |
| **Application** | `APP_ENV` | Environment (development/production) | - |
| | `APP_LOG_LEVEL` | Log level (debug/info/warn/error) | info |
| | `APP_LOG_FORMAT` | Log format (json/text) | json (prod), text (dev) |
| **Rate Limiting** | `RATE_LIMIT_REQUESTS_PER_WINDOW` | Max requests per window | 100 |
| | `RATE_LIMIT_WINDOW_SECONDS` | Time window | 60 |
| **CORS** | `CORS_ALLOWED_ORIGINS` | Allowed origins | * |
| | `CORS_ALLOWED_METHODS` | Allowed methods | GET,POST,PUT,DELETE,OPTIONS |

See `.env.example` for the complete configuration reference.

## Project Structure

```
gostarterkit/
├── cmd/
│   └── app/               # Application entry point
│       └── main.go
├── internal/
│   ├── config/            # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── handlers/          # HTTP handlers
│   │   └── handlers.go
│   ├── logger/            # Structured logging
│   │   ├── logger.go
│   │   └── logger_test.go
│   ├── middlewares/       # HTTP middlewares
│   │   ├── middlewares.go
│   │   └── middlewares_test.go
│   ├── models/            # Data models
│   │   └── models.go
│   ├── routes/            # Route registration
│   │   ├── routes.go
│   │   └── routes_test.go
│   ├── server/            # HTTP server
│   │   ├── server.go
│   │   └── server_test.go
│   └── templates/         # Template rendering
│       ├── templates.go
│       └── templates_test.go
├── templates/             # HTML templates
│   ├── base.html
│   └── home.html
├── .env.example           # Environment variable template
├── Makefile               # Build automation
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Development

### Makefile Targets

```bash
make help      # Show all available targets
make init      # Initialize Go modules
make tidy      # Download and organize dependencies
make build     # Build the application
make run       # Build and run the application
make watch     # Run with hot-reload (requires Air)
make clean     # Clean build artifacts
```

### Hot-Reload Development

For rapid development, use Air for automatic reloading:

```bash
# Install Air (if not already installed)
go install github.com/cosmtrek/air@latest

# Run with hot-reload
make watch
```

Changes to Go files will automatically rebuild and restart the server.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/config

# Run tests with verbose output
go test -v ./...
```

## API Endpoints

### Health Endpoints

All health endpoints are accessible without authentication:

- `GET /healthz` - Basic health information
- `GET /livez` - Liveness check (application is running)
- `GET /readyz` - Readiness check (dependencies are ready)

Responses are in JSON format:

```json
{
  "status": "ok"
}
```

### Routing

Routes are registered in `internal/routes/routes.go` using Go 1.22+ http.ServeMux with named routes. Handlers are organized by feature in `internal/handlers/`.

## Key Features

### Authentication

The starter kit supports dual authentication methods:

1. **JWT Authentication** - Token-based authentication with HS256 signing
   - Configurable expiration time
   - Secret can be set directly or via `_FILE` suffix for Docker Swarm

2. **Session Authentication** - Secure cookie-based sessions
   - HttpOnly and Secure flags for security
   - Configurable session lifetime
   - SameSite attribute for CSRF protection

### Logging

Structured logging using stdlib `log/slog`:

- **JSON format** in production for log aggregation
- **Text format** in development for readability
- **Correlation IDs** automatically generated and propagated
- **Request IDs** logged with all entries for traceability
- Configurable log levels (debug, info, warn, error)

### Middleware

Comprehensive middleware stack:

- **Request ID** - Generates and propagates unique request IDs
- **Recovery** - Panic recovery with proper logging
- **Rate Limiting** - Token bucket algorithm with configurable limits
- **CORS** - Configurable CORS policies for cross-origin requests
- **Logging** - Access logging with correlation IDs

### Database

SQLite database with connection pooling:

- **CGO-free** using modernc.org/sqlite
- Configurable connection pool settings
- Automatic connection lifetime management

### Graceful Shutdown

The server implements graceful shutdown:

- Handles SIGTERM and SIGINT signals
- Waits for in-flight requests up to configurable timeout
- Properly closes database connections
- Clean resource cleanup

## Design Philosophy

### Stdlib-First

This project follows a stdlib-first philosophy, preferring Go's standard library over external dependencies. External libraries are only used when strongly justified and must be well-maintained and idiomatic.

### Environment Variable Naming

All environment variables use descriptive, contextual names:

- Prefix with domain/category (e.g., `HTTP_`, `SQLITE_`, `JWT_`)
- Use full words instead of abbreviations
- Include `_FILE` suffix for Docker Swarm secret references

Examples:
- ✅ `HTTP_PORT` (not `PORT`)
- ✅ `SQLITE_DB_FILE` (not `DB_FILE`)
- ✅ `JWT_SIGNING_SECRET` (not `JWT_SECRET`)

### Non-Goals

The following features are intentionally out of scope:

- ORM/database layer abstraction - Use database/sql directly
- OpenAPI/Swagger code generation - Manual API documentation
- Metrics collection - No Prometheus, etc. beyond logging
- Distributed tracing - No Jaeger, etc.
- WebSocket support - Not included
- Database GUI/migration tools - Basic migration system only
- Comprehensive test suite - Critical path tests only
- CI/CD configuration - Not included
- Dockerfile/containerization setup - Not included
- GraphQL support - REST only
- Caching layer - Beyond HTTP headers only

## License

See [LICENSE](LICENSE) for details.

## Contributing

This project follows the coding standards outlined in [AGENTS.md](AGENTS.md). When contributing, please:

1. Follow the existing project structure
2. Write tests for critical paths (unit + integration + negative + edge cases)
3. Use structured logging with correlation IDs
4. Implement graceful shutdown for long-running processes
5. Use Go 1.22+ features where applicable
6. Keep it simple - avoid adding features from the "Non-Goals" section