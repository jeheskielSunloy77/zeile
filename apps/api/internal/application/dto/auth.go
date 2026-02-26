package dto

import "strings"

type RegisterInput struct {
	Email    string
	Username string
	Password string
}

type LoginInput struct {
	Identifier string
	Password   string
}

type VerifyEmailInput struct {
	Email string
	Code  string
}

type DeviceAuthPollInput struct {
	DeviceCode string
}

func (d DeviceAuthPollInput) Normalized() DeviceAuthPollInput {
	d.DeviceCode = strings.TrimSpace(d.DeviceCode)
	return d
}

type DeviceAuthApproveInput struct {
	UserCode string
}

func (d DeviceAuthApproveInput) Normalized() DeviceAuthApproveInput {
	d.UserCode = strings.TrimSpace(strings.ToUpper(d.UserCode))
	return d
}
