package seeder

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	DefaultUserCount   = 10
	defaultPasswordRaw = "Password123!"
)

func SeedUsers(ctx context.Context, db *gorm.DB, count int) (int, error) {
	if count <= 0 {
		return 0, nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPasswordRaw), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	users := make([]domain.User, 0, count)
	for i := range count {
		username := normalizeUsername(faker.Username(), i)
		email := uniqueEmail(faker.Email(), i)
		users = append(users, domain.User{
			ID:           uuid.New(),
			Email:        email,
			Username:     username,
			PasswordHash: string(passwordHash),
		})
	}

	if err := db.WithContext(ctx).Create(&users).Error; err != nil {
		return 0, err
	}

	return len(users), nil
}

func uniqueEmail(email string, idx int) string {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return fmt.Sprintf("user%d@example.com", idx)
	}
	if !strings.Contains(email, "@") {
		return fmt.Sprintf("user%d@example.com", idx)
	}
	return strings.Replace(email, "@", fmt.Sprintf("+%d@", idx), 1)
}

func normalizeUsername(username string, idx int) string {
	username = strings.ToLower(strings.TrimSpace(username))
	username = strings.ReplaceAll(username, " ", "_")
	if username == "" {
		username = fmt.Sprintf("user%d", idx)
	}

	suffix := fmt.Sprintf("_%d", idx)
	maxBaseLen := max(50-len(suffix), 1)
	if len(username) > maxBaseLen {
		username = username[:maxBaseLen]
	}
	username = username + suffix
	if len(username) < 3 {
		username = "user" + suffix
	}
	return username
}
