import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export type UpstreamType = 'sub2api' | 'newapi' | 'openai_compatible' | 'custom'
export type UpstreamStatus = 'active' | 'degraded' | 'half_open' | 'circuit_open' | 'disabled'
export type UpstreamRoutingMode = 'stability' | 'balanced' | 'cost' | 'speed' | 'manual'
export type UpstreamAdminAuthMode = 'password' | 'token' | 'none'

export interface UpstreamForwardCredential {
  id?: number
  upstream_id?: number
  name: string
  auth_type: string
  api_key?: string
  api_key_masked?: string
  enabled: boolean
  expires_at?: string | null
  metadata?: Record<string, unknown>
  decrypt_failed?: boolean
}

export interface UpstreamAdminAuth {
  upstream_id?: number
  auth_mode: UpstreamAdminAuthMode
  login_url?: string
  username?: string
  username_masked?: string
  password?: string
  password_configured?: boolean
  access_token?: string
  access_token_masked?: string
  refresh_token?: string
  refresh_token_masked?: string
  token_expires_at?: string | null
  last_login_at?: string | null
  last_login_error?: string
  metadata?: Record<string, unknown>
  secret_decrypt_failed?: boolean
}

export interface UpstreamRemoteGroup {
  id: number
  upstream_id: number
  remote_group_id: string
  remote_group_name: string
  rate_multiplier: number
  status: string
  raw_snapshot?: Record<string, unknown>
  last_synced_at: string
}

export interface UpstreamRemoteAPIKey {
  id: number
  upstream_id: number
  remote_api_key_id: string
  remote_api_key_name: string
  masked_key: string
  api_key_configured?: boolean
  synced_remote_group_id?: string
  remote_group_id: string
  local_group_ids?: number[]
  status: string
  quota?: number | null
  used_quota?: number | null
  raw_snapshot?: Record<string, unknown>
  last_synced_at: string
}

export interface UpstreamAccountBalanceResult {
  upstream_id: number
  balance?: number | null
  quota?: number | null
  used_quota?: number | null
  remaining_quota?: number | null
  source?: string
  has_balance: boolean
  message: string
  checked_at: string
}

export interface UpstreamSyncRun {
  id: number
  upstream_id: number
  status: string
  groups_count: number
  api_keys_count: number
  message: string
  started_at: string
  finished_at?: string | null
  raw_result?: Record<string, unknown>
}

export interface UpstreamSyncDiffItem {
  kind: string
  id: string
  name?: string
  before?: Record<string, unknown>
  after?: Record<string, unknown>
  impact?: string
  local_group_ids?: number[]
}

export interface UpstreamSyncDiff {
  added_groups: UpstreamSyncDiffItem[]
  removed_groups: UpstreamSyncDiffItem[]
  changed_groups: UpstreamSyncDiffItem[]
  added_api_keys: UpstreamSyncDiffItem[]
  removed_api_keys: UpstreamSyncDiffItem[]
  changed_api_keys: UpstreamSyncDiffItem[]
  affected_local_group_ids: number[]
  unschedulable_api_key_ids: string[]
  cost_multiplier_change_count: number
}

export interface UpstreamSyncPreview {
  id: number
  upstream_id: number
  preview_token: string
  status: string
  diff: UpstreamSyncDiff
  groups?: UpstreamRemoteGroup[]
  api_keys?: UpstreamRemoteAPIKey[]
  created_at: string
  applied_at?: string | null
  expires_at: string
}

export interface UpstreamSchedulerSnapshot {
  health_score: number
  performance_score: number
  cost_score: number
  capacity_score: number
}

export interface UpstreamHealthWindow {
  window_seconds: number
  success_count: number
  error_count: number
  success_rate: number
  ttft_p50_ms?: number | null
  ttft_p90_ms?: number | null
  ttft_p95_ms?: number | null
  ttft_p99_ms?: number | null
}

export interface UpstreamAlert {
  id: number
  upstream_id?: number | null
  alert_type: string
  severity: 'info' | 'warning' | 'critical' | string
  status: 'active' | 'resolved' | string
  title: string
  message: string
  evidence?: Record<string, unknown>
  created_at: string
  resolved_at?: string | null
}

