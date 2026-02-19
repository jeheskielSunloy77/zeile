package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/logger"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

const (
	UserIDKey      = "user_id"
	UserRoleKey    = "user_role"
	UserEmailKey   = "user_email"
	UserIsAdminKey = "user_is_admin"
	LoggerKey      = "logger"
)

type ContextEnhancer struct {
	server *server.Server
}

func NewContextEnhancer(s *server.Server) *ContextEnhancer {
	return &ContextEnhancer{server: s}
}

func (ce *ContextEnhancer) WithTimeout() fiber.Handler {
	return func(c *fiber.Ctx) error {
		timeout := ce.timeoutForMethod(c.Method())
		if timeout <= 0 {
			return c.Next()
		}

		ctx, cancel := context.WithTimeout(c.UserContext(), timeout)
		defer cancel()

		c.SetUserContext(ctx)

		return c.Next()
	}
}

func (ce *ContextEnhancer) EnhanceContext() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := GetRequestID(c)

		contextLogger := ce.server.Logger.With().
			Str("request_id", requestID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Logger()

		if txn := newrelic.FromContext(c.UserContext()); txn != nil {
			contextLogger = logger.WithTraceContext(contextLogger, txn)
		}

		if userID := ce.extractUserID(c); userID != "" {
			contextLogger = contextLogger.With().Str("user_id", userID).Logger()
		}

		if userRole := ce.extractUserRole(c); userRole != "" {
			contextLogger = contextLogger.With().Str("user_role", userRole).Logger()
		}

		c.Locals(LoggerKey, &contextLogger)

		ctx := context.WithValue(c.UserContext(), LoggerKey, &contextLogger)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

func (ce *ContextEnhancer) extractUserID(c *fiber.Ctx) string {
	if userID, ok := c.Locals(UserIDKey).(string); ok && userID != "" {
		return userID
	}
	return ""
}

func (ce *ContextEnhancer) extractUserRole(c *fiber.Ctx) string {
	if userRole, ok := c.Locals(UserRoleKey).(string); ok && userRole != "" {
		return userRole
	}
	return ""
}

func (ce *ContextEnhancer) timeoutForMethod(method string) time.Duration {
	switch method {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return ce.server.Config.Server.ReadTimeout
	default:
		return ce.server.Config.Server.WriteTimeout
	}
}

func GetUserID(c *fiber.Ctx) string {
	if userID, ok := c.Locals(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func GetUserEmail(c *fiber.Ctx) string {
	if email, ok := c.Locals(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

func GetUserIsAdmin(c *fiber.Ctx) bool {
	if isAdmin, ok := c.Locals(UserIsAdminKey).(bool); ok {
		return isAdmin
	}
	return false
}

func GetLogger(c *fiber.Ctx) *zerolog.Logger {
	if logger, ok := c.Locals(LoggerKey).(*zerolog.Logger); ok {
		return logger
	}
	logger := zerolog.Nop()
	return &logger
}
