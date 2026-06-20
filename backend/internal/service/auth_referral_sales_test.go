//go:build unit

package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestResolveRegistrationAffiliationSalesCreatorOwnsLegacySnapshotLink(t *testing.T) {
	ctx := context.Background()
	client, db := newAuthReferralSalesTestClient(t, "resolve_snapshot")

	salesA, err := client.User.Create().
		SetEmail("sales-a@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleSales).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	salesB, err := client.User.Create().
		SetEmail("sales-b@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleSales).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.InviteLink.Create().
		SetCode("sales-link").
		SetCreatedByUserID(salesA.ID).
		SetCreatorRole(service.RoleUser).
		SetOwnerSalesID(salesB.ID).
		SetStatus(service.InviteLinkStatusActive).
		Save(ctx)
	require.NoError(t, err)

	cfg := &config.Config{}
	settingSvc := service.NewSettingService(&authReferralSalesSettingRepoStub{}, cfg)
	svc := service.NewAuthService(client, repository.NewUserRepository(client, db), nil, nil, cfg, settingSvc, nil, nil, nil, nil, nil, nil, nil)

	affiliation, err := svc.ResolveRegistrationAffiliation(ctx, "sales-link")
	require.NoError(t, err)
	require.NotNil(t, affiliation)
	require.NotNil(t, affiliation.InvitedByUserID)
	require.NotNil(t, affiliation.OwnerSalesID)
	require.Equal(t, salesA.ID, *affiliation.InvitedByUserID)
	require.Equal(t, salesA.ID, *affiliation.OwnerSalesID)
}

func TestResolveRegistrationAffiliationUserInviteUsesOwnerSalesSnapshot(t *testing.T) {
	ctx := context.Background()
	client, db := newAuthReferralSalesTestClient(t, "resolve_user_snapshot")

	sales, err := client.User.Create().
		SetEmail("sales@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleSales).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	inviter, err := client.User.Create().
		SetEmail("customer@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetOwnerSalesID(sales.ID).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.InviteLink.Create().
		SetCode("customer-link").
		SetCreatedByUserID(inviter.ID).
		SetCreatorRole(service.RoleUser).
		SetOwnerSalesID(sales.ID).
		SetStatus(service.InviteLinkStatusActive).
		Save(ctx)
	require.NoError(t, err)

	cfg := &config.Config{}
	settingSvc := service.NewSettingService(&authReferralSalesSettingRepoStub{}, cfg)
	svc := service.NewAuthService(client, repository.NewUserRepository(client, db), nil, nil, cfg, settingSvc, nil, nil, nil, nil, nil, nil, nil)

	affiliation, err := svc.ResolveRegistrationAffiliation(ctx, "customer-link")
	require.NoError(t, err)
	require.NotNil(t, affiliation)
	require.NotNil(t, affiliation.InvitedByUserID)
	require.NotNil(t, affiliation.OwnerSalesID)
	require.Equal(t, inviter.ID, *affiliation.InvitedByUserID)
	require.Equal(t, sales.ID, *affiliation.OwnerSalesID)
}

func TestRegisterVerifiedOAuthEmailAccountPersistsInviteLinkSalesOwner(t *testing.T) {
	ctx := context.Background()
	client, db := newAuthReferralSalesTestClient(t, "verified_oauth")
	repo := repository.NewUserRepository(client, db)

	sales, err := client.User.Create().
		SetEmail("sales@example.com").
		SetPasswordHash("hash").
		SetRole(service.RoleSales).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.InviteLink.Create().
		SetCode("oauth-sales-link").
		SetCreatedByUserID(sales.ID).
		SetCreatorRole(service.RoleSales).
		SetOwnerSalesID(sales.ID).
		SetStatus(service.InviteLinkStatusActive).
		Save(ctx)
	require.NoError(t, err)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                   "test-secret",
			ExpireHour:               1,
			AccessTokenExpireMinutes: 60,
			RefreshTokenExpireDays:   7,
		},
		Default: config.DefaultConfig{
			UserBalance:     3.5,
			UserConcurrency: 2,
		},
	}
	settingSvc := service.NewSettingService(&authReferralSalesSettingRepoStub{values: map[string]string{
		service.SettingKeyRegistrationEnabled: "true",
	}}, cfg)
	svc := service.NewAuthService(
		client,
		repo,
		nil,
		&authReferralSalesRefreshTokenCacheStub{},
		cfg,
		settingSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	tokenPair, createdUser, err := svc.RegisterVerifiedOAuthEmailAccount(
		ctx,
		"oauth-customer@example.com",
		"secret-123",
		"oauth-sales-link",
		"github",
	)
	require.NoError(t, err)
	require.NotNil(t, tokenPair)
	require.NotNil(t, createdUser)

	stored, err := client.User.Get(ctx, createdUser.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.InvitedByUserID)
	require.NotNil(t, stored.OwnerSalesID)
	require.Equal(t, sales.ID, *stored.InvitedByUserID)
	require.Equal(t, sales.ID, *stored.OwnerSalesID)
}

func newAuthReferralSalesTestClient(t *testing.T, name string) (*dbent.Client, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:auth_referral_sales_%s?mode=memory&cache=shared&_fk=1", name))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return client, db
}

type authReferralSalesSettingRepoStub struct {
	values map[string]string
}

func (s *authReferralSalesSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *authReferralSalesSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *authReferralSalesSettingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *authReferralSalesSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *authReferralSalesSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *authReferralSalesSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *authReferralSalesSettingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type authReferralSalesRefreshTokenCacheStub struct{}

func (s *authReferralSalesRefreshTokenCacheStub) StoreRefreshToken(context.Context, string, *service.RefreshTokenData, time.Duration) error {
	return nil
}

func (s *authReferralSalesRefreshTokenCacheStub) GetRefreshToken(context.Context, string) (*service.RefreshTokenData, error) {
	return nil, service.ErrRefreshTokenNotFound
}

func (s *authReferralSalesRefreshTokenCacheStub) DeleteRefreshToken(context.Context, string) error {
	return nil
}

func (s *authReferralSalesRefreshTokenCacheStub) DeleteUserRefreshTokens(context.Context, int64) error {
	return nil
}

func (s *authReferralSalesRefreshTokenCacheStub) DeleteTokenFamily(context.Context, string) error {
	return nil
}

func (s *authReferralSalesRefreshTokenCacheStub) AddToUserTokenSet(context.Context, int64, string, time.Duration) error {
	return nil
}

func (s *authReferralSalesRefreshTokenCacheStub) AddToFamilyTokenSet(context.Context, string, string, time.Duration) error {
	return nil
}

func (s *authReferralSalesRefreshTokenCacheStub) GetUserTokenHashes(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (s *authReferralSalesRefreshTokenCacheStub) GetFamilyTokenHashes(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s *authReferralSalesRefreshTokenCacheStub) IsTokenInFamily(context.Context, string, string) (bool, error) {
	return false, nil
}
