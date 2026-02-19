package application

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

//go:embed authorization_model.conf
var authorizationModelConf string

type AuthorizationSubject struct {
	ID      string `json:"id"`
	Email   string `json:"email,omitempty"`
	IsAdmin bool   `json:"is_admin"`
}

type AuthorizationObject struct {
	Route  string            `json:"route"`
	Path   string            `json:"path,omitempty"`
	Params map[string]string `json:"params,omitempty"`
	Query  map[string]string `json:"query,omitempty"`
}

type AuthorizationEnforcer interface {
	Enforce(rvals ...any) (bool, error)
}

type AuthorizationService struct {
	enforcer AuthorizationEnforcer
	logger   *zerolog.Logger
}

func NewAuthorizationService(db *gorm.DB, logger *zerolog.Logger) (*AuthorizationService, error) {
	if db == nil {
		return nil, errors.New("authorization: db is nil")
	}

	modelConf, err := model.NewModelFromString(authorizationModelConf)
	if err != nil {
		return nil, fmt.Errorf("authorization: load model: %w", err)
	}

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("authorization: init adapter: %w", err)
	}

	enforcer, err := casbin.NewSyncedEnforcer(modelConf, adapter)
	if err != nil {
		return nil, fmt.Errorf("authorization: init enforcer: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("authorization: load policy: %w", err)
	}

	enforcer.EnableAutoSave(true)

	return &AuthorizationService{
		enforcer: enforcer,
		logger:   logger,
	}, nil
}

func NewAuthorizationServiceWithEnforcer(enforcer AuthorizationEnforcer, logger *zerolog.Logger) *AuthorizationService {
	return &AuthorizationService{
		enforcer: enforcer,
		logger:   logger,
	}
}

func (a *AuthorizationService) Enforce(ctx context.Context, sub AuthorizationSubject, obj AuthorizationObject, act string) (bool, error) {
	_ = ctx
	if a == nil || a.enforcer == nil {
		return false, errors.New("authorization: enforcer not initialized")
	}

	allowed, err := a.enforcer.Enforce(sub, obj, act)
	if err != nil && a.logger != nil {
		a.logger.Error().Err(err).Msg("authorization enforcement failed")
	}
	return allowed, err
}
