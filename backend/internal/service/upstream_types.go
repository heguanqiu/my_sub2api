package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	UpstreamTypeSub2API          = "sub2api"
	UpstreamTypeNewAPI           = "newapi"
	UpstreamTypeOpenAICompatible = "openai_compatible"
	UpstreamTypeCustom           = "custom"

	UpstreamStatusActive      = "active"
	UpstreamStatusDegraded    = "degraded"
	UpstreamStatusHalfOpen    = "half_open"
	UpstreamStatusCircuitOpen = "circuit_open"
	UpstreamStatusDisabled    = "disabled"

	UpstreamRoutingStability = "stability"
	UpstreamRoutingBalanced  = "balanced"
	UpstreamRoutingCost      = "cost"
	UpstreamRoutingSpeed     = "speed"
	UpstreamRoutingManual    = "manual"
	UpstreamRoutingModeKey   = "upstream.routing_mode"

	UpstreamAdminAuthPassword = "password"
	UpstreamAdminAuthToken    = "token"
	UpstreamAdminAuthNone     = "none"

	UpstreamForwardAuthBearer = "bearer"
	UpstreamForwardAuthOpenAI = "openai"
	UpstreamForwardAuthCustom = "custom"
)

var (
	ErrUpstreamNotFound = infraerrors.NotFound("UPSTREAM_NOT_FOUND", "upstream not found")
	ErrUpstreamExists   = infraerrors.Conflict("UPSTREAM_EXISTS", "upstream already exists")

	ErrUpstreamInvalidInput = infraerrors.BadRequest("UPSTREAM_INVALID_INPUT", "invalid upstream input")
	ErrUpstreamSyncFailed   = infraerrors.ServiceUnavailable("UPSTREAM_SYNC_FAILED", "upstream sync failed")
	ErrUpstreamLoginFailed  = infraerrors.ServiceUnavailable("UPSTREAM_LOGIN_FAILED", "upstream login failed")
)

type Upstream struct {
	ID                   int64                      `json:"id"`
	Name                 string                     `json:"name"`
	Type                 string                     `json:"type"`
	BaseURL              string                     `json:"base_url"`
	Status               string                     `json:"status"`
	Priority             int                        `json:"priority"`
	Weight               int                        `json:"weight"`
	CostMultiplier       float64                    `json:"cost_multiplier"`
	TimeoutMS            int                        `json:"timeout_ms"`
	ConnectTimeoutMS     int                        `json:"connect_timeout_ms"`
	RetryMax             int                        `json:"retry_max"`
	ProbeEnabled         bool                       `json:"probe_enabled"`
	ProbeModel           string                     `json:"probe_model"`
	ProbeIntervalSeconds int                        `json:"probe_interval_seconds"`
	RoutingMode          string                     `json:"routing_mode"`
	Notes                string                     `json:"notes"`
	LastSyncedAt         *time.Time                 `json:"last_synced_at"`
	LastSyncStatus       string                     `json:"last_sync_status"`
	LastSyncError        string                     `json:"last_sync_error"`
	CreatedAt            time.Time                  `json:"created_at"`
	UpdatedAt            time.Time                  `json:"updated_at"`
	DeletedAt            *time.Time                 `json:"deleted_at,omitempty"`
	GroupsCount          int                        `json:"groups_count"`
	APIKeysCount         int                        `json:"api_keys_count"`
	LatestHealthScore    float64                    `json:"latest_health_score"`
	ForwardCredential    *UpstreamForwardCredential `json:"forward_credential,omitempty"`
	AdminAuth            *UpstreamAdminAuth         `json:"admin_auth,omitempty"`
	RemoteGroups         []*UpstreamRemoteGroup     `json:"remote_groups,omitempty"`
	RemoteAPIKeys        []*UpstreamRemoteAPIKey    `json:"remote_api_keys,omitempty"`
	LatestSyncRun        *UpstreamSyncRun           `json:"latest_sync_run,omitempty"`
	DecryptFailed        bool                       `json:"decrypt_failed,omitempty"`
	Metadata             map[string]any             `json:"metadata,omitempty"`
	LocalGroupIDs        []int64                    `json:"local_group_ids,omitempty"`
	SchedulerSnapshot    *UpstreamSchedulerSnapshot `json:"scheduler_snapshot,omitempty"`
}

