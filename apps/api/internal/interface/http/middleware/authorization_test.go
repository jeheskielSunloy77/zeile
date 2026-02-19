package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/application"
	"github.com/stretchr/testify/require"
)

type stubAuthorizationService struct {
	allowed bool
	err     error
	called  bool
	subject application.AuthorizationSubject
	object  application.AuthorizationObject
	action  string
}

func (s *stubAuthorizationService) Enforce(_ context.Context, sub application.AuthorizationSubject, obj application.AuthorizationObject, act string) (bool, error) {
	s.called = true
	s.subject = sub
	s.object = obj
	s.action = act
	return s.allowed, s.err
}

func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			var httpErr *errs.ErrorResponse
			if errors.As(err, &httpErr) {
				return c.SendStatus(httpErr.Status)
			}
			return c.SendStatus(http.StatusInternalServerError)
		},
	})
}

func TestAuthorizationMiddleware(t *testing.T) {
	t.Run("missing user id", func(t *testing.T) {
		app := newTestApp()
		mw := NewAuthorizationMiddleware(&stubAuthorizationService{allowed: true})
		app.Get("/test", mw.RequireAuthorization(), func(c *fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("admin bypass", func(t *testing.T) {
		authorizer := &stubAuthorizationService{allowed: false}
		app := newTestApp()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals(UserIDKey, "admin-user")
			c.Locals(UserIsAdminKey, true)
			return c.Next()
		})
		mw := NewAuthorizationMiddleware(authorizer)
		app.Get("/test", mw.RequireAuthorization(), func(c *fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.False(t, authorizer.called)
	})

	t.Run("denied", func(t *testing.T) {
		authorizer := &stubAuthorizationService{allowed: false}
		app := newTestApp()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals(UserIDKey, "user-1")
			return c.Next()
		})
		mw := NewAuthorizationMiddleware(authorizer)
		app.Get("/test", mw.RequireAuthorization(), func(c *fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
		require.True(t, authorizer.called)
	})

	t.Run("error", func(t *testing.T) {
		authorizer := &stubAuthorizationService{err: errors.New("boom")}
		app := newTestApp()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals(UserIDKey, "user-1")
			return c.Next()
		})
		mw := NewAuthorizationMiddleware(authorizer)
		app.Get("/test", mw.RequireAuthorization(), func(c *fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		require.True(t, authorizer.called)
	})

	t.Run("allowed", func(t *testing.T) {
		authorizer := &stubAuthorizationService{allowed: true}
		app := newTestApp()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals(UserIDKey, "user-1")
			c.Locals(UserEmailKey, "user@example.com")
			return c.Next()
		})
		mw := NewAuthorizationMiddleware(authorizer)
		app.Get("/test/:id", mw.RequireAuthorization(), func(c *fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test/123?foo=bar", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.True(t, authorizer.called)
		require.Equal(t, "user-1", authorizer.subject.ID)
		require.Equal(t, "user@example.com", authorizer.subject.Email)
		require.Equal(t, "/test/:id", authorizer.object.Route)
		require.Equal(t, "/test/123", authorizer.object.Path)
		require.Equal(t, "GET", authorizer.action)
	})
}
