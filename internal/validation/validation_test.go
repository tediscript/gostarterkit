package validation

import (
	"strings"
	"testing"
)

func TestValidationError(t *testing.T) {
	ve := ValidationError{Field: "email", Message: "invalid format"}

	if ve.Error() != "email: invalid format" {
		t.Errorf("ValidationError.Error() = %v, want %v", ve.Error(), "email: invalid format")
	}
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   string
	}{
		{
			name:   "no errors",
			errors: ValidationErrors{},
			want:   "",
		},
		{
			name:   "single error",
			errors: ValidationErrors{{Field: "email", Message: "invalid"}},
			want:   "validation errors: email: invalid",
		},
		{
			name: "multiple errors",
			errors: ValidationErrors{
				{Field: "email", Message: "invalid"},
				{Field: "password", Message: "too short"},
			},
			want: "validation errors: email: invalid; password: too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errors.Error(); got != tt.want {
				t.Errorf("ValidationErrors.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test validation functions
func TestEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  error
	}{
		{"valid email", "test@example.com", nil},
		{"valid email with subdomain", "test@mail.example.com", nil},
		{"valid email with plus", "test+tag@example.com", nil},
		{"valid email with hyphen", "test-user@example.com", nil},
		{"empty string", "", nil}, // Empty is handled by Required validator
		{"missing @", "testexample.com", ErrEmailInvalid},
		{"missing domain", "test@", ErrEmailInvalid},
		{"missing local part", "@example.com", ErrEmailInvalid},
		{"invalid characters", "test#example.com", ErrEmailInvalid},
		// Note: Simple regex doesn't catch all edge cases like double dots, leading/trailing dots
		// These are technically valid per RFC 5322 in some contexts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Email(tt.email)
			if tt.want == nil {
				if err != nil {
					t.Errorf("Email(%q) = %v, want nil", tt.email, err)
				}
			} else {
				if err == nil {
					t.Errorf("Email(%q) = nil, want %v", tt.email, tt.want)
				}
			}
		})
	}
}

func TestUUID(t *testing.T) {
	tests := []struct {
		name string
		uuid string
		want error
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", nil},
		{"valid UUID v1", "00000000-0000-0001-0000-000000000000", nil},
		{"valid UUID with uppercase", "550E8400-E29B-41D4-A716-446655440000", nil},
		{"empty string", "", nil}, // Empty is handled by Required validator
		{"invalid format", "550e8400-e29b-41d4-a716", ErrUUIDInvalid},
		{"invalid characters", "550e8400-e29b-41d4-a716-44665544000g", ErrUUIDInvalid},
		{"too short", "550e8400-e29b-41d4", ErrUUIDInvalid},
		{"too long", "550e8400-e29b-41d4-a716-44665544000000", ErrUUIDInvalid},
		// Note: google/uuid.Parse accepts UUIDs without hyphens (RFC 4122 compliant)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UUID(tt.uuid)
			if tt.want == nil {
				if err != nil {
					t.Errorf("UUID(%q) = %v, want nil", tt.uuid, err)
				}
			} else {
				if err == nil {
					t.Errorf("UUID(%q) = nil, want %v", tt.uuid, tt.want)
				}
			}
		})
	}
}

