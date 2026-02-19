package dto

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