type UpstreamForwardCredential struct {
	ID            int64          `json:"id"`
	UpstreamID    int64          `json:"upstream_id"`
	Name          string         `json:"name"`
	AuthType      string         `json:"auth_type"`
	APIKey        string         `json:"api_key,omitempty"`
	APIKeyMasked  string         `json:"api_key_masked"`
	Enabled       bool           `json:"enabled"`
	ExpiresAt     *time.Time     `json:"expires_at"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DecryptFailed bool           `json:"decrypt_failed,omitempty"`
}

type UpstreamAdminAuth struct {
	UpstreamID          int64          `json:"upstream_id"`
	AuthMode            string         `json:"auth_mode"`
	LoginURL            string         `json:"login_url"`
	Username            string         `json:"username,omitempty"`
	UsernameMasked      string         `json:"username_masked,omitempty"`
	Password            string         `json:"password,omitempty"`
	PasswordConfigured  bool           `json:"password_configured"`
	AccessToken         string         `json:"access_token,omitempty"`
	AccessTokenMasked   string         `json:"access_token_masked,omitempty"`
	RefreshToken        string         `json:"refresh_token,omitempty"`
	RefreshTokenMasked  string         `json:"refresh_token_masked,omitempty"`
	TokenExpiresAt      *time.Time     `json:"token_expires_at"`
	LastLoginAt         *time.Time     `json:"last_login_at"`
	LastLoginError      string         `json:"last_login_error"`
	Metadata            map[string]any `json:"metadata"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	SecretDecryptFailed bool           `json:"secret_decrypt_failed,omitempty"`
}

type UpstreamRemoteGroup struct {
	ID              int64          `json:"id"`
	UpstreamID      int64          `json:"upstream_id"`
	RemoteGroupID   string         `json:"remote_group_id"`
	RemoteGroupName string         `json:"remote_group_name"`
	RateMultiplier  float64        `json:"rate_multiplier"`
	Status          string         `json:"status"`
	RawSnapshot     map[string]any `json:"raw_snapshot,omitempty"`
	LastSyncedAt    time.Time      `json:"last_synced_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type UpstreamRemoteAPIKey struct {
	ID                  int64          `json:"id"`
	UpstreamID          int64          `json:"upstream_id"`
	RemoteAPIKeyID      string         `json:"remote_api_key_id"`
	RemoteAPIKeyName    string         `json:"remote_api_key_name"`
	APIKey              string         `json:"-"`
	APIKeyConfigured    bool           `json:"api_key_configured"`
	MaskedKey           string         `json:"masked_key"`
	SyncedRemoteGroupID string         `json:"synced_remote_group_id"`
	RemoteGroupID       string         `json:"remote_group_id"`
	LocalGroupIDs       []int64        `json:"local_group_ids,omitempty"`
	Status              string         `json:"status"`
	Quota               *float64       `json:"quota"`
	UsedQuota           *float64       `json:"used_quota"`
	RawSnapshot         map[string]any `json:"raw_snapshot,omitempty"`
	LastSyncedAt        time.Time      `json:"last_synced_at"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	GovernanceStatus    string         `json:"governance_status,omitempty"`
	GovernanceReason    string         `json:"governance_reason,omitempty"`
}

type UpstreamSyncRun struct {
	ID           int64          `json:"id"`
	UpstreamID   int64          `json:"upstream_id"`
	Status       string         `json:"status"`
	GroupsCount  int            `json:"groups_count"`
	APIKeysCount int            `json:"api_keys_count"`
	Message      string         `json:"message"`
	StartedAt    time.Time      `json:"started_at"`
	FinishedAt   *time.Time     `json:"finished_at"`
	RawResult    map[string]any `json:"raw_result,omitempty"`
}

