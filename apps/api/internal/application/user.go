package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/app/sqlerr"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	ResourceService[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput]
}

type userService struct {
	ResourceService[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput]
	repo port.UserRepository
}

func NewUserService(repo port.UserRepository) UserService {
	return &userService{
		ResourceService: NewResourceService[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput]("user", repo),
		repo:            repo,
	}
}

func (s *userService) Store(ctx context.Context, dto *applicationdto.StoreUserInput) (*domain.User, error) {
	user := dto.ToModel()

	if dto.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, errs.NewInternalServerError()
		}
		user.PasswordHash = string(hash)
	}

	if err := s.repo.Store(ctx, user); err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return user, nil
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, dto *applicationdto.UpdateUserInput) (*domain.User, error) {
	if dto == nil {
		return s.GetByID(ctx, id, nil)
	}

	updates := dto.ToMap()

	if email, ok := updates["email"].(string); ok {
		email = normalizeEmail(email)
		if email == "" {
			delete(updates, "email")
		} else {
			updates["email"] = email
		}
	}

	if username, ok := updates["username"].(string); ok {
		username = strings.TrimSpace(username)
		if username == "" {
			delete(updates, "username")
		} else {
			updates["username"] = username
		}
	}

	if password, ok := updates["password_hash"].(string); ok {
		if password == "" {
			delete(updates, "password_hash")
		} else {
			if len(password) < minPasswordLength {
				return nil, errs.NewBadRequestError(
					fmt.Sprintf("Password must be at least %d characters", minPasswordLength),
					true,
					[]errs.FieldError{{Field: "password", Error: "too short"}},
					nil,
				)
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return nil, errs.NewInternalServerError()
			}
			updates["password_hash"] = string(hash)
		}
	}

	entity, err := s.repo.GetByID(ctx, id, nil)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	if len(updates) == 0 {
		return entity, nil
	}

	updatedUser, err := s.repo.Update(ctx, *entity, updates)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return updatedUser, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
