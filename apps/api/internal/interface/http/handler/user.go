package handler

import (
	"github.com/jeheskielSunloy77/zeile/internal/application"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	httpdto "github.com/jeheskielSunloy77/zeile/internal/interface/http/dto"
)

type UserHandler struct {
	*ResourceHandler[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput, *httpdto.StoreUserRequest, *httpdto.UpdateUserRequest]
}

func NewUserHandler(h Handler, service application.UserService) *UserHandler {
	return &UserHandler{
		ResourceHandler: NewResourceHandler[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput, *httpdto.StoreUserRequest, *httpdto.UpdateUserRequest]("user", h, service),
	}
}
