package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubAuthorizationEnforcer struct {
	allowed bool
	err     error
	called  bool
	args    []interface{}
}

func (s *stubAuthorizationEnforcer) Enforce(rvals ...interface{}) (bool, error) {
	s.called = true
	s.args = append([]interface{}{}, rvals...)
	return s.allowed, s.err
}

func TestAuthorizationServiceEnforce(t *testing.T) {
	subject := AuthorizationSubject{ID: "user-1", Email: "user@example.com", IsAdmin: false}
	obj := AuthorizationObject{Route: "/users/:id", Path: "/users/123"}
	action := "GET"

	t.Run("returns allowed", func(t *testing.T) {
		enforcer := &stubAuthorizationEnforcer{allowed: true}
		svc := NewAuthorizationServiceWithEnforcer(enforcer, nil)

		allowed, err := svc.Enforce(context.Background(), subject, obj, action)

		require.NoError(t, err)
		require.True(t, allowed)
		require.True(t, enforcer.called)
		require.Len(t, enforcer.args, 3)
		require.Equal(t, subject, enforcer.args[0])
		require.Equal(t, obj, enforcer.args[1])
		require.Equal(t, action, enforcer.args[2])
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("boom")
		enforcer := &stubAuthorizationEnforcer{allowed: false, err: expectedErr}
		svc := NewAuthorizationServiceWithEnforcer(enforcer, nil)

		allowed, err := svc.Enforce(context.Background(), subject, obj, action)

		require.ErrorIs(t, err, expectedErr)
		require.False(t, allowed)
		require.True(t, enforcer.called)
	})

	t.Run("missing enforcer", func(t *testing.T) {
		svc := &AuthorizationService{}

		allowed, err := svc.Enforce(context.Background(), subject, obj, action)

		require.Error(t, err)
		require.False(t, allowed)
	})
}
