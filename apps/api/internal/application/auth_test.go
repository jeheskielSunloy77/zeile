package application

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/job"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"

	"github.com/stretchr/testify/require"
)

type mockAuthRepo struct {
	createUserFn            func(ctx context.Context, user *domain.User) error
	getByIDFn               func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	getByEmailFn            func(ctx context.Context, email string) (*domain.User, error)
	getByUsernameFn         func(ctx context.Context, username string) (*domain.User, error)
	getByGoogleIDFn         func(ctx context.Context, googleID string) (*domain.User, error)
	saveFn                  func(ctx context.Context, user *domain.User) error
	updateLoginAtFn         func(ctx context.Context, id uuid.UUID, ts time.Time) error
	updateEmailVerifiedAtFn func(ctx context.Context, id uuid.UUID, ts time.Time) error
}

type mockVerificationRepo struct {
	createFn       func(ctx context.Context, verification *domain.EmailVerification) error
	getActiveFn    func(ctx context.Context, userID uuid.UUID, codeHash string, now time.Time) (*domain.EmailVerification, error)
	expireActiveFn func(ctx context.Context, userID uuid.UUID, now time.Time) error
	markVerifiedFn func(ctx context.Context, id uuid.UUID, verifiedAt time.Time) error
}

type mockSessionRepo struct {
	createFn         func(ctx context.Context, session *domain.AuthSession) error
	getByHashFn      func(ctx context.Context, hash string) (*domain.AuthSession, error)
	revokeByIDFn     func(ctx context.Context, id uuid.UUID, revokedAt time.Time) error
	revokeByUserIDFn func(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type mockTaskEnqueuer struct {
	called bool
	task   *asynq.Task
}

type mockOAuthConfig struct {
	authURL    string
	state      string
	exchangeFn func(ctx context.Context, code string) (*oauth2.Token, error)
}

func (m *mockOAuthConfig) AuthCodeURL(state string, _ ...oauth2.AuthCodeOption) string {
	m.state = state
	return m.authURL
}

func (m *mockOAuthConfig) Exchange(ctx context.Context, code string, _ ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	if m.exchangeFn != nil {
		return m.exchangeFn(ctx, code)
	}
	return nil, nil
}

func (m *mockTaskEnqueuer) EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	m.called = true
	m.task = task
	return &asynq.TaskInfo{ID: "task-id"}, nil
}

func (m *mockAuthRepo) Save(ctx context.Context, user *domain.User) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, user)
	}
	return nil
}

func (m *mockAuthRepo) CreateUser(ctx context.Context, user *domain.User) error {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, user)
	}
	return nil
}

