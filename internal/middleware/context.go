package middleware

import (
	"context"

	"github.com/Barry-dE/go-backend-boilerplate/internal/logger"
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

const (
	
	UserRoleKey = "user_role"
	UserIDkEY   = "user_id"
)


// contextKey is unexported so other packages can't collide with our keys.
// the pointer value ensures a unique, comparable key.
type contextKey struct{ name string }

var (
	loggerKey     = &contextKey{name: "logger"} // for context.WithValue
	echoLoggerKey = "logger"              // for echo's context
)

// ContextEnhancer is a middleware responsible for enriching the request context
// with additional metadata (request ID, trace info, user info, etc.)
// and a contextual logger. This improves observability and makes debugging easier.
type ContextEnhancer struct {
	server *server.Server
}

// NewContextEnhancer returns a new instance of ContextEnhancer tied to the server.
// The server reference gives access to shared dependencies
func NewContextEnhancer(s *server.Server) *ContextEnhancer {
	return &ContextEnhancer{
		server: s,
	}
}

func (ce *ContextEnhancer) EnhanceContext() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			requestID := GetRequestID(c)

			// Create a structured logger tied to each incoming  request that includes request-scoped metadata such as HTTP method, route path, and client IP.
			contextLogger := ce.server.Logger.With().Str("request_id", requestID).Str("method", c.Request().Method).Str("path", c.Path()).Str("ip", c.RealIP()).Logger()

			// If this request is part of a distributed trace, extract and attach the trace/span IDs for cross-service correlation.
			if txn := newrelic.FromContext(c.Request().Context()); txn != nil {
				contextLogger = logger.WithTraceContext(contextLogger, txn)
			}

			// Extract user info from JWT (if available) to enrich the transaction logs.This enables per-user observability and better audit trails.
			userID := ce.getUserID(c)
			if userID != "" {
				contextLogger = contextLogger.With().Str(UserIDkEY, userID).Logger()
			}

			userRole := ce.getUserRole(c)
			if userRole != "" {
				contextLogger = contextLogger.With().Str(UserRoleKey, userRole).Logger()
			}

			// Store the enhanced logger in Echo’s context so handlers can access it
			c.Set(echoLoggerKey, &contextLogger)

			// create a new context with the logger
			ctx := context.WithValue(c.Request().Context(), loggerKey, &contextLogger)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}

}

func (ce *ContextEnhancer) getUserID(c echo.Context) string {
	if userID, ok := c.Get(UserIDkEY).(string); ok && userID != "" {
		return userID
	}

	return ""
}

func (ce *ContextEnhancer) getUserRole(c echo.Context) string {
	if role, ok := c.Get(UserRoleKey).(string); ok && role != "" {
		return role
	}

	return ""
}

func GetLogger(c echo.Context) *zerolog.Logger {
	if lg, ok := c.Get(echoLoggerKey).(*zerolog.Logger); ok && lg != nil {
		return lg
	}

	// nop is a no-op zerolog Logger used as a safe default.
	nop := zerolog.Nop()
	return &nop
}

func GetUserID(c echo.Context) string {
	userID, ok := c.Get(UserIDkEY).(string)
	if ok {
		return userID
	}
	return ""
}
