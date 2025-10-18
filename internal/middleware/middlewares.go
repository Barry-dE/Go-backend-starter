package middleware

import (
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type Middlewares struct {
	GlobalMiddleware     *GlobalMiddleware
	AuthMiddleware        *AuthMiddleware
	TracingMiddleware     *TracingMiddleware
	RateLimiterMiddleware *RateLimiterMiddleware
	ContextEnhancer       *ContextEnhancer
}

func NewMiddlewares(s *server.Server) *Middlewares{
	var newrelicApp *newrelic.Application
	if s.LoggerService != nil{
		newrelicApp = s.LoggerService.GetNewRelicApp()
	}

	return &Middlewares{
		GlobalMiddleware: NewGlobalMiddleWare(s),
		AuthMiddleware: NewAuthMiddleware(s),
		TracingMiddleware: NewTracingMiddleware(s, newrelicApp),
		RateLimiterMiddleware: NewRateLimiter(s),
		ContextEnhancer: NewContextEnhancer(s),
	}

}

