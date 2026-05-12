package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type PlaygroundHandler struct {
	userService   *service.UserService
	apiKeyService *service.APIKeyService
	cfg           *config.Config
}

func NewPlaygroundHandler(
	userService *service.UserService,
	apiKeyService *service.APIKeyService,
	cfg *config.Config,
) *PlaygroundHandler {
	return &PlaygroundHandler{
		userService:   userService,
		apiKeyService: apiKeyService,
		cfg:           cfg,
	}
}

type PlaygroundEmbedSessionRequest struct {
	APIKeyID int64  `json:"api_key_id" binding:"required"`
	APIBase  string `json:"api_base_url"`
}

type PlaygroundEmbedSessionResponse struct {
	Source            string         `json:"source"`
	Version           int            `json:"version"`
	ExpiresAt         int64          `json:"expires_at"`
	Signature         string         `json:"signature"`
	APIBaseURL        string         `json:"api_base_url"`
	SelectedKeyID     int64          `json:"selected_key_id"`
	SelectedKeyName   string         `json:"selected_key_name"`
	User              playgroundUser `json:"user"`
	DirectConnections directPayload  `json:"direct_connections"`
}

type playgroundUser struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type directPayload struct {
	OpenAIAPIBaseURLs []string                       `json:"OPENAI_API_BASE_URLS"`
	OpenAIAPIKeys     []string                       `json:"OPENAI_API_KEYS"`
	OpenAIAPIConfigs  map[string]directConfigPayload `json:"OPENAI_API_CONFIGS"`
}

type directConfigPayload struct {
	Enable   bool     `json:"enable"`
	PrefixID string   `json:"prefix_id"`
	ModelIDs []string `json:"model_ids"`
}

// CreateEmbedSession signs the current Sub2API user's Open WebUI iframe payload.
// POST /api/v1/playground/embed-session
func (h *PlaygroundHandler) CreateEmbedSession(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req PlaygroundEmbedSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	apiKey, err := h.apiKeyService.GetByID(c.Request.Context(), req.APIKeyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if apiKey.UserID != subject.UserID {
		response.Forbidden(c, "Not authorized to use this API key")
		return
	}
	if !apiKey.IsActive() || apiKey.IsExpired() || apiKey.IsQuotaExhausted() {
		response.ErrorFrom(c, infraerrors.Forbidden("API_KEY_UNAVAILABLE", "api key is not available"))
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	apiBaseURL, err := normalizePlaygroundAPIBaseURL(req.APIBase, c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ttl := h.cfg.Playground.EmbedSessionTTLSeconds
	if ttl <= 0 {
		ttl = 300
	}

	payload := PlaygroundEmbedSessionResponse{
		Source:          "sub2api",
		Version:         1,
		ExpiresAt:       time.Now().Add(time.Duration(ttl) * time.Second).Unix(),
		APIBaseURL:      apiBaseURL,
		SelectedKeyID:   apiKey.ID,
		SelectedKeyName: apiKey.Name,
		User: playgroundUser{
			ID:       strconv.FormatInt(user.ID, 10),
			Email:    user.Email,
			Username: user.Username,
			Name:     displayNameForPlayground(user),
		},
		DirectConnections: directPayload{
			OpenAIAPIBaseURLs: []string{apiBaseURL},
			OpenAIAPIKeys:     []string{apiKey.Key},
			OpenAIAPIConfigs: map[string]directConfigPayload{
				"0": {
					Enable:   true,
					PrefixID: "",
					ModelIDs: []string{},
				},
			},
		},
	}

	signature, err := signPlaygroundEmbedSession(payload, h.cfg.Playground.EmbedSecret)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	payload.Signature = signature

	response.Success(c, payload)
}

func displayNameForPlayground(user *service.User) string {
	if user == nil {
		return ""
	}
	if strings.TrimSpace(user.Username) != "" {
		return strings.TrimSpace(user.Username)
	}
	if strings.TrimSpace(user.Email) != "" {
		return strings.TrimSpace(user.Email)
	}
	return strconv.FormatInt(user.ID, 10)
}

func signPlaygroundEmbedSession(payload PlaygroundEmbedSessionResponse, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("playground embed secret is not configured")
	}

	payload.Signature = ""
	body, err := marshalPlaygroundEmbedSession(payload)
	if err != nil {
		return "", fmt.Errorf("marshal playground embed payload: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func marshalPlaygroundEmbedSession(payload PlaygroundEmbedSessionResponse) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func normalizePlaygroundAPIBaseURL(value string, c *gin.Context) (string, error) {
	base := strings.TrimSpace(value)
	if base == "" {
		scheme := "http"
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
			base = scheme + "://" + forwardedHost
		} else {
			base = scheme + "://" + c.Request.Host
		}
	}

	base = strings.TrimRight(base, "/")
	if !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return "", errors.New("api_base_url must be an absolute HTTP(S) URL")
	}
	return base, nil
}
