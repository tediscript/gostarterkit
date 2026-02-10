package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{"debug level", "debug", slog.LevelDebug},
		{"info level", "info", slog.LevelInfo},
		{"warn level", "warn", slog.LevelWarn},
		{"error level", "error", slog.LevelError},
		{"invalid level defaults to info", "invalid", slog.LevelInfo},
		{"empty string defaults to info", "", slog.LevelInfo},
		{"uppercase level", "INFO", slog.LevelInfo}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLogLevel(tt.level)
			if result != tt.expected {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.level, result, tt.expected)
			}
		})
	}
}

func TestValidateLogFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{"json format", "json", "json"},
		{"text format", "text", "text"},
		{"invalid format defaults to json", "invalid", "json"},
		{"empty string defaults to json", "", "json"},
		{"uppercase JSON", "JSON", "json"}, // case insensitive
		{"uppercase TEXT", "TEXT", "json"}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateLogFormat(tt.format)
			if result != tt.expected {
				t.Errorf("ValidateLogFormat(%q) = %q, want %q", tt.format, result, tt.expected)
			}
		})
	}
}

func TestGenerateCorrelationID(t *testing.T) {
	id := GenerateCorrelationID()
	if id == "" {
		t.Error("GenerateCorrelationID() returned empty string")
	}

	// Generate another ID and ensure they're different
	id2 := GenerateCorrelationID()
	if id == id2 {
		t.Error("GenerateCorrelationID() returned duplicate IDs")
	}

	// UUIDs should be 36 characters (including hyphens)
	if len(id) != 36 {
		t.Errorf("GenerateCorrelationID() returned ID with length %d, want 36", len(id))
	}
}

func TestNewContextWithCorrelationID(t *testing.T) {
	ctx := context.Background()
	id := GenerateCorrelationID()

	newCtx := NewContextWithCorrelationID(ctx, id)
	retrievedID := GetCorrelationID(newCtx)

	if retrievedID != id {
		t.Errorf("GetCorrelationID() = %q, want %q", retrievedID, id)
	}

	// Original context should not have the ID
	originalID := GetCorrelationID(ctx)
	if originalID != "" {
		t.Errorf("Original context should not have correlation ID, got %q", originalID)
	}
}

func TestGetCorrelationID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "context with correlation ID",
			ctx:      NewContextWithCorrelationID(context.Background(), "test-id"),
			expected: "test-id",
		},
		{
			name:     "context without correlation ID",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCorrelationID(tt.ctx)
			if result != tt.expected {
				t.Errorf("GetCorrelationID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLoggerInitialization(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		format string
	}{
		{"json info", "info", "json"},
		{"json debug", "debug", "json"},
		{"text info", "info", "text"},
		{"text debug", "debug", "text"},
		{"invalid format", "info", "invalid"},
		{"invalid level", "invalid", "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			log := Init(tt.level, tt.format)
			if log == nil {
				t.Error("Init() returned nil logger")
			}
		})
	}
}

func TestLoggerWithJSONFormat(t *testing.T) {
	// Capture stdout
	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)
	log.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("Expected JSON format with 'msg' field, got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("Expected key-value pair in JSON, got: %s", output)
	}
}

func TestLoggerWithTextFormat(t *testing.T) {
	// Capture stdout
	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "text",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)
	log.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Expected key=value in output, got: %s", output)
	}
}

func TestLoggerWithContext(t *testing.T) {
	ctx := context.Background()
	id := GenerateCorrelationID()
	ctx = NewContextWithCorrelationID(ctx, id)

	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)
	logWithContext := log.WithContext(ctx)
	logWithContext.Info("test message")

	output := buf.String()
	if !strings.Contains(output, id) {
		t.Errorf("Expected correlation ID %q in output, got: %s", id, output)
	}
}

