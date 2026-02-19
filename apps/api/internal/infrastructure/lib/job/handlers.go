package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/email"
	"github.com/rs/zerolog"
)

var emailClient *email.Client

func (j *JobService) InitHandlers(config *config.Config, logger *zerolog.Logger) {
	emailClient = email.NewClient(config, logger)
}

func (j *JobService) handleEmailVerificationTask(ctx context.Context, t *asynq.Task) error {
	var p EmailVerificationPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal email verification payload: %w", err)
	}

	j.logger.Info().
		Str("type", "email_verification").
		Str("to", p.To).
		Msg("Processing email verification task")

	err := emailClient.SendEmailVerification(
		p.To,
		p.Username,
		p.Code,
		p.ExpiresInMinutes,
	)
	if err != nil {
		j.logger.Error().
			Str("type", "email_verification").
			Str("to", p.To).
			Err(err).
			Msg("Failed to send email verification email")
		return err
	}

	j.logger.Info().
		Str("type", "email_verification").
		Str("to", p.To).
		Msg("Successfully sent email verification email")
	return nil
}
