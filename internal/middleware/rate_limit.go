package middleware

import "github.com/Barry-dE/go-backend-boilerplate/internal/server"

type RateLimiterMiddleware struct {
	server *server.Server
}

func (rl *RateLimiterMiddleware) NewRateLimiter(s *server.Server) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		server: s,
	}
}

// RecordHit records a rate limit breach event to New Relic
func (rl *RateLimiterMiddleware) RecordHit(endpoint string) {
	if rl.server.LoggerService != nil && rl.server.LoggerService.GetNewRelicApp() != nil {
		rl.server.LoggerService.GetNewRelicApp().RecordCustomEvent("RateLimitHit", map[string]interface{}{
			"endpoint": endpoint,
		})
	}
}
