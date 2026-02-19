package middleware

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/app/sqlerr"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type GlobalMiddlewares struct {
	server *server.Server
}

func NewGlobalMiddlewares(s *server.Server) *GlobalMiddlewares {
	return &GlobalMiddlewares{
		server: s,
	}
}

func (global *GlobalMiddlewares) RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		statusCode := c.Response().StatusCode()
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		if err != nil {
			var httpErr *errs.ErrorResponse
			var fiberErr *fiber.Error
			switch {
			case errors.As(err, &httpErr):
				statusCode = httpErr.Status
			case errors.As(err, &fiberErr):
				statusCode = fiberErr.Code
			default:
				statusCode = http.StatusInternalServerError
			}
		}

		logger := GetLogger(c)

		var e *zerolog.Event

		switch {
		case statusCode >= 500:
			e = logger.Error().Err(err)
		case statusCode >= 400:
			e = logger.Warn()
		default:
			e = logger.Info()
		}

		if requestID := GetRequestID(c); requestID != "" {
			e = e.Str("request_id", requestID)
		}

		if userID := GetUserID(c); userID != "" {
			e = e.Str("user_id", userID)
		}

		e.
			Dur("latency", time.Since(start)).
			Int("status", statusCode).
			Str("method", c.Method()).
			Str("uri", c.OriginalURL()).
			Str("host", c.Hostname()).
			Str("ip", c.IP()).
			Str("user_agent", c.Get(fiber.HeaderUserAgent)).
			Msg("API")

		return err
	}
}

func (global *GlobalMiddlewares) Recover() fiber.Handler {
	return fiberrecover.New()
}

func (global *GlobalMiddlewares) GlobalErrorHandler(c *fiber.Ctx, err error) error {
	// First try to handle database errors and convert them to appropriate HTTP errors
	originalErr := err

	// Try to handle known database errors
	// Only do this for errors that haven't already been converted to HTTPError
	var httpErr *errs.ErrorResponse
	if !errors.As(err, &httpErr) {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			if fiberErr.Code == http.StatusNotFound {
				err = errs.NewNotFoundError("Route not found", false)
			}
		} else {
			// Here we call our sqlerr handler which will convert database errors
			// to appropriate application errors
			err = sqlerr.HandleError(err)
		}
	}

	// Now process the possibly converted error
	var fiberErr *fiber.Error
	var status int
	var code string
	var message string
	var fieldErrors []errs.FieldError
	var action *errs.Action

	switch {
	case errors.As(err, &httpErr):
		status = httpErr.Status
		message = httpErr.Message
		fieldErrors = httpErr.Errors
		action = httpErr.Action

	case errors.As(err, &fiberErr):
		status = fiberErr.Code
		code = errs.MakeUpperCaseWithUnderscores(http.StatusText(status))
		message = fiberErr.Message

	default:
		status = http.StatusInternalServerError
		code = errs.MakeUpperCaseWithUnderscores(
			http.StatusText(http.StatusInternalServerError))
		message = http.StatusText(http.StatusInternalServerError)
	}

	// Log the original error to help with debugging
	// Use enhanced logger from context which already includes request_id, method, path, ip, user context, and trace context
	logger := *GetLogger(c)

	logger.Error().Stack().
		Err(originalErr).
		Int("status", status).
		Str("error_code", code).
		Msg(message)

	_ = c.Status(status).JSON(errs.ErrorResponse{
		Success:  false,
		Message:  message,
		Status:   status,
		Override: httpErr != nil && httpErr.Override,
		Errors:   fieldErrors,
		Action:   action,
	})

	return nil
}
