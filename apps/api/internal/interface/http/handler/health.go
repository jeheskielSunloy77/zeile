package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/middleware"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/response"
)

type HealthHandler struct {
	Handler
}

func NewHealthHandler(h Handler) *HealthHandler {
	return &HealthHandler{
		Handler: h,
	}
}

func (h *HealthHandler) GetHealth(c *fiber.Ctx) error {
	start := time.Now()
	logger := middleware.GetLogger(c).With().
		Str("operation", "health_check").
		Logger()

	result := map[string]any{
		"status":      "healthy",
		"timestamp":   time.Now().UTC(),
		"environment": h.server.Config.Primary.Env,
		"checks":      make(map[string]any),
	}

	checks := result["checks"].(map[string]any)
	isHealthy := true

	// Check database connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sqlDB := h.server.DB.SQLDB
	dbStart := time.Now()
	if sqlDB == nil {
		checks["database"] = map[string]any{
			"status":        "unhealthy",
			"response_time": time.Since(dbStart).String(),
			"error":         "database connection not initialized",
		}
		isHealthy = false
		logger.Error().Msg("database health check failed: no connection")
	} else if err := sqlDB.PingContext(ctx); err != nil {
		checks["database"] = map[string]any{
			"status":        "unhealthy",
			"response_time": time.Since(dbStart).String(),
			"error":         err.Error(),
		}
		isHealthy = false
		logger.Error().Err(err).Dur("response_time", time.Since(dbStart)).Msg("database health check failed")
		if h.server.LoggerService != nil && h.server.LoggerService.GetApplication() != nil {
			h.server.LoggerService.GetApplication().RecordCustomEvent(
				"HealthCheckError", map[string]any{
					"check_type":       "database",
					"operation":        "health_check",
					"error_type":       "database_unhealthy",
					"response_time_ms": time.Since(dbStart).Milliseconds(),
					"error_message":    err.Error(),
				})
		}
	} else {
		checks["database"] = map[string]any{
			"status":        "healthy",
			"response_time": time.Since(dbStart).String(),
		}
		logger.Info().Dur("response_time", time.Since(dbStart)).Msg("database health check passed")
	}

	// Check Redis connectivity
	if h.server.Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		redisStart := time.Now()
		if err := h.server.Redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = map[string]any{
				"status":        "unhealthy",
				"response_time": time.Since(redisStart).String(),
				"error":         err.Error(),
			}
			logger.Error().Err(err).Dur("response_time", time.Since(redisStart)).Msg("redis health check failed")
			if h.server.LoggerService != nil && h.server.LoggerService.GetApplication() != nil {
				h.server.LoggerService.GetApplication().RecordCustomEvent(
					"HealthCheckError", map[string]any{
						"check_type":       "redis",
						"operation":        "health_check",
						"error_type":       "redis_unhealthy",
						"response_time_ms": time.Since(redisStart).Milliseconds(),
						"error_message":    err.Error(),
					})
			}
		} else {
			checks["redis"] = map[string]any{
				"status":        "healthy",
				"response_time": time.Since(redisStart).String(),
			}
			logger.Info().Dur("response_time", time.Since(redisStart)).Msg("redis health check passed")
		}
	}

	resp := response.Response[map[string]any]{
		Message: "Health check completed, all systems operational",
		Data:    &result,
		Status:  http.StatusOK,
		Success: true,
	}

	// Set overall status
	if !isHealthy {
		result["status"] = "unhealthy"
		resp.Message = "Health check completed, some systems are unhealthy"
		resp.Data = &result

		logger.Warn().
			Dur("total_duration", time.Since(start)).
			Msg("health check failed")
		if h.server.LoggerService != nil && h.server.LoggerService.GetApplication() != nil {
			h.server.LoggerService.GetApplication().RecordCustomEvent(
				"HealthCheckError", map[string]any{
					"check_type":        "overall",
					"operation":         "health_check",
					"error_type":        "overall_unhealthy",
					"total_duration_ms": time.Since(start).Milliseconds(),
				})
		}

		return c.Status(http.StatusServiceUnavailable).JSON(resp)
	}

	logger.Info().
		Dur("total_duration", time.Since(start)).
		Msg("health check passed")

	err := c.Status(http.StatusOK).JSON(resp)
	if err != nil {
		logger.Error().Err(err).Msg("failed to write JSON response")
		if h.server.LoggerService != nil && h.server.LoggerService.GetApplication() != nil {
			h.server.LoggerService.GetApplication().RecordCustomEvent(
				"HealthCheckError", map[string]any{
					"check_type":    "response",
					"operation":     "health_check",
					"error_type":    "json_response_error",
					"error_message": err.Error(),
				})
		}
		return fmt.Errorf("failed to write JSON response: %w", err)
	}

	return nil
}
