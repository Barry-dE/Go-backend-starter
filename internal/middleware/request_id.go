package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

// RequestID is middleware that ensures each incoming HTTP request
// has a unique identifier. If the client doesn’t send one,
// it generates a new UUID and attaches it to both the request context
// and the response header for traceability.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if the client already provided a request ID.
			requestID := c.Request().Header.Get(RequestIDHeader)
			// If not, create a new one.
			if requestID == "" {
				requestID = uuid.New().String()
			}
			// Store the request ID in the context so other parts of the app (like logs) can access it.
			c.Set(RequestIDKey, requestID)
			// Add the request ID to the response header
			c.Response().Header().Set(RequestIDHeader, requestID)
			// Proceed to the next middleware or handler.
			return next(c)
		}
	}
}

// GetRequestID retrieves the request ID stored in the request context.
// Returns an empty string if none is found.
func GetRequestID(c echo.Context) string {
	if requestID, ok := c.Get(RequestIDKey).(string); ok {
		return requestID
	}

	return ""
}
