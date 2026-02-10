package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/google/uuid"
)

// contextKey is a custom type to avoid context key collisions
type contextKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey contextKey = "correlation_id"
)

// Logger wraps slog.Logger with correlation ID support
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level       slog.Leveler
	Format      string // "json" or "text"
	Output      io.Writer
	ErrorOutput io.Writer
}

// New creates a new logger with the given configuration
func New(cfg Config) *Logger {
	var handler slog.Handler

	// Determine output writer based on log level
	// We'll use a custom handler to route to stdout/stderr
	handler = NewLevelWriterHandler(
		cfg.Output,
		cfg.ErrorOutput,
		cfg.Format,
		cfg.Level,
	)

	return &Logger{
		Logger: slog.New(handler),
	}
}

// FromContext extracts a logger with correlation ID from context
func FromContext(ctx context.Context) *Logger {
	logger := slog.Default()

	// Add correlation ID to logger context if present
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		if id, ok := correlationID.(string); ok {
			logger = logger.With(slog.String("correlation_id", id))
		}
	}

	return &Logger{Logger: logger}
}

// WithContext returns a logger with correlation ID from context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	logger := l.Logger

	// Add correlation ID to logger context if present
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		if id, ok := correlationID.(string); ok {
			logger = logger.With(slog.String("correlation_id", id))
		}
	}

	return &Logger{Logger: logger}
}

// InfoCtx logs an info message with correlation ID from context
func InfoCtx(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

// DebugCtx logs a debug message with correlation ID from context
func DebugCtx(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}

// WarnCtx logs a warning message with correlation ID from context
func WarnCtx(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

// ErrorCtx logs an error message with correlation ID from context
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
}

// GenerateCorrelationID generates a new correlation ID
func GenerateCorrelationID() string {
	return uuid.New().String()
}

// NewContextWithCorrelationID creates a new context with correlation ID
func NewContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// GetCorrelationID extracts the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if correlationID := ctx.Value(CorrelationIDKey); correlationID != nil {
		if id, ok := correlationID.(string); ok {
			return id
		}
	}
	return ""
}

// ParseLogLevel converts string log level to slog.Level
func ParseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ValidateLogFormat validates log format string
func ValidateLogFormat(format string) string {
	if format != "json" && format != "text" {
		return "json"
	}
	return format
}

// LevelWriterHandler routes log records to different writers based on level
type LevelWriterHandler struct {
	infoHandler  slog.Handler
	errorHandler slog.Handler
	mu           sync.Mutex
}

// NewLevelWriterHandler creates a new LevelWriterHandler
func NewLevelWriterHandler(output, errorOutput io.Writer, format string, level slog.Leveler) *LevelWriterHandler {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create handlers for both outputs
	var baseHandler slog.Handler
	if format == "json" {
		baseHandler = slog.NewJSONHandler(output, opts)
	} else {
		baseHandler = slog.NewTextHandler(output, opts)
	}

	var errorBaseHandler slog.Handler
	if format == "json" {
		errorBaseHandler = slog.NewJSONHandler(errorOutput, opts)
	} else {
		errorBaseHandler = slog.NewTextHandler(errorOutput, opts)
	}

	return &LevelWriterHandler{
		infoHandler:  baseHandler,
		errorHandler: errorBaseHandler,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *LevelWriterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.infoHandler.Enabled(ctx, level) || h.errorHandler.Enabled(ctx, level)
}

// Handle handles the Record
func (h *LevelWriterHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Error and warning levels go to stderr, others to stdout
	if r.Level >= slog.LevelWarn {
		return h.errorHandler.Handle(ctx, r)
	}
	return h.infoHandler.Handle(ctx, r)
}

// WithAttrs returns a new Handler with attributes added
func (h *LevelWriterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	return &LevelWriterHandler{
		infoHandler:  h.infoHandler.WithAttrs(attrs),
		errorHandler: h.errorHandler.WithAttrs(attrs),
	}
}

// WithGroup returns a new Handler with a group
func (h *LevelWriterHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	return &LevelWriterHandler{
		infoHandler:  h.infoHandler.WithGroup(name),
		errorHandler: h.errorHandler.WithGroup(name),
	}
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	slog.SetDefault(l.Logger)
}

// Init initializes the default logger from environment variables
func Init(level, format string) *Logger {
	cfg := Config{
		Level:       ParseLogLevel(level),
		Format:      ValidateLogFormat(format),
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
	}

	logger := New(cfg)
	SetDefault(logger)

	return logger
}

// Default returns the default logger
func Default() *Logger {
	return &Logger{Logger: slog.Default()}
}
