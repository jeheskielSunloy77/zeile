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

type DeviceAuthPollRequest struct {
	DeviceCode string `json:"deviceCode" validate:"required,min=16,max=256"`
}

func (d *DeviceAuthPollRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *DeviceAuthPollRequest) ToUsecase() dto.DeviceAuthPollInput {
	return dto.DeviceAuthPollInput{
		DeviceCode: d.DeviceCode,
	}
}

type DeviceAuthApproveRequest struct {
	UserCode string `json:"userCode" validate:"required,min=4,max=32"`
}

func (d *DeviceAuthApproveRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *DeviceAuthApproveRequest) ToUsecase() dto.DeviceAuthApproveInput {
	return dto.DeviceAuthApproveInput{
		UserCode: d.UserCode,
	}
}

type RefreshRequest struct {
	RefreshToken *string `json:"refreshToken" validate:"omitempty,min=32,max=256"`
}

func (d *RefreshRequest) Validate() error {
	return validator.New().Struct(d)
}

type LogoutRequest struct {
	RefreshToken *string `json:"refreshToken" validate:"omitempty,min=32,max=256"`
}

func (d *LogoutRequest) Validate() error {
	return validator.New().Struct(d)
}
