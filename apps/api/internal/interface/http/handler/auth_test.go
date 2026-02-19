package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/application"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/middleware"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/response"

	"github.com/stretchr/testify/require"
)

type stubAuthService struct {
	registerFn           func(ctx context.Context, input applicationdto.RegisterInput, userAgent, ipAddress string) (*application.AuthResult, error)
	loginFn              func(ctx context.Context, input applicationdto.LoginInput, userAgent, ipAddress string) (*application.AuthResult, error)
	startGoogleAuthFn    func(ctx context.Context) (*application.GoogleAuthStart, error)
	completeGoogleAuthFn func(ctx context.Context, code, state, stateCookie, userAgent, ipAddress string) (*application.AuthResult, error)
	verifyEmailFn        func(ctx context.Context, input applicationdto.VerifyEmailInput) (*domain.User, error)
	refreshFn            func(ctx context.Context, refreshToken, userAgent, ipAddress string) (*application.AuthResult, error)
	logoutFn             func(ctx context.Context, refreshToken string) error
	logoutAllFn          func(ctx context.Context, userID uuid.UUID) error
	currentUserFn        func(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	resendVerificationFn func(ctx context.Context, userID uuid.UUID) error
}

func (s *stubAuthService) Register(ctx context.Context, input applicationdto.RegisterInput, userAgent, ipAddress string) (*application.AuthResult, error) {
	if s.registerFn != nil {
		return s.registerFn(ctx, input, userAgent, ipAddress)
	}
	return nil, nil
}

func (s *stubAuthService) Login(ctx context.Context, input applicationdto.LoginInput, userAgent, ipAddress string) (*application.AuthResult, error) {
	if s.loginFn != nil {
		return s.loginFn(ctx, input, userAgent, ipAddress)
	}
	return nil, nil
}

func (s *stubAuthService) StartGoogleAuth(ctx context.Context) (*application.GoogleAuthStart, error) {
	if s.startGoogleAuthFn != nil {
		return s.startGoogleAuthFn(ctx)
	}
	return nil, nil
}

func (s *stubAuthService) CompleteGoogleAuth(ctx context.Context, code, state, stateCookie, userAgent, ipAddress string) (*application.AuthResult, error) {
	if s.completeGoogleAuthFn != nil {
		return s.completeGoogleAuthFn(ctx, code, state, stateCookie, userAgent, ipAddress)
	}
	return nil, nil
}

func (s *stubAuthService) VerifyEmail(ctx context.Context, input applicationdto.VerifyEmailInput) (*domain.User, error) {
	if s.verifyEmailFn != nil {
		return s.verifyEmailFn(ctx, input)
	}
	return nil, nil
}

func (s *stubAuthService) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (*application.AuthResult, error) {
	if s.refreshFn != nil {
		return s.refreshFn(ctx, refreshToken, userAgent, ipAddress)
	}
	return nil, nil
}

func (s *stubAuthService) Logout(ctx context.Context, refreshToken string) error {
	if s.logoutFn != nil {
		return s.logoutFn(ctx, refreshToken)
	}
	return nil
}

func (s *stubAuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if s.logoutAllFn != nil {
		return s.logoutAllFn(ctx, userID)
	}
	return nil
}

func (s *stubAuthService) CurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	if s.currentUserFn != nil {
		return s.currentUserFn(ctx, userID)
	}
	return nil, nil
}

func (s *stubAuthService) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	if s.resendVerificationFn != nil {
		return s.resendVerificationFn(ctx, userID)
	}
	return nil
}