func (m *mockAuthRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.getByUsernameFn != nil {
		return m.getByUsernameFn(ctx, username)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	if m.getByGoogleIDFn != nil {
		return m.getByGoogleIDFn(ctx, googleID)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthRepo) UpdateLoginAt(ctx context.Context, id uuid.UUID, ts time.Time) error {
	if m.updateLoginAtFn != nil {
		return m.updateLoginAtFn(ctx, id, ts)
	}
	return nil
}

func (m *mockAuthRepo) UpdateEmailVerifiedAt(ctx context.Context, id uuid.UUID, ts time.Time) error {
	if m.updateEmailVerifiedAtFn != nil {
		return m.updateEmailVerifiedAtFn(ctx, id, ts)
	}
	return nil
}

func (m *mockVerificationRepo) Create(ctx context.Context, verification *domain.EmailVerification) error {
	if m.createFn != nil {
		return m.createFn(ctx, verification)
	}
	return nil
}

func (m *mockVerificationRepo) GetActiveByUserIDAndCodeHash(ctx context.Context, userID uuid.UUID, codeHash string, now time.Time) (*domain.EmailVerification, error) {
	if m.getActiveFn != nil {
		return m.getActiveFn(ctx, userID, codeHash, now)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockVerificationRepo) ExpireActiveByUserID(ctx context.Context, userID uuid.UUID, now time.Time) error {
	if m.expireActiveFn != nil {
		return m.expireActiveFn(ctx, userID, now)
	}
	return nil
}

func (m *mockVerificationRepo) MarkVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) error {
	if m.markVerifiedFn != nil {
		return m.markVerifiedFn(ctx, id, verifiedAt)
	}
	return nil
}

func (m *mockSessionRepo) Create(ctx context.Context, session *domain.AuthSession) error {
	if m.createFn != nil {
		return m.createFn(ctx, session)
	}
	return nil
}

func (m *mockSessionRepo) GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.AuthSession, error) {
	if m.getByHashFn != nil {
		return m.getByHashFn(ctx, hash)
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockSessionRepo) RevokeByID(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	if m.revokeByIDFn != nil {
		return m.revokeByIDFn(ctx, id, revokedAt)
	}
	return nil
}

func (m *mockSessionRepo) RevokeByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	if m.revokeByUserIDFn != nil {
		return m.revokeByUserIDFn(ctx, userID, revokedAt)
	}
	return nil
}

// Ensures Register hashes passwords and returns a signed token tied to the user ID.
func TestAuthServiceRegister_HashesPasswordAndReturnsToken(t *testing.T) {
	secret := "test-secret"
	ttl := 15 * time.Minute
	ctx := context.Background()
	var createdUser *domain.User

	repo := &mockAuthRepo{
		createUserFn: func(_ context.Context, user *domain.User) error {
			if user.ID == uuid.Nil {
				user.ID = uuid.New()
			}
			createdUser = user
			return nil
		},
	}

	sessionRepo := &mockSessionRepo{
		createFn: func(_ context.Context, session *domain.AuthSession) error {
			if session.ID == uuid.Nil {
				session.ID = uuid.New()
			}
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: secret, AccessTokenTTL: ttl}, repo, sessionRepo, nil, nil, nil)

	result, err := svc.Register(ctx, applicationdto.RegisterInput{
		Email:    "user@example.com",
		Username: "user",
		Password: "password123",
	}, "agent", "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, createdUser)

	require.NotEmpty(t, result.User.PasswordHash)
	require.NotEqual(t, "password123", result.User.PasswordHash)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(result.User.PasswordHash), []byte("password123")))
	require.NotEmpty(t, result.RefreshToken.Token)
	require.False(t, result.RefreshToken.ExpiresAt.IsZero())

	claims := &domain.AuthClaims{}
	parsed, err := jwt.ParseWithClaims(result.Token.Token, claims, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	require.Equal(t, result.User.ID.String(), claims.Subject)
	require.Equal(t, result.User.Email, claims.Email)
	require.False(t, claims.IsAdmin)
}

// Ensures Register rejects short passwords before hitting the repository.
func TestAuthServiceRegister_ShortPassword(t *testing.T) {
	ctx := context.Background()
	called := false

	repo := &mockAuthRepo{
		createUserFn: func(_ context.Context, user *domain.User) error {
			called = true
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, repo, nil, nil, nil, nil)

	_, err := svc.Register(ctx, applicationdto.RegisterInput{
		Email:    "user@example.com",
		Username: "user",
		Password: "short",
	}, "agent", "127.0.0.1")
	require.Error(t, err)
	require.False(t, called)

	var httpErr *errs.ErrorResponse
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusBadRequest, httpErr.Status)
}

// Ensures Login returns unauthorized when the user lookup fails.
func TestAuthServiceLogin_UserNotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockAuthRepo{
		getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, repo, nil, nil, nil, nil)

	_, err := svc.Login(ctx, applicationdto.LoginInput{
		Identifier: "user@example.com",
		Password:   "password123",
	}, "agent", "127.0.0.1")
	require.Error(t, err)

	var httpErr *errs.ErrorResponse
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusUnauthorized, httpErr.Status)
}