type UpstreamListParams struct {
	Page     int
	PageSize int
	Type     string
	Status   string
	Search   string
}

type UpstreamRoutingConfig struct {
	Mode string `json:"mode"`
}

type UpstreamCreateParams struct {
	Name                 string
	Type                 string
	BaseURL              string
	Status               string
	Priority             int
	Weight               int
	CostMultiplier       float64
	TimeoutMS            int
	ConnectTimeoutMS     int
	RetryMax             int
	ProbeEnabled         bool
	ProbeModel           string
	ProbeIntervalSeconds int
	RoutingMode          string
	Notes                string
	Metadata             map[string]any
	LocalGroupIDs        []int64
	ForwardCredential    *UpstreamForwardCredentialInput
	AdminAuth            *UpstreamAdminAuthInput
}

type UpstreamUpdateParams struct {
	Name                 *string
	Type                 *string
	BaseURL              *string
	Status               *string
	Priority             *int
	Weight               *int
	CostMultiplier       *float64
	TimeoutMS            *int
	ConnectTimeoutMS     *int
	RetryMax             *int
	ProbeEnabled         *bool
	ProbeModel           *string
	ProbeIntervalSeconds *int
	RoutingMode          *string
	Notes                *string
	Metadata             *map[string]any
	LocalGroupIDs        *[]int64
	ForwardCredential    *UpstreamForwardCredentialInput
	AdminAuth            *UpstreamAdminAuthInput
}

type UpstreamForwardCredentialInput struct {
	Name      string
	AuthType  string
	APIKey    *string
	Enabled   bool
	ExpiresAt *time.Time
	Metadata  map[string]any
}

type UpstreamAdminAuthInput struct {
	AuthMode       string
	LoginURL       string
	Username       *string
	Password       *string
	AccessToken    *string
	RefreshToken   *string
	TokenExpiresAt *time.Time
	Metadata       map[string]any
}

type UpstreamAdminSession struct {
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time
	UserID         string
}

type UpstreamAdminAdapter interface {
	Login(ctx context.Context, upstream *Upstream) (*UpstreamAdminSession, error)
	GetAccountBalance(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) (*UpstreamAccountBalanceResult, error)
	ListGroups(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) ([]*UpstreamRemoteGroup, error)
	ListAPIKeys(ctx context.Context, upstream *Upstream, session *UpstreamAdminSession) ([]*UpstreamRemoteAPIKey, error)
}

type UpstreamSyncResult struct {
	Run     *UpstreamSyncRun        `json:"run"`
	Groups  []*UpstreamRemoteGroup  `json:"groups"`
	APIKeys []*UpstreamRemoteAPIKey `json:"api_keys"`
}

type UpstreamSyncDiffItem struct {
	Kind        string         `json:"kind"`
	ID          string         `json:"id"`
	Name        string         `json:"name,omitempty"`
	Before      map[string]any `json:"before,omitempty"`
	After       map[string]any `json:"after,omitempty"`
	Impact      string         `json:"impact,omitempty"`
	LocalGroups []int64        `json:"local_group_ids,omitempty"`
}

type UpstreamSyncDiff struct {
	AddedGroups               []UpstreamSyncDiffItem `json:"added_groups"`
	RemovedGroups             []UpstreamSyncDiffItem `json:"removed_groups"`
	ChangedGroups             []UpstreamSyncDiffItem `json:"changed_groups"`
	AddedAPIKeys              []UpstreamSyncDiffItem `json:"added_api_keys"`
	RemovedAPIKeys            []UpstreamSyncDiffItem `json:"removed_api_keys"`
	ChangedAPIKeys            []UpstreamSyncDiffItem `json:"changed_api_keys"`
	AffectedLocalGroupIDs     []int64                `json:"affected_local_group_ids"`
	UnschedulableAPIKeyIDs    []string               `json:"unschedulable_api_key_ids"`
	CostMultiplierChangeCount int                    `json:"cost_multiplier_change_count"`
}

