package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Locals(RequestIDKey, requestID)
		c.Set(RequestIDHeader, requestID)

		return c.Next()
	}
}

func GetRequestID(c *fiber.Ctx) string {
	if requestID, ok := c.Locals(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
