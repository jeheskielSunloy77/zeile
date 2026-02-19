package router

import (
	"github.com/gofiber/fiber/v2"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	httpdto "github.com/jeheskielSunloy77/zeile/internal/interface/http/dto"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/handler"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/middleware"
)

func registerRoutes(
	r *fiber.App,
	h *handler.Handlers,
	middlewares *middleware.Middlewares,
) {
	// system routes
	r.Get("/health", h.Health.GetHealth)
	r.Static("/static", "static")
	r.Get("/api/docs", h.OpenAPI.ServeOpenAPIUI)

	// versioned routes
	api := r.Group("/api/v1")

	authGroup := api.Group("/auth")
	authGroup.Post("/register", h.Auth.Register())
	authGroup.Post("/login", h.Auth.Login())
	authGroup.Get("/google", h.Auth.GoogleLogin())
	authGroup.Get("/google/callback", h.Auth.GoogleCallback())
	authGroup.Post("/verify-email", h.Auth.VerifyEmail())
	authGroup.Post("/refresh", h.Auth.Refresh())
	authGroup.Post("/logout", h.Auth.Logout())

	authProtected := authGroup.Group("", middlewares.Auth.RequireAuth())
	authProtected.Get("/me", h.Auth.Me())
	authProtected.Post("/resend-verification", h.Auth.ResendVerification())
	authProtected.Post("/logout-all", h.Auth.LogoutAll())

	// protected routes
	protected := api.Group("", middlewares.Auth.RequireAuth())

	resource(protected, "/users", h.User.ResourceHandler)
}

func resource[T domain.BaseModel, S applicationdto.StoreDTO[T], U applicationdto.UpdateDTO[T], TS httpdto.StoreDTO[S], TU httpdto.UpdateDTO[U]](group fiber.Router, path string, h *handler.ResourceHandler[T, S, U, TS, TU], authMiddleware ...fiber.Handler) {
	g := group.Group(path, authMiddleware...)
	g.Get("/", h.GetMany())
	g.Get("/:id", h.GetByID())
	g.Post("/", h.Store())
	g.Delete("/:id", h.Destroy())
	g.Delete("/:id/kill", h.Kill())
	g.Patch("/:id/restore", h.Restore())
	g.Patch("/:id", h.Update())
}