export interface UpstreamAttributionSignal {
  scope: string
  id: string
  error_count: number
  total_count: number
  error_rate: number
  confidence: number
  suggestion: string
}

export interface UpstreamHealthDashboard {
  upstream_id: number
  status: UpstreamStatus
  latest_probe_status: string
  latest_probe_first_token_ms?: number | null
  latest_probe_checked_at?: string | null
  recent_error_reason?: string
  recent_error_at?: string | null
  windows: UpstreamHealthWindow[]
  scheduler_snapshot: UpstreamSchedulerSnapshot
  degraded: boolean
  circuit_open: boolean
  recovering: boolean
  schedulable_api_keys: number
  servable_local_groups: number
  active_alerts?: UpstreamAlert[]
  attribution?: UpstreamAttributionSignal[]
}

export interface UpstreamEvent {
  id: number
  upstream_id: number
  event_type: string
  reason: string
  account_id?: number | null
  remote_api_key_id?: string
  remote_group_id?: string
  local_group_id?: number | null
  model?: string
  status_code: number
  first_token_ms?: number | null
  duration_ms: number
  user_id?: number | null
  stream_interrupted: boolean
  retried: boolean
  confidence: number
  evidence?: Record<string, unknown>
  created_at: string
}

export interface UpstreamGovernancePolicy {
  consecutive_failures_to_circuit_open: number
  first_token_degrade_threshold_ms: number
  error_rate_degrade_threshold: number
  recovery_probe_interval_seconds: number
  recovery_successes_required: number
  ignored_status_codes: number[]
  immediate_circuit_status_codes: number[]
  probe_failure_weight: number
  runtime_failure_weight: number
  alert_enabled: boolean
}

export interface UpstreamCostDimension {
  upstream_id: number
  upstream_name: string
  remote_group_id?: string
  remote_api_key_id?: string
  local_group_id?: number | null
  user_id?: number | null
  model?: string
  request_count: number
  local_billed_cost: number
  upstream_cost: number
  cost_delta: number
  gross_profit: number
  avg_multiplier: number
}

export interface UpstreamCostReport {
  start: string
  end: string
  dimension: string
  reset_at?: string | null
  items: UpstreamCostDimension[]
  totals: UpstreamCostDimension
}

export interface UpstreamCostReportResetResult {
  upstream_id: number
  reset_at: string
}

export interface Upstream {
  id: number
  name: string
  type: UpstreamType
  base_url: string
  status: UpstreamStatus
  priority: number
  weight: number
  cost_multiplier: number
  timeout_ms: number
  connect_timeout_ms: number
  retry_max: number
  probe_enabled: boolean
  probe_model: string
  probe_interval_seconds: number
  routing_mode: UpstreamRoutingMode
  notes: string
  last_synced_at?: string | null
  last_sync_status?: string
  last_sync_error?: string
  created_at: string
  updated_at: string
  groups_count: number
  api_keys_count: number
  latest_health_score: number
  forward_credential?: UpstreamForwardCredential | null
  admin_auth?: UpstreamAdminAuth | null
  remote_groups?: UpstreamRemoteGroup[]
  remote_api_keys?: UpstreamRemoteAPIKey[]
  latest_sync_run?: UpstreamSyncRun | null
  scheduler_snapshot?: UpstreamSchedulerSnapshot | null
  metadata?: Record<string, unknown>
  local_group_ids?: number[]
  decrypt_failed?: boolean
}

export interface UpstreamPayload {
  name?: string
  type?: UpstreamType
  base_url?: string
  status?: UpstreamStatus
  priority?: number
  weight?: number
  cost_multiplier?: number
  timeout_ms?: number
  connect_timeout_ms?: number
  retry_max?: number
  probe_enabled?: boolean
  probe_model?: string
  probe_interval_seconds?: number
  routing_mode?: UpstreamRoutingMode
  notes?: string
  metadata?: Record<string, unknown>
  local_group_ids?: number[]
  forward_credential?: Partial<UpstreamForwardCredential>
  admin_auth?: Partial<UpstreamAdminAuth>
}

export interface UpstreamSyncResult {
  run: UpstreamSyncRun
  groups: UpstreamRemoteGroup[]
  api_keys: UpstreamRemoteAPIKey[]
}

export interface UpstreamLoginTestResult {
  success: boolean
  has_token: boolean
  token_expires_at?: string | null
  message: string
}

