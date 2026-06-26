package admin

import (
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type UpstreamHandler struct {
	upstreamService *service.UpstreamService
}

func NewUpstreamHandler(upstreamService *service.UpstreamService) *UpstreamHandler {
	return &UpstreamHandler{upstreamService: upstreamService}
}

type upstreamForwardCredentialRequest struct {
	Name      string         `json:"name"`
	AuthType  string         `json:"auth_type"`
	APIKey    *string        `json:"api_key"`
	Enabled   *bool          `json:"enabled"`
	ExpiresAt *time.Time     `json:"expires_at"`
	Metadata  map[string]any `json:"metadata"`
}

type upstreamAdminAuthRequest struct {
	AuthMode       string         `json:"auth_mode"`
	LoginURL       string         `json:"login_url"`
	Username       *string        `json:"username"`
	Password       *string        `json:"password"`
	AccessToken    *string        `json:"access_token"`
	RefreshToken   *string        `json:"refresh_token"`
	TokenExpiresAt *time.Time     `json:"token_expires_at"`
	Metadata       map[string]any `json:"metadata"`
}

type createUpstreamRequest struct {
	Name                 string                            `json:"name" binding:"required,max=120"`
	Type                 string                            `json:"type"`
	BaseURL              string                            `json:"base_url" binding:"required"`
	Status               string                            `json:"status"`
	Priority             int                               `json:"priority"`
	Weight               int                               `json:"weight"`
	CostMultiplier       float64                           `json:"cost_multiplier"`
	TimeoutMS            int                               `json:"timeout_ms"`
	ConnectTimeoutMS     int                               `json:"connect_timeout_ms"`
	RetryMax             int                               `json:"retry_max"`
	ProbeEnabled         *bool                             `json:"probe_enabled"`
	ProbeModel           string                            `json:"probe_model"`
	ProbeIntervalSeconds int                               `json:"probe_interval_seconds"`
	RoutingMode          string                            `json:"routing_mode"`
	Notes                string                            `json:"notes"`
	Metadata             map[string]any                    `json:"metadata"`
	LocalGroupIDs        []int64                           `json:"local_group_ids"`
	ForwardCredential    *upstreamForwardCredentialRequest `json:"forward_credential"`
	AdminAuth            *upstreamAdminAuthRequest         `json:"admin_auth"`
}

type updateUpstreamRequest struct {
	Name                 *string                           `json:"name" binding:"omitempty,max=120"`
	Type                 *string                           `json:"type"`
	BaseURL              *string                           `json:"base_url"`
	Status               *string                           `json:"status"`
	Priority             *int                              `json:"priority"`
	Weight               *int                              `json:"weight"`
	CostMultiplier       *float64                          `json:"cost_multiplier"`
	TimeoutMS            *int                              `json:"timeout_ms"`
	ConnectTimeoutMS     *int                              `json:"connect_timeout_ms"`
	RetryMax             *int                              `json:"retry_max"`
	ProbeEnabled         *bool                             `json:"probe_enabled"`
	ProbeModel           *string                           `json:"probe_model"`
	ProbeIntervalSeconds *int                              `json:"probe_interval_seconds"`
	RoutingMode          *string                           `json:"routing_mode"`
	Notes                *string                           `json:"notes"`
	Metadata             *map[string]any                   `json:"metadata"`
	LocalGroupIDs        *[]int64                          `json:"local_group_ids"`
	ForwardCredential    *upstreamForwardCredentialRequest `json:"forward_credential"`
	AdminAuth            *upstreamAdminAuthRequest         `json:"admin_auth"`
}

type schedulePreviewRequest struct {
	Model         string                      `json:"model"`
	RemoteGroupID string                      `json:"remote_group_id"`
	LocalGroupID  int64                       `json:"local_group_id"`
	Mode          string                      `json:"mode"`
	RandomSeed    int64                       `json:"random_seed"`
	Candidates    []service.UpstreamCandidate `json:"candidates"`
}

type updateUpstreamRemoteAPIKeyConfigRequest struct {
	RemoteGroupID string  `json:"remote_group_id"`
	LocalGroupIDs []int64 `json:"local_group_ids"`
	APIKey        *string `json:"api_key"`
}

type applySyncPreviewRequest struct {
	PreviewToken string `json:"preview_token" binding:"required"`
}

type updateGovernancePolicyRequest struct {
	ConsecutiveFailuresToCircuitOpen int     `json:"consecutive_failures_to_circuit_open"`
	FirstTokenDegradeThresholdMS     int     `json:"first_token_degrade_threshold_ms"`
	ErrorRateDegradeThreshold        float64 `json:"error_rate_degrade_threshold"`
	RecoveryProbeIntervalSeconds     int     `json:"recovery_probe_interval_seconds"`
	RecoverySuccessesRequired        int     `json:"recovery_successes_required"`
	IgnoredStatusCodes               []int   `json:"ignored_status_codes"`
	ImmediateCircuitStatusCodes      []int   `json:"immediate_circuit_status_codes"`
	ProbeFailureWeight               float64 `json:"probe_failure_weight"`
	RuntimeFailureWeight             float64 `json:"runtime_failure_weight"`
	AlertEnabled                     *bool   `json:"alert_enabled"`
}

func (h *UpstreamHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.upstreamService.List(c.Request.Context(), service.UpstreamListParams{
		Page:     page,
		PageSize: pageSize,
		Type:     strings.TrimSpace(c.Query("type")),
		Status:   strings.TrimSpace(c.Query("status")),
		Search:   strings.TrimSpace(c.Query("search")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *UpstreamHandler) Get(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	upstream, err := h.upstreamService.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, upstream)
}

func (h *UpstreamHandler) Create(c *gin.Context) {
	var req createUpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	probeEnabled := true
	if req.ProbeEnabled != nil {
		probeEnabled = *req.ProbeEnabled
	}
	upstream, err := h.upstreamService.Create(c.Request.Context(), service.UpstreamCreateParams{
		Name:                 req.Name,
		Type:                 req.Type,
		BaseURL:              req.BaseURL,
		Status:               req.Status,
		Priority:             req.Priority,
		Weight:               req.Weight,
		CostMultiplier:       req.CostMultiplier,
		TimeoutMS:            req.TimeoutMS,
		ConnectTimeoutMS:     req.ConnectTimeoutMS,
		RetryMax:             req.RetryMax,
		ProbeEnabled:         probeEnabled,
		ProbeModel:           req.ProbeModel,
		ProbeIntervalSeconds: req.ProbeIntervalSeconds,
		RoutingMode:          req.RoutingMode,
		Notes:                req.Notes,
		Metadata:             req.Metadata,
		LocalGroupIDs:        req.LocalGroupIDs,
		ForwardCredential:    forwardCredentialInput(req.ForwardCredential),
		AdminAuth:            adminAuthInput(req.AdminAuth),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, upstream)
}

func (h *UpstreamHandler) Update(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req updateUpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	upstream, err := h.upstreamService.Update(c.Request.Context(), id, service.UpstreamUpdateParams{
		Name:                 req.Name,
		Type:                 req.Type,
		BaseURL:              req.BaseURL,
		Status:               req.Status,
		Priority:             req.Priority,
		Weight:               req.Weight,
		CostMultiplier:       req.CostMultiplier,
		TimeoutMS:            req.TimeoutMS,
		ConnectTimeoutMS:     req.ConnectTimeoutMS,
		RetryMax:             req.RetryMax,
		ProbeEnabled:         req.ProbeEnabled,
		ProbeModel:           req.ProbeModel,
		ProbeIntervalSeconds: req.ProbeIntervalSeconds,
		RoutingMode:          req.RoutingMode,
		Notes:                req.Notes,
		Metadata:             req.Metadata,
		LocalGroupIDs:        req.LocalGroupIDs,
		ForwardCredential:    forwardCredentialInput(req.ForwardCredential),
		AdminAuth:            adminAuthInput(req.AdminAuth),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, upstream)
}

func (h *UpstreamHandler) Delete(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	if err := h.upstreamService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "upstream deleted"})
}

func (h *UpstreamHandler) Sync(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	result, err := h.upstreamService.SyncRemoteResources(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamHandler) SyncPreview(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	result, err := h.upstreamService.PreviewSyncRemoteResources(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamHandler) ApplySyncPreview(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req applySyncPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	result, err := h.upstreamService.ApplySyncPreview(c.Request.Context(), id, req.PreviewToken)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamHandler) TestLogin(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	result, err := h.upstreamService.TestAdminLogin(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamHandler) Probe(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	result, err := h.upstreamService.RunProbe(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamHandler) Health(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	result, err := h.upstreamService.HealthDashboard(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamHandler) Events(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	events, err := h.upstreamService.ListEvents(c.Request.Context(), id, limit, strings.TrimSpace(c.Query("event_type")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, events)
}

func (h *UpstreamHandler) Groups(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	groups, err := h.upstreamService.ListRemoteGroups(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, groups)
}

func (h *UpstreamHandler) APIKeys(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	keys, err := h.upstreamService.ListRemoteAPIKeys(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, keys)
}

func (h *UpstreamHandler) UpdateAPIKeyConfig(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	remoteAPIKeyID := strings.TrimSpace(c.Param("remote_key_id"))
	if remoteAPIKeyID == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_REMOTE_API_KEY_ID", "invalid remote api key id"))
		return
	}
	var req updateUpstreamRemoteAPIKeyConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	key, err := h.upstreamService.UpdateRemoteAPIKeyConfig(c.Request.Context(), id, remoteAPIKeyID, req.RemoteGroupID, req.LocalGroupIDs, req.APIKey)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, key)
}

func (h *UpstreamHandler) RefreshBalance(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	result, err := h.upstreamService.RefreshAccountBalance(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *UpstreamHandler) SchedulePreview(c *gin.Context) {
	var req schedulePreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	decision, err := h.upstreamService.SchedulePreview(c.Request.Context(), service.UpstreamScheduleRequest{
		Model:         req.Model,
		RemoteGroupID: req.RemoteGroupID,
		LocalGroupID:  req.LocalGroupID,
		Mode:          req.Mode,
		RandomSeed:    req.RandomSeed,
		Candidates:    req.Candidates,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, decision)
}

func (h *UpstreamHandler) GetPolicy(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	policy, err := h.upstreamService.GovernancePolicy(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, policy)
}

func (h *UpstreamHandler) UpdatePolicy(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req updateGovernancePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	alertEnabled := true
	if req.AlertEnabled != nil {
		alertEnabled = *req.AlertEnabled
	}
	policy, err := h.upstreamService.UpdateGovernancePolicy(c.Request.Context(), id, service.UpstreamGovernancePolicy{
		ConsecutiveFailuresToCircuitOpen: req.ConsecutiveFailuresToCircuitOpen,
		FirstTokenDegradeThresholdMS:     req.FirstTokenDegradeThresholdMS,
		ErrorRateDegradeThreshold:        req.ErrorRateDegradeThreshold,
		RecoveryProbeIntervalSeconds:     req.RecoveryProbeIntervalSeconds,
		RecoverySuccessesRequired:        req.RecoverySuccessesRequired,
		IgnoredStatusCodes:               req.IgnoredStatusCodes,
		ImmediateCircuitStatusCodes:      req.ImmediateCircuitStatusCodes,
		ProbeFailureWeight:               req.ProbeFailureWeight,
		RuntimeFailureWeight:             req.RuntimeFailureWeight,
		AlertEnabled:                     alertEnabled,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, policy)
}

func (h *UpstreamHandler) Alerts(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	activeOnly := strings.TrimSpace(c.Query("active")) != "false"
	alerts, err := h.upstreamService.ListAlerts(c.Request.Context(), id, activeOnly)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, alerts)
}

func (h *UpstreamHandler) ResolveAlert(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	alertType := strings.TrimSpace(c.Param("alert_type"))
	if alertType == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_ALERT_TYPE", "invalid alert type"))
		return
	}
	if err := h.upstreamService.ResolveAlert(c.Request.Context(), id, alertType); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "alert resolved"})
}

func (h *UpstreamHandler) CostReport(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	end := time.Now().UTC()
	start := end.Add(-24 * time.Hour)
	if raw := strings.TrimSpace(c.Query("start")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.ErrorFrom(c, infraerrors.BadRequest("INVALID_START", "start must be RFC3339"))
			return
		}
		start = parsed
	}
	if raw := strings.TrimSpace(c.Query("end")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.ErrorFrom(c, infraerrors.BadRequest("INVALID_END", "end must be RFC3339"))
			return
		}
		end = parsed
	}
	report, err := h.upstreamService.CostReport(c.Request.Context(), id, start, end, strings.TrimSpace(c.Query("dimension")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, report)
}

func (h *UpstreamHandler) ResetCostReport(c *gin.Context) {
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	result, err := h.upstreamService.ResetCostReport(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func parseUpstreamID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_UPSTREAM_ID", "invalid upstream id"))
		return 0, false
	}
	return id, true
}

func forwardCredentialInput(req *upstreamForwardCredentialRequest) *service.UpstreamForwardCredentialInput {
	if req == nil {
		return nil
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return &service.UpstreamForwardCredentialInput{
		Name:      req.Name,
		AuthType:  req.AuthType,
		APIKey:    req.APIKey,
		Enabled:   enabled,
		ExpiresAt: req.ExpiresAt,
		Metadata:  req.Metadata,
	}
}

func adminAuthInput(req *upstreamAdminAuthRequest) *service.UpstreamAdminAuthInput {
	if req == nil {
		return nil
	}
	return &service.UpstreamAdminAuthInput{
		AuthMode:       req.AuthMode,
		LoginURL:       req.LoginURL,
		Username:       req.Username,
		Password:       req.Password,
		AccessToken:    req.AccessToken,
		RefreshToken:   req.RefreshToken,
		TokenExpiresAt: req.TokenExpiresAt,
		Metadata:       req.Metadata,
	}
}
