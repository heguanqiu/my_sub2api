//go:build unit

package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type authInvitationValidationSettingRepoStub struct {
	values map[string]string
}

func (s *authInvitationValidationSettingRepoStub) Get(_ context.Context, key string) (*service.Setting, error) {
	if value, ok := s.values[key]; ok {
		return &service.Setting{Key: key, Value: value}, nil
	}
	return nil, service.ErrSettingNotFound
}

func (s *authInvitationValidationSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *authInvitationValidationSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *authInvitationValidationSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *authInvitationValidationSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *authInvitationValidationSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *authInvitationValidationSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestValidateInvitationCode_AllowsInviteLinksWhenInvitationCodeSignupDisabled(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	db, err := sql.Open("sqlite", "file:auth_validate_invite_link?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	inviter, err := client.User.Create().
		SetEmail("inviter@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SetUsername("inviter").
		Save(context.Background())
	require.NoError(t, err)

	_, err = client.InviteLink.Create().
		SetCode("link-code-123").
		SetCreatedByUserID(inviter.ID).
		SetCreatorRole(service.RoleUser).
		SetStatus(service.InviteLinkStatusActive).
		Save(context.Background())
	require.NoError(t, err)

	cfg := &config.Config{}
	settingSvc := service.NewSettingService(&authInvitationValidationSettingRepoStub{
		values: map[string]string{
			service.SettingKeyInvitationCodeEnabled: "false",
		},
	}, cfg)
	userRepo := repository.NewUserRepository(client, nil)
	authSvc := service.NewAuthService(
		client,
		userRepo,
		nil,
		nil,
		cfg,
		settingSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	handler := &AuthHandler{
		authService: authSvc,
		settingSvc:  settingSvc,
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/validate-invitation-code",
		bytes.NewBufferString(`{"code":"link-code-123"}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler.ValidateInvitationCode(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Valid     bool   `json:"valid"`
			ErrorCode string `json:"error_code"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.True(t, resp.Data.Valid)
	require.Empty(t, resp.Data.ErrorCode)
}
