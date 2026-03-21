package application

import (
	"github.com/jeheskielSunloy77/kern/internal/application/port"
	"github.com/jeheskielSunloy77/kern/internal/infrastructure/lib/job"
	"github.com/jeheskielSunloy77/kern/internal/infrastructure/server"
)

type Services struct {
	Auth          AuthService
	User          UserService
	Library       LibraryService
	Community     CommunityService
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
	libraryService := NewLibraryService(repos.Library, s.Storage)
	communityService := NewCommunityService(repos.Library)
	authorizationService, err := NewAuthorizationService(s.DB.DB, s.Logger)
	if err != nil {
		return nil, err
	}

	return &Services{
		Job:           s.Job,
		Auth:          authService,
		User:          userService,
		Library:       libraryService,
		Community:     communityService,
		Authorization: authorizationService,
	}, nil
}
