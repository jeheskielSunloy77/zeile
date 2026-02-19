package dto

import (
	"github.com/go-playground/validator/v10"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
)

type StoreUserRequest struct {
	Email    string  `json:"email" validate:"required,email"`
	Username string  `json:"username" validate:"required,min=3,max=50"`
	Password string  `json:"password" validate:"min=8,max=128"`
	GoogleID *string `json:"googleId" validate:"omitempty"`
}

func (d *StoreUserRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *StoreUserRequest) ToUsecase() *applicationdto.StoreUserInput {
	return &applicationdto.StoreUserInput{
		Email:    d.Email,
		Username: d.Username,
		Password: d.Password,
		GoogleID: d.GoogleID,
	}
}

type UpdateUserRequest struct {
	Email    *string `json:"email" validate:"omitempty,email"`
	Username *string `json:"username" validate:"omitempty,min=3,max=50"`
	Password *string `json:"password" validate:"omitempty,min=8,max=128"`
}

func (d *UpdateUserRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *UpdateUserRequest) ToUsecase() *applicationdto.UpdateUserInput {
	return &applicationdto.UpdateUserInput{
		Email:    d.Email,
		Username: d.Username,
		Password: d.Password,
	}
}
