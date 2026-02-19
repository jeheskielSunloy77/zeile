package job

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TaskEmailVerification = "email:verification"
)

type EmailVerificationPayload struct {
	To               string `json:"to"`
	Username         string `json:"username"`
	Code             string `json:"code"`
	ExpiresInMinutes int    `json:"expires_in_minutes"`
}

func NewEmailVerificationTask(payload EmailVerificationPayload) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TaskEmailVerification, payloadBytes,
		asynq.MaxRetry(3),
		asynq.Queue("default"),
		asynq.Timeout(30*time.Second)), nil
}