// Ensures Register returns validation errors without invoking the application.
func TestAuthHandlerRegister_ValidationError(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	called := false
	authService := &stubAuthService{
		registerFn: func(ctx context.Context, input applicationdto.RegisterInput, userAgent, ipAddress string) (*application.AuthResult, error) {
			called = true
			return nil, nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/register", h.Register())

	req, err := http.NewRequest(http.MethodPost, "/register", bytes.NewReader(mustJSON(t, map[string]any{})))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.False(t, called)
}

// Ensures Register returns a 201 response with the auth payload on success.
func TestAuthHandlerRegister_Success(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	userID := uuid.New()
	authService := &stubAuthService{
		registerFn: func(ctx context.Context, input applicationdto.RegisterInput, userAgent, ipAddress string) (*application.AuthResult, error) {
			return &application.AuthResult{
				User:         &domain.User{ID: userID, Email: input.Email, Username: input.Username},
				Token:        application.AuthToken{Token: "token", ExpiresAt: time.Now().Add(time.Hour)},
				RefreshToken: application.AuthToken{Token: "refresh", ExpiresAt: time.Now().Add(24 * time.Hour)},
			}, nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/register", h.Register())

	body := mustJSON(t, map[string]any{
		"email":    "user@example.com",
		"username": "user",
		"password": "password123",
	})

	req, err := http.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var got domain.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, userID, got.ID)
	require.Equal(t, "user@example.com", got.Email)
}

// Ensures Login normalizes email identifiers and maps auth errors to HTTP responses.
func TestAuthHandlerLogin_NormalizesEmail(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	var gotIdentifier string
	authService := &stubAuthService{
		loginFn: func(ctx context.Context, input applicationdto.LoginInput, userAgent, ipAddress string) (*application.AuthResult, error) {
			gotIdentifier = input.Identifier
			return nil, errs.NewUnauthorizedError("Invalid credentials", true)
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/login", h.Login())

	body := mustJSON(t, map[string]any{
		"identifier": "USER@Example.COM",
		"password":   "password123",
	})

	req, err := http.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	fmt.Printf("comparing user@example.com to %s", gotIdentifier)
	require.Equal(t, "user@example.com", gotIdentifier)
}

// Ensures Google login start redirects to the provider and sets a state cookie.
func TestAuthHandlerGoogleLogin_RedirectsToProvider(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	authService := &stubAuthService{
		startGoogleAuthFn: func(ctx context.Context) (*application.GoogleAuthStart, error) {
			return &application.GoogleAuthStart{
				AuthURL:        "https://accounts.google.com/o/oauth2/auth?state=abc",
				StateCookie:    "cookie-value",
				StateExpiresAt: time.Now().Add(10 * time.Minute),
			}, nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Get("/google", h.GoogleLogin())

	req, err := http.NewRequest(http.MethodGet, "/google", nil)
	require.NoError(t, err)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, "https://accounts.google.com/o/oauth2/auth?state=abc", resp.Header.Get("Location"))

	found := false
	for _, header := range resp.Header["Set-Cookie"] {
		if strings.HasPrefix(header, googleStateCookieName+"=") {
			found = true
			break
		}
	}
	require.True(t, found)
}

// Ensures Google callback sets auth cookies and redirects to the web success URL.
func TestAuthHandlerGoogleCallback_RedirectsToSuccess(t *testing.T) {
	srv := newTestServer()
	srv.Config.Auth.GoogleSuccessRedirectURL = "http://localhost:3000/auth/me"
	app := newTestApp(srv)

	userID := uuid.New()
	authService := &stubAuthService{
		completeGoogleAuthFn: func(ctx context.Context, code, state, stateCookie, userAgent, ipAddress string) (*application.AuthResult, error) {
			require.Equal(t, "code", code)
			require.Equal(t, "state", state)
			require.Equal(t, "cookie-value", stateCookie)
			return &application.AuthResult{
				User:         &domain.User{ID: userID, Email: "user@example.com"},
				Token:        application.AuthToken{Token: "token", ExpiresAt: time.Now().Add(time.Hour)},
				RefreshToken: application.AuthToken{Token: "refresh", ExpiresAt: time.Now().Add(24 * time.Hour)},
			}, nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Get("/google/callback", h.GoogleCallback())

	req, err := http.NewRequest(http.MethodGet, "/google/callback?code=code&state=state", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: googleStateCookieName, Value: "cookie-value"})

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, "http://localhost:3000/auth/me", resp.Header.Get("Location"))

	cookieHeaders := strings.Join(resp.Header["Set-Cookie"], "; ")
	require.Contains(t, cookieHeaders, "access_token=")
	require.Contains(t, cookieHeaders, "refresh_token=")
}

// Ensures VerifyEmail validates required fields.
func TestAuthHandlerVerifyEmail_ValidationError(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	called := false
	authService := &stubAuthService{
		verifyEmailFn: func(ctx context.Context, input applicationdto.VerifyEmailInput) (*domain.User, error) {
			called = true
			return nil, nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/verify-email", h.VerifyEmail())

	req, err := http.NewRequest(http.MethodPost, "/verify-email", bytes.NewReader(mustJSON(t, map[string]any{})))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.False(t, called)
}

// Ensures VerifyEmail returns a user on success.
func TestAuthHandlerVerifyEmail_Success(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	userID := uuid.New()
	authService := &stubAuthService{
		verifyEmailFn: func(ctx context.Context, input applicationdto.VerifyEmailInput) (*domain.User, error) {
			return &domain.User{ID: userID, Email: input.Email, Username: "user"}, nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/verify-email", h.VerifyEmail())

	body := mustJSON(t, map[string]any{
		"email": "user@example.com",
		"code":  "123456",
	})

	req, err := http.NewRequest(http.MethodPost, "/verify-email", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var got domain.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, userID, got.ID)
	require.Equal(t, "user@example.com", got.Email)
}

// Ensures Refresh pulls the refresh cookie and sets new auth cookies.
func TestAuthHandlerRefresh_Success(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	userID := uuid.New()
	refreshToken := "refresh-token"
	var gotToken string
	authService := &stubAuthService{
		refreshFn: func(ctx context.Context, token, userAgent, ipAddress string) (*application.AuthResult, error) {
			gotToken = token
			return &application.AuthResult{
				User:         &domain.User{ID: userID, Email: "user@example.com"},
				Token:        application.AuthToken{Token: "access", ExpiresAt: time.Now().Add(time.Hour)},
				RefreshToken: application.AuthToken{Token: "refresh-new", ExpiresAt: time.Now().Add(24 * time.Hour)},
			}, nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/refresh", h.Refresh())

	req, err := http.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(mustJSON(t, map[string]any{})))
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, refreshToken, gotToken)

	var got domain.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, userID, got.ID)

	require.NotNil(t, cookieByName(resp.Cookies(), "access_token"))
	require.NotNil(t, cookieByName(resp.Cookies(), "refresh_token"))
}

// Ensures Logout clears cookies and forwards the refresh token.
func TestAuthHandlerLogout_Success(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	refreshToken := "refresh-token"
	var gotToken string
	authService := &stubAuthService{
		logoutFn: func(ctx context.Context, token string) error {
			gotToken = token
			return nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/logout", h.Logout())

	req, err := http.NewRequest(http.MethodPost, "/logout", bytes.NewReader(mustJSON(t, map[string]any{})))
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, refreshToken, gotToken)

	var got response.Response[any]
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, "Logged out successfully.", got.Message)

	accessCookie := cookieByName(resp.Cookies(), "access_token")
	refreshCookie := cookieByName(resp.Cookies(), "refresh_token")
	require.NotNil(t, accessCookie)
	require.NotNil(t, refreshCookie)
	require.Empty(t, accessCookie.Value)
	require.Empty(t, refreshCookie.Value)
}

// Ensures Me rejects requests without a user ID in context.
func TestAuthHandlerMe_MissingUserID(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	called := false
	authService := &stubAuthService{
		currentUserFn: func(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
			called = true
			return nil, nil
		},
	}

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Get("/me", h.Me())

	req, err := http.NewRequest(http.MethodGet, "/me", nil)
	require.NoError(t, err)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.False(t, called)
}

// Ensures Me returns the current user when authenticated.
func TestAuthHandlerMe_Success(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	userID := uuid.New()
	authService := &stubAuthService{
		currentUserFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			require.Equal(t, userID, id)
			return &domain.User{ID: userID, Email: "user@example.com"}, nil
		},
	}

	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.UserIDKey, userID.String())
		return c.Next()
	})

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Get("/me", h.Me())

	req, err := http.NewRequest(http.MethodGet, "/me", nil)
	require.NoError(t, err)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var got domain.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, userID, got.ID)
}

// Ensures ResendVerification uses the user ID from context.
func TestAuthHandlerResendVerification_Success(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	userID := uuid.New()
	var gotID uuid.UUID
	authService := &stubAuthService{
		resendVerificationFn: func(ctx context.Context, id uuid.UUID) error {
			gotID = id
			return nil
		},
	}

	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.UserIDKey, userID.String())
		return c.Next()
	})

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/resend-verification", h.ResendVerification())

	req, err := http.NewRequest(http.MethodPost, "/resend-verification", bytes.NewReader(mustJSON(t, map[string]any{})))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, userID, gotID)

	var got response.Response[any]
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, "Verification email sent if needed.", got.Message)
}

// Ensures LogoutAll revokes sessions and clears cookies.
func TestAuthHandlerLogoutAll_Success(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	userID := uuid.New()
	var gotID uuid.UUID
	authService := &stubAuthService{
		logoutAllFn: func(ctx context.Context, id uuid.UUID) error {
			gotID = id
			return nil
		},
	}

	app.Use(func(c *fiber.Ctx) error {
		c.Locals(middleware.UserIDKey, userID.String())
		return c.Next()
	})

	h := NewAuthHandler(NewHandler(srv), authService)
	app.Post("/logout-all", h.LogoutAll())

	req, err := http.NewRequest(http.MethodPost, "/logout-all", bytes.NewReader(mustJSON(t, map[string]any{})))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, userID, gotID)

	var got response.Response[any]
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, "Logged out from all sessions.", got.Message)

	require.NotNil(t, cookieByName(resp.Cookies(), "access_token"))
	require.NotNil(t, cookieByName(resp.Cookies(), "refresh_token"))
}

func cookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
