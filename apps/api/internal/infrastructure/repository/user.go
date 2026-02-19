package repository

import (
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/cache"
	"gorm.io/gorm"
)

type UserRepository = port.UserRepository

type userRepository struct {
	ResourceRepository[domain.User]
}

func NewUserRepository(cfg *config.Config, db *gorm.DB, cacheClient cache.Cache) UserRepository {
	return &userRepository{
		ResourceRepository: NewResourceRepository[domain.User](cfg, db, cacheClient),
	}
}
