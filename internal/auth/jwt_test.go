package auth

import (
	"testing"
	"time"

	"github.com/tediscript/gostarterkit/internal/config"
	"github.com/tediscript/gostarterkit/internal/logger"
)

// Test setup helper
func setupJWT(t *testing.T) *config.Config {
	t.Helper()

	cfg := &config.Config{}
	cfg.App.Env = "test"
	cfg.JWT.SigningSecret = "test-secret-key-for-testing"
	cfg.JWT.ExpirationSeconds = 3600

	SetConfigForTesting(cfg)
	return cfg
}

func TestInitializeJWT(t *testing.T) {
	t.Run("with valid secret", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.App.Env = "test"
		cfg.JWT.SigningSecret = "test-secret"

		InitializeJWT(cfg)
		// Should not panic
	})

	t.Run("without secret in development", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("InitializeJWT should not panic in development, got: %v", r)
			}
		}()

		cfg := &config.Config{}
		cfg.App.Env = "development"
		cfg.JWT.SigningSecret = ""

		InitializeJWT(cfg)
	})

	t.Run("without secret in production", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("InitializeJWT should panic in production without secret")
			}
		}()

		cfg := &config.Config{}
		cfg.App.Env = "production"
		cfg.JWT.SigningSecret = ""

		InitializeJWT(cfg)
	})
}

func TestGenerateToken(t *testing.T) {
	_ = setupJWT(t)

	t.Run("valid token generation", func(t *testing.T) {
		userID := "testuser123"
		token, err := GenerateToken(userID)

		if err != nil {
			t.Errorf("GenerateToken() error = %v", err)
			return
		}

		if token == "" {
			t.Error("GenerateToken() returned empty token")
		}
	})

	t.Run("token with HS256 algorithm", func(t *testing.T) {
		userID := "testuser456"
		token, err := GenerateToken(userID)

		if err != nil {
			t.Errorf("GenerateToken() error = %v", err)
			return
		}

		// Validate the token was generated correctly
		_, err = ValidateToken(token)
		if err != nil {
			t.Errorf("Generated token is invalid: %v", err)
		}
	})

	t.Run("not initialized", func(t *testing.T) {
		// Reset config
		ResetConfigForTesting()

		_, err := GenerateToken("testuser")
		if err == nil {
			t.Error("GenerateToken() should return error when not initialized")
		}

		// Reinitialize for other tests
		setupJWT(t)
	})
}