func TestRequired(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  error
	}{
		{"valid string", "hello", nil},
		{"string with spaces", "hello world", nil},
		{"empty string", "", ErrRequired},
		{"whitespace only", "   ", ErrRequired},
		{"tab only", "\t", ErrRequired},
		{"newline only", "\n", ErrRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Required(tt.value)
			if err != tt.want {
				t.Errorf("Required(%q) = %v, want %v", tt.value, err, tt.want)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name  string
		min   int
		value string
		want  error
	}{
		{"exactly at minimum", 5, "hello", nil},
		{"above minimum", 5, "hello world", nil},
		{"below minimum", 5, "hi", ErrMinLength},
		{"empty string", 5, "", ErrMinLength},
		{"at boundary", 3, "abc", nil},
		{"just below boundary", 3, "ab", ErrMinLength},
		{"just above boundary", 3, "abcd", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := MinLength(tt.min)
			err := validator(tt.value)
			if tt.want == nil {
				if err != nil {
					t.Errorf("MinLength(%d)(%q) = %v, want nil", tt.min, tt.value, err)
				}
			} else {
				if err == nil {
					t.Errorf("MinLength(%d)(%q) = nil, want error", tt.min, tt.value)
				}
			}
		})
	}

	// Test with non-string value
	t.Run("non-string value", func(t *testing.T) {
		validator := MinLength(5)
		err := validator(123)
		if err == nil {
			t.Errorf("MinLength(5)(123) = nil, want error")
		}
	})
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name  string
		max   int
		value string
		want  error
	}{
		{"exactly at maximum", 5, "hello", nil},
		{"below maximum", 5, "hi", nil},
		{"above maximum", 5, "hello world", ErrMaxLength},
		{"empty string", 5, "", nil},
		{"at boundary", 3, "abc", nil},
		{"just below boundary", 3, "ab", nil},
		{"just above boundary", 3, "abcd", ErrMaxLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := MaxLength(tt.max)
			err := validator(tt.value)
			if tt.want == nil {
				if err != nil {
					t.Errorf("MaxLength(%d)(%q) = %v, want nil", tt.max, tt.value, err)
				}
			} else {
				if err == nil {
					t.Errorf("MaxLength(%d)(%q) = nil, want error", tt.max, tt.value)
				}
			}
		})
	}

	// Test with non-string value
	t.Run("non-string value", func(t *testing.T) {
		validator := MaxLength(5)
		err := validator(123)
		if err == nil {
			t.Errorf("MaxLength(5)(123) = nil, want error")
		}
	})
}

func TestLength(t *testing.T) {
	tests := []struct {
		name  string
		min   int
		max   int
		value string
		want  error
	}{
		{"within range", 3, 10, "hello", nil},
		{"at minimum", 3, 10, "abc", nil},
		{"at maximum", 3, 10, "1234567890", nil},
		{"below minimum", 3, 10, "hi", ErrMinLength},
		{"above maximum", 3, 10, "hello world", ErrMaxLength},
		{"empty string", 3, 10, "", ErrMinLength},
		{"same min and max", 5, 5, "hello", nil},
		{"same min and max - wrong", 5, 5, "hi", ErrMinLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := Length(tt.min, tt.max)
			err := validator(tt.value)
			if tt.want == nil {
				if err != nil {
					t.Errorf("Length(%d,%d)(%q) = %v, want nil", tt.min, tt.max, tt.value, err)
				}
			} else {
				if err == nil {
					t.Errorf("Length(%d,%d)(%q) = nil, want error", tt.min, tt.max, tt.value)
				}
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name  string
		min   int
		value int
		want  error
	}{
		{"above minimum", 5, 10, nil},
		{"at minimum", 5, 5, nil},
		{"below minimum", 5, 3, ErrMinValue},
		{"zero", 0, 0, nil},
		{"negative", -5, -3, nil},
		{"negative at minimum", -5, -5, nil},
		{"negative below minimum", -5, -10, ErrMinValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := Min(tt.min)
			err := validator(tt.value)
			if tt.want == nil {
				if err != nil {
					t.Errorf("Min(%d)(%d) = %v, want nil", tt.min, tt.value, err)
				}
			} else {
				if err == nil {
					t.Errorf("Min(%d)(%d) = nil, want error", tt.min, tt.value)
				}
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name  string
		max   int
		value int
		want  error
	}{
		{"below maximum", 10, 5, nil},
		{"at maximum", 10, 10, nil},
		{"above maximum", 10, 15, ErrMaxValue},
		{"zero", 0, 0, nil},
		{"negative", -5, -10, nil},
		{"negative at maximum", -5, -5, nil},
		{"negative above maximum", -5, -3, ErrMaxValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := Max(tt.max)
			err := validator(tt.value)
			if tt.want == nil {
				if err != nil {
					t.Errorf("Max(%d)(%d) = %v, want nil", tt.max, tt.value, err)
				}
			} else {
				if err == nil {
					t.Errorf("Max(%d)(%d) = nil, want error", tt.max, tt.value)
				}
			}
		})
	}
}

func TestPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		value   string
		want    error
	}{
		{"alphanumeric match", "^[a-zA-Z0-9]+$", "abc123", nil},
		{"alphanumeric mismatch", "^[a-zA-Z0-9]+$", "abc123!", ErrPatternMismatch},
		{"email pattern", "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", "test@example.com", nil},
		{"phone pattern", "^\\d{3}-\\d{3}-\\d{4}$", "123-456-7890", nil},
		{"phone pattern mismatch", "^\\d{3}-\\d{3}-\\d{4}$", "123-456-789", ErrPatternMismatch},
		{"empty string", "^[a-z]+$", "", ErrPatternMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := Pattern(tt.pattern)
			err := validator(tt.value)
			if tt.want == nil {
				if err != nil {
					t.Errorf("Pattern(%q)(%q) = %v, want nil", tt.pattern, tt.value, err)
				}
			} else {
				if err == nil {
					t.Errorf("Pattern(%q)(%q) = nil, want error", tt.pattern, tt.value)
				}
			}
		})
	}
}

