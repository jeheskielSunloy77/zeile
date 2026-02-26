package application

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/jeheskielSunloy77/zeile/internal/app/sqlerr"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/lib/job"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
)

var (
	minPasswordLength = 8
	emailRegex        = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

const googleStateTTL = 10 * time.Minute

const (
	deviceAuthCodeTTL       = 10 * time.Minute
	deviceAuthPollInterval  = 5 * time.Second
	deviceAuthUserCodeSize  = 8
	deviceAuthUserCodeGroup = 4
)

type googleOAuthConfig interface {
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
}

type googleTokenValidator func(ctx context.Context, idToken, audience string) (*idtoken.Payload, error)

type googleStatePayload struct {
	State     string    `json:"state"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type deviceAuthRecord struct {
	DeviceCodeHash   string
	UserCodeHash     string
	IntervalSeconds  int
	ExpiresAt        time.Time
	ApprovedByUserID *uuid.UUID
	ConsumedAt       *time.Time
}

type authService struct {
	repo                 port.AuthRepository
	sessionRepo          port.AuthSessionRepository
	verificationRepo     port.EmailVerificationRepository
	taskEnqueuer         TaskEnqueuer
	logger               *zerolog.Logger
	secretKey            []byte
	accessTokenTTL       time.Duration
	refreshTokenTTL      time.Duration
	googleClientID       string
	googleClientSecret   string
	googleRedirectURL    string
	googleOAuthConfig    googleOAuthConfig
	googleTokenValidator googleTokenValidator
	emailVerificationTTL time.Duration
	now                  func() time.Time

	deviceAuthMu     sync.Mutex
	deviceAuthByCode map[string]deviceAuthRecord
	deviceAuthByUser map[string]string
}

type AuthToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AuthResult struct {
	User         *domain.User `json:"user"`
	Token        AuthToken    `json:"token"`
	RefreshToken AuthToken    `json:"refreshToken"`
}

type DeviceAuthStartResult struct {
	DeviceCode      string    `json:"deviceCode"`
	UserCode        string    `json:"userCode"`
	VerificationURI string    `json:"verificationUri,omitempty"`
	ExpiresAt       time.Time `json:"expiresAt"`
	IntervalSeconds int       `json:"intervalSeconds"`
}

type DeviceAuthPollResult struct {
	Status          string       `json:"status"`
	ExpiresAt       *time.Time   `json:"expiresAt,omitempty"`
	IntervalSeconds int          `json:"intervalSeconds,omitempty"`
	User            *domain.User `json:"user,omitempty"`
	Token           *AuthToken   `json:"token,omitempty"`
	RefreshToken    *AuthToken   `json:"refreshToken,omitempty"`
}

type AuthService interface {
	Register(ctx context.Context, input applicationdto.RegisterInput, userAgent, ipAddress string) (*AuthResult, error)
	Login(ctx context.Context, input applicationdto.LoginInput, userAgent, ipAddress string) (*AuthResult, error)
	StartGoogleAuth(ctx context.Context) (*GoogleAuthStart, error)
	CompleteGoogleAuth(ctx context.Context, code, state, stateCookie, userAgent, ipAddress string) (*AuthResult, error)
	VerifyEmail(ctx context.Context, input applicationdto.VerifyEmailInput) (*domain.User, error)
	StartDeviceAuth(ctx context.Context) (*DeviceAuthStartResult, error)
	PollDeviceAuth(ctx context.Context, input applicationdto.DeviceAuthPollInput, userAgent, ipAddress string) (*DeviceAuthPollResult, error)
	ApproveDeviceAuth(ctx context.Context, userID uuid.UUID, input applicationdto.DeviceAuthApproveInput) error
	Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (*AuthResult, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	CurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	ResendVerification(ctx context.Context, userID uuid.UUID) error
}

type TaskEnqueuer interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type GoogleAuthStart struct {
	AuthURL        string
	StateCookie    string
	StateExpiresAt time.Time
}

func NewAuthService(cfg *config.AuthConfig, repo port.AuthRepository, sessionRepo port.AuthSessionRepository, verificationRepo port.EmailVerificationRepository, taskEnqueuer TaskEnqueuer, logger *zerolog.Logger) AuthService {
	refreshTTL := cfg.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     googleoauth.Endpoint,
	}

	return &authService{
		repo:                 repo,
		sessionRepo:          sessionRepo,
		verificationRepo:     verificationRepo,
		taskEnqueuer:         taskEnqueuer,
		logger:               logger,
		secretKey:            []byte(cfg.SecretKey),
		accessTokenTTL:       cfg.AccessTokenTTL,
		refreshTokenTTL:      refreshTTL,
		googleClientID:       cfg.GoogleClientID,
		googleClientSecret:   cfg.GoogleClientSecret,
		googleRedirectURL:    cfg.GoogleRedirectURL,
		googleOAuthConfig:    oauthConfig,
		googleTokenValidator: idtoken.Validate,
		emailVerificationTTL: cfg.EmailVerificationTTL,
		now:                  time.Now,
		deviceAuthByCode:     make(map[string]deviceAuthRecord),
		deviceAuthByUser:     make(map[string]string),
	}
}

func (s *authService) Register(ctx context.Context, input applicationdto.RegisterInput, userAgent, ipAddress string) (*AuthResult, error) {
	if len(input.Password) < minPasswordLength {
		return nil, errs.NewBadRequestError(
			fmt.Sprintf("Password must be at least %d characters", minPasswordLength),
			true,
			[]errs.FieldError{{Field: "password", Error: "too short"}},
			nil,
		)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	user := &domain.User{
		Email:        input.Email,
		Username:     input.Username,
		PasswordHash: string(passwordHash),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, sqlerr.HandleError(err)
	}

	if err := s.queueEmailVerification(ctx, user); err != nil {
		s.logVerificationQueueError(err)
	}

	token, exp, err := s.generateToken(user)
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	refreshToken, refreshExp, err := s.createSession(ctx, user, userAgent, ipAddress)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		Token:        AuthToken{Token: token, ExpiresAt: exp},
		RefreshToken: AuthToken{Token: refreshToken, ExpiresAt: refreshExp},
	}, nil
}

func (s *authService) Login(ctx context.Context, input applicationdto.LoginInput, userAgent, ipAddress string) (*AuthResult, error) {
	user, err := s.lookupUser(ctx, input.Identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NewUnauthorizedError("Invalid credentials", true)
		}
		return nil, sqlerr.HandleError(err)
	}

	if user.PasswordHash == "" {
		return nil, errs.NewUnauthorizedError("Password login not available for this account", true)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, errs.NewUnauthorizedError("Invalid credentials", true)
	}

	now := time.Now().UTC()
	_ = s.repo.UpdateLoginAt(ctx, user.ID, now)

	token, exp, err := s.generateToken(user)
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	refreshToken, refreshExp, err := s.createSession(ctx, user, userAgent, ipAddress)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		Token:        AuthToken{Token: token, ExpiresAt: exp},
		RefreshToken: AuthToken{Token: refreshToken, ExpiresAt: refreshExp},
	}, nil
}

func (s *authService) StartGoogleAuth(ctx context.Context) (*GoogleAuthStart, error) {
	if !s.googleConfigReady() {
		return nil, errs.NewBadRequestError("Google login is not configured", false, nil, nil)
	}

	state, cookieValue, expiresAt, err := s.buildGoogleStateCookie()
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	authURL := s.googleOAuthConfig.AuthCodeURL(state)

	return &GoogleAuthStart{
		AuthURL:        authURL,
		StateCookie:    cookieValue,
		StateExpiresAt: expiresAt,
	}, nil
}

func (s *authService) CompleteGoogleAuth(ctx context.Context, code, state, stateCookie, userAgent, ipAddress string) (*AuthResult, error) {
	if !s.googleConfigReady() {
		return nil, errs.NewBadRequestError("Google login is not configured", false, nil, nil)
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" || strings.TrimSpace(stateCookie) == "" {
		return nil, errs.NewBadRequestError("Invalid Google login request", false, nil, nil)
	}

	cookiePayload, err := s.parseGoogleStateCookie(stateCookie)
	if err != nil {
		return nil, errs.NewBadRequestError("Invalid Google login state", false, nil, nil)
	}
	if cookiePayload.State != state {
		return nil, errs.NewBadRequestError("Invalid Google login state", false, nil, nil)
	}

	token, err := s.googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, errs.NewUnauthorizedError("Invalid Google token", false)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		return nil, errs.NewUnauthorizedError("Invalid Google token", false)
	}

	claims, err := s.googleTokenValidator(ctx, rawIDToken, s.googleClientID)
	if err != nil {
		return nil, errs.NewUnauthorizedError("Invalid Google token", false)
	}

	subject := claims.Subject
	emailClaim, _ := claims.Claims["email"].(string)
	emailVerified, _ := claims.Claims["email_verified"].(bool)

	return s.loginWithGoogleClaims(ctx, subject, emailClaim, emailVerified, userAgent, ipAddress)
}

func (s *authService) loginWithGoogleClaims(ctx context.Context, subject, emailClaim string, emailVerified bool, userAgent, ipAddress string) (*AuthResult, error) {
	if subject == "" {
		return nil, errs.NewUnauthorizedError("Invalid Google token", false)
	}
	if emailClaim == "" || !emailVerified {
		return nil, errs.NewUnauthorizedError("Google account email is not verified", true)
	}

	user, findErr := s.repo.GetByGoogleID(ctx, subject)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, sqlerr.HandleError(findErr)
	}

	if user == nil {
		// Try to link existing account by email
		user, findErr = s.repo.GetByEmail(ctx, emailClaim)
		switch {
		case findErr == nil:
			user.GoogleID = &subject
			if err := s.repo.Save(ctx, user); err != nil {
				return nil, sqlerr.HandleError(err)
			}
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			username := deriveUsername(emailClaim)
			user = &domain.User{
				Email:    emailClaim,
				Username: username,
				GoogleID: &subject,
			}
			if err := s.repo.CreateUser(ctx, user); err != nil {
				return nil, sqlerr.HandleError(err)
			}
		default:
			return nil, sqlerr.HandleError(findErr)
		}
	}

	now := time.Now().UTC()
	_ = s.repo.UpdateLoginAt(ctx, user.ID, now)
	if user.EmailVerifiedAt == nil {
		_ = s.repo.UpdateEmailVerifiedAt(ctx, user.ID, now)
		user.EmailVerifiedAt = &now
	}

	token, exp, err := s.generateToken(user)
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	refreshToken, refreshExp, err := s.createSession(ctx, user, userAgent, ipAddress)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		Token:        AuthToken{Token: token, ExpiresAt: exp},
		RefreshToken: AuthToken{Token: refreshToken, ExpiresAt: refreshExp},
	}, nil
}

func (s *authService) googleConfigReady() bool {
	return s.googleClientID != "" && s.googleClientSecret != "" && s.googleRedirectURL != "" && s.googleOAuthConfig != nil
}

func (s *authService) buildGoogleStateCookie() (string, string, time.Time, error) {
	state, err := generateStateToken()
	if err != nil {
		return "", "", time.Time{}, err
	}

	now := time.Now
	if s.now != nil {
		now = s.now
	}

	expiresAt := now().UTC().Add(googleStateTTL)
	payload := googleStatePayload{State: state, ExpiresAt: expiresAt}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return "", "", time.Time{}, err
	}

	encoded := base64.RawURLEncoding.EncodeToString(rawPayload)
	signature := s.signGoogleState(encoded)
	cookieValue := encoded + "." + hex.EncodeToString(signature)

	return state, cookieValue, expiresAt, nil
}

func (s *authService) parseGoogleStateCookie(cookieValue string) (*googleStatePayload, error) {
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid state cookie format")
	}

	payloadPart := parts[0]
	signaturePart := parts[1]

	signature, err := hex.DecodeString(signaturePart)
	if err != nil {
		return nil, errors.New("invalid state cookie signature")
	}

	expectedSignature := s.signGoogleState(payloadPart)
	if !hmac.Equal(signature, expectedSignature) {
		return nil, errors.New("invalid state cookie signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return nil, errors.New("invalid state cookie payload")
	}

	var payload googleStatePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.New("invalid state cookie payload")
	}

	if payload.State == "" {
		return nil, errors.New("invalid state cookie payload")
	}

	now := time.Now
	if s.now != nil {
		now = s.now
	}

	if payload.ExpiresAt.Before(now().UTC()) {
		return nil, errors.New("state cookie expired")
	}

	return &payload, nil
}

func (s *authService) signGoogleState(payload string) []byte {
	mac := hmac.New(sha256.New, s.secretKey)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func (s *authService) lookupUser(ctx context.Context, identifier string) (*domain.User, error) {
	if emailRegex.MatchString(identifier) {
		return s.repo.GetByEmail(ctx, identifier)
	}
	return s.repo.GetByUsername(ctx, identifier)
}

func (s *authService) VerifyEmail(ctx context.Context, input applicationdto.VerifyEmailInput) (*domain.User, error) {
	email := normalizeEmail(input.Email)
	if email == "" {
		return nil, errs.NewBadRequestError("Invalid email", true, []errs.FieldError{{Field: "email", Error: "invalid email"}}, nil)
	}

	code := strings.TrimSpace(input.Code)
	if code == "" {
		return nil, errs.NewBadRequestError("Invalid code", true, []errs.FieldError{{Field: "code", Error: "invalid code"}}, nil)
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalidVerificationError()
		}
		return nil, sqlerr.HandleError(err)
	}

	if user.EmailVerifiedAt != nil {
		return user, nil
	}

	if s.verificationRepo == nil {
		return nil, errs.NewInternalServerError()
	}

	codeHash := hashVerificationCode(code)
	now := time.Now().UTC()
	verification, err := s.verificationRepo.GetActiveByUserIDAndCodeHash(ctx, user.ID, codeHash, now)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalidVerificationError()
		}
		return nil, sqlerr.HandleError(err)
	}

	if err := s.verificationRepo.MarkVerified(ctx, verification.ID, now); err != nil {
		return nil, sqlerr.HandleError(err)
	}

	if err := s.repo.UpdateEmailVerifiedAt(ctx, user.ID, now); err != nil {
		return nil, sqlerr.HandleError(err)
	}

	user.EmailVerifiedAt = &now
	return user, nil
}

func (s *authService) StartDeviceAuth(ctx context.Context) (*DeviceAuthStartResult, error) {
	_ = ctx
	now := s.nowUTC()
	deviceCode, err := generateDeviceCode()
	if err != nil {
		return nil, errs.NewInternalServerError()
	}
	userCode, err := generateUserCode(deviceAuthUserCodeSize, deviceAuthUserCodeGroup)
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	deviceCodeHash := hashDeviceAuthToken(deviceCode)
	userCodeHash := hashDeviceAuthToken(normalizeUserCode(userCode))
	expiresAt := now.Add(deviceAuthCodeTTL)
	record := deviceAuthRecord{
		DeviceCodeHash:  deviceCodeHash,
		UserCodeHash:    userCodeHash,
		IntervalSeconds: int(deviceAuthPollInterval.Seconds()),
		ExpiresAt:       expiresAt,
	}

	s.deviceAuthMu.Lock()
	s.cleanupExpiredDeviceAuthLocked(now)
	s.deviceAuthByCode[deviceCodeHash] = record
	s.deviceAuthByUser[userCodeHash] = deviceCodeHash
	s.deviceAuthMu.Unlock()

	return &DeviceAuthStartResult{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		ExpiresAt:       expiresAt,
		IntervalSeconds: record.IntervalSeconds,
	}, nil
}

func (s *authService) PollDeviceAuth(ctx context.Context, input applicationdto.DeviceAuthPollInput, userAgent, ipAddress string) (*DeviceAuthPollResult, error) {
	normalized := input.Normalized()
	if normalized.DeviceCode == "" {
		return nil, errs.NewBadRequestError("deviceCode is required", true, []errs.FieldError{{Field: "deviceCode", Error: "is required"}}, nil)
	}

	now := s.nowUTC()
	deviceCodeHash := hashDeviceAuthToken(normalized.DeviceCode)

	s.deviceAuthMu.Lock()
	s.cleanupExpiredDeviceAuthLocked(now)

	record, ok := s.deviceAuthByCode[deviceCodeHash]
	if !ok {
		s.deviceAuthMu.Unlock()
		return nil, errs.NewBadRequestError("invalid_device_code", true, nil, nil)
	}

	if record.ConsumedAt != nil {
		s.deviceAuthMu.Unlock()
		return nil, &errs.ErrorResponse{
			Message:  "device_code_already_used",
			Status:   409,
			Success:  false,
			Override: true,
		}
	}

	if record.ApprovedByUserID == nil {
		expiresAt := record.ExpiresAt
		interval := record.IntervalSeconds
		s.deviceAuthMu.Unlock()
		return &DeviceAuthPollResult{
			Status:          "pending",
			ExpiresAt:       &expiresAt,
			IntervalSeconds: interval,
		}, nil
	}

	approvedUserID := *record.ApprovedByUserID
	consumedAt := now
	record.ConsumedAt = &consumedAt
	s.deviceAuthByCode[deviceCodeHash] = record
	delete(s.deviceAuthByUser, record.UserCodeHash)
	s.deviceAuthMu.Unlock()

	user, err := s.repo.GetByID(ctx, approvedUserID)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	refreshToken, refreshExp, err := s.createSession(ctx, user, userAgent, ipAddress)
	if err != nil {
		return nil, err
	}
	accessToken, accessExp, err := s.generateToken(user)
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	return &DeviceAuthPollResult{
		Status: "approved",
		User:   user,
		Token: &AuthToken{
			Token:     accessToken,
			ExpiresAt: accessExp,
		},
		RefreshToken: &AuthToken{
			Token:     refreshToken,
			ExpiresAt: refreshExp,
		},
	}, nil
}

func (s *authService) ApproveDeviceAuth(ctx context.Context, userID uuid.UUID, input applicationdto.DeviceAuthApproveInput) error {
	normalized := input.Normalized()
	if normalized.UserCode == "" {
		return errs.NewBadRequestError("userCode is required", true, []errs.FieldError{{Field: "userCode", Error: "is required"}}, nil)
	}

	now := s.nowUTC()
	userCodeHash := hashDeviceAuthToken(normalizeUserCode(normalized.UserCode))

	s.deviceAuthMu.Lock()
	defer s.deviceAuthMu.Unlock()
	s.cleanupExpiredDeviceAuthLocked(now)

	deviceCodeHash, ok := s.deviceAuthByUser[userCodeHash]
	if !ok {
		return errs.NewBadRequestError("invalid_user_code", true, nil, nil)
	}

	record, ok := s.deviceAuthByCode[deviceCodeHash]
	if !ok {
		delete(s.deviceAuthByUser, userCodeHash)
		return errs.NewBadRequestError("invalid_user_code", true, nil, nil)
	}

	if record.ConsumedAt != nil {
		return &errs.ErrorResponse{
			Message:  "device_code_already_used",
			Status:   409,
			Success:  false,
			Override: true,
		}
	}

	if record.ApprovedByUserID != nil {
		if *record.ApprovedByUserID == userID {
			return nil
		}
		return &errs.ErrorResponse{
			Message:  "device_code_already_approved",
			Status:   409,
			Success:  false,
			Override: true,
		}
	}

	record.ApprovedByUserID = &userID
	s.deviceAuthByCode[deviceCodeHash] = record
	return nil
}

func (s *authService) CurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return user, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken, userAgent, ipAddress string) (*AuthResult, error) {
	if refreshToken == "" {
		return nil, errs.NewUnauthorizedError("Unauthorized", false)
	}
	if s.sessionRepo == nil {
		return nil, errs.NewInternalServerError()
	}

	tokenHash := hashRefreshToken(refreshToken)
	session, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NewUnauthorizedError("Unauthorized", false)
		}
		return nil, sqlerr.HandleError(err)
	}

	now := time.Now().UTC()
	if session.RevokedAt != nil || session.ExpiresAt.Before(now) {
		return nil, errs.NewUnauthorizedError("Unauthorized", false)
	}

	user, err := s.repo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	if err := s.sessionRepo.RevokeByID(ctx, session.ID, now); err != nil {
		return nil, sqlerr.HandleError(err)
	}

	rotatedToken, rotatedExp, err := s.createSession(ctx, user, userAgent, ipAddress)
	if err != nil {
		return nil, err
	}

	accessToken, accessExp, err := s.generateToken(user)
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	return &AuthResult{
		User:         user,
		Token:        AuthToken{Token: accessToken, ExpiresAt: accessExp},
		RefreshToken: AuthToken{Token: rotatedToken, ExpiresAt: rotatedExp},
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" || s.sessionRepo == nil {
		return nil
	}

	tokenHash := hashRefreshToken(refreshToken)
	session, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return sqlerr.HandleError(err)
	}

	if session.RevokedAt != nil {
		return nil
	}

	now := time.Now().UTC()
	return sqlerr.HandleError(s.sessionRepo.RevokeByID(ctx, session.ID, now))
}

func (s *authService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if s.sessionRepo == nil {
		return nil
	}
	now := time.Now().UTC()
	return sqlerr.HandleError(s.sessionRepo.RevokeByUserID(ctx, userID, now))
}

func (s *authService) ResendVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return sqlerr.HandleError(err)
	}

	if user.EmailVerifiedAt != nil {
		return nil
	}

	if err := s.queueEmailVerification(ctx, user); err != nil {
		s.logVerificationQueueError(err)
		return errs.NewInternalServerError()
	}

	return nil
}

func (s *authService) generateToken(user *domain.User) (string, time.Time, error) {
	if user == nil {
		return "", time.Time{}, errs.NewInternalServerError()
	}

	exp := time.Now().Add(s.accessTokenTTL)
	claims := domain.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email:   user.Email,
		IsAdmin: user.IsAdmin,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

func (s *authService) createSession(ctx context.Context, user *domain.User, userAgent, ipAddress string) (string, time.Time, error) {
	if s.sessionRepo == nil || user == nil {
		return "", time.Time{}, errs.NewInternalServerError()
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return "", time.Time{}, errs.NewInternalServerError()
	}

	expiresAt := time.Now().UTC().Add(s.refreshTokenTTL)
	var agent *string
	if strings.TrimSpace(userAgent) != "" {
		clean := strings.TrimSpace(userAgent)
		agent = &clean
	}
	var ip *string
	if strings.TrimSpace(ipAddress) != "" {
		clean := strings.TrimSpace(ipAddress)
		ip = &clean
	}

	session := &domain.AuthSession{
		UserID:           user.ID,
		RefreshTokenHash: hashRefreshToken(refreshToken),
		UserAgent:        agent,
		IPAddress:        ip,
		ExpiresAt:        expiresAt,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", time.Time{}, sqlerr.HandleError(err)
	}

	return refreshToken, expiresAt, nil
}

func deriveUsername(email string) string {
	parts := regexp.MustCompile("@").Split(email, 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return fmt.Sprintf("user-%s", uuid.New().String()[:8])
}

func generateStateToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

func generateRefreshToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func hashDeviceAuthToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeUserCode(value string) string {
	trimmed := strings.TrimSpace(strings.ToUpper(value))
	return strings.ReplaceAll(trimmed, "-", "")
}

func generateDeviceCode() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

func generateUserCode(length int, groupSize int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if length <= 0 {
		return "", errors.New("invalid user code length")
	}
	if groupSize <= 0 {
		groupSize = length
	}

	builder := strings.Builder{}
	builder.Grow(length + (length-1)/groupSize)
	for idx := 0; idx < length; idx++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		if idx > 0 && idx%groupSize == 0 {
			builder.WriteByte('-')
		}
		builder.WriteByte(alphabet[n.Int64()])
	}
	return builder.String(), nil
}

func (s *authService) nowUTC() time.Time {
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	return now().UTC()
}

func (s *authService) cleanupExpiredDeviceAuthLocked(now time.Time) {
	if len(s.deviceAuthByCode) == 0 {
		return
	}

	for codeHash, record := range s.deviceAuthByCode {
		if !record.ExpiresAt.Before(now) {
			continue
		}
		delete(s.deviceAuthByCode, codeHash)
		if record.UserCodeHash != "" {
			delete(s.deviceAuthByUser, record.UserCodeHash)
		}
	}
}

func (s *authService) queueEmailVerification(ctx context.Context, user *domain.User) error {
	if user == nil || user.Email == "" || user.EmailVerifiedAt != nil || s.verificationRepo == nil {
		return nil
	}

	code, err := generateVerificationCode()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	ttl := s.emailVerificationTTL
	if err := s.verificationRepo.ExpireActiveByUserID(ctx, user.ID, now); err != nil {
		return err
	}

	verification := &domain.EmailVerification{
		UserID:    user.ID,
		Email:     user.Email,
		CodeHash:  hashVerificationCode(code),
		ExpiresAt: now.Add(ttl),
	}
	if err := s.verificationRepo.Create(ctx, verification); err != nil {
		return err
	}

	if s.taskEnqueuer == nil {
		return nil
	}

	expiresInMinutes := int(ttl.Minutes())
	if expiresInMinutes <= 0 {
		expiresInMinutes = 1
	}
	task, err := job.NewEmailVerificationTask(job.EmailVerificationPayload{
		To:               user.Email,
		Username:         user.Username,
		Code:             code,
		ExpiresInMinutes: expiresInMinutes,
	})
	if err != nil {
		return err
	}

	_, err = s.taskEnqueuer.EnqueueContext(ctx, task)
	return err
}

func (s *authService) logVerificationQueueError(err error) {
	if err == nil || s.logger == nil {
		return
	}
	s.logger.Error().Err(err).Msg("failed to queue email verification")
}

func generateVerificationCode() (string, error) {
	const codeLength = 6
	const maxDigit = 10

	code := make([]byte, 0, codeLength)
	for range codeLength {
		n, err := rand.Int(rand.Reader, big.NewInt(maxDigit))
		if err != nil {
			return "", err
		}
		code = append(code, byte('0'+n.Int64()))
	}

	return string(code), nil
}

func hashVerificationCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func invalidVerificationError() *errs.ErrorResponse {
	return errs.NewBadRequestError(
		"Invalid or expired verification code",
		true,
		[]errs.FieldError{{Field: "code", Error: "invalid or expired"}},
		nil,
	)
}