type UpstreamSyncPreview struct {
	ID           int64                   `json:"id"`
	UpstreamID   int64                   `json:"upstream_id"`
	PreviewToken string                  `json:"preview_token"`
	Status       string                  `json:"status"`
	Diff         UpstreamSyncDiff        `json:"diff"`
	Groups       []*UpstreamRemoteGroup  `json:"groups,omitempty"`
	APIKeys      []*UpstreamRemoteAPIKey `json:"api_keys,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	AppliedAt    *time.Time              `json:"applied_at,omitempty"`
	ExpiresAt    time.Time               `json:"expires_at"`
}

type UpstreamHealthWindow struct {
	WindowSeconds int     `json:"window_seconds"`
	SuccessCount  int     `json:"success_count"`
	ErrorCount    int     `json:"error_count"`
	SuccessRate   float64 `json:"success_rate"`
	TTFTP50MS     *int    `json:"ttft_p50_ms,omitempty"`
	TTFTP90MS     *int    `json:"ttft_p90_ms,omitempty"`
	TTFTP95MS     *int    `json:"ttft_p95_ms,omitempty"`
	TTFTP99MS     *int    `json:"ttft_p99_ms,omitempty"`
}

type UpstreamHealthDashboard struct {
	UpstreamID              int64                       `json:"upstream_id"`
	Status                  string                      `json:"status"`
	LatestProbeStatus       string                      `json:"latest_probe_status"`
	LatestProbeFirstTokenMS *int                        `json:"latest_probe_first_token_ms,omitempty"`
	LatestProbeCheckedAt    *time.Time                  `json:"latest_probe_checked_at,omitempty"`
	RecentErrorReason       string                      `json:"recent_error_reason"`
	RecentErrorAt           *time.Time                  `json:"recent_error_at,omitempty"`
	Windows                 []UpstreamHealthWindow      `json:"windows"`
	SchedulerSnapshot       UpstreamSchedulerSnapshot   `json:"scheduler_snapshot"`
	Degraded                bool                        `json:"degraded"`
	CircuitOpen             bool                        `json:"circuit_open"`
	Recovering              bool                        `json:"recovering"`
	SchedulableAPIKeys      int                         `json:"schedulable_api_keys"`
	ServableLocalGroups     int                         `json:"servable_local_groups"`
	ActiveAlerts            []UpstreamAlert             `json:"active_alerts,omitempty"`
	Attribution             []UpstreamAttributionSignal `json:"attribution,omitempty"`
}

type UpstreamEvent struct {
	ID                int64          `json:"id"`
	UpstreamID        int64          `json:"upstream_id"`
	EventType         string         `json:"event_type"`
	Reason            string         `json:"reason"`
	AccountID         *int64         `json:"account_id,omitempty"`
	RemoteAPIKeyID    string         `json:"remote_api_key_id,omitempty"`
	RemoteGroupID     string         `json:"remote_group_id,omitempty"`
	LocalGroupID      *int64         `json:"local_group_id,omitempty"`
	Model             string         `json:"model,omitempty"`
	StatusCode        int            `json:"status_code"`
	FirstTokenMS      *int           `json:"first_token_ms,omitempty"`
	DurationMS        int64          `json:"duration_ms"`
	UserID            *int64         `json:"user_id,omitempty"`
	StreamInterrupted bool           `json:"stream_interrupted"`
	Retried           bool           `json:"retried"`
	Confidence        float64        `json:"confidence"`
	Evidence          map[string]any `json:"evidence,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
}

type UpstreamAttributionSignal struct {
	Scope      string  `json:"scope"`
	ID         string  `json:"id"`
	ErrorCount int     `json:"error_count"`
	TotalCount int     `json:"total_count"`
	ErrorRate  float64 `json:"error_rate"`
	Confidence float64 `json:"confidence"`
	Suggestion string  `json:"suggestion"`
}