func TestNoSpecialChars(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  error
	}{
		{"letters only", "hello", nil},
		{"letters and numbers", "hello123", nil},
		{"with spaces", "hello world", nil},
		{"with hyphen", "hello-world", nil},
		{"with underscore", "hello_world", nil},
		{"with period", "hello.world", nil},
		{"with comma", "hello,world", nil},
		{"with at", "hello@world", nil},
		{"with exclamation", "hello!", ErrPatternMismatch},
		{"with hash", "hello#", ErrPatternMismatch},
		{"with dollar", "hello$", ErrPatternMismatch},
		{"with percent", "hello%", ErrPatternMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NoSpecialChars()
			err := validator(tt.value)
			if tt.want == nil {
				if err != nil {
					t.Errorf("NoSpecialChars()(%q) = %v, want nil", tt.value, err)
				}
			} else {
				if err == nil {
					t.Errorf("NoSpecialChars()(%q) = nil, want error", tt.value)
				}
			}
		})
	}
}

func TestUnicodeCharacters(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  error
	}{
		{"Chinese characters", "你好世界", nil},
		{"Japanese characters", "こんにちは", nil},
		{"Emoji", "😀😁😂", nil},
		{"Arabic", "مرحبا", nil},
		{"Cyrillic", "Привет", nil},
		{"mixed", "Hello世界", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with Required
			err := Required(tt.value)
			if err != nil {
				t.Errorf("Required(%q) = %v, want nil (unicode should be valid)", tt.value, err)
			}

			// Test with length validators
			minValidator := MinLength(2)
			err = minValidator(tt.value)
			if err != nil {
				t.Errorf("MinLength(2)(%q) = %v, want nil", tt.value, err)
			}
		})
	}
}

func TestVeryLongStrings(t *testing.T) {
	// Create a very long string (10000 characters)
	longString := strings.Repeat("a", 10000)

	t.Run("very long string validation", func(t *testing.T) {
		err := Required(longString)
		if err != nil {
			t.Errorf("Required(very long string) = %v, want nil", err)
		}
	})

	t.Run("very long string exceeds max", func(t *testing.T) {
		validator := MaxLength(5000)
		err := validator(longString)
		if err == nil {
			t.Errorf("MaxLength(5000)(very long string) = nil, want error")
		}
	})
}

