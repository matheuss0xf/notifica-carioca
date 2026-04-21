package httpx

import "github.com/gin-gonic/gin"

// ErrorResponse is the standardized error payload returned by HTTP adapters.
type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

// MessageResponse is a simple success payload for message-only responses.
type MessageResponse struct {
	Message string `json:"message"`
}

// JSONError writes a standardized error response.
func JSONError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Code: code, Error: message})
}

// JSONFieldError writes a standardized field-scoped error response.
func JSONFieldError(c *gin.Context, status int, code, message, field string) {
	c.JSON(status, ErrorResponse{Code: code, Error: message, Field: field})
}

// AbortError aborts the request with a standardized error response.
func AbortError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Code: code, Error: message})
}
