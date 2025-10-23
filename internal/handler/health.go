package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Barry-dE/go-backend-boilerplate/internal/middleware"
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	Handler
}

func NewHealthHandler(s *server.Server) *HealthHandler {
	return &HealthHandler{
		Handler: NewHandler(s),
	}
}

func (h *HealthHandler) HealthCheck(c echo.Context) error {
	start := time.Now()
	logger := middleware.GetLogger(c).With().Str("operation", "health_check").Logger()

	response := map[string]interface{}{
		"status":      "healthy",
		"environment": h.server.Config.Primary.Env,
		"timestamp":   time.Now().UTC(),
		"checks":      make(map[string]interface{}),
	}

	// Assert type for checks map
	checks := response["checks"].(map[string]interface{})

	isHealthy := true

	// Add database connectivity check
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	databaseTimerStart := time.Now()
	err := h.server.DB.Pool.Ping(ctx)
	if err != nil {
		// populate the checks map with database health status
		checks["database"] = map[string]interface{}{
			"status":        "unhealthy",
			"error":         err.Error(),
			"response_time": time.Since(databaseTimerStart).String(),
		}
		isHealthy = false
		logger.Error().Err(err).Dur("response_time", time.Since(databaseTimerStart)).Msg("database health check failed")

		// Record New Relic custom event for database health check failure
		if h.server.LoggerService != nil && h.server.LoggerService.GetNewRelicApp() != nil {
			h.server.LoggerService.GetNewRelicApp().RecordCustomEvent("HealthCheckError", map[string]interface{}{
				"operation":        "health_check",
				"check_type":       "database_health",
				"error_type":       "database_unhealthy",
				"response_time_ms": time.Since(databaseTimerStart).Milliseconds(),
				"error_message":    err.Error(),
			})
		}

	} else {
		checks["database"] = map[string]interface{}{
			"status":           "healthy",
			"response_time": time.Since(databaseTimerStart).String(),
		}
		logger.Info().Dur("response_time_ms", time.Since(databaseTimerStart)).Msg("database health check succeeded")
	}

	// check Redis connectivity if enabled
	if h.server.Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		redisStartTimer := time.Now()
		err := h.server.Redis.Ping(ctx).Err()

		if err != nil {
			checks["redis"] = map[string]interface{}{
				"status":        "unhealthy",
				"error":         err.Error(),
				"response_time": time.Since(redisStartTimer).String(),
			}

			logger.Error().Err(err).Dur("response_time", time.Since(redisStartTimer)).Msg("redis health check failed")

			
			if h.server.LoggerService != nil && h.server.LoggerService.GetNewRelicApp() != nil {
				h.server.LoggerService.GetNewRelicApp().RecordCustomEvent("HealthCheckError", map[string]interface{}{
					"operation":        "health_check",
					"check_type":       "redis_health",
					"error_type":       "redis_unhealthy",
					"response_time_ms": time.Since(redisStartTimer).Milliseconds(),
					"error_message":    err.Error(),
				})
			}
		}else{
			checks["redis"] = map[string]interface{}{
				"status": "healthy",
				"response_time": time.Since(redisStartTimer).String(),
			}

			logger.Info().Dur("response_time", time.Since(redisStartTimer)).Msg("redis health check succeeded")
		}
	}

	// Overall health status
	if !isHealthy{
		
		response["status"] = "unhealthy"
		
		logger.Warn().Dur("total_duration", time.Since(start)).Msg("health check failed")

		if h.server.LoggerService != nil && h.server.LoggerService.GetNewRelicApp() != nil {
			h.server.LoggerService.GetNewRelicApp().RecordCustomEvent("HealthCheckError", map[string]interface{}{
				"operation":        "health_check",
				"check_type":       "overall_health",
				"error_type":       "overall_unhealthy",
				"total_response_time_ms": time.Since(start).Milliseconds(),
			})
		}

		return c.JSON(http.StatusServiceUnavailable, response)
	}

	logger.Info().Dur("total_duration", time.Since(start)).Msg("health check succeeded")
	
	if err := c.JSON(http.StatusOK, response); err != nil{
		
		logger.Error().Err(err).Msg("failed to write JSON response")
	  
		if h.server.LoggerService != nil && h.server.LoggerService.GetNewRelicApp() != nil {
		h.server.LoggerService.GetNewRelicApp().RecordCustomEvent("HealthCheckError", map[string]interface{}{
			"operation": "health_check",
			"check_type": "response",
			"error_type": "json_response",
			"error_message": err.Error(),

		})
	  }

	  return fmt.Errorf("failed to write JSON response: %w", err)
	}


	return nil
}