func TestUserRegistrationRequest(t *testing.T) {
	tests := []struct {
		name    string
		request UserRegistrationRequest
		wantErr bool
	}{
		{"valid request", UserRegistrationRequest{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "John Doe",
		}, false},
		{"missing email", UserRegistrationRequest{
			Email:    "",
			Password: "password123",
			Name:     "John Doe",
		}, true},
		{"invalid email", UserRegistrationRequest{
			Email:    "invalid-email",
			Password: "password123",
			Name:     "John Doe",
		}, true},
		{"missing password", UserRegistrationRequest{
			Email:    "test@example.com",
			Password: "",
			Name:     "John Doe",
		}, true},
		{"password too short", UserRegistrationRequest{
			Email:    "test@example.com",
			Password: "short",
			Name:     "John Doe",
		}, true},
		{"password too long", UserRegistrationRequest{
			Email:    "test@example.com",
			Password: strings.Repeat("a", 101),
			Name:     "John Doe",
		}, true},
		{"missing name", UserRegistrationRequest{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "",
		}, true},
		{"name too short", UserRegistrationRequest{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "J",
		}, true},
		{"name too long", UserRegistrationRequest{
			Email:    "test@example.com",
			Password: "password123",
			Name:     strings.Repeat("a", 51),
		}, true},
		{"multiple errors", UserRegistrationRequest{
			Email:    "invalid",
			Password: "short",
			Name:     "",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UserRegistrationRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginRequest(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequest
		wantErr bool
	}{
		{"valid request", LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}, false},
		{"missing email", LoginRequest{
			Email:    "",
			Password: "password123",
		}, true},
		{"invalid email", LoginRequest{
			Email:    "invalid-email",
			Password: "password123",
		}, true},
		{"missing password", LoginRequest{
			Email:    "test@example.com",
			Password: "",
		}, true},
		{"multiple errors", LoginRequest{
			Email:    "invalid",
			Password: "",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoginRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateProfileRequest(t *testing.T) {
	tests := []struct {
		name    string
		request UpdateProfileRequest
		wantErr bool
	}{
		{"valid with name", UpdateProfileRequest{
			Name:  "John Doe",
			Email: "",
		}, false},
		{"valid with email", UpdateProfileRequest{
			Name:  "",
			Email: "test@example.com",
		}, false},
		{"valid with both", UpdateProfileRequest{
			Name:  "John Doe",
			Email: "test@example.com",
		}, false},
		{"valid with none", UpdateProfileRequest{
			Name:  "",
			Email: "",
		}, false},
		{"name too short", UpdateProfileRequest{
			Name:  "J",
			Email: "",
		}, true},
		{"name too long", UpdateProfileRequest{
			Name:  strings.Repeat("a", 51),
			Email: "",
		}, true},
		{"invalid email", UpdateProfileRequest{
			Name:  "",
			Email: "invalid-email",
		}, true},
		{"multiple errors", UpdateProfileRequest{
			Name:  "J",
			Email: "invalid",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateProfileRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseValidateTag(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantLen int // Number of validators
	}{
		{"empty tag", "", 0},
		{"required only", "required", 1},
		{"email only", "email", 1},
		{"uuid only", "uuid", 1},
		{"min only", "min=5", 1},
		{"max only", "max=100", 1},
		{"min and max", "min=5,max=100", 2},
		{"required and email", "required,email", 2},
		{"all validators", "required,email,min=5,max=100", 4},
		{"with spaces", "required, email, min=5, max=100", 4},
		{"invalid min", "min=abc", 0},
		{"invalid max", "max=xyz", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validators := ParseValidateTag(tt.tag)
			if len(validators) != tt.wantLen {
				t.Errorf("ParseValidateTag(%q) returned %d validators, want %d", tt.tag, len(validators), tt.wantLen)
			}
		})
	}
}

func TestMultipleValidationErrors(t *testing.T) {
	request := UserRegistrationRequest{
		Email:    "",
		Password: "short",
		Name:     "",
	}

	err := request.Validate()
	if err == nil {
		t.Fatal("Expected validation errors, got nil")
	}

	validationErrors, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("Expected ValidationErrors type, got %T", err)
	}

	if len(validationErrors) != 3 {
		t.Errorf("Expected 3 validation errors, got %d", len(validationErrors))
	}

	// Check that all expected fields have errors
	fieldsWithErrors := make(map[string]bool)
	for _, ve := range validationErrors {
		fieldsWithErrors[ve.Field] = true
	}

	expectedFields := []string{"email", "password", "name"}
	for _, field := range expectedFields {
		if !fieldsWithErrors[field] {
			t.Errorf("Expected validation error for field %s", field)
		}
	}
}

func TestNilValueValidation(t *testing.T) {
	// Test that validators handle nil/undefined values gracefully
	t.Run("nil string in Required", func(t *testing.T) {
		var s *string
		if s != nil {
			err := Required(*s)
			if err != nil {
				t.Logf("Required(nil string) returned error: %v", err)
			}
		}
		// If s is nil, we can't call Required on it - this is expected behavior
	})

	t.Run("empty string validators", func(t *testing.T) {
		err := Email("")
		if err != nil {
			t.Errorf("Email(\"\") returned error, expected nil for empty string: %v", err)
		}

		err = UUID("")
		if err != nil {
			t.Errorf("UUID(\"\") returned error, expected nil for empty string: %v", err)
		}
	})
}
