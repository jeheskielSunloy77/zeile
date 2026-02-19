package middleware

import (
	"github.com/jeheskielSunloy77/zeile/internal/application"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type Middlewares struct {
	Global          *GlobalMiddlewares
	Auth            *AuthMiddleware
	Authorization   *AuthorizationMiddleware
	ContextEnhancer *ContextEnhancer
	Tracing         *TracingMiddleware
	RateLimit       *RateLimitMiddleware
}

func NewMiddlewares(s *server.Server, services *application.Services) *Middlewares {
	// Get New Relic application instance from server
	var nrApp *newrelic.Application
	if s.LoggerService != nil {
		nrApp = s.LoggerService.GetApplication()
	}

	var authorizer AuthorizationEnforcer
	if services != nil {
		authorizer = services.Authorization
	}

	return &Middlewares{
		Global:          NewGlobalMiddlewares(s),
		Auth:            NewAuthMiddleware(s),
		Authorization:   NewAuthorizationMiddleware(authorizer),
		ContextEnhancer: NewContextEnhancer(s),
		Tracing:         NewTracingMiddleware(s, nrApp),
		RateLimit:       NewRateLimitMiddleware(s),
	}
}