type UpstreamGovernancePolicy struct {
	ConsecutiveFailuresToCircuitOpen int     `json:"consecutive_failures_to_circuit_open"`
	FirstTokenDegradeThresholdMS     int     `json:"first_token_degrade_threshold_ms"`
	ErrorRateDegradeThreshold        float64 `json:"error_rate_degrade_threshold"`
	RecoveryProbeIntervalSeconds     int     `json:"recovery_probe_interval_seconds"`
	RecoverySuccessesRequired        int     `json:"recovery_successes_required"`
	IgnoredStatusCodes               []int   `json:"ignored_status_codes"`
	ImmediateCircuitStatusCodes      []int   `json:"immediate_circuit_status_codes"`
	ProbeFailureWeight               float64 `json:"probe_failure_weight"`
	RuntimeFailureWeight             float64 `json:"runtime_failure_weight"`
	AlertEnabled                     bool    `json:"alert_enabled"`
}

type UpstreamAlert struct {
	ID         int64          `json:"id"`
	UpstreamID *int64         `json:"upstream_id,omitempty"`
	AlertType  string         `json:"alert_type"`
	Severity   string         `json:"severity"`
	Status     string         `json:"status"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
	Evidence   map[string]any `json:"evidence,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	ResolvedAt *time.Time     `json:"resolved_at,omitempty"`
}

type UpstreamCostDimension struct {
	UpstreamID      int64   `json:"upstream_id"`
	UpstreamName    string  `json:"upstream_name"`
	RemoteGroupID   string  `json:"remote_group_id,omitempty"`
	RemoteAPIKeyID  string  `json:"remote_api_key_id,omitempty"`
	LocalGroupID    *int64  `json:"local_group_id,omitempty"`
	UserID          *int64  `json:"user_id,omitempty"`
	Model           string  `json:"model,omitempty"`
	RequestCount    int64   `json:"request_count"`
	LocalBilledCost float64 `json:"local_billed_cost"`
	UpstreamCost    float64 `json:"upstream_cost"`
	CostDelta       float64 `json:"cost_delta"`
	GrossProfit     float64 `json:"gross_profit"`
	AvgMultiplier   float64 `json:"avg_multiplier"`
}

type UpstreamCostReport struct {
	Start     time.Time               `json:"start"`
	End       time.Time               `json:"end"`
	Dimension string                  `json:"dimension"`
	ResetAt   *time.Time              `json:"reset_at,omitempty"`
	Items     []UpstreamCostDimension `json:"items"`
	Totals    UpstreamCostDimension   `json:"totals"`
}

type UpstreamCostReportResetResult struct {
	UpstreamID int64     `json:"upstream_id"`
	ResetAt    time.Time `json:"reset_at"`
}

type UpstreamLoginTestResult struct {
	Success        bool       `json:"success"`
	HasToken       bool       `json:"has_token"`
	TokenExpiresAt *time.Time `json:"token_expires_at"`
	Message        string     `json:"message"`
}

type UpstreamProbeResult struct {
	UpstreamID        int64                      `json:"upstream_id"`
	AccountID         int64                      `json:"account_id,omitempty"`
	RemoteAPIKeyID    string                     `json:"remote_api_key_id,omitempty"`
	RemoteAPIKeyName  string                     `json:"remote_api_key_name,omitempty"`
	RemoteGroupID     string                     `json:"remote_group_id,omitempty"`
	Model             string                     `json:"model"`
	Success           bool                       `json:"success"`
	Ignored           bool                       `json:"ignored"`
	Reason            string                     `json:"reason"`
	StatusCode        int                        `json:"status_code"`
	FirstTokenMS      *int                       `json:"first_token_ms,omitempty"`
	DurationMS        int64                      `json:"duration_ms"`
	ErrorMessage      string                     `json:"error_message,omitempty"`
	SchedulerSnapshot *UpstreamSchedulerSnapshot `json:"scheduler_snapshot,omitempty"`
	CheckedAt         time.Time                  `json:"checked_at"`
}

