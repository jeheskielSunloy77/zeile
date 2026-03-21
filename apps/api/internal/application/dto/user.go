package dto

import "github.com/jeheskielSunloy77/kern/internal/domain"

type StoreUserInput struct {
	Email     string
	Username  string
	AvatarURL *string
	Password  string
	GoogleID  *string
}

func (d *StoreUserInput) ToModel() *domain.User {
	return &domain.User{
		Email:     d.Email,
		Username:  d.Username,
		AvatarURL: d.AvatarURL,
		GoogleID:  d.GoogleID,
	}
}

type UpdateUserInput struct {
	Email     *string
	Username  *string
	AvatarURL *string
	Password  *string
}

func (d *UpdateUserInput) ToModel() *domain.User {
	user := &domain.User{}
	if d.Email != nil {
		user.Email = *d.Email
	}
	if d.Username != nil {
		user.Username = *d.Username
	}
	if d.AvatarURL != nil {
		user.AvatarURL = d.AvatarURL
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
	if d.AvatarURL != nil {
		updates["avatar_url"] = *d.AvatarURL
	}
	if d.Password != nil {
		updates["password_hash"] = *d.Password
	}
	return updates
}
