package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/application"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	httpdto "github.com/jeheskielSunloy77/zeile/internal/interface/http/dto"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/response"
	"github.com/stretchr/testify/require"
)

// Ensures Store returns validation errors without calling the application.
func TestResourceHandlerStore_ValidationError(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	called := false
	mockService := application.NewMockResourceService[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput]()
	mockService.StoreFn = func(ctx context.Context, dto *applicationdto.StoreUserInput) (*domain.User, error) {
		called = true
		return nil, nil
	}

	h := NewResourceHandler[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput, *httpdto.StoreUserRequest, *httpdto.UpdateUserRequest]("user", NewHandler(srv), mockService)
	app.Post("/users", h.Store())

	req, err := http.NewRequest(http.MethodPost, "/users", bytes.NewReader(mustJSON(t, map[string]any{})))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.False(t, called)
}

// Ensures Store parses the payload and returns the created user with 201 status.
func TestResourceHandlerStore_Success(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	userID := uuid.New()
	mockService := application.NewMockResourceService[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput]()
	mockService.StoreFn = func(ctx context.Context, dto *applicationdto.StoreUserInput) (*domain.User, error) {
		require.Equal(t, "user@example.com", dto.Email)
		require.Equal(t, "user", dto.Username)
		return &domain.User{ID: userID, Email: dto.Email, Username: dto.Username}, nil
	}

	h := NewResourceHandler[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput, *httpdto.StoreUserRequest, *httpdto.UpdateUserRequest]("user", NewHandler(srv), mockService)
	app.Post("/users", h.Store())

	body := mustJSON(t, map[string]any{
		"email":    "user@example.com",
		"username": "user",
		"password": "password123",
	})

	req, err := http.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var got response.Response[domain.User]
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.NotNil(t, got.Data)
	require.Equal(t, userID, got.Data.ID)
	require.Equal(t, "user@example.com", got.Data.Email)
	require.Equal(t, "user", got.Data.Username)
}

// Ensures GetMany uses request pagination values and returns computed page metadata.
func TestResourceHandlerGetMany_Paginates(t *testing.T) {
	srv := newTestServer()
	app := newTestApp(srv)

	var captured port.GetManyOptions
	mockService := application.NewMockResourceService[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput]()
	mockService.GetManyFn = func(ctx context.Context, opts port.GetManyOptions) ([]domain.User, int64, error) {
		captured = opts
		return []domain.User{{ID: uuid.New()}, {ID: uuid.New()}}, 5, nil
	}

	h := NewResourceHandler[domain.User, *applicationdto.StoreUserInput, *applicationdto.UpdateUserInput, *httpdto.StoreUserRequest, *httpdto.UpdateUserRequest]("user", NewHandler(srv), mockService)
	app.Get("/users", h.GetMany())

	req, err := http.NewRequest(http.MethodGet, "/users?limit=2&offset=2", nil)
	require.NoError(t, err)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var got response.PaginatedResponse[domain.User]
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, 2, captured.Limit)
	require.Equal(t, 2, captured.Offset)
	require.Equal(t, 2, got.Page)
	require.Equal(t, 2, got.Limit)
	require.Equal(t, 5, got.Total)
	require.Equal(t, 3, got.TotalPages)
}
