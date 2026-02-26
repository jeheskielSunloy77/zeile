package handler

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/application"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	httpdto "github.com/jeheskielSunloy77/zeile/internal/interface/http/dto"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/middleware"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/response"
)

type AuthHandler struct {
	Handler
	authService application.AuthService
}

func NewAuthHandler(h Handler, authService application.AuthService) *AuthHandler {
	return &AuthHandler{
		Handler:     h,
		authService: authService,
	}
}

func (h *AuthHandler) Register() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, req *httpdto.RegisterRequest) (*domain.User, error) {
		result, err := h.authService.Register(c.UserContext(), req.ToUsecase(), c.Get(fiber.HeaderUserAgent), c.IP())
		if err != nil {
			return nil, err
		}
		h.setAuthCookies(c, result)
		return result.User, nil
	}, http.StatusCreated, &httpdto.RegisterRequest{})
}

func (h *AuthHandler) Login() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, req *httpdto.LoginRequest) (*domain.User, error) {
		identifier := req.Identifier
		if isEmail(identifier) {
			identifier = normalizeEmail(identifier)
		}

		input := req.ToUsecase()
		input.Identifier = identifier
		result, err := h.authService.Login(c.UserContext(), input, c.Get(fiber.HeaderUserAgent), c.IP())
		if err != nil {
			return nil, err
		}
		h.setAuthCookies(c, result)
		return result.User, nil
	}, http.StatusOK, &httpdto.LoginRequest{})
}

func (h *AuthHandler) GoogleLogin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start, err := h.authService.StartGoogleAuth(c.UserContext())
		if err != nil {
			return h.redirectGoogleFailure(c, err)
		}

		h.setGoogleStateCookie(c, start)
		return c.Redirect(start.AuthURL, http.StatusFound)
	}
}

func (h *AuthHandler) GoogleCallback() fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Query("code")
		state := c.Query("state")
		stateCookie := c.Cookies(googleStateCookieName)

		result, err := h.authService.CompleteGoogleAuth(
			c.UserContext(),
			code,
			state,
			stateCookie,
			c.Get(fiber.HeaderUserAgent),
			c.IP(),
		)

		h.clearGoogleStateCookie(c)

		if err != nil {
			return h.redirectGoogleFailure(c, err)
		}

		h.setAuthCookies(c, result)

		return c.Redirect(h.server.Config.Auth.GoogleSuccessRedirectURL, http.StatusFound)
	}
}

func (h *AuthHandler) VerifyEmail() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, req *httpdto.VerifyEmailRequest) (*domain.User, error) {
		return h.authService.VerifyEmail(c.UserContext(), req.ToUsecase())
	}, http.StatusOK, &httpdto.VerifyEmailRequest{})
}

func (h *AuthHandler) DeviceStart() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*application.DeviceAuthStartResult, error) {
		start, err := h.authService.StartDeviceAuth(c.UserContext())
		if err != nil {
			return nil, err
		}
		if start == nil {
			return nil, errs.NewInternalServerError()
		}

		verificationURI := "/api/v1/auth/device"
		if baseURL := strings.TrimSpace(c.BaseURL()); baseURL != "" {
			if parsedURL, err := url.Parse(baseURL); err == nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
				verificationURI = (&url.URL{
					Scheme: parsedURL.Scheme,
					Host:   parsedURL.Host,
					Path:   verificationURI,
				}).String()
			}
		}

		startCopy := *start
		startCopy.UserCode = strings.TrimSpace(startCopy.UserCode)
		startCopy.DeviceCode = strings.TrimSpace(startCopy.DeviceCode)
		startCopy.VerificationURI = verificationURI
		return &startCopy, nil
	}, http.StatusCreated, &httpdto.Empty{})
}

