package middleware

import (
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v4"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type TracingMiddleware struct {
	server      *server.Server
	newRelicApp *newrelic.Application
}

func NewTracingMiddleware(s *server.Server, newRelicApp *newrelic.Application) *TracingMiddleware {
	return &TracingMiddleware{
		server:      s,
		newRelicApp: newRelicApp,
	}
}

// NewRelicMiddleware initializes the New Relic middleware for distributed tracing.
// If New Relic is not configured (newRelicApp is nil), returns a no-op middleware
// to ensure the application continues to function without tracing.
func (tm *TracingMiddleware) NewRelicMiddleware() echo.MiddlewareFunc {
	if tm.newRelicApp == nil {
		// No-op middleware
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}
	return nrecho.Middleware(tm.newRelicApp)
}

// EnchanceTracing enriches New Relic transactions with additional context and error tracking.
// It adds request and user attributes to each transaction and reports errors with stack traces.
func (tm *TracingMiddleware) EnchanceTracing() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract the New Relic transaction from the request context
			txn := newrelic.FromContext(c.Request().Context())
			if txn == nil {
				return next(c)
			}

			// Add custom attributes
			tm.addRequestAttributes(txn, c)
			tm.addUserAttributes(txn, c)

			// Execute the next handler and capture any errors
			err := next(c)

			// Report errors to New Relic with stack traces
			if err != nil {
				txn.NoticeError(nrpkgerrors.Wrap(err))
			}

			// Record the final status code
			// This is useful in:  Filtering transactions by response code in New Relic
			// Create alerts based on error rates
			// Generate reports on API health
			// Correlate performance issues with specific response types
			txn.AddAttribute("http.status_code", c.Response().Status)

			return err
		}

	}
}

func (tm *TracingMiddleware) addRequestAttributes(txn *newrelic.Transaction, c echo.Context) {
	txn.AddAttribute("service.name", tm.server.Config.Observability.ServiceName)
	txn.AddAttribute("service.environment", tm.server.Config.Observability.Environment)
	txn.AddAttribute("http.user_agent", c.Request().UserAgent())
	txn.AddAttribute("http.real_ip", c.RealIP())

	requestID := GetRequestID(c)
	if requestID != "" {
		txn.AddAttribute("request_id", requestID)
	}
}

func (tm *TracingMiddleware) addUserAttributes(txn *newrelic.Transaction, c echo.Context) {
	userID := c.Get("user_id")
	if userID == nil {
		return
	}

	if userIDStr, ok := userID.(string); ok {
		txn.AddAttribute("user_id", userIDStr)
	}
}