type UpstreamAccountBalanceResult struct {
	UpstreamID     int64     `json:"upstream_id"`
	Balance        *float64  `json:"balance"`
	Quota          *float64  `json:"quota"`
	UsedQuota      *float64  `json:"used_quota"`
	RemainingQuota *float64  `json:"remaining_quota"`
	Source         string    `json:"source,omitempty"`
	HasBalance     bool      `json:"has_balance"`
	Message        string    `json:"message"`
	CheckedAt      time.Time `json:"checked_at"`
}

type UpstreamRepository interface {
	Create(ctx context.Context, upstream *Upstream) error
	Update(ctx context.Context, upstream *Upstream) error
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (*Upstream, error)
	List(ctx context.Context, params UpstreamListParams) ([]*Upstream, int64, error)
	UpsertForwardCredential(ctx context.Context, credential *UpstreamForwardCredential) error
	UpsertAdminAuth(ctx context.Context, auth *UpstreamAdminAuth) error
	ReplaceRemoteResources(ctx context.Context, upstreamID int64, groups []*UpstreamRemoteGroup, keys []*UpstreamRemoteAPIKey, run *UpstreamSyncRun) error
	ListRemoteGroups(ctx context.Context, upstreamID int64) ([]*UpstreamRemoteGroup, error)
	ListRemoteAPIKeys(ctx context.Context, upstreamID int64) ([]*UpstreamRemoteAPIKey, error)
	UpdateRemoteAPIKeyConfig(ctx context.Context, upstreamID int64, remoteAPIKeyID string, remoteGroupID string, localGroupIDs []int64, apiKeyEncrypted *string) (*UpstreamRemoteAPIKey, error)
	ClearRemoteAPIKeyLocalConfig(ctx context.Context, upstreamID int64, remoteAPIKeyID string) error
	LatestSyncRun(ctx context.Context, upstreamID int64) (*UpstreamSyncRun, error)
	RecordRuntimeEvent(ctx context.Context, event UpstreamRuntimeEvent) (*UpstreamSchedulerSnapshot, error)
	GetHealthDashboard(ctx context.Context, upstreamID int64) (*UpstreamHealthDashboard, error)
	ListEvents(ctx context.Context, upstreamID int64, limit int, eventType string) ([]UpstreamEvent, error)
	CreateSyncPreview(ctx context.Context, preview *UpstreamSyncPreview) error
	GetSyncPreview(ctx context.Context, upstreamID int64, token string) (*UpstreamSyncPreview, error)
	MarkSyncPreviewApplied(ctx context.Context, upstreamID int64, token string, appliedAt time.Time) error
	ListAlerts(ctx context.Context, upstreamID int64, activeOnly bool) ([]UpstreamAlert, error)
	UpsertAlert(ctx context.Context, alert UpstreamAlert) error
	ResolveAlert(ctx context.Context, upstreamID int64, alertType string) error
	GetCostReport(ctx context.Context, upstreamID int64, start, end time.Time, dimension string) (*UpstreamCostReport, error)
	GetRoutingMode(ctx context.Context) (string, error)
	SetRoutingMode(ctx context.Context, mode string) error
}

type UpstreamSchedulerSnapshot struct {
	HealthScore      float64 `json:"health_score"`
	PerformanceScore float64 `json:"performance_score"`
	CostScore        float64 `json:"cost_score"`
	CapacityScore    float64 `json:"capacity_score"`
}

type UpstreamRuntimeEvent struct {
	UpstreamID        int64
	AccountID         int64
	RemoteAPIKeyID    string
	RemoteGroupID     string
	LocalGroupID      *int64
	Model             string
	UserID            *int64
	Success           bool
	FirstTokenMs      *int
	DurationMs        int64
	StatusCode        int
	ErrorMessage      string
	Ignored           bool
	Reason            string
	StreamInterrupted bool
	Retried           bool
	Confidence        float64
	ObservedAt        time.Time
}
