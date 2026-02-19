package handler

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/middleware"
	"github.com/rs/zerolog"

	"github.com/stretchr/testify/require"
)

func newTestServer() *server.Server {
	logger := zerolog.New(io.Discard)
	cfg := &config.Config{
		Primary: config.Primary{Env: config.EnvDevelopment},
		Server: config.ServerConfig{
			Port:               "0",
			ReadTimeout:        time.Second,
			WriteTimeout:       time.Second,
			IdleTimeout:        time.Second,
			CORSAllowedOrigins: []string{"*"},
		},
		Auth: config.AuthConfig{
			CookieSameSite:    config.CookieSameSiteLax,
			AccessCookieName:  "access_token",
			RefreshCookieName: "refresh_token",
		},
	}

	return &server.Server{Config: cfg, Logger: &logger}
}

func newTestApp(srv *server.Server) *fiber.App {
	global := middleware.NewGlobalMiddlewares(srv)
	return fiber.New(fiber.Config{ErrorHandler: global.GlobalErrorHandler})
}

func mustJSON(t *testing.T, v any) []byte {
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
