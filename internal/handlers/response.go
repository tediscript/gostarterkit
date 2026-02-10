package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// JSONResponse sends a JSON success response with the given status code and data
func JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := SuccessResponse{
		Status: "success",
		Data:   data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// If we can't encode the response, log it (but we can't send another response)
		// In production, this would go to a logger
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// JSONResponseWithMessage sends a JSON success response with a custom message
func JSONResponseWithMessage(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := SuccessResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// ErrorResponse sends a JSON error response with the given status code and error message
func ErrorResponseFunc(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error: message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// ErrorResponseWithDetails sends a JSON error response with additional details
func ErrorResponseWithDetails(w http.ResponseWriter, statusCode int, message string, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   message,
		Details: details,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// ErrorResponseFromError sends a JSON error response from an error object
func ErrorResponseFromError(w http.ResponseWriter, statusCode int, err error) {
	if err == nil {
		ErrorResponseFunc(w, statusCode, "Unknown error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error: err.Error(),
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// ValidationError sends a JSON error response for validation errors
func ValidationError(w http.ResponseWriter, field, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	details := fmt.Sprintf("%s: %s", field, message)
	response := ErrorResponse{
		Error:   "Validation failed",
		Details: details,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// DecodeJSONBody decodes a JSON request body into the provided struct
func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}
