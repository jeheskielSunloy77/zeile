package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type APIError struct {
	Message string
	Status  int
}

func (e *APIError) Error() string {
	if e == nil {
		return "api error"
	}
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Sprintf("api request failed with status %d", e.Status)
	}
	return fmt.Sprintf("%s (status %d)", e.Message, e.Status)
}

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type AuthToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type DeviceAuthStartResponse struct {
	DeviceCode      string    `json:"deviceCode"`
	UserCode        string    `json:"userCode"`
	VerificationURI string    `json:"verificationUri"`
	ExpiresAt       time.Time `json:"expiresAt"`
	IntervalSeconds int       `json:"intervalSeconds"`
}

type DeviceAuthPollResponse struct {
	Status          string     `json:"status"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	IntervalSeconds int        `json:"intervalSeconds,omitempty"`
	User            *User      `json:"user,omitempty"`
	Token           *AuthToken `json:"token,omitempty"`
	RefreshToken    *AuthToken `json:"refreshToken,omitempty"`
}

func NewClient(baseURL string) *Client {
	cleanBaseURL := strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	return &Client{
		baseURL: cleanBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && strings.TrimSpace(c.baseURL) != ""
}

func (c *Client) StartDeviceAuth(ctx context.Context) (DeviceAuthStartResponse, error) {
	reqBody := struct{}{}
	respBody := DeviceAuthStartResponse{}
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/device/start", reqBody, "", &respBody)
	return respBody, err
}

func (c *Client) PollDeviceAuth(ctx context.Context, deviceCode string) (DeviceAuthPollResponse, error) {
	reqBody := struct {
		DeviceCode string `json:"deviceCode"`
	}{
		DeviceCode: strings.TrimSpace(deviceCode),
	}
	respBody := DeviceAuthPollResponse{}
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/device/poll", reqBody, "", &respBody)
	return respBody, err
}

func (c *Client) Logout(ctx context.Context, refreshToken string) error {
	reqBody := struct {
		RefreshToken string `json:"refreshToken"`
	}{
		RefreshToken: strings.TrimSpace(refreshToken),
	}
	return c.doJSON(ctx, http.MethodPost, "/api/v1/auth/logout", reqBody, "", nil)
}

func (c *Client) Me(ctx context.Context, accessToken string) (User, error) {
	respBody := User{}
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/auth/me", nil, strings.TrimSpace(accessToken), &respBody)
	return respBody, err
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, bearerToken string, out any) error {
	if !c.Enabled() {
		return &APIError{
			Message: "api base url is not configured",
			Status:  http.StatusBadRequest,
		}
	}

	targetURL := c.baseURL + path
	var bodyReader io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encode request payload: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer := strings.TrimSpace(bearerToken); bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		apiErr := parseAPIError(resp.StatusCode, responseBytes)
		return apiErr
	}

	if out == nil || len(responseBytes) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBytes, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func parseAPIError(status int, payload []byte) error {
	if len(payload) == 0 {
		return &APIError{Message: "request failed", Status: status}
	}

	type errorPayload struct {
		Message string `json:"message"`
	}
	var decoded errorPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return &APIError{Message: "request failed", Status: status}
	}
	if strings.TrimSpace(decoded.Message) == "" {
		decoded.Message = "request failed"
	}
	return &APIError{Message: decoded.Message, Status: status}
}

