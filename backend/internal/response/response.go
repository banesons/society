package response

import (
	"encoding/json"
	"net/http"
)

// SuccessEnvelope defines the standard success JSON envelope.
type SuccessEnvelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    interface{} `json:"meta"`
}

// ErrorDetail defines the error payload structure.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorEnvelope defines the standard error JSON envelope.
type ErrorEnvelope struct {
	Success   bool        `json:"success"`
	Error     ErrorDetail `json:"error"`
	RequestID string      `json:"request_id"`
}

// JSON sends a JSON response with the provided status code.
func JSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

// Success writes a standard 2xx response.
func Success(w http.ResponseWriter, statusCode int, data interface{}, meta interface{}) {
	JSON(w, statusCode, SuccessEnvelope{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error writes a standard error response containing the request ID.
func Error(w http.ResponseWriter, statusCode int, code string, message string, requestID string) {
	JSON(w, statusCode, ErrorEnvelope{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
		RequestID: requestID,
	})
}
