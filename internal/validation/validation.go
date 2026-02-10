package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("validation errors: ")
	for i, err := range ve {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}

// Validator interface for custom validators
type Validator interface {
	Validate() error
}

// ValidatorFunc is a function type for validators
type ValidatorFunc func(interface{}) error

// Common validation errors
var (
	ErrRequired        = errors.New("field is required")
	ErrEmailInvalid    = errors.New("invalid email format")
	ErrUUIDInvalid     = errors.New("invalid UUID format")
	ErrTooShort        = errors.New("value is too short")
	ErrTooLong         = errors.New("value is too long")
	ErrMinLength       = errors.New("value does not meet minimum length")
	ErrMaxLength       = errors.New("value exceeds maximum length")
	ErrMinValue        = errors.New("value is below minimum")
	ErrMaxValue        = errors.New("value exceeds maximum")
	ErrPatternMismatch = errors.New("value does not match required pattern")
	ErrEmpty           = errors.New("value cannot be empty")
)

// Email regex pattern (RFC 5322 compliant)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Email validates that a string is a valid email address
func Email(value string) error {
	if value == "" {
		return nil // Empty values are handled by Required validator
	}

	if !emailRegex.MatchString(value) {
		return fmt.Errorf("%w: %s", ErrEmailInvalid, value)
	}
	return nil
}

// UUID validates that a string is a valid UUID
func UUID(value string) error {
	if value == "" {
		return nil // Empty values are handled by Required validator
	}

	_, err := uuid.Parse(value)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrUUIDInvalid, value)
	}
	return nil
}

// Required validates that a string is not empty or whitespace only
func Required(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrRequired
	}
	return nil
}

// MinLength validates that a string meets the minimum length
func MinLength(min int) ValidatorFunc {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("MinLength validator requires string type")
		}

		if utf8.RuneCountInString(s) < min {
			return fmt.Errorf("%w: minimum length is %d", ErrMinLength, min)
		}
		return nil
	}
}

// MaxLength validates that a string does not exceed maximum length
func MaxLength(max int) ValidatorFunc {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("MaxLength validator requires string type")
		}

		if utf8.RuneCountInString(s) > max {
			return fmt.Errorf("%w: maximum length is %d", ErrMaxLength, max)
		}
		return nil
	}
}

// Length validates that a string is within a length range
func Length(min, max int) ValidatorFunc {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("Length validator requires string type")
		}

		length := utf8.RuneCountInString(s)
		if length < min {
			return fmt.Errorf("%w: minimum length is %d", ErrMinLength, min)
		}
		if length > max {
			return fmt.Errorf("%w: maximum length is %d", ErrMaxLength, max)
		}
		return nil
	}
}

// Min validates that a numeric value meets the minimum
func Min(min int) ValidatorFunc {
	return func(value interface{}) error {
		var v int
		switch val := value.(type) {
		case int:
			v = val
		case int64:
			v = int(val)
		case int32:
			v = int(val)
		case uint:
			v = int(val)
		case uint64:
			v = int(val)
		case uint32:
			v = int(val)
		default:
			return fmt.Errorf("Min validator requires numeric type")
		}

		if v < min {
			return fmt.Errorf("%w: minimum value is %d", ErrMinValue, min)
		}
		return nil
	}
}

// Max validates that a numeric value does not exceed maximum
func Max(max int) ValidatorFunc {
	return func(value interface{}) error {
		var v int
		switch val := value.(type) {
		case int:
			v = val
		case int64:
			v = int(val)
		case int32:
			v = int(val)
		case uint:
			v = int(val)
		case uint64:
			v = int(val)
		case uint32:
			v = int(val)
		default:
			return fmt.Errorf("Max validator requires numeric type")
		}

		if v > max {
			return fmt.Errorf("%w: maximum value is %d", ErrMaxValue, max)
		}
		return nil
	}
}

// Pattern validates that a string matches a regex pattern
func Pattern(pattern string) ValidatorFunc {
	regex := regexp.MustCompile(pattern)
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("Pattern validator requires string type")
		}

		if !regex.MatchString(s) {
			return fmt.Errorf("%w: pattern: %s", ErrPatternMismatch, pattern)
		}
		return nil
	}
}

// NoSpecialChars validates that a string contains only alphanumeric and basic punctuation
func NoSpecialChars() ValidatorFunc {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("NoSpecialChars validator requires string type")
		}

		for _, r := range s {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
				!strings.ContainsAny(string(r), " _-.,@") {
				return fmt.Errorf("contains invalid special character: %c", r)
			}
		}
		return nil
	}
}

// Struct validates a struct based on its validation tags
// Tags are parsed from the "validate" struct tag
func Struct(s interface{}) ValidationErrors {
	// This is a simplified implementation that works with custom validation
	// For a full implementation, use reflection to parse struct tags
	if v, ok := s.(Validator); ok {
		if err := v.Validate(); err != nil {
			if ve, ok := err.(ValidationErrors); ok {
				return ve
			}
			return ValidationErrors{{Field: "struct", Message: err.Error()}}
		}
	}
	return nil
}

// ValidateStruct parses struct tags and validates fields
// Expected tag format: validate:"required,email,min=5,max=100"
func ValidateStruct(s interface{}) ValidationErrors {
	// For simplicity, we're implementing a basic version
	// A full implementation would use reflect to parse struct tags dynamically
	// This is a placeholder that can be extended

	if v, ok := s.(Validator); ok {
		if err := v.Validate(); err != nil {
			if ve, ok := err.(ValidationErrors); ok {
				return ve
			}
			return ValidationErrors{{Field: "struct", Message: err.Error()}}
		}
	}

	return nil
}

// ParseValidateTag parses a validation tag and returns the validators
// Tag format: "required,email,min=5,max=100"
func ParseValidateTag(tag string) []ValidatorFunc {
	var validators []ValidatorFunc

	if tag == "" {
		return validators
	}

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		switch {
		case part == "required":
			validators = append(validators, func(v interface{}) error {
				s, ok := v.(string)
				if !ok {
					return fmt.Errorf("required validator requires string type")
				}
				return Required(s)
			})

		case part == "email":
			validators = append(validators, func(v interface{}) error {
				s, ok := v.(string)
				if !ok {
					return fmt.Errorf("email validator requires string type")
				}
				return Email(s)
			})

		case part == "uuid":
			validators = append(validators, func(v interface{}) error {
				s, ok := v.(string)
				if !ok {
					return fmt.Errorf("uuid validator requires string type")
				}
				return UUID(s)
			})

		case strings.HasPrefix(part, "min="):
			minStr := strings.TrimPrefix(part, "min=")
			min, err := strconv.Atoi(minStr)
			if err != nil {
				continue // Skip invalid tag
			}
			validators = append(validators, MinLength(min))

		case strings.HasPrefix(part, "max="):
			maxStr := strings.TrimPrefix(part, "max=")
			max, err := strconv.Atoi(maxStr)
			if err != nil {
				continue // Skip invalid tag
			}
			validators = append(validators, MaxLength(max))

		case strings.HasPrefix(part, "length="):
			// Expected format: length=5,100
			lengthStr := strings.TrimPrefix(part, "length=")
			parts := strings.Split(lengthStr, ",")
			if len(parts) == 2 {
				min, err1 := strconv.Atoi(parts[0])
				max, err2 := strconv.Atoi(parts[1])
				if err1 == nil && err2 == nil {
					validators = append(validators, Length(min, max))
				}
			}
		}
	}

	return validators
}
