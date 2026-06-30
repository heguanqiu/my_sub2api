package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type imSSOClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
	Role   string `json:"role,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	jwt.RegisteredClaims
}

type imSSOTokenResponse struct {
	Token      string `json:"token"`
	TokenType  string `json:"token_type"`
	ExpiresIn  int    `json:"expires_in"`
	WebURL     string `json:"web_url"`
	ServiceURL string `json:"service_url,omitempty"`
}

// IssueIMSSOToken issues a short-lived login ticket for the embedded IM system.
func (h *AuthHandler) IssueIMSSOToken(c *gin.Context) {
	if h == nil || h.cfg == nil || !h.cfg.IM.Enabled {
		response.NotFound(c, "IM integration is not enabled")
		return
	}

	secret := strings.TrimSpace(h.cfg.IM.SharedSecret)
	if secret == "" {
		response.InternalError(c, "IM SSO is not configured")
		return
	}

	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := ensureLoginUserActive(user); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	ttlSeconds := h.cfg.IM.SSOTokenTTLSeconds
	if ttlSeconds <= 0 {
		ttlSeconds = 60
	}
	if ttlSeconds > 300 {
		ttlSeconds = 300
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(ttlSeconds) * time.Second)
	displayName := strings.TrimSpace(user.Username)
	if displayName == "" {
		displayName = strings.Split(user.Email, "@")[0]
	}

	claims := imSSOClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   displayName,
		Role:   user.Role,
		Avatar: strings.TrimSpace(user.AvatarURL),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.cfg.IM.Issuer,
			Subject:   strconv.FormatInt(user.ID, 10),
			Audience:  jwt.ClaimStrings{h.cfg.IM.Audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		response.InternalError(c, "Failed to generate IM SSO token")
		return
	}

	response.Success(c, imSSOTokenResponse{
		Token:      token,
		TokenType:  "Bearer",
		ExpiresIn:  ttlSeconds,
		WebURL:     h.cfg.IM.WebURL,
		ServiceURL: h.cfg.IM.ServiceURL,
	})
}
