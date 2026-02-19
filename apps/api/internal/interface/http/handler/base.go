package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/middleware"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/validation"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// Handler provides base functionality for all handlers
type Handler struct {
	server *server.Server
}

// NewHandler creates a new base handler
func NewHandler(s *server.Server) Handler {
	return Handler{server: s}
}

// HandlerFunc represents a typed handler function that processes a request and returns a response
type HandlerFunc[Req validation.Validatable, Res any] func(c *fiber.Ctx, req Req) (Res, error)

// HandlerFuncNoContent represents a typed handler function that processes a request without returning content
type HandlerFuncNoContent[Req validation.Validatable] func(c *fiber.Ctx, req Req) error

// ResponseHandler defines the interface for handling different response types
type ResponseHandler interface {
	Handle(c *fiber.Ctx, result any) error
	GetOperation() string
	AddAttributes(txn *newrelic.Transaction, result any)
}

// JSONResponseHandler handles JSON responses
type JSONResponseHandler struct {
	status int
}

func (h JSONResponseHandler) Handle(c *fiber.Ctx, result any) error {
	return c.Status(h.status).JSON(result)
}

func (h JSONResponseHandler) GetOperation() string {
	return "handler"
}

func (h JSONResponseHandler) AddAttributes(txn *newrelic.Transaction, result any) {
	// http.status_code is already set by tracing middleware
}

// NoContentResponseHandler handles no-content responses
type NoContentResponseHandler struct {
	status int
}

func (h NoContentResponseHandler) Handle(c *fiber.Ctx, result any) error {
	return c.SendStatus(h.status)
}

func (h NoContentResponseHandler) GetOperation() string {
	return "handler_no_content"
}

func (h NoContentResponseHandler) AddAttributes(txn *newrelic.Transaction, result any) {
	// http.status_code is already set by tracing middleware
}

// FileResponseHandler handles file responses
type FileResponseHandler struct {
	status      int
	filename    string
	contentType string
}

func (h FileResponseHandler) Handle(c *fiber.Ctx, result any) error {
	data := result.([]byte)
	c.Response().Header.Set("Content-Disposition", "attachment; filename="+h.filename)
	c.Set(fiber.HeaderContentType, h.contentType)
	return c.Status(h.status).Send(data)
}

func (h FileResponseHandler) GetOperation() string {
	return "handler_file"
}

func (h FileResponseHandler) AddAttributes(txn *newrelic.Transaction, result any) {
	if txn != nil {
		// http.status_code is already set by tracing middleware
		txn.AddAttribute("file.name", h.filename)
		txn.AddAttribute("file.content_type", h.contentType)
		if data, ok := result.([]byte); ok {
			txn.AddAttribute("file.size_bytes", len(data))
		}
	}
}

// handleRequest is the unified handler function that eliminates code duplication
func handleRequest[Req validation.Validatable](
	c *fiber.Ctx,
	req Req,
	handler func(c *fiber.Ctx, req Req) (any, error),
	responseHandler ResponseHandler,
) error {
	start := time.Now()
	method := c.Method()
	path := c.Path()
	route := path
	if c.Route() != nil && c.Route().Path != "" {
		route = c.Route().Path
	}

	// Get New Relic transaction from context
	txn := newrelic.FromContext(c.UserContext())
	if txn != nil {
		txn.AddAttribute("handler.name", route)
		txn.AddAttribute("http.method", method)
		txn.AddAttribute("http.route", route)
		responseHandler.AddAttributes(txn, nil)
	}

	// Get context-enhanced logger
	loggerBuilder := middleware.GetLogger(c).With().
		Str("operation", responseHandler.GetOperation()).
		Str("method", method).
		Str("path", path).
		Str("route", route)

	// Add file-specific fields to logger if it's a file handler
	if fileHandler, ok := responseHandler.(FileResponseHandler); ok {
		loggerBuilder = loggerBuilder.
			Str("filename", fileHandler.filename).
			Str("content_type", fileHandler.contentType)
	}

	logger := loggerBuilder.Logger()

	// user.id is already set by tracing middleware

	logger.Info().Msg("handling request")

	// Validation with observability
	validationStart := time.Now()
	if err := validation.BindAndValidate(c, req); err != nil {
		validationDuration := time.Since(validationStart)

		logger.Error().
			Err(err).
			Dur("validation_duration", validationDuration).
			Msg("request validation failed")

		if txn != nil {
			txn.NoticeError(nrpkgerrors.Wrap(err))
			txn.AddAttribute("validation.status", "failed")
			txn.AddAttribute("validation.duration_ms", validationDuration.Milliseconds())
		}
		return err
	}

	validationDuration := time.Since(validationStart)
	if txn != nil {
		txn.AddAttribute("validation.status", "success")
		txn.AddAttribute("validation.duration_ms", validationDuration.Milliseconds())
	}

	logger.Debug().
		Dur("validation_duration", validationDuration).
		Msg("request validation successful")

	// Execute handler with observability
	handlerStart := time.Now()
	result, err := handler(c, req)
	handlerDuration := time.Since(handlerStart)

	if err != nil {
		totalDuration := time.Since(start)

		logger.Error().
			Err(err).
			Dur("handler_duration", handlerDuration).
			Dur("total_duration", totalDuration).
			Msg("handler execution failed")

		if txn != nil {
			txn.NoticeError(nrpkgerrors.Wrap(err))
			txn.AddAttribute("handler.status", "error")
			txn.AddAttribute("handler.duration_ms", handlerDuration.Milliseconds())
			txn.AddAttribute("total.duration_ms", totalDuration.Milliseconds())
		}
		return err
	}

	totalDuration := time.Since(start)

	// Record success metrics and tracing
	if txn != nil {
		txn.AddAttribute("handler.status", "success")
		txn.AddAttribute("handler.duration_ms", handlerDuration.Milliseconds())
		txn.AddAttribute("total.duration_ms", totalDuration.Milliseconds())
		responseHandler.AddAttributes(txn, result)
	}

	logger.Info().
		Dur("handler_duration", handlerDuration).
		Dur("validation_duration", validationDuration).
		Dur("total_duration", totalDuration).
		Msg("request completed successfully")

	return responseHandler.Handle(c, result)
}

// Handle wraps a handler with validation, error handling, logging, metrics, and tracing
func Handle[Req validation.Validatable, Res any](
	h Handler,
	handler HandlerFunc[Req, Res],
	status int,
	req Req,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return handleRequest(c, req, func(c *fiber.Ctx, req Req) (any, error) {
			return handler(c, req)
		}, JSONResponseHandler{status: status})
	}
}

func HandleFile[Req validation.Validatable](
	h Handler,
	handler HandlerFunc[Req, []byte],
	status int,
	req Req,
	filename string,
	contentType string,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return handleRequest(c, req, func(c *fiber.Ctx, req Req) (any, error) {
			return handler(c, req)
		}, FileResponseHandler{
			status:      status,
			filename:    filename,
			contentType: contentType,
		})
	}
}

// HandleNoContent wraps a handler with validation, error handling, logging, metrics, and tracing for endpoints that don't return content
func HandleNoContent[Req validation.Validatable](
	h Handler,
	handler HandlerFuncNoContent[Req],
	status int,
	req Req,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return handleRequest(c, req, func(c *fiber.Ctx, req Req) (any, error) {
			err := handler(c, req)
			return nil, err
		}, NoContentResponseHandler{status: status})
	}
}
