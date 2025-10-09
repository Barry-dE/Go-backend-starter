package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Barry-dE/go-backend-boilerplate/internal/errs"
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/clerk/clerk-sdk-go/v2"
	clerkHttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	server *server.Server
}

// NewAuthMiddleware creates a new AuthMiddleware instance
// with access to the main server's dependencies (logger, config, etc.).
func NewAuthMiddleware(s *server.Server) *AuthMiddleware {
	return &AuthMiddleware{
		server: s,
	}
}
// Authenticate is an Echo middleware that checks if the incoming request is authenticated via Clerk.
// It wraps Clerk's HTTP middleware to handle Authorization headers and session validation.
// On authentication failure, it returns a JSON 401 response and logs the error.
// On success, it extracts user claims from the context and stores them for downstream handlers.
func (auth *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return echo.WrapMiddleware(
		// This wraps Clerk’s HTTP middleware to handle Authorization headers and manage session validation automatically.
		clerkHttp.WithHeaderAuthorization(
			// Custom handler for when Clerk authentication fails.
			clerkHttp.AuthorizationFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()

				// Respond with a JSON-formatted 401 Unauthorized message.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)

				response := map[string]string{
					"code":     "UNAUTHORIZED",
					"message":  "Unauthorized",
					"override": "false",
					"status":   "401",
				}
				// Attempt to send the JSON response to the client.
				if err := json.NewEncoder(w).Encode(response); err != nil {
					auth.server.Logger.Error().Err(err).Str("function", "Authenticte").Dur(
						"duration", time.Since(start)).Msg("failed to write JSON response")
				} else {
					// Secondary error log if JSON was written successfully
					auth.server.Logger.Error().Str("function", "Authenticate").Dur("duration", time.Since(start)).Msg(
						"could not get session claims from context")
				}
			}))))(func(c echo.Context) error {
		start := time.Now()
		// Extract session claims (user info) from the request context.
		// This only works if the request passed Clerk authentication.
		claims, ok := clerk.SessionClaimsFromContext(c.Request().Context())
		// If session claims are missing, authentication failed.
		if !ok {
			auth.server.Logger.Error().
				Str("function", "Authenticate").
				Str("request_id", GetRequestID(c)).
				Dur("duration", time.Since(start)).
				Msg("could not get session claims from context")

			return errs.UnauthorizedError("Unauthorized", false)
		}

		// Store user information from Clerk in the context so downstream handlers can access it
		c.Set("user_id", claims.Subject)
		c.Set("user_role", claims.ActiveOrganizationRole)
		c.Set("permissions", claims.Claims.ActiveOrganizationPermissions)

		// Log successful authentication for visibility and debugging.
		auth.server.Logger.Info().
			Str("function", "Authenticate").
			Str("user_id", claims.Subject).
			Str("request_id", GetRequestID(c)).
			Dur("duration", time.Since(start)).
			Msg("user authenticated successfully")

		return next(c)
	})
}
