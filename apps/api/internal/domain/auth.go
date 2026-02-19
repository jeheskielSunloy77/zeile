package domain

import "github.com/golang-jwt/jwt/v5"

type AuthClaims struct {
	jwt.RegisteredClaims
	Email   string `json:"email,omitempty"`
	IsAdmin bool   `json:"is_admin"`
}