func (h *AuthHandler) DeviceApprovePage() fiber.Handler {
	return func(c *fiber.Ctx) error {
		const page = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Device Authorization</title>
  <style>
    :root { color-scheme: light dark; }
    body { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; max-width: 720px; margin: 4rem auto; padding: 0 1rem; line-height: 1.5; }
    input { width: 100%; padding: 0.75rem; font-size: 1rem; margin: 0.5rem 0 1rem; text-transform: uppercase; }
    button { padding: 0.75rem 1rem; font-size: 1rem; cursor: pointer; }
    .muted { opacity: 0.75; }
    .result { margin-top: 1rem; font-weight: 600; }
  </style>
</head>
<body>
  <h1>Approve Device</h1>
  <p class="muted">Sign in first (if needed), then enter the code shown in your terminal.</p>
  <label for="code">User code</label>
  <input id="code" placeholder="ABCD-EFGH" autocomplete="off" />
  <button id="approve">Approve</button>
  <p id="result" class="result"></p>
  <script>
    const codeInput = document.getElementById('code');
    const approveButton = document.getElementById('approve');
    const resultEl = document.getElementById('result');

    async function approve() {
      const userCode = (codeInput.value || '').trim().toUpperCase();
      if (!userCode) {
        resultEl.textContent = 'Enter a code first.';
        return;
      }

      approveButton.disabled = true;
      resultEl.textContent = 'Submitting...';

      try {
        const response = await fetch('/api/v1/auth/device/approve', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ userCode }),
        });

        let payload = {};
        try {
          payload = await response.json();
        } catch (e) {}

        if (!response.ok) {
          const message = (payload && payload.message) ? payload.message : 'Approval failed';
          resultEl.textContent = message + ' (status ' + response.status + ')';
          return;
        }

        resultEl.textContent = 'Device approved. Return to your terminal.';
      } finally {
        approveButton.disabled = false;
      }
    }

    approveButton.addEventListener('click', approve);
    codeInput.addEventListener('keydown', (event) => {
      if (event.key === 'Enter') {
        approve();
      }
    });
  </script>
