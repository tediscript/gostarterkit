package validation

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ValidationErrorResponse is the response format for validation errors
type ValidationErrorResponse struct {
	Errors []ValidationError `json:"errors"`
}

// Middleware returns a middleware function that validates requests
// The middleware expects a ValidatorFunc to extract and validate the request data
func Middleware(validator func(*http.Request) (Validator, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the validator for this request
			v, err := validator(r)
			if err != nil {
				http.Error(w, "invalid request format", http.StatusBadRequest)
				return
			}

			// Validate the request
			if err := v.Validate(); err != nil {
				// Check if it's a ValidationErrors type
				var validationErrors ValidationErrors
				switch e := err.(type) {
				case ValidationErrors:
					validationErrors = e
				default:
					validationErrors = ValidationErrors{
						{Field: "request", Message: err.Error()},
					}
				}

				// Return validation errors as JSON
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)

				response := ValidationErrorResponse{Errors: validationErrors}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, "failed to encode validation errors", http.StatusInternalServerError)
				}
				return
			}

			// Validation passed, continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// JSONBodyValidator returns a validator function that decodes JSON request body
// Example usage: validation.Middleware(validation.JSONBodyValidator(&MyStruct{}))
func JSONBodyValidator(v Validator) func(*http.Request) (Validator, error) {
	return func(r *http.Request) (Validator, error) {
		// Only validate POST, PUT, PATCH requests
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
			return v, nil
		}

		// Check Content-Type header
		contentType := r.Header.Get("Content-Type")
		if contentType != "" && !strings.Contains(contentType, "application/json") {
			return nil, http.ErrNotSupported
		}

		// Decode JSON body
		if err := json.NewDecoder(r.Body).Decode(v); err != nil {
			return nil, err
		}

		return v, nil
	}
}

// QueryParamValidator returns a validator function that validates query parameters
// Example usage: validation.Middleware(validation.QueryParamValidator("email", validation.Email))
func QueryParamValidator(param string, validator func(string) error) func(*http.Request) (Validator, error) {
	return func(r *http.Request) (Validator, error) {
		// Create a simple validator struct
		return &QueryParamValidatorStruct{
			param:     param,
			validator: validator,
			request:   r,
		}, nil
	}
}

// QueryParamValidatorStruct is a validator for query parameters
type QueryParamValidatorStruct struct {
	param     string
	validator func(string) error
	request   *http.Request
}

// Validate implements the Validator interface
func (qv *QueryParamValidatorStruct) Validate() error {
	value := qv.request.URL.Query().Get(qv.param)
	return qv.validator(value)
}

// Field returns the field name for error messages
func (qv *QueryParamValidatorStruct) Field() string {
	return qv.param
}

// Example usage structs

// UserRegistrationRequest is an example request struct for user registration
type UserRegistrationRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	Name     string `json:"name" validate:"required,min=2,max=50"`
}

// Validate implements the Validator interface for UserRegistrationRequest
func (r *UserRegistrationRequest) Validate() error {
	var errors ValidationErrors

	// Validate Email
	if err := Required(r.Email); err != nil {
		errors = append(errors, ValidationError{Field: "email", Message: err.Error()})
	} else if err := Email(r.Email); err != nil {
		errors = append(errors, ValidationError{Field: "email", Message: err.Error()})
	}

	// Validate Password
	if err := Required(r.Password); err != nil {
		errors = append(errors, ValidationError{Field: "password", Message: err.Error()})
	} else if err := MinLength(8)(r.Password); err != nil {
		errors = append(errors, ValidationError{Field: "password", Message: err.Error()})
	} else if err := MaxLength(100)(r.Password); err != nil {
		errors = append(errors, ValidationError{Field: "password", Message: err.Error()})
	}

	// Validate Name
	if err := Required(r.Name); err != nil {
		errors = append(errors, ValidationError{Field: "name", Message: err.Error()})
	} else if err := Length(2, 50)(r.Name); err != nil {
		errors = append(errors, ValidationError{Field: "name", Message: err.Error()})
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// LoginRequest is an example request struct for login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// Validate implements the Validator interface for LoginRequest
func (r *LoginRequest) Validate() error {
	var errors ValidationErrors

	// Validate Email
	if err := Required(r.Email); err != nil {
		errors = append(errors, ValidationError{Field: "email", Message: err.Error()})
	} else if err := Email(r.Email); err != nil {
		errors = append(errors, ValidationError{Field: "email", Message: err.Error()})
	}

	// Validate Password
	if err := Required(r.Password); err != nil {
		errors = append(errors, ValidationError{Field: "password", Message: err.Error()})
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// UpdateProfileRequest is an example request struct for updating user profile
type UpdateProfileRequest struct {
	Name  string `json:"name,omitempty" validate:"omitempty,min=2,max=50"`
	Email string `json:"email,omitempty" validate:"omitempty,email"`
}

// Validate implements the Validator interface for UpdateProfileRequest
func (r *UpdateProfileRequest) Validate() error {
	var errors ValidationErrors

	// Validate Name (optional)
	if r.Name != "" {
		if err := Length(2, 50)(r.Name); err != nil {
			errors = append(errors, ValidationError{Field: "name", Message: err.Error()})
		}
	}

	// Validate Email (optional)
	if r.Email != "" {
		if err := Email(r.Email); err != nil {
			errors = append(errors, ValidationError{Field: "email", Message: err.Error()})
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}