export interface UpstreamProbeResult {
  upstream_id: number
  account_id?: number
  remote_api_key_id?: string
  remote_api_key_name?: string
  remote_group_id?: string
  model: string
  success: boolean
  ignored: boolean
  reason: string
  status_code: number
  first_token_ms?: number | null
  duration_ms: number
  error_message?: string
  scheduler_snapshot?: UpstreamSchedulerSnapshot | null
  checked_at: string
}

export interface UpstreamScoreWeights {
  health: number
  performance: number
  cost: number
  capacity: number
  priority: number
}

export interface UpstreamScoreBreakdown {
  upstream_id: number
  name: string
  score: number
  weighted_ticket: number
  health_score: number
  performance_score: number
  cost_score: number
  capacity_score: number
  priority_score: number
  weights: UpstreamScoreWeights
  filtered: boolean
  filter_reason?: string
  candidate_api_keys?: UpstreamAPIKeyScheduleCandidate[]
}

export interface UpstreamScheduleDecision {
  selected_id: number
  selected_name: string
  selected_remote_api_key_id?: string
  selected_remote_group_id?: string
  local_group_id?: number
  mode: UpstreamRoutingMode
  reason: string
  candidate_scores: UpstreamScoreBreakdown[]
  filtered: UpstreamScoreBreakdown[]
  candidate_api_keys?: UpstreamAPIKeyScheduleCandidate[]
  filtered_api_keys?: UpstreamAPIKeyScheduleCandidate[]
}

export interface UpstreamAPIKeyScheduleCandidate {
  upstream_id: number
  upstream_name: string
  remote_api_key_id: string
  remote_api_key_name: string
  remote_group_id: string
  local_group_ids?: number[]
  status: string
  schedulable: boolean
  filter_reason?: string
}

export interface UpstreamListFilters {
  type?: string
  status?: string
  search?: string
}

export interface UpstreamRoutingConfig {
  mode: UpstreamRoutingMode
}

export interface SchedulePreviewRequest {
  model?: string
  local_group_id?: number
  mode?: UpstreamRoutingMode
  random_seed?: number
}

export interface UpdateRemoteAPIKeyConfigPayload {
  local_group_ids: number[]
  api_key?: string
}

