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
	authGroup.Get("/device", h.Auth.DeviceApprovePage())
	authGroup.Post("/device/start", h.Auth.DeviceStart())
	authGroup.Post("/device/poll", h.Auth.DevicePoll())
	authGroup.Post("/refresh", h.Auth.Refresh())
	authGroup.Post("/logout", h.Auth.Logout())

	authProtected := authGroup.Group("", middlewares.Auth.RequireAuth())
	authProtected.Get("/me", h.Auth.Me())
	authProtected.Post("/device/approve", h.Auth.DeviceApprove())
	authProtected.Post("/resend-verification", h.Auth.ResendVerification())
	authProtected.Post("/logout-all", h.Auth.LogoutAll())

	// protected routes
	protected := api.Group("", middlewares.Auth.RequireAuth())

	resource(protected, "/users", h.User.ResourceHandler)

	library := protected.Group("/library")
	library.Post("/catalog/books", h.Library.CreateCatalogBook())
	library.Get("/catalog/books", h.Library.ListCatalogBooks())
	library.Post("/assets/upload", h.Library.UploadBookAsset())
	library.Post("/books", h.Library.UpsertLibraryBook())
	library.Get("/books", h.Library.ListLibraryBooks())
	library.Patch("/books/:id", h.Library.UpdateLibraryBook())
	library.Delete("/books/:id", h.Library.DeleteLibraryBook())
	library.Get("/books/:id/reading-states/:mode", h.Library.GetReadingState())
	library.Put("/books/:id/reading-states/:mode", h.Library.UpsertReadingState())
	library.Get("/books/:id/highlights", h.Library.ListHighlights())
	library.Post("/books/:id/highlights", h.Library.CreateHighlight())
	library.Patch("/highlights/:highlightId", h.Library.UpdateHighlight())
	library.Delete("/highlights/:highlightId", h.Library.DeleteHighlight())

	sharing := protected.Group("/sharing")
	sharing.Post("/lists", h.Sharing.CreateShareList())
	sharing.Get("/lists", h.Sharing.ListShareLists())
	sharing.Patch("/lists/:id", h.Sharing.UpdateShareList())
	sharing.Post("/lists/:id/items", h.Sharing.CreateShareListItem())
	sharing.Get("/lists/:id/items", h.Sharing.ListShareListItems())
	sharing.Put("/book-share-policies", h.Sharing.UpsertBookSharePolicy())
	sharing.Post("/links", h.Sharing.CreateShareLink())
	sharing.Post("/links/:id/revoke", h.Sharing.RevokeShareLink())
	sharing.Get("/resolve/:token", h.Sharing.ResolveShareLink())

	community := protected.Group("/community")
	community.Get("/profiles/:userId", h.Community.GetProfile())
	community.Patch("/profile", h.Community.UpdateMyProfile())
	community.Get("/profiles/:userId/activity", h.Community.ListActivity())

	moderation := protected.Group("/moderation")
	moderation.Post("/reviews", h.Moderation.CreateReview())
	moderation.Get("/reviews", h.Moderation.ListReviews())
	moderation.Patch("/reviews/:id/decision", h.Moderation.DecideReview())
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
