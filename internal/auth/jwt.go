package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tediscript/gostarterkit/internal/config"
)

var (
	// cfg holds the application configuration
	cfg *config.Config

	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken is returned when a token is expired
	ErrExpiredToken = errors.New("token expired")
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// InitializeJWT initializes the JWT authentication system
func InitializeJWT(c *config.Config) {
	cfg = c

	// Validate JWT signing secret
	if cfg.JWT.SigningSecret == "" {
		if cfg.App.Env == "production" {
			panic("JWT_SIGNING_SECRET or JWT_SIGNING_SECRET_FILE is required in production")
		}
		// For development, use a temporary secret
		cfg.JWT.SigningSecret = "development-jwt-secret-change-in-production"
	}
}

// SetConfigForTesting sets the config for testing purposes
// This is exported for testing only
func SetConfigForTesting(c *config.Config) {
	cfg = c
}

// ResetConfigForTesting resets the config to nil for testing
func ResetConfigForTesting() {
	cfg = nil
}

// GenerateToken generates a JWT token for the given user ID
func GenerateToken(userID string) (string, error) {
	if cfg == nil {
		return "", errors.New("JWT not initialized")
	}

	// Create claims with user ID and expiration time
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWT.ExpirationSeconds) * time.Second)),
			Issuer:    "gostarterkit",
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(cfg.JWT.SigningSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the user ID
func ValidateToken(tokenString string) (string, error) {
	if cfg == nil {
		return "", errors.New("JWT not initialized")
	}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return signing secret
		return []byte(cfg.JWT.SigningSecret), nil
	})

	if err != nil {
		// Check if error is due to expired token
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}

	// Extract claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, nil
	}

	return "", ErrInvalidToken
}

// GetExpirationSeconds returns the token expiration time in seconds
func GetExpirationSeconds() int {
	if cfg == nil {
		return 3600 // Default
	}
	return cfg.JWT.ExpirationSeconds
}