</body>
</html>`
		c.Type("html")
		return c.Status(http.StatusOK).SendString(page)
	}
}

func (h *AuthHandler) DevicePoll() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, req *httpdto.DeviceAuthPollRequest) (*application.DeviceAuthPollResult, error) {
		return h.authService.PollDeviceAuth(c.UserContext(), req.ToUsecase(), c.Get(fiber.HeaderUserAgent), c.IP())
	}, http.StatusOK, &httpdto.DeviceAuthPollRequest{})
}

func (h *AuthHandler) DeviceApprove() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, req *httpdto.DeviceAuthApproveRequest) (*response.Response[any], error) {
		userID, err := h.parseUserID(c)
		if err != nil {
			return nil, err
		}

		if err := h.authService.ApproveDeviceAuth(c.UserContext(), userID, req.ToUsecase()); err != nil {
			return nil, err
		}

		resp := response.Response[any]{
			Status:  http.StatusOK,
			Success: true,
			Message: "Device approved successfully.",
		}
		return &resp, nil
	}, http.StatusOK, &httpdto.DeviceAuthApproveRequest{})
}

func (h *AuthHandler) Refresh() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, req *httpdto.RefreshRequest) (*domain.User, error) {
		refreshToken := c.Cookies(h.refreshCookieName())
		if req != nil && req.RefreshToken != nil && strings.TrimSpace(*req.RefreshToken) != "" {
			refreshToken = strings.TrimSpace(*req.RefreshToken)
		}
		result, err := h.authService.Refresh(c.UserContext(), refreshToken, c.Get(fiber.HeaderUserAgent), c.IP())
		if err != nil {
			return nil, err
		}
		h.setAuthCookies(c, result)
		return result.User, nil
	}, http.StatusOK, &httpdto.RefreshRequest{})
}

func (h *AuthHandler) Me() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*domain.User, error) {
		userID, err := h.parseUserID(c)
		if err != nil {
			return nil, err
		}
		return h.authService.CurrentUser(c.UserContext(), userID)
	}, http.StatusOK, &httpdto.Empty{})
}

func (h *AuthHandler) ResendVerification() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*response.Response[any], error) {
		userID, err := h.parseUserID(c)
		if err != nil {
			return nil, err
		}

		if err := h.authService.ResendVerification(c.UserContext(), userID); err != nil {
			return nil, err
		}

		resp := response.Response[any]{
			Status:  http.StatusOK,
			Success: true,
			Message: "Verification email sent if needed.",
		}
		return &resp, nil
	}, http.StatusOK, &httpdto.Empty{})
}

func (h *AuthHandler) Logout() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, req *httpdto.LogoutRequest) (*response.Response[any], error) {
		refreshToken := c.Cookies(h.refreshCookieName())
		if req != nil && req.RefreshToken != nil && strings.TrimSpace(*req.RefreshToken) != "" {
			refreshToken = strings.TrimSpace(*req.RefreshToken)
		}
		if err := h.authService.Logout(c.UserContext(), refreshToken); err != nil {
			return nil, err
		}
		h.clearAuthCookies(c)

		resp := response.Response[any]{
			Status:  http.StatusOK,
			Success: true,
			Message: "Logged out successfully.",
		}
		return &resp, nil
	}, http.StatusOK, &httpdto.LogoutRequest{})
}

func (h *AuthHandler) LogoutAll() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*response.Response[any], error) {
		userID, err := h.parseUserID(c)
		if err != nil {
			return nil, err
		}

		if err := h.authService.LogoutAll(c.UserContext(), userID); err != nil {
			return nil, err
		}
		h.clearAuthCookies(c)

		resp := response.Response[any]{
			Status:  http.StatusOK,
			Success: true,
			Message: "Logged out from all sessions.",
		}
		return &resp, nil
	}, http.StatusOK, &httpdto.Empty{})
}

func (h *AuthHandler) parseUserID(c *fiber.Ctx) (uuid.UUID, error) {
	raw := middleware.GetUserID(c)
	if raw == "" {
		return uuid.Nil, errs.NewUnauthorizedError("Unauthorized", false)
	}
	userID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, errs.NewUnauthorizedError("Unauthorized", false)
	}
	return userID, nil
}

func (h *AuthHandler) setAuthCookies(c *fiber.Ctx, result *application.AuthResult) {
	if result == nil {
		return
	}

	sameSite := string(h.server.Config.Auth.CookieSameSite)
	secure := h.server.Config.Primary.Env == config.EnvProduction

	accessCookie := &fiber.Cookie{
		Name:     h.accessCookieName(),
		Value:    result.Token.Token,
		Expires:  result.Token.ExpiresAt,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
		Domain:   h.server.Config.Auth.CookieDomain,
	}
	refreshCookie := &fiber.Cookie{
		Name:     h.refreshCookieName(),
		Value:    result.RefreshToken.Token,
		Expires:  result.RefreshToken.ExpiresAt,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
		Domain:   h.server.Config.Auth.CookieDomain,
	}

	c.Cookie(accessCookie)
	c.Cookie(refreshCookie)
}

func (h *AuthHandler) clearAuthCookies(c *fiber.Ctx) {
	sameSite := string(h.server.Config.Auth.CookieSameSite)
	secure := h.server.Config.Primary.Env == config.EnvProduction
	expired := time.Unix(0, 0)

	c.Cookie(&fiber.Cookie{
		Name:     h.accessCookieName(),
		Value:    "",
		Expires:  expired,
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
		Domain:   h.server.Config.Auth.CookieDomain,
	})
	c.Cookie(&fiber.Cookie{
		Name:     h.refreshCookieName(),
		Value:    "",
		Expires:  expired,
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/",
		Domain:   h.server.Config.Auth.CookieDomain,
	})
}

func (h *AuthHandler) accessCookieName() string {
	if h.server != nil && h.server.Config.Auth.AccessCookieName != "" {
		return h.server.Config.Auth.AccessCookieName
	}
	return "access_token"
}

func (h *AuthHandler) refreshCookieName() string {
	if h.server != nil && h.server.Config.Auth.RefreshCookieName != "" {
		return h.server.Config.Auth.RefreshCookieName
	}
	return "refresh_token"
}

const googleStateCookieName = "google_auth_state"

func (h *AuthHandler) setGoogleStateCookie(c *fiber.Ctx, start *application.GoogleAuthStart) {
	if start == nil {
		return
	}

	sameSite := string(h.server.Config.Auth.CookieSameSite)
	secure := h.server.Config.Primary.Env == config.EnvProduction

	c.Cookie(&fiber.Cookie{
		Name:     googleStateCookieName,
		Value:    start.StateCookie,
		Expires:  start.StateExpiresAt,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/api/v1/auth/google",
		Domain:   h.server.Config.Auth.CookieDomain,
	})
}

func (h *AuthHandler) clearGoogleStateCookie(c *fiber.Ctx) {
	sameSite := string(h.server.Config.Auth.CookieSameSite)
	secure := h.server.Config.Primary.Env == config.EnvProduction
	expired := time.Unix(0, 0)

	c.Cookie(&fiber.Cookie{
		Name:     googleStateCookieName,
		Value:    "",
		Expires:  expired,
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Path:     "/api/v1/auth/google",
		Domain:   h.server.Config.Auth.CookieDomain,
	})
}

func (h *AuthHandler) defaultAuthOrigin() string {
	if h.server == nil || h.server.Config == nil {
		return ""
	}

	if len(h.server.Config.Server.CORSAllowedOrigins) == 0 {
		return ""
	}

	origin := strings.TrimSpace(h.server.Config.Server.CORSAllowedOrigins[0])
	if origin == "" || origin == "*" {
		return ""
	}

	return origin
}

func (h *AuthHandler) redirectGoogleFailure(c *fiber.Ctx, err error) error {
	middleware.GetLogger(c).Warn().Err(err).Msg("google auth failed")
	redirectURL := appendQueryParam(h.server.Config.Auth.GoogleFailureRedirectURL, "error", "google_auth_failed")
	return c.Redirect(redirectURL, http.StatusFound)
}

func appendQueryParam(rawURL, key, value string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()

	return parsed.String()
}

func isEmail(identifier string) bool {
	emailRegex := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	return emailRegex.MatchString(identifier)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
