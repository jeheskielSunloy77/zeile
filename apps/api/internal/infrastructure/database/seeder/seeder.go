package seeder

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type Options struct {
	UsersCount int
}

func Run(ctx context.Context, db *gorm.DB, log *zerolog.Logger, opts Options) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}

	total := 0

	if opts.UsersCount > 0 {
		count, err := SeedUsers(ctx, db, opts.UsersCount)
		if err != nil {
			return fmt.Errorf("seed users: %w", err)
		}
		total += count
		if log != nil {
			log.Info().Int("count", count).Msg("seeded users")
		}
	}

	if log != nil {
		log.Info().Int("count", total).Msg("seed completed")
	}

	return nil
}
