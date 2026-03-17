package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorCode is a machine-readable identifier for an API error.
// It is stable across releases and safe for clients to switch on.
//
// Documented error codes:
//
//   - INVALID_REQUEST   – The request body or parameters could not be parsed / validated.
//   - INTERNAL_ERROR    – An unexpected server-side error occurred.
//   - UNAUTHORIZED      – Authentication credentials are missing or invalid.
//   - TOO_MANY_REQUESTS – The client has exceeded the allowed request rate.
//   - READONLY          – The server is running in readonly mode; write operations are rejected.
type ErrorCode string

const (
	ErrCodeInvalidRequest  ErrorCode = "INVALID_REQUEST"
	ErrCodeInternalError   ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrCodeTooManyRequests ErrorCode = "TOO_MANY_REQUESTS"
	ErrCodeReadonly        ErrorCode = "READONLY"
)

// ErrorDetail is the canonical error payload embedded in every error response.
//
// Error envelope shape (HTTP 4xx / 5xx):
//
//	{
//	  "error": {
//	    "code":    "<ErrorCode>",
//	    "message": "<human-readable description>"
//	  }
//	}
//
// Successful responses are unaffected and do not include this envelope.
type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// ErrResponse returns a gin.H map that wraps detail in the standard error envelope.
// Use with c.JSON(statusCode, ErrResponse(...)).
func ErrResponse(code ErrorCode, message string) gin.H {
	return gin.H{
		"error": ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

// ErrInvalidRequest returns a 400-appropriate error envelope for bad request payloads.
func ErrInvalidRequest(message string) gin.H {
	return ErrResponse(ErrCodeInvalidRequest, message)
}

// ErrInternal returns a 500-appropriate error envelope for unexpected server errors.
func ErrInternal(message string) gin.H {
	return ErrResponse(ErrCodeInternalError, message)
}

// ErrUnauthorized returns a 401-appropriate error envelope for auth failures.
func ErrUnauthorized(message string) gin.H {
	return ErrResponse(ErrCodeUnauthorized, message)
}

// ErrTooManyRequests returns a 429-appropriate error envelope for rate-limit violations.
func ErrTooManyRequests(message string) gin.H {
	return ErrResponse(ErrCodeTooManyRequests, message)
}

// AbortWithError writes the standard error envelope and aborts the handler chain.
func AbortWithError(c *gin.Context, status int, code ErrorCode, message string) {
	c.AbortWithStatusJSON(status, ErrResponse(code, message))
}

// RespondError writes the standard error envelope without aborting.
func RespondError(c *gin.Context, status int, code ErrorCode, message string) {
	c.JSON(status, ErrResponse(code, message))
}

// BindJSONOrError attempts to bind the request body to dst.
// On failure it writes a 400 INVALID_REQUEST response and returns false.
// On success it returns true and does not write anything.
func BindJSONOrError(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
		return false
	}
	return true
}
