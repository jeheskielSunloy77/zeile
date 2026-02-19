package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
)

type TracingMiddleware struct {
	server *server.Server
	nrApp  *newrelic.Application
}

func NewTracingMiddleware(s *server.Server, nrApp *newrelic.Application) *TracingMiddleware {
	return &TracingMiddleware{
		server: s,
		nrApp:  nrApp,
	}
}

// NewRelicMiddleware instruments fiber requests with New Relic.
func (tm *TracingMiddleware) NewRelicMiddleware() fiber.Handler {
	if tm.nrApp == nil {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}
	return func(c *fiber.Ctx) error {
		routeName := c.Path()
		if c.Route() != nil && c.Route().Path != "" {
			routeName = c.Route().Path
		}

		txn := tm.nrApp.StartTransaction(fmt.Sprintf("%s %s", c.Method(), routeName))
		defer txn.End()

		ctx := newrelic.NewContext(c.UserContext(), txn)
		c.SetUserContext(ctx)

		err := c.Next()

		txn.AddAttribute("http.method", c.Method())
		txn.AddAttribute("http.route", routeName)
		txn.AddAttribute("http.url", c.OriginalURL())
		txn.AddAttribute("http.real_ip", c.IP())
		txn.AddAttribute("http.status_code", c.Response().StatusCode())

		if err != nil {
			txn.NoticeError(nrpkgerrors.Wrap(err))
		}

		return err
	}
}

// EnhanceTracing adds custom attributes to New Relic transactions
func (tm *TracingMiddleware) EnhanceTracing() fiber.Handler {
	return func(c *fiber.Ctx) error {
		txn := newrelic.FromContext(c.UserContext())
		if txn == nil {
			return c.Next()
		}

		txn.AddAttribute("http.real_ip", c.IP())
		txn.AddAttribute("http.user_agent", c.Get(fiber.HeaderUserAgent))

		if requestID := GetRequestID(c); requestID != "" {
			txn.AddAttribute("request.id", requestID)
		}

		if userID := c.Locals(UserIDKey); userID != nil {
			if userIDStr, ok := userID.(string); ok {
				txn.AddAttribute("user.id", userIDStr)
			}
		}

		err := c.Next()
		if err != nil {
			txn.NoticeError(nrpkgerrors.Wrap(err))
		}

		txn.AddAttribute("http.status_code", c.Response().StatusCode())

		return err
	}
}