// Ensures Login rejects invalid passwords without updating login timestamps.
func TestAuthServiceLogin_PasswordMismatch(t *testing.T) {
	ctx := context.Background()
	called := false

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mockAuthRepo{
		getByUsernameFn: func(_ context.Context, username string) (*domain.User, error) {
			return &domain.User{ID: uuid.New(), Username: username, PasswordHash: string(hash)}, nil
		},
		updateLoginAtFn: func(_ context.Context, id uuid.UUID, ts time.Time) error {
			called = true
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, repo, nil, nil, nil, nil)

	_, err = svc.Login(ctx, applicationdto.LoginInput{
		Identifier: "user",
		Password:   "wrong-password",
	}, "agent", "127.0.0.1")
	require.Error(t, err)
	require.False(t, called)

	var httpErr *errs.ErrorResponse
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusUnauthorized, httpErr.Status)
}

// Ensures Login updates login timestamps and returns a valid token on success.
func TestAuthServiceLogin_Success(t *testing.T) {
	secret := "test-secret"
	ctx := context.Background()
	called := false

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	userID := uuid.New()
	repo := &mockAuthRepo{
		getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: userID, Email: email, PasswordHash: string(hash)}, nil
		},
		updateLoginAtFn: func(_ context.Context, id uuid.UUID, ts time.Time) error {
			called = true
			return nil
		},
	}

	sessionRepo := &mockSessionRepo{
		createFn: func(_ context.Context, session *domain.AuthSession) error {
			if session.ID == uuid.Nil {
				session.ID = uuid.New()
			}
			return nil
		},
	}
	svc := NewAuthService(&config.AuthConfig{SecretKey: secret, AccessTokenTTL: time.Minute}, repo, sessionRepo, nil, nil, nil)

	result, err := svc.Login(ctx, applicationdto.LoginInput{
		Identifier: "user@example.com",
		Password:   "password123",
	}, "agent", "127.0.0.1")
	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, result)
	require.NotEmpty(t, result.RefreshToken.Token)

	claims := &domain.AuthClaims{}
	parsed, err := jwt.ParseWithClaims(result.Token.Token, claims, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	require.Equal(t, userID.String(), claims.Subject)
	require.Equal(t, "user@example.com", claims.Email)
	require.False(t, claims.IsAdmin)
}

// Ensures StartGoogleAuth fails fast when Google auth is not configured.
func TestAuthServiceStartGoogleAuth_ConfigMissing(t *testing.T) {
	ctx := context.Background()

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, &mockAuthRepo{}, nil, nil, nil, nil)

	_, err := svc.StartGoogleAuth(ctx)
	require.Error(t, err)

	var httpErr *errs.ErrorResponse
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusBadRequest, httpErr.Status)
}

// Ensures StartGoogleAuth returns the provider URL and signed state cookie.
func TestAuthServiceStartGoogleAuth_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)

	svc := NewAuthService(
		&config.AuthConfig{
			SecretKey:          "secret",
			GoogleClientID:     "client",
			GoogleClientSecret: "secret",
			GoogleRedirectURL:  "http://localhost:8080/api/v1/auth/google/callback",
		},
		&mockAuthRepo{},
		nil,
		nil,
		nil,
		nil,
	).(*authService)

	mockOAuth := &mockOAuthConfig{authURL: "https://accounts.google.com/o/oauth2/auth"}
	svc.googleOAuthConfig = mockOAuth
	svc.now = func() time.Time { return now }

	result, err := svc.StartGoogleAuth(ctx)
	require.NoError(t, err)
	require.Equal(t, "https://accounts.google.com/o/oauth2/auth", result.AuthURL)
	require.NotEmpty(t, result.StateCookie)
	require.WithinDuration(t, now.Add(googleStateTTL), result.StateExpiresAt, time.Second)

	payload, err := svc.parseGoogleStateCookie(result.StateCookie)
	require.NoError(t, err)
	require.Equal(t, mockOAuth.state, payload.State)
}

