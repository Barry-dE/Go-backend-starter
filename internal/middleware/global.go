package middleware

import (
	"errors"
	"net/http"

	"github.com/Barry-dE/go-backend-boilerplate/internal/errs"
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/Barry-dE/go-backend-boilerplate/internal/sqlerr"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

// GlobalMiddleWares bundles all middlewares used across the application.
// It keeps a reference to the main server, giving middlewares access to configuration and utilities.
type GlobalMiddleware struct {
	server *server.Server
}

// NewGlobalMiddleWares initializes and returns a GlobalMiddleWares instance.
func NewGlobalMiddleWare(s *server.Server) *GlobalMiddleware {
	return &GlobalMiddleware{
		server: s,
	}
}

// CORS configures Cross-Origin Resource Sharing using allowed origins from server config.
// This enables browsers to safely call the API from specified domains.
func (gm *GlobalMiddleware) CORS() echo.MiddlewareFunc {
	return echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: gm.server.Config.Server.CORSAllowedOrigin,
	})
}


// RequestLogger logs every HTTP request passing through the server.
// It captures request details, latency, and errors, using structured logging via zerolog.
func (gm *GlobalMiddleware) RequestLogger() echo.MiddlewareFunc {
	return echoMiddleware.RequestLoggerWithConfig(echoMiddleware.RequestLoggerConfig{
		LogURI:     true,
		LogMethod:  true,
		LogURIPath: true,
		LogLatency: true,
		LogError:   true,
		LogHost:    true,
		LogStatus:  true,
		// Custom log handler for shaping the log output and behavior
		LogValuesFunc: func(c echo.Context, v echoMiddleware.RequestLoggerValues) error {

			statusCode := v.Status

			// Detect and normalize error types to extract the proper status code
			if v.Error != nil {
				var httpErr *errs.HttpError
				var EchoErr *echo.HTTPError

				if errors.As(v.Error, &httpErr) {
					statusCode = httpErr.Status
				} else if errors.As(v.Error, &EchoErr) {
					statusCode = EchoErr.Code
				}

			}

			// Retrieve context-aware logger (may include trace IDs or user data)
			logger := GetLogger(c)

			var e *zerolog.Event

			// Choose log level based on response status
			switch {
			case statusCode >= 500:
				e = logger.Error().Err(v.Error) // Server errors
			case statusCode >= 400:
				e = logger.Warn() // Client errors
			default:
				e = logger.Info() // Successful requests

			}

			// Include request ID for traceability
			requestID := GetRequestID(c)
			if requestID != "" {
				e = e.Str("request_id", requestID)
			}

			// Add user ID
			userId := GetUserID(c)
			if userId != "" {
				e = e.Str("user_id", userId)
			}

			// Log full structured data
			e.Dur("latency", v.Latency).Int("status", statusCode).Str("method", v.Method).Str("uri", v.URI).Str("route", c.Path()).Str("host", v.Host).Str("ip", c.RealIP()).Str("user_agent", c.Request().UserAgent()).Msg("API")
			return nil
		},
	})
}

// Secure adds security-related headers to all responses (e.g., preventing clickjacking, XSS, etc.)
func (gm *GlobalMiddleware) Secure() echo.MiddlewareFunc {
	return echoMiddleware.Secure()
}

// Recover gracefully handles panics to prevent the server from crashing.
// It logs the panic and returns a generic 500 error to the client.
func (gm *GlobalMiddleware) Recover() echo.MiddlewareFunc {
	return echoMiddleware.Recover()
}

// GlobalErrorHandler provides centralized handling for any unhandled error in the app.
// It ensures consistent JSON error responses and detailed server-side logging.
func (gm *GlobalMiddleware) GlobalErrorHandler(err error, c echo.Context) {
	
	// Preserve stack trace and raw diagnostic info of original error for logging.
	originalErr := err

	// Convert different error types into standardized HTTP errors
	var httpErr *errs.HttpError

	if !errors.As(err, &httpErr) {
		var echoErr *echo.HTTPError
		if errors.As(err, &echoErr) {
			if echoErr.Code == http.StatusNotFound {
				err = errs.NotFoundError("Route not found", false, nil)
			}
		} else {
			/// Handle possible database errors
			sqlerr.HandleError(err)
		}
	}

	// Extract relevant data to build the HTTP response
	var code string
	var echoErr *echo.HTTPError
	var message string
	var status int
	var fieldErrors []errs.FieldError
	var action *errs.Action

	switch {
	case errors.As(err, &httpErr):
		status = httpErr.Status
		code = httpErr.Code
		message = httpErr.Message
		fieldErrors = httpErr.Errors
		action = httpErr.Action

	case errors.As(err, &echoErr):
		status = echoErr.Code
		code = errs.MakeUpperCaseWithUnderscores(http.StatusText(status))
		if msg, ok := echoErr.Message.(string); ok{
			message = msg
		}else{
			message = http.StatusText(echoErr.Code)
		}
	// Fallback for unknown errors
	default:
		status = http.StatusInternalServerError
		code = errs.MakeUpperCaseWithUnderscores(http.StatusText(http.StatusInternalServerError))
		message = http.StatusText(http.StatusInternalServerError)
	
}

// Log the original error with all relevant context
logger := *GetLogger(c)

logger.Error().Stack().Err(originalErr).Int("status", status).Str("error_code", code).Msg(message)

// Send a structured JSON error response if nothing has been sent yet
if !c.Response().Committed{
	_ = c.JSON(status, errs.HttpError{
		Code: code,
		Message: message,
		Status: status,
		Override: httpErr != nil && httpErr.Override,
		Errors: fieldErrors,
		Action: action,
	})
}

}
