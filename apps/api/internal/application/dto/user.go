package dto

import "github.com/jeheskielSunloy77/zeile/internal/domain"

type StoreUserInput struct {
	Email    string
	Username string
	Password string
	GoogleID *string
}

func (d *StoreUserInput) ToModel() *domain.User {
	return &domain.User{
		Email:    d.Email,
		Username: d.Username,
		GoogleID: d.GoogleID,
	}
}

type UpdateUserInput struct {
	Email    *string
	Username *string
	Password *string
}

func (d *UpdateUserInput) ToModel() *domain.User {
	user := &domain.User{}
	if d.Email != nil {
		user.Email = *d.Email
	}
	if d.Username != nil {
		user.Username = *d.Username
	}
	if d.Password != nil {
		user.PasswordHash = *d.Password
	}
	return user
}

func (d *UpdateUserInput) ToMap() map[string]any {
	updates := make(map[string]any)
	if d.Email != nil {
		updates["email"] = *d.Email
	}
	if d.Username != nil {
		updates["username"] = *d.Username
	}
	if d.Password != nil {
		updates["password_hash"] = *d.Password
	}
	return updates
}