// Ensures CompleteGoogleAuth exchanges the code, validates claims, and creates a session.
func TestAuthServiceCompleteGoogleAuth_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	repo := &mockAuthRepo{
		getByGoogleIDFn: func(_ context.Context, googleID string) (*domain.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
		getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
		createUserFn: func(_ context.Context, user *domain.User) error {
			user.ID = userID
			return nil
		},
	}

	sessionRepo := &mockSessionRepo{
		createFn: func(_ context.Context, session *domain.AuthSession) error {
			session.ID = uuid.New()
			return nil
		},
	}

	svc := NewAuthService(
		&config.AuthConfig{
			SecretKey:          "secret",
			AccessTokenTTL:     time.Minute,
			GoogleClientID:     "client",
			GoogleClientSecret: "secret",
			GoogleRedirectURL:  "http://localhost:8080/api/v1/auth/google/callback",
		},
		repo,
		sessionRepo,
		nil,
		nil,
		nil,
	).(*authService)

	oauthConfig := &mockOAuthConfig{
		exchangeFn: func(_ context.Context, code string) (*oauth2.Token, error) {
			require.Equal(t, "code", code)
			token := (&oauth2.Token{AccessToken: "access"}).WithExtra(map[string]interface{}{
				"id_token": "id-token",
			})
			return token, nil
		},
	}

	svc.googleOAuthConfig = oauthConfig
	svc.googleTokenValidator = func(_ context.Context, token, audience string) (*idtoken.Payload, error) {
		require.Equal(t, "id-token", token)
		require.Equal(t, "client", audience)
		return &idtoken.Payload{
			Subject: "google-sub",
			Claims: map[string]interface{}{
				"email":          "user@example.com",
				"email_verified": true,
			},
		}, nil
	}

	state, cookieValue, _, err := svc.buildGoogleStateCookie()
	require.NoError(t, err)

	result, err := svc.CompleteGoogleAuth(ctx, "code", state, cookieValue, "agent", "127.0.0.1")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, userID, result.User.ID)
	require.NotEmpty(t, result.RefreshToken.Token)
}

// Ensures VerifyEmail marks the user as verified when the code is valid.
func TestAuthServiceVerifyEmail_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	code := "123456"
	codeHash := hashVerificationCode(code)
	verifiedCalled := false

	repo := &mockAuthRepo{
		getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: userID, Email: email}, nil
		},
		updateEmailVerifiedAtFn: func(_ context.Context, id uuid.UUID, ts time.Time) error {
			verifiedCalled = true
			return nil
		},
	}

	verificationRepo := &mockVerificationRepo{
		getActiveFn: func(_ context.Context, id uuid.UUID, hash string, now time.Time) (*domain.EmailVerification, error) {
			require.Equal(t, userID, id)
			require.Equal(t, codeHash, hash)
			return &domain.EmailVerification{ID: uuid.New(), UserID: id}, nil
		},
		markVerifiedFn: func(_ context.Context, id uuid.UUID, verifiedAt time.Time) error {
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, repo, nil, verificationRepo, nil, nil)

	user, err := svc.VerifyEmail(ctx, applicationdto.VerifyEmailInput{
		Email: "user@example.com",
		Code:  code,
	})
	require.NoError(t, err)
	require.True(t, verifiedCalled)
	require.NotNil(t, user.EmailVerifiedAt)
}

// Ensures VerifyEmail rejects invalid codes.
func TestAuthServiceVerifyEmail_InvalidCode(t *testing.T) {
	ctx := context.Background()

	repo := &mockAuthRepo{
		getByEmailFn: func(_ context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: uuid.New(), Email: email}, nil
		},
	}

	verificationRepo := &mockVerificationRepo{
		getActiveFn: func(_ context.Context, id uuid.UUID, hash string, now time.Time) (*domain.EmailVerification, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, repo, nil, verificationRepo, nil, nil)

	_, err := svc.VerifyEmail(ctx, applicationdto.VerifyEmailInput{
		Email: "user@example.com",
		Code:  "bad-code",
	})
	require.Error(t, err)

	var httpErr *errs.ErrorResponse
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusBadRequest, httpErr.Status)
}