export async function list(
  page = 1,
  pageSize = 20,
  filters?: UpstreamListFilters,
  options?: { signal?: AbortSignal }
): Promise<BasePaginationResponse<Upstream>> {
  const { data } = await apiClient.get<BasePaginationResponse<Upstream>>('/admin/upstreams', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

export async function get(id: number): Promise<Upstream> {
  const { data } = await apiClient.get<Upstream>(`/admin/upstreams/${id}`)
  return data
}

export async function create(payload: UpstreamPayload): Promise<Upstream> {
  const { data } = await apiClient.post<Upstream>('/admin/upstreams', payload)
  return data
}

export async function update(id: number, payload: UpstreamPayload): Promise<Upstream> {
  const { data } = await apiClient.put<Upstream>(`/admin/upstreams/${id}`, payload)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/upstreams/${id}`)
}

export async function sync(id: number): Promise<UpstreamSyncResult> {
  const { data } = await apiClient.post<UpstreamSyncResult>(`/admin/upstreams/${id}/sync`)
  return data
}

export async function syncPreview(id: number): Promise<UpstreamSyncPreview> {
  const { data } = await apiClient.post<UpstreamSyncPreview>(`/admin/upstreams/${id}/sync-preview`)
  return data
}

export async function applySyncPreview(id: number, previewToken: string): Promise<UpstreamSyncResult> {
  const { data } = await apiClient.post<UpstreamSyncResult>(`/admin/upstreams/${id}/sync-preview/apply`, {
    preview_token: previewToken
  })
  return data
}

export async function testLogin(id: number): Promise<UpstreamLoginTestResult> {
  const { data } = await apiClient.post<UpstreamLoginTestResult>(`/admin/upstreams/${id}/test-login`)
  return data
}

export async function probe(id: number): Promise<UpstreamProbeResult> {
  const { data } = await apiClient.post<UpstreamProbeResult>(`/admin/upstreams/${id}/probe`)
  return data
}

export async function health(id: number): Promise<UpstreamHealthDashboard> {
  const { data } = await apiClient.get<UpstreamHealthDashboard>(`/admin/upstreams/${id}/health`)
  return data
}

export async function events(id: number, params?: { limit?: number; event_type?: string }): Promise<UpstreamEvent[]> {
  const { data } = await apiClient.get<UpstreamEvent[]>(`/admin/upstreams/${id}/events`, { params })
  return data
}

export async function listRemoteGroups(id: number): Promise<UpstreamRemoteGroup[]> {
  const { data } = await apiClient.get<UpstreamRemoteGroup[]>(`/admin/upstreams/${id}/remote-groups`)
  return data
}

export async function listRemoteAPIKeys(id: number): Promise<UpstreamRemoteAPIKey[]> {
  const { data } = await apiClient.get<UpstreamRemoteAPIKey[]>(`/admin/upstreams/${id}/remote-api-keys`)
  return data
}

export async function updateRemoteAPIKeyConfig(
  id: number,
  remoteAPIKeyID: string,
  payload: UpdateRemoteAPIKeyConfigPayload
): Promise<UpstreamRemoteAPIKey> {
  const { data } = await apiClient.put<UpstreamRemoteAPIKey>(
    `/admin/upstreams/${id}/remote-api-keys/${remoteAPIKeyID}`,
    payload
  )
  return data
}

export async function refreshBalance(id: number): Promise<UpstreamAccountBalanceResult> {
  const { data } = await apiClient.post<UpstreamAccountBalanceResult>(`/admin/upstreams/${id}/balance`)
  return data
}

export async function getRoutingConfig(): Promise<UpstreamRoutingConfig> {
  const { data } = await apiClient.get<UpstreamRoutingConfig>('/admin/upstreams/routing-config')
  return data
}

export async function updateRoutingConfig(mode: UpstreamRoutingMode): Promise<UpstreamRoutingConfig> {
  const { data } = await apiClient.put<UpstreamRoutingConfig>('/admin/upstreams/routing-config', { mode })
  return data
}

export async function schedulePreview(payload: SchedulePreviewRequest): Promise<UpstreamScheduleDecision> {
  const { data } = await apiClient.post<UpstreamScheduleDecision>('/admin/upstreams/schedule-preview', payload)
  return data
}

export async function getPolicy(id: number): Promise<UpstreamGovernancePolicy> {
  const { data } = await apiClient.get<UpstreamGovernancePolicy>(`/admin/upstreams/${id}/policy`)
  return data
}

export async function updatePolicy(id: number, payload: UpstreamGovernancePolicy): Promise<UpstreamGovernancePolicy> {
  const { data } = await apiClient.put<UpstreamGovernancePolicy>(`/admin/upstreams/${id}/policy`, payload)
  return data
}

export async function listAlerts(id: number, active = true): Promise<UpstreamAlert[]> {
  const { data } = await apiClient.get<UpstreamAlert[]>(`/admin/upstreams/${id}/alerts`, { params: { active } })
  return data
}

export async function resolveAlert(id: number, alertType: string): Promise<void> {
  await apiClient.put(`/admin/upstreams/${id}/alerts/${encodeURIComponent(alertType)}/resolve`)
}

export async function costReport(
  id: number,
  params?: { start?: string; end?: string; dimension?: string }
): Promise<UpstreamCostReport> {
  const { data } = await apiClient.get<UpstreamCostReport>(`/admin/upstreams/${id}/cost-report`, { params })
  return data
}

export async function resetCostReport(id: number): Promise<UpstreamCostReportResetResult> {
  const { data } = await apiClient.post<UpstreamCostReportResetResult>(`/admin/upstreams/${id}/cost-report/reset`)
  return data
}

const upstreamsAPI = {
  list,
  get,
  create,
  update,
  remove,
  sync,
  syncPreview,
  applySyncPreview,
  testLogin,
  probe,
  health,
  events,
  listRemoteGroups,
  listRemoteAPIKeys,
  updateRemoteAPIKeyConfig,
  refreshBalance,
  getRoutingConfig,
  updateRoutingConfig,
  schedulePreview,
  getPolicy,
  updatePolicy,
  listAlerts,
  resolveAlert,
  costReport,
  resetCostReport
}

export default upstreamsAPI
