package repository

import (
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/cache"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
)

type Repositories = port.Repositories

func NewRepositories(s *server.Server, cacheClient cache.Cache) *Repositories {
	return &Repositories{
		Auth:              NewAuthRepository(s.DB.DB),
		AuthSession:       NewAuthSessionRepository(s.DB.DB),
		User:              NewUserRepository(s.Config, s.DB.DB, cacheClient),
		EmailVerification: NewEmailVerificationRepository(s.DB.DB),
	}
}