// Ensures Refresh rejects missing refresh tokens before hitting repositories.
func TestAuthServiceRefresh_MissingToken(t *testing.T) {
	ctx := context.Background()
	called := false

	sessionRepo := &mockSessionRepo{
		getByHashFn: func(_ context.Context, hash string) (*domain.AuthSession, error) {
			called = true
			return nil, nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test", AccessTokenTTL: time.Minute}, &mockAuthRepo{}, sessionRepo, nil, nil, nil)

	_, err := svc.Refresh(ctx, "", "agent", "127.0.0.1")
	require.Error(t, err)
	require.False(t, called)

	var httpErr *errs.ErrorResponse
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusUnauthorized, httpErr.Status)
}

// Ensures Refresh rejects missing, revoked, or expired sessions.
func TestAuthServiceRefresh_InvalidSession(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	tests := []struct {
		name    string
		session *domain.AuthSession
		getErr  error
	}{
		{name: "not_found", getErr: gorm.ErrRecordNotFound},
		{name: "revoked", session: &domain.AuthSession{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: now.Add(time.Hour), RevokedAt: &now}},
		{name: "expired", session: &domain.AuthSession{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: now.Add(-time.Hour)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookedUp := false
			repo := &mockAuthRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.User, error) {
					lookedUp = true
					return &domain.User{ID: id}, nil
				},
			}
			sessionRepo := &mockSessionRepo{
				getByHashFn: func(_ context.Context, hash string) (*domain.AuthSession, error) {
					return tt.session, tt.getErr
				},
			}

			svc := NewAuthService(&config.AuthConfig{SecretKey: "test", AccessTokenTTL: time.Minute}, repo, sessionRepo, nil, nil, nil)

			_, err := svc.Refresh(ctx, "refresh-token", "agent", "127.0.0.1")
			require.Error(t, err)
			require.False(t, lookedUp)

			var httpErr *errs.ErrorResponse
			require.ErrorAs(t, err, &httpErr)
			require.Equal(t, http.StatusUnauthorized, httpErr.Status)
		})
	}
}

