package application

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zeile/tui/internal/infrastructure/config"
	"github.com/zeile/tui/internal/infrastructure/remote"
	"github.com/zeile/tui/internal/infrastructure/storage"
)

type AuthService struct {
	client  *remote.Client
	store   *remote.SessionStore
	session *remote.Session

	mu sync.RWMutex
}

func NewAuthService(cfg config.Config, paths storage.Paths) (*AuthService, error) {
	client := remote.NewClient(cfg.APIBaseURL)
	store := remote.NewSessionStore(filepath.Join(paths.BaseDir, "auth-session.json"))

	svc := &AuthService{
		client: client,
		store:  store,
	}

	session, err := store.Load()
	if err != nil {
		return nil, err
	}
	if session != nil {
		svc.session = session
	}

	return svc, nil
}

func (s *AuthService) Enabled() bool {
	if s == nil || s.client == nil {
		return false
	}
	return s.client.Enabled()
}

func (s *AuthService) StartDeviceAuth(ctx context.Context) (remote.DeviceAuthStartResponse, error) {
	if !s.Enabled() {
		return remote.DeviceAuthStartResponse{}, fmt.Errorf("remote API is not configured")
	}
	return s.client.StartDeviceAuth(ctx)
}

func (s *AuthService) PollDeviceAuth(ctx context.Context, deviceCode string) (remote.DeviceAuthPollResponse, error) {
	if !s.Enabled() {
		return remote.DeviceAuthPollResponse{}, fmt.Errorf("remote API is not configured")
	}

	result, err := s.client.PollDeviceAuth(ctx, deviceCode)
	if err != nil {
		return result, err
	}

	if result.Status == "approved" && result.User != nil && result.Token != nil && result.RefreshToken != nil {
		session := remote.Session{
			User:             *result.User,
			AccessToken:      result.Token.Token,
			AccessExpiresAt:  result.Token.ExpiresAt.UTC(),
			RefreshToken:     result.RefreshToken.Token,
			RefreshExpiresAt: result.RefreshToken.ExpiresAt.UTC(),
		}
		if err := s.store.Save(session); err != nil {
			return remote.DeviceAuthPollResponse{}, err
		}
		s.mu.Lock()
		s.session = &session
		s.mu.Unlock()
	}

	return result, nil
}

func (s *AuthService) Disconnect(ctx context.Context) error {
	if s == nil {
		return nil
	}

	session, ok := s.Session()
	if ok && s.client != nil && s.client.Enabled() && strings.TrimSpace(session.RefreshToken) != "" {
		_ = s.client.Logout(ctx, session.RefreshToken)
	}

	s.mu.Lock()
	s.session = nil
	s.mu.Unlock()

	return s.store.Clear()
}

func (s *AuthService) Session() (remote.Session, bool) {
	if s == nil {
		return remote.Session{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.session == nil {
		return remote.Session{}, false
	}
	return *s.session, true
}

func (s *AuthService) IsConnected() bool {
	session, ok := s.Session()
	if !ok {
		return false
	}
	return strings.TrimSpace(session.AccessToken) != "" && session.AccessExpiresAt.After(time.Now().UTC())
}

func (s *AuthService) ConnectionLabel() string {
	session, ok := s.Session()
	if !ok {
		return "Local-only"
	}
	if !session.AccessExpiresAt.After(time.Now().UTC()) {
		return "Local-only"
	}
	if strings.TrimSpace(session.User.Username) != "" {
		return "Connected: @" + session.User.Username
	}
	if strings.TrimSpace(session.User.Email) != "" {
		return "Connected: " + session.User.Email
	}
	if strings.TrimSpace(session.User.ID) != "" {
		return "Connected: " + session.User.ID
	}
	return "Connected"
}
