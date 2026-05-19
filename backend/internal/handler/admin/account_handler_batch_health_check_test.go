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

type fakeBatchAccountTester struct {
	results map[int64]*service.ScheduledTestResult
	errs    map[int64]error
	calls   []struct {
		accountID int64
		modelID   string
	}
}

func (f *fakeBatchAccountTester) TestAccountConnection(_ *gin.Context, _ int64, _ string, _ string, _ string) error {
	panic("unexpected TestAccountConnection call")
}

func (f *fakeBatchAccountTester) RunTestBackground(_ context.Context, accountID int64, modelID string) (*service.ScheduledTestResult, error) {
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

func (f *fakeBatchAccountTester) ProbeOpenAIAPIKeyResponsesSupport(_ context.Context, _ int64) {}

func (f *fakeBatchAccountTester) FetchUpstreamSupportedModels(_ context.Context, _ *service.Account) ([]string, error) {
	panic("unexpected FetchUpstreamSupportedModels call")
}

func setupBatchHealthCheckRouter(adminSvc service.AdminService, tester accountTester) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &AccountHandler{
		adminService:       adminSvc,
		accountTestService: tester,
	}
	router.POST("/api/v1/admin/accounts/batch-health-check", handler.BatchHealthCheck)
	return router
}

func TestAccountHandlerBatchHealthCheck_ReturnsStructuredResults(t *testing.T) {
	svc := newStubAdminService()
	tester := &fakeBatchAccountTester{
		results: map[int64]*service.ScheduledTestResult{
			3: {
				Status:       "success",
				ResponseText: "pong",
				LatencyMs:    120,
			},
			4: {
				Status:       "failed",
				ErrorMessage: "401 unauthorized",
				LatencyMs:    35,
			},
		},
	}
	router := setupBatchHealthCheckRouter(svc, tester)

	body, err := json.Marshal(BatchHealthCheckRequest{
		AccountIDs: []int64{3, 4},
		ModelID:    "claude-sonnet-4-5",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-health-check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data BatchHealthCheckResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 2, resp.Data.Total)
	require.Equal(t, 1, resp.Data.SuccessCount)
	require.Equal(t, 1, resp.Data.FailedCount)
	require.Len(t, resp.Data.Items, 2)
	require.True(t, resp.Data.Items[0].Success)
	require.Equal(t, "success", resp.Data.Items[0].Status)
	require.Equal(t, "pong", resp.Data.Items[0].ResponseText)
	require.False(t, resp.Data.Items[1].Success)
	require.Equal(t, "failed", resp.Data.Items[1].Status)
	require.Equal(t, "401 unauthorized", resp.Data.Items[1].ErrorMessage)
	require.Len(t, tester.calls, 2)
	require.Equal(t, "claude-sonnet-4-5", tester.calls[0].modelID)
	require.Equal(t, "claude-sonnet-4-5", tester.calls[1].modelID)
}

func TestAccountHandlerBatchHealthCheck_RejectsEmptyIDs(t *testing.T) {
	router := setupBatchHealthCheckRouter(newStubAdminService(), &fakeBatchAccountTester{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-health-check", bytes.NewReader([]byte(`{"account_ids":[]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