// Ensures Refresh rotates sessions and returns new tokens on success.
func TestAuthServiceRefresh_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	refreshToken := "refresh-token"
	expectedHash := hashRefreshToken(refreshToken)
	sessionID := uuid.New()

	repo := &mockAuthRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.User, error) {
			require.Equal(t, userID, id)
			return &domain.User{ID: userID, Email: "user@example.com"}, nil
		},
	}

	revoked := false
	created := false
	sessionRepo := &mockSessionRepo{
		getByHashFn: func(_ context.Context, hash string) (*domain.AuthSession, error) {
			require.Equal(t, expectedHash, hash)
			return &domain.AuthSession{ID: sessionID, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		revokeByIDFn: func(_ context.Context, id uuid.UUID, revokedAt time.Time) error {
			require.Equal(t, sessionID, id)
			require.False(t, revokedAt.IsZero())
			revoked = true
			return nil
		},
		createFn: func(_ context.Context, session *domain.AuthSession) error {
			created = true
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test", AccessTokenTTL: time.Minute}, repo, sessionRepo, nil, nil, nil)

	result, err := svc.Refresh(ctx, refreshToken, "agent", "127.0.0.1")
	require.NoError(t, err)
	require.True(t, revoked)
	require.True(t, created)
	require.NotNil(t, result)
	require.Equal(t, userID, result.User.ID)
	require.NotEmpty(t, result.Token.Token)
	require.NotEmpty(t, result.RefreshToken.Token)
}

// Ensures Logout revokes active sessions.
func TestAuthServiceLogout_RevokesSession(t *testing.T) {
	ctx := context.Background()
	refreshToken := "refresh-token"
	sessionID := uuid.New()

	called := false
	sessionRepo := &mockSessionRepo{
		getByHashFn: func(_ context.Context, hash string) (*domain.AuthSession, error) {
			require.Equal(t, hashRefreshToken(refreshToken), hash)
			return &domain.AuthSession{ID: sessionID, ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		revokeByIDFn: func(_ context.Context, id uuid.UUID, revokedAt time.Time) error {
			require.Equal(t, sessionID, id)
			called = true
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, &mockAuthRepo{}, sessionRepo, nil, nil, nil)

	err := svc.Logout(ctx, refreshToken)
	require.NoError(t, err)
	require.True(t, called)
}

// Ensures Logout skips revocation for already revoked sessions.
func TestAuthServiceLogout_AlreadyRevoked(t *testing.T) {
	ctx := context.Background()
	refreshToken := "refresh-token"
	revokedAt := time.Now().UTC()

	called := false
	sessionRepo := &mockSessionRepo{
		getByHashFn: func(_ context.Context, hash string) (*domain.AuthSession, error) {
			return &domain.AuthSession{ID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour), RevokedAt: &revokedAt}, nil
		},
		revokeByIDFn: func(_ context.Context, id uuid.UUID, revokedAt time.Time) error {
			called = true
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, &mockAuthRepo{}, sessionRepo, nil, nil, nil)

	err := svc.Logout(ctx, refreshToken)
	require.NoError(t, err)
	require.False(t, called)
}

// Ensures LogoutAll revokes all sessions for the user.
func TestAuthServiceLogoutAll(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	called := false
	sessionRepo := &mockSessionRepo{
		revokeByUserIDFn: func(_ context.Context, id uuid.UUID, revokedAt time.Time) error {
			require.Equal(t, userID, id)
			called = true
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, &mockAuthRepo{}, sessionRepo, nil, nil, nil)

	err := svc.LogoutAll(ctx, userID)
	require.NoError(t, err)
	require.True(t, called)
}

// Ensures CurrentUser fetches the user by ID.
func TestAuthServiceCurrentUser(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	repo := &mockAuthRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.User, error) {
			require.Equal(t, userID, id)
			return &domain.User{ID: userID, Email: "user@example.com"}, nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, repo, nil, nil, nil, nil)

	user, err := svc.CurrentUser(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, userID, user.ID)
}

// Ensures ResendVerification is a no-op when the user is already verified.
func TestAuthServiceResendVerification_AlreadyVerified(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	verifiedAt := time.Now().UTC()

	repo := &mockAuthRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.User, error) {
			return &domain.User{ID: userID, Email: "user@example.com", EmailVerifiedAt: &verifiedAt}, nil
		},
	}

	expiredCalled := false
	createdCalled := false
	verificationRepo := &mockVerificationRepo{
		expireActiveFn: func(_ context.Context, id uuid.UUID, now time.Time) error {
			expiredCalled = true
			return nil
		},
		createFn: func(_ context.Context, verification *domain.EmailVerification) error {
			createdCalled = true
			return nil
		},
	}

	svc := NewAuthService(&config.AuthConfig{SecretKey: "test"}, repo, nil, verificationRepo, nil, nil)

	err := svc.ResendVerification(ctx, userID)
	require.NoError(t, err)
	require.False(t, expiredCalled)
	require.False(t, createdCalled)
}

// Ensures ResendVerification queues a new verification for unverified users.
func TestAuthServiceResendVerification_QueuesVerification(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	repo := &mockAuthRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.User, error) {
			return &domain.User{ID: userID, Email: "user@example.com", Username: "user"}, nil
		},
	}

	expiredCalled := false
	createdCalled := false
	verificationRepo := &mockVerificationRepo{
		expireActiveFn: func(_ context.Context, id uuid.UUID, now time.Time) error {
			require.Equal(t, userID, id)
			expiredCalled = true
			return nil
		},
		createFn: func(_ context.Context, verification *domain.EmailVerification) error {
			require.Equal(t, userID, verification.UserID)
			require.Equal(t, "user@example.com", verification.Email)
			require.NotEmpty(t, verification.CodeHash)
			require.False(t, verification.ExpiresAt.IsZero())
			createdCalled = true
			return nil
		},
	}

	enqueuer := &mockTaskEnqueuer{}
	svc := NewAuthService(&config.AuthConfig{SecretKey: "test", EmailVerificationTTL: time.Hour}, repo, nil, verificationRepo, enqueuer, nil)

	err := svc.ResendVerification(ctx, userID)
	require.NoError(t, err)
	require.True(t, expiredCalled)
	require.True(t, createdCalled)
	require.True(t, enqueuer.called)
	require.Equal(t, job.TaskEmailVerification, enqueuer.task.Type())
}
