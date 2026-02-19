package dto

import (
	"github.com/go-playground/validator/v10"
	"github.com/jeheskielSunloy77/zeile/internal/application/dto"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

func (d *RegisterRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *RegisterRequest) ToUsecase() dto.RegisterInput {
	return dto.RegisterInput{
		Email:    d.Email,
		Username: d.Username,
		Password: d.Password,
	}
}

type LoginRequest struct {
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password" validate:"required"`
}

func (d *LoginRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *LoginRequest) ToUsecase() dto.LoginInput {
	return dto.LoginInput{
		Identifier: d.Identifier,
		Password:   d.Password,
	}
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,min=4,max=10"`
}

func (d *VerifyEmailRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *VerifyEmailRequest) ToUsecase() dto.VerifyEmailInput {
	return dto.VerifyEmailInput{
		Email: d.Email,
		Code:  d.Code,
	}
}