func TestContextLoggingFunctions(t *testing.T) {
	ctx := context.Background()
	id := GenerateCorrelationID()
	ctx = NewContextWithCorrelationID(ctx, id)

	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)
	SetDefault(log)

	// Test all context logging functions
	InfoCtx(ctx, "info message", "key", "value")
	DebugCtx(ctx, "debug message")
	WarnCtx(ctx, "warn message")
	ErrorCtx(ctx, "error message")

	output := buf.String()
	if !strings.Contains(output, id) {
		t.Errorf("Expected correlation ID %q in output, got: %s", id, output)
	}
	if !strings.Contains(output, "info message") {
		t.Errorf("Expected 'info message' in output, got: %s", output)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name          string
		level         slog.Leveler
		expectedDebug bool
		expectedInfo  bool
		expectedWarn  bool
		expectedError bool
	}{
		{
			name:          "debug level",
			level:         slog.LevelDebug,
			expectedDebug: true,
			expectedInfo:  true,
			expectedWarn:  true,
			expectedError: true,
		},
		{
			name:          "info level",
			level:         slog.LevelInfo,
			expectedDebug: false,
			expectedInfo:  true,
			expectedWarn:  true,
			expectedError: true,
		},
		{
			name:          "warn level",
			level:         slog.LevelWarn,
			expectedDebug: false,
			expectedInfo:  false,
			expectedWarn:  true,
			expectedError: true,
		},
		{
			name:          "error level",
			level:         slog.LevelError,
			expectedDebug: false,
			expectedInfo:  false,
			expectedWarn:  false,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := Config{
				Level:       tt.level,
				Format:      "text",
				Output:      &buf,
				ErrorOutput: &buf,
			}

			log := New(cfg)

			log.Debug("debug")
			log.Info("info")
			log.Warn("warn")
			log.Error("error")

			output := buf.String()

			if tt.expectedDebug && !strings.Contains(output, "debug") {
				t.Error("Expected debug message")
			}
			if !tt.expectedDebug && strings.Contains(output, "debug") {
				t.Error("Did not expect debug message")
			}

			if tt.expectedInfo && !strings.Contains(output, "info") {
				t.Error("Expected info message")
			}
			if !tt.expectedInfo && strings.Contains(output, "info") {
				t.Error("Did not expect info message")
			}

			if tt.expectedWarn && !strings.Contains(output, "warn") {
				t.Error("Expected warn message")
			}
			if !tt.expectedWarn && strings.Contains(output, "warn") {
				t.Error("Did not expect warn message")
			}

			if tt.expectedError && !strings.Contains(output, "error") {
				t.Error("Expected error message")
			}
			if !tt.expectedError && strings.Contains(output, "error") {
				t.Error("Did not expect error message")
			}
		})
	}
}

func TestLevelWriterHandler_Routing(t *testing.T) {
	var stdoutBuf, stderrBuf bytes.Buffer

	// Create a logger with the custom handler
	log := New(Config{
		Level:       slog.LevelInfo,
		Format:      "text",
		Output:      &stdoutBuf,
		ErrorOutput: &stderrBuf,
	})

	log.Info("info message")
	log.Warn("warn message")
	log.Error("error message")

	stdoutOutput := stdoutBuf.String()
	stderrOutput := stderrBuf.String()

	// Info should go to stdout
	if !strings.Contains(stdoutOutput, "info message") {
		t.Errorf("Expected 'info message' in stdout, got: %s", stdoutOutput)
	}

	// Warn should go to stderr
	if !strings.Contains(stderrOutput, "warn message") {
		t.Errorf("Expected 'warn message' in stderr, got: %s", stderrOutput)
	}

	// Error should go to stderr
	if !strings.Contains(stderrOutput, "error message") {
		t.Errorf("Expected 'error message' in stderr, got: %s", stderrOutput)
	}
}

func TestVeryLongLogMessages(t *testing.T) {
	longMessage := strings.Repeat("a", 10000)

	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "text",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)

	// Should not panic
	log.Info(longMessage)

	output := buf.String()
	if !strings.Contains(output, longMessage) {
		t.Error("Very long message not in output")
	}
}

func TestConcurrentLogWrites(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)

	done := make(chan bool)

	// Write logs concurrently
	for i := 0; i < 100; i++ {
		go func(n int) {
			log.Info("message", "count", n)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should not panic
}

func TestLoggingNilValues(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)

	// Should not panic
	log.Info("test", "nil", nil)
	log.Info("test", "empty", "")

	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Error("Expected 'test' in output")
	}
}

func TestLoggingUnicodeAndEmoji(t *testing.T) {
	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "json",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)

	// Should handle unicode and emoji
	log.Info("test 🚀", "emoji", "😀", "unicode", "日本語")

	output := buf.String()
	if !strings.Contains(output, "🚀") {
		t.Error("Expected emoji in output")
	}
}

func TestInvalidCorrelationIDInContext(t *testing.T) {
	ctx := context.Background()

	// Manually set invalid type in context
	ctx = context.WithValue(ctx, CorrelationIDKey, 123)

	var buf bytes.Buffer
	cfg := Config{
		Level:       slog.LevelInfo,
		Format:      "text",
		Output:      &buf,
		ErrorOutput: &buf,
	}

	log := New(cfg)

	// Should not panic
	logWithContext := log.WithContext(ctx)
	logWithContext.Info("test message")

	// Should still work, just without correlation ID
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Expected message in output")
	}
}
