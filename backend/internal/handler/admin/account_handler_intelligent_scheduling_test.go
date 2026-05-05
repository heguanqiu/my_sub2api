package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeIntelligentSchedulingTester struct {
	results map[int64]*service.ScheduledTestResult
	errs    map[int64]error
	calls   []struct {
		accountID int64
		modelID   string
	}
}

func (f *fakeIntelligentSchedulingTester) TestAccountConnection(_ *gin.Context, _ int64, _ string, _ string, _ string) error {
	panic("unexpected TestAccountConnection call")
}

func (f *fakeIntelligentSchedulingTester) RunTestBackground(_ context.Context, accountID int64, modelID string) (*service.ScheduledTestResult, error) {
	f.calls = append(f.calls, struct {
		accountID int64
		modelID   string
	}{accountID: accountID, modelID: modelID})
	if err, ok := f.errs[accountID]; ok {
		return f.results[accountID], err
	}
	if result, ok := f.results[accountID]; ok {
		return result, nil
	}
	return &service.ScheduledTestResult{Status: "failed", ErrorMessage: "missing result"}, errors.New("missing result")
}

func (f *fakeIntelligentSchedulingTester) ProbeOpenAIAPIKeyResponsesSupport(_ context.Context, _ int64) {}

func setupIntelligentSchedulingRouter(adminSvc service.AdminService, tester accountTester) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &AccountHandler{
		adminService:       adminSvc,
		accountTestService: tester,
	}
	router.POST("/api/v1/admin/accounts/intelligent-scheduling", handler.IntelligentScheduling)
	return router
}

func TestAccountHandlerIntelligentScheduling_AppliesPriorityAndLoadFactorByLatency(t *testing.T) {
	svc := newStubAdminService()
	baseLoadFactor := 2
	svc.accounts = []service.Account{
		{
			ID:          3,
			Name:        "slow",
			Platform:    service.PlatformAnthropic,
			Type:        service.AccountTypeOAuth,
			Status:      service.StatusActive,
			Concurrency: 2,
			Priority:    9,
			LoadFactor:  &baseLoadFactor,
		},
		{
			ID:          4,
			Name:        "fast",
			Platform:    service.PlatformAnthropic,
			Type:        service.AccountTypeOAuth,
			Status:      service.StatusActive,
			Concurrency: 3,
			Priority:    8,
		},
	}
	tester := &fakeIntelligentSchedulingTester{
		results: map[int64]*service.ScheduledTestResult{
			3: {Status: "success", ResponseText: "slow ok", LatencyMs: 180},
			4: {Status: "success", ResponseText: "fast ok", LatencyMs: 60},
		},
	}
	router := setupIntelligentSchedulingRouter(svc, tester)

	body, err := json.Marshal(IntelligentSchedulingRequest{
		AccountIDs: []int64{3, 4},
		ModelID:    "claude-sonnet-4-5",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/intelligent-scheduling", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data IntelligentSchedulingResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 2, resp.Data.Total)
	require.Equal(t, 2, resp.Data.TestSuccessCount)
	require.Equal(t, 0, resp.Data.TestFailedCount)
	require.Equal(t, 2, resp.Data.AppliedCount)
	require.Len(t, resp.Data.Items, 2)

	updatedFast := svc.updatedAccounts[4]
	require.NotNil(t, updatedFast)
	require.NotNil(t, updatedFast.Priority)
	require.NotNil(t, updatedFast.LoadFactor)
	require.Equal(t, 1, *updatedFast.Priority)
	require.Equal(t, 9, *updatedFast.LoadFactor)

	updatedSlow := svc.updatedAccounts[3]
	require.NotNil(t, updatedSlow)
	require.NotNil(t, updatedSlow.Priority)
	require.NotNil(t, updatedSlow.LoadFactor)
	require.Equal(t, 2, *updatedSlow.Priority)
	require.Equal(t, 2, *updatedSlow.LoadFactor)

	require.Len(t, tester.calls, 2)
	require.Equal(t, "claude-sonnet-4-5", tester.calls[0].modelID)
	require.Equal(t, "claude-sonnet-4-5", tester.calls[1].modelID)
}

func TestAccountHandlerIntelligentScheduling_SkipsFailedTests(t *testing.T) {
	svc := newStubAdminService()
	svc.accounts = []service.Account{
		{
			ID:          3,
			Name:        "ok",
			Platform:    service.PlatformAnthropic,
			Type:        service.AccountTypeOAuth,
			Status:      service.StatusActive,
			Concurrency: 2,
			Priority:    5,
		},
		{
			ID:          4,
			Name:        "bad",
			Platform:    service.PlatformAnthropic,
			Type:        service.AccountTypeOAuth,
			Status:      service.StatusActive,
			Concurrency: 2,
			Priority:    6,
		},
	}
	tester := &fakeIntelligentSchedulingTester{
		results: map[int64]*service.ScheduledTestResult{
			3: {Status: "success", ResponseText: "ok", LatencyMs: 120},
			4: {Status: "failed", ErrorMessage: "timeout", LatencyMs: 500},
		},
		errs: map[int64]error{
			4: errors.New("timeout"),
		},
	}
	router := setupIntelligentSchedulingRouter(svc, tester)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/intelligent-scheduling", bytes.NewReader([]byte(`{"account_ids":[3,4]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data IntelligentSchedulingResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 2, resp.Data.Total)
	require.Equal(t, 1, resp.Data.TestSuccessCount)
	require.Equal(t, 1, resp.Data.TestFailedCount)
	require.Equal(t, 1, resp.Data.AppliedCount)

	require.NotNil(t, svc.updatedAccounts[3])
	require.Nil(t, svc.updatedAccounts[4])

	require.Len(t, resp.Data.Items, 2)
	require.Equal(t, int64(4), resp.Data.Items[1].AccountID)
	require.Equal(t, "test_failed", resp.Data.Items[1].Status)
	require.False(t, resp.Data.Items[1].Updated)
	require.Equal(t, "timeout", resp.Data.Items[1].ErrorMessage)
}
