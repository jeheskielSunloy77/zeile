package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Session struct {
	User             User      `json:"user"`
	AccessToken      string    `json:"accessToken"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshToken     string    `json:"refreshToken"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type SessionStore struct {
	path string
}

func NewSessionStore(path string) *SessionStore {
	return &SessionStore{path: path}
}

func (s *SessionStore) Load() (*Session, error) {
	if s == nil || s.path == "" {
		return nil, nil
	}

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, fmt.Errorf("decode session file: %w", err)
	}
	if session.AccessToken == "" {
		return nil, nil
	}
	return &session, nil
}

func (s *SessionStore) Save(session Session) error {
	if s == nil || s.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	session.UpdatedAt = time.Now().UTC()
	payload, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0o600); err != nil {
		return fmt.Errorf("write session temp file: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace session file: %w", err)
	}
	return nil
}

func (s *SessionStore) Clear() error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove session file: %w", err)
	}
	return nil
}

