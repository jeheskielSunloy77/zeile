package application

import (
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/job"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/server"
)

type Services struct {
	Auth          AuthService
	User          UserService
	Authorization *AuthorizationService
	Job           *job.JobService
}

func NewServices(s *server.Server, repos *port.Repositories) (*Services, error) {
	var enqueuer TaskEnqueuer
	if s.Job != nil {
		enqueuer = s.Job.Client
	}
	authService := NewAuthService(&s.Config.Auth, repos.Auth, repos.AuthSession, repos.EmailVerification, enqueuer, s.Logger)
	userService := NewUserService(repos.User)
	authorizationService, err := NewAuthorizationService(s.DB.DB, s.Logger)
	if err != nil {
		return nil, err
	}

	return &Services{
		Job:           s.Job,
		Auth:          authService,
		User:          userService,
		Authorization: authorizationService,
	}, nil
}
