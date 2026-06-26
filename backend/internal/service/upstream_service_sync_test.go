package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSyncRemoteResourcesPassesNilKeysWhenRemoteAPIKeyListFails(t *testing.T) {
	repo := &upstreamSyncRepo{upstream: &Upstream{ID: 7, Name: "north", Type: UpstreamTypeSub2API, BaseURL: "https://up.example.com"}}
	adapter := &upstreamSyncAdapter{
		groups: []*UpstreamRemoteGroup{{RemoteGroupID: "default", RemoteGroupName: "Default", RateMultiplier: 1}},
		keyErr: errors.New("keys endpoint unavailable"),
	}
	svc := NewUpstreamService(repo, nil, nil, nil, adapter)

	result, err := svc.SyncRemoteResources(context.Background(), 7)
	if err != nil {
		t.Fatalf("SyncRemoteResources() error = %v", err)
	}
	if !repo.replaceCalled {
		t.Fatal("ReplaceRemoteResources was not called")
	}
	if repo.replacedKeys != nil {
		t.Fatalf("replaced keys = %#v, want nil so existing remote keys are preserved", repo.replacedKeys)
	}
	if result == nil || result.Run == nil || !strings.Contains(result.Run.Message, "without remote api keys") {
		t.Fatalf("sync message = %#v, want partial key-list message", result)
	}
}

func TestSyncRemoteResourcesPassesEmptyKeysWhenUpstreamReturnsEmptyList(t *testing.T) {
	repo := &upstreamSyncRepo{upstream: &Upstream{ID: 7, Name: "north", Type: UpstreamTypeSub2API, BaseURL: "https://up.example.com"}}
	adapter := &upstreamSyncAdapter{
		groups: []*UpstreamRemoteGroup{{RemoteGroupID: "default", RemoteGroupName: "Default", RateMultiplier: 1}},
		keys:   []*UpstreamRemoteAPIKey{},
	}
	svc := NewUpstreamService(repo, nil, nil, nil, adapter)

	_, err := svc.SyncRemoteResources(context.Background(), 7)
	if err != nil {
		t.Fatalf("SyncRemoteResources() error = %v", err)
	}
	if !repo.replaceCalled {
		t.Fatal("ReplaceRemoteResources was not called")
	}
	if repo.replacedKeys == nil {
		t.Fatal("replaced keys = nil, want empty slice so stale remote keys are cleared")
	}
	if len(repo.replacedKeys) != 0 {
		t.Fatalf("replaced keys len = %d, want 0", len(repo.replacedKeys))
	}
}

type upstreamSyncRepo struct {
	UpstreamRepository
	upstream      *Upstream
	replaceCalled bool
	replacedKeys  []*UpstreamRemoteAPIKey
}

func (r *upstreamSyncRepo) Get(ctx context.Context, id int64) (*Upstream, error) {
	if r.upstream == nil || r.upstream.ID != id {
		return nil, ErrUpstreamNotFound
	}
	copy := *r.upstream
	return &copy, nil
}

func (r *upstreamSyncRepo) ReplaceRemoteResources(ctx context.Context, upstreamID int64, groups []*UpstreamRemoteGroup, keys []*UpstreamRemoteAPIKey, run *UpstreamSyncRun) error {
	r.replaceCalled = true
	r.replacedKeys = keys
	return nil
}

type upstreamSyncAdapter struct {
	groups []*UpstreamRemoteGroup
	keys   []*UpstreamRemoteAPIKey
	keyErr error
}

func (a *upstreamSyncAdapter) Login(ctx context.Context, upstream *Upstream) (*UpstreamAdminSession, error) {
	return &UpstreamAdminSession{}, nil
}

func (a *upstreamSyncAdapter) GetAccountBalance(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) (*UpstreamAccountBalanceResult, error) {
	return nil, nil
}

func (a *upstreamSyncAdapter) ListGroups(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) ([]*UpstreamRemoteGroup, error) {
	return a.groups, nil
}

func (a *upstreamSyncAdapter) ListAPIKeys(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) ([]*UpstreamRemoteAPIKey, error) {
	if a.keyErr != nil {
		return nil, a.keyErr
	}
	return a.keys, nil
}
