package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/application"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/handler"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/middleware"
)

func NewRouter(s *server.Server, h *handler.Handlers, services *application.Services) *fiber.App {
	middlewares := middleware.NewMiddlewares(s, services)

	router := fiber.New(fiber.Config{
		ErrorHandler:          middlewares.Global.GlobalErrorHandler,
		ReadTimeout:           s.Config.Server.ReadTimeout,
		WriteTimeout:          s.Config.Server.WriteTimeout,
		IdleTimeout:           s.Config.Server.IdleTimeout,
		DisableStartupMessage: true,
	})

	// global middlewares
	router.Use(
		limiter.New(limiter.Config{
			Max:        20,
			Expiration: time.Second,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				if rateLimitMiddleware := middlewares.RateLimit; rateLimitMiddleware != nil {
					rateLimitMiddleware.RecordRateLimitHit(c.Path())
				}

				s.Logger.Warn().
					Str("request_id", middleware.GetRequestID(c)).
					Str("path", c.Path()).
					Str("method", c.Method()).
					Str("ip", c.IP()).
					Msg("rate limit exceeded")

				return c.Status(http.StatusTooManyRequests).JSON(
					errs.ErrorResponse{
						Success:  false,
						Message:  "Rate limit exceeded",
						Status:   http.StatusTooManyRequests,
						Override: false,
					})
			},
		}),
		cors.New(cors.Config{
			AllowOrigins:     strings.Join(s.Config.Server.CORSAllowedOrigins, ","),
			AllowCredentials: true,
			AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		}),
		helmet.New(),
		middleware.RequestID(),
		middlewares.Tracing.NewRelicMiddleware(),
		middlewares.Tracing.EnhanceTracing(),
		middlewares.ContextEnhancer.EnhanceContext(),
		middlewares.ContextEnhancer.WithTimeout(),
		middlewares.Global.RequestLogger(),
		middlewares.Global.Recover(),
	)

	// register application routes
	registerRoutes(router, h, middlewares)

	return router
}