func TestValidateToken(t *testing.T) {
	_ = setupJWT(t)

	t.Run("valid token", func(t *testing.T) {
		userID := "validuser"
		token, err := GenerateToken(userID)

		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		extractedUserID, err := ValidateToken(token)
		if err != nil {
			t.Errorf("ValidateToken() error = %v", err)
			return
		}

		if extractedUserID != userID {
			t.Errorf("ValidateToken() userID = %v, want %v", extractedUserID, userID)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Create a token with 0 expiration
		testCfg := setupJWT(t)
		testCfg.JWT.ExpirationSeconds = 0
		SetConfigForTesting(testCfg)
		token, err := GenerateToken("expireduser")

		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		// Wait a moment to ensure token is considered expired
		time.Sleep(10 * time.Millisecond)

		_, err = ValidateToken(token)
		if err != ErrExpiredToken {
			t.Errorf("ValidateToken() error = %v, want ErrExpiredToken", err)
		}

		// Restore expiration
		testCfg.JWT.ExpirationSeconds = 3600
		SetConfigForTesting(testCfg)
	})

	t.Run("malformed token", func(t *testing.T) {
		malformedTokens := []string{
			"",
			"not.a.valid.jwt",
			"invalid.token.string",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid",
		}

		for _, token := range malformedTokens {
			_, err := ValidateToken(token)
			if err == nil {
				t.Errorf("ValidateToken() should return error for malformed token: %s", token)
			}
		}
	})

	t.Run("token with wrong signature", func(t *testing.T) {
		// Generate token with one secret
		testCfg := setupJWT(t)
		testCfg.JWT.SigningSecret = "secret1"
		SetConfigForTesting(testCfg)
		token, err := GenerateToken("wrongsiguser")

		if err != nil {
			t.Fatalf("GenerateToken() error = %v", err)
		}

		// Try to validate with different secret
		testCfg.JWT.SigningSecret = "secret2"
		SetConfigForTesting(testCfg)

		_, err = ValidateToken(token)
		if err == nil {
			t.Error("ValidateToken() should return error for token with wrong signature")
		}

		// Restore secret
		testCfg.JWT.SigningSecret = "test-secret-key-for-testing"
		SetConfigForTesting(testCfg)
	})

	t.Run("not initialized", func(t *testing.T) {
		// Reset config
		ResetConfigForTesting()

		_, err := ValidateToken("sometoken")
		if err == nil {
			t.Error("ValidateToken() should return error when not initialized")
		}

		// Reinitialize for other tests
		setupJWT(t)
	})
}

func TestGetExpirationSeconds(t *testing.T) {
	_ = setupJWT(t)

	t.Run("returns configured expiration", func(t *testing.T) {
		testCfg := setupJWT(t)
		testCfg.JWT.ExpirationSeconds = 7200
		SetConfigForTesting(testCfg)
		expiration := GetExpirationSeconds()

		if expiration != 7200 {
			t.Errorf("GetExpirationSeconds() = %v, want 7200", expiration)
		}
	})

	t.Run("returns default when not initialized", func(t *testing.T) {
		// Reset config
		ResetConfigForTesting()
		expiration := GetExpirationSeconds()

		if expiration != 3600 {
			t.Errorf("GetExpirationSeconds() = %v, want 3600", expiration)
		}

		// Reinitialize for other tests
		setupJWT(t)
	})
}

func TestJWTEdgeCases(t *testing.T) {
	_ = setupJWT(t)

	t.Run("token with very large payload", func(t *testing.T) {
		// Create a very large user ID
		largeUserID := string(make([]byte, 10000))
		for i := range largeUserID {
			largeUserID = largeUserID[:i] + "a" + largeUserID[i+1:]
		}

		token, err := GenerateToken(largeUserID)
		if err != nil {
			t.Errorf("GenerateToken() with large payload error = %v", err)
			return
		}

		extractedUserID, err := ValidateToken(token)
		if err != nil {
			t.Errorf("ValidateToken() with large payload error = %v", err)
			return
		}

		if extractedUserID != largeUserID {
			t.Error("Extracted userID doesn't match large payload")
		}
	})

	t.Run("token with unusual characters in userID", func(t *testing.T) {
		unusualUserIDs := []string{
			"user@domain.com",
			"user+tag",
			"user=123",
			"用户123",           // Chinese characters
			"ユーザー123",         // Japanese characters
			"utilisateur_123", // French
			"benutzer_123",    // German
			"u$er#123",
			"user/name",
			"user.name",
		}

		for _, userID := range unusualUserIDs {
			token, err := GenerateToken(userID)
			if err != nil {
				t.Errorf("GenerateToken() with unusual characters error = %v for userID: %s", err, userID)
				continue
			}

			extractedUserID, err := ValidateToken(token)
			if err != nil {
				t.Errorf("ValidateToken() with unusual characters error = %v for userID: %s", err, userID)
				continue
			}

			if extractedUserID != userID {
				t.Errorf("Extracted userID doesn't match for unusual characters: got %v, want %v", extractedUserID, userID)
			}
		}
	})

	t.Run("concurrent token generation and validation", func(t *testing.T) {
		concurrentOps := 100
		done := make(chan bool, concurrentOps)

		for i := 0; i < concurrentOps; i++ {
			go func(index int) {
				userID := "concurrentuser" + string(rune('0'+index%10))
				token, err := GenerateToken(userID)
				if err != nil {
					t.Errorf("Concurrent GenerateToken() error = %v", err)
					done <- false
					return
				}

				extractedUserID, err := ValidateToken(token)
				if err != nil {
					t.Errorf("Concurrent ValidateToken() error = %v", err)
					done <- false
					return
				}

				if extractedUserID != userID {
					t.Errorf("Concurrent token validation mismatch: got %v, want %v", extractedUserID, userID)
					done <- false
					return
				}

				done <- true
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < concurrentOps; i++ {
			if !<-done {
				t.Error("One or more concurrent operations failed")
				break
			}
		}
	})
}

func TestJWTWithLogger(t *testing.T) {
	// This test ensures JWT works with logger package
	_ = setupJWT(t)
	_ = logger.Init("debug", "text")

	t.Run("token generation with logger initialized", func(t *testing.T) {
		userID := "logtestuser"
		token, err := GenerateToken(userID)

		if err != nil {
			t.Errorf("GenerateToken() with logger error = %v", err)
			return
		}

		if token == "" {
			t.Error("GenerateToken() returned empty token with logger")
		}
	})
}
