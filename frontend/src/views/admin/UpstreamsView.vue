<template>
  <AppLayout>
    <div class="flex h-[calc(100vh-64px-4rem)] flex-col gap-4">
      <div class="flex flex-wrap items-center gap-3">
        <div class="min-w-56 flex-1 sm:max-w-72">
          <input
            v-model="searchQuery"
            type="text"
            class="input"
            :placeholder="tr('admin.upstreams.searchPlaceholder', 'Search upstreams')"
            @input="handleSearch"
          />
        </div>
        <Select
          v-model="filters.type"
          :options="typeFilterOptions"
          class="w-44"
          @change="handleFilterChange"
        />
        <Select
          v-model="filters.status"
          :options="statusFilterOptions"
          class="w-44"
          @change="handleFilterChange"
        />
        <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            :title="tr('common.refresh', 'Refresh')"
            @click="loadUpstreams"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button type="button" class="btn btn-primary" @click="openCreateDialog">
            <Icon name="plus" size="md" />
            {{ tr('admin.upstreams.create', 'Create upstream') }}
          </button>
        </div>
      </div>

      <div class="min-h-0 flex-1 overflow-hidden">
        <div class="flex h-full min-h-0 flex-col overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <DataTable
            :columns="columns"
            :data="upstreams"
            :loading="loading"
            row-key="id"
            :estimate-row-height="76"
            default-sort-key="priority"
            default-sort-order="asc"
          >
            <template #cell-name="{ row }">
              <button
                type="button"
                class="flex min-w-0 items-center gap-3 text-left"
                @click="selectUpstream(row)"
              >
                <span class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300">
                  <Icon name="server" size="md" />
                </span>
                <span class="min-w-0">
                  <span class="block truncate font-medium text-gray-900 dark:text-white">
                    {{ row.name }}
                  </span>
                  <span class="mt-1 block max-w-[260px] truncate text-xs text-gray-500 dark:text-dark-400">
                    {{ row.base_url }}
                  </span>
                </span>
              </button>
            </template>

            <template #cell-type="{ value }">
              <span class="badge badge-gray">{{ typeLabel(String(value)) }}</span>
            </template>

            <template #cell-status="{ value }">
              <span :class="['badge', statusBadgeClass(String(value))]">
                {{ statusLabel(String(value)) }}
              </span>
            </template>

            <template #cell-routing_mode="{ value }">
              <span class="badge badge-primary">{{ routingModeLabel(String(value)) }}</span>
            </template>

            <template #cell-weight="{ value, row }">
              <div class="text-sm tabular-nums text-gray-700 dark:text-gray-200">
                {{ value }}
                <span class="text-xs text-gray-400">/ P{{ row.priority }}</span>
              </div>
            </template>

            <template #cell-cost_multiplier="{ value }">
              <span class="font-mono text-sm text-gray-700 dark:text-gray-200">
                x{{ formatNumber(value, 3) }}
              </span>
            </template>

            <template #cell-groups_count="{ row }">
              <div class="text-sm text-gray-700 dark:text-gray-200">
                {{ row.groups_count }}
                <span class="text-xs text-gray-400">/ {{ row.api_keys_count }}</span>
              </div>
            </template>

            <template #cell-last_synced_at="{ row }">
              <div class="min-w-32 text-sm">
                <div class="text-gray-700 dark:text-gray-200">
                  {{ formatOptionalDate(row.last_synced_at) }}
                </div>
                <div v-if="row.last_sync_status" :class="['mt-1 text-xs', syncStatusClass(row.last_sync_status)]">
                  {{ syncStatusLabel(row.last_sync_status) }}
                </div>
              </div>
            </template>

            <template #cell-actions="{ row }">
              <div class="flex items-center gap-1">
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                  :title="tr('admin.upstreams.viewDetails', 'View details')"
                  @click="selectUpstream(row)"
                >
                  <Icon name="eye" size="sm" />
                </button>
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-300"
                  :title="tr('admin.upstreams.sync', 'Sync')"
                  :disabled="isSyncing(row.id)"
                  @click="syncUpstream(row)"
                >
                  <Icon name="sync" size="sm" :class="isSyncing(row.id) ? 'animate-spin' : ''" />
                </button>
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                  :title="tr('admin.upstreams.visit', '访问')"
                  :disabled="!upstreamVisitURL(row)"
                  @click="visitUpstream(row)"
                >
                  <Icon name="externalLink" size="sm" />
                </button>
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-300"
                  :title="tr('admin.upstreams.testLogin', 'Test login')"
                  :disabled="testingId === row.id"
                  @click="testLogin(row)"
                >
                  <Icon name="login" size="sm" />
                </button>
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-900/20 dark:hover:text-amber-300"
                  :title="tr('admin.upstreams.probe', 'Probe')"
                  :disabled="probeId === row.id"
                  @click="probeUpstream(row)"
                >
                  <Icon name="bolt" size="sm" :class="probeId === row.id ? 'animate-pulse' : ''" />
                </button>
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                  :title="tr('common.edit', 'Edit')"
                  @click="openEditDialog(row)"
                >
                  <Icon name="edit" size="sm" />
                </button>
                <button
                  type="button"
                  class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-300"
                  :title="tr('common.delete', 'Delete')"
                  @click="requestDelete(row)"
                >
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </template>

            <template #empty>
              <EmptyState
                :title="tr('admin.upstreams.emptyTitle', 'No upstreams yet')"
                :description="tr('admin.upstreams.emptyDescription', 'Create an external upstream to sync groups, API keys, multipliers, and preview routing decisions.')"
                :action-text="tr('admin.upstreams.create', 'Create upstream')"
                @action="openCreateDialog"
              />
            </template>
          </DataTable>
        </div>

      </div>

      <BaseDialog
        :show="showDetailDialog && !!selectedUpstream"
        :title="tr('admin.upstreams.detailTitle', '上游详情')"
        width="extra-wide"
        @close="closeDetailDialog"
      >
          <div v-if="selectedUpstream" class="mx-auto flex min-h-0 w-full max-w-6xl flex-col">
            <div class="border-b border-gray-200 pb-4 dark:border-dark-700">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                    {{ selectedUpstream.name }}
                  </h2>
                  <p class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">
                    {{ selectedUpstream.base_url }}
                  </p>
                </div>
                <span :class="['badge', statusBadgeClass(selectedUpstream.status)]">
                  {{ statusLabel(selectedUpstream.status) }}
                </span>
              </div>
              <div class="mt-3 grid grid-cols-2 gap-2 text-center sm:grid-cols-5">
                <div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
                  <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.apiKeys', 'API 密钥') }}</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ remoteAPIKeys.length }}</div>
                </div>
                <div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
                  <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.schedulableKeys', '可调度密钥') }}</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">{{ healthDashboard?.schedulable_api_keys ?? 0 }}</div>
                </div>
                <div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
                  <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.availableModels', '可用模型') }}</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">
                    {{ supportedModels.length > 0 ? supportedModels.length : tr('admin.upstreams.allModels', '全部') }}
                  </div>
                </div>
                <div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
                  <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.accountBalance', '上游余额') }}</div>
                  <div class="mt-1 font-mono text-sm font-semibold text-gray-900 dark:text-white">
                    {{ formatQuotaValue(upstreamAccountBalance) }}
                  </div>
                  <div class="mt-0.5 text-[11px] text-gray-500 dark:text-dark-400">
                    {{ tr('admin.upstreams.usedQuota', '已用') }} {{ formatQuotaValue(upstreamAccountUsedQuota) }}
                  </div>
                  <div class="mt-0.5 text-[11px] text-gray-500 dark:text-dark-400">
                    {{ formatOptionalDate(upstreamAccountBalanceCheckedAt) }}
                  </div>
                </div>
                <div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
                  <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.schedulerCost', '调度成本') }}</div>
                  <div class="mt-1 font-semibold text-gray-900 dark:text-white">x{{ formatNumber(selectedUpstream.cost_multiplier, 2) }}</div>
                </div>
              </div>
              <div class="mt-3 flex flex-wrap gap-2">
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="detailLoading || isSyncing(selectedUpstream.id)"
                  @click="syncUpstream(selectedUpstream)"
                >
                  <Icon name="sync" size="sm" :class="isSyncing(selectedUpstream.id) ? 'animate-spin' : ''" />
                  {{ tr('admin.upstreams.syncNow', '立即同步') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="detailLoading || syncPreviewLoading"
                  @click="previewSync(selectedUpstream)"
                >
                  <Icon name="eye" size="sm" />
                  {{ tr('admin.upstreams.previewSync', '预览同步') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="testingId === selectedUpstream.id"
                  @click="testLogin(selectedUpstream)"
                >
                  <Icon name="login" size="sm" />
                  {{ tr('admin.upstreams.testLogin', '测试登录') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="refreshingBalance || detailLoading"
                  @click="refreshUpstreamBalance"
                >
                  <Icon name="refresh" size="sm" :class="refreshingBalance ? 'animate-spin' : ''" />
                  {{ tr('admin.upstreams.queryBalance', '查询余额') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="!upstreamVisitURL(selectedUpstream)"
                  @click="visitUpstream(selectedUpstream)"
                >
                  <Icon name="externalLink" size="sm" />
                  {{ tr('admin.upstreams.visit', '访问') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="probeId === selectedUpstream.id"
                  @click="probeUpstream(selectedUpstream)"
                >
                  <Icon name="bolt" size="sm" :class="probeId === selectedUpstream.id ? 'animate-pulse' : ''" />
                  {{ tr('admin.upstreams.probe', '探测') }}
                </button>
                <button type="button" class="btn btn-secondary btn-sm" @click="openEditDialog(selectedUpstream)">
                  <Icon name="edit" size="sm" />
                  {{ tr('common.edit', '编辑') }}
                </button>
              </div>
            </div>

            <div class="min-h-0 flex-1 py-4">
              <div v-if="detailLoading" class="space-y-3">
                <div class="skeleton h-16"></div>
                <div class="skeleton h-32"></div>
                <div class="skeleton h-32"></div>
              </div>
              <template v-else>
                <section class="space-y-3">
                  <div class="flex items-center justify-between">
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                      {{ tr('admin.upstreams.healthBoard', '健康看板') }}
                    </h3>
                    <button type="button" class="btn btn-secondary btn-sm" @click="refreshEventsAndHealth">
                      <Icon name="refresh" size="sm" />
                    </button>
                  </div>
                  <div class="grid grid-cols-2 gap-2">
                    <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900">
                      <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.healthScore', '健康分') }}</div>
                      <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                        {{ formatPercent(healthDashboard?.scheduler_snapshot.health_score) }}
                      </div>
                    </div>
                    <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900">
                      <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.performanceScore', '性能分') }}</div>
                      <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                        {{ formatPercent(healthDashboard?.scheduler_snapshot.performance_score) }}
                      </div>
                    </div>
                    <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900">
                      <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.latestProbe', '最近探针') }}</div>
                      <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
                        {{ eventTypeLabel(healthDashboard?.latest_probe_status || 'unknown') }}
                      </div>
                      <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                        {{ formatMS(healthDashboard?.latest_probe_first_token_ms) }}
                      </div>
                    </div>
                    <div class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900">
                      <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.schedulableKeys', '可调度密钥') }}</div>
                      <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                        {{ healthDashboard?.schedulable_api_keys ?? 0 }}
                        <span class="text-xs text-gray-400">/ {{ healthDashboard?.servable_local_groups ?? 0 }}</span>
                      </div>
                    </div>
                  </div>
                  <div class="grid grid-cols-3 gap-2 text-xs">
                    <div
                      v-for="win in healthDashboard?.windows || []"
                      :key="win.window_seconds"
                      class="rounded-lg border border-gray-100 px-2 py-2 dark:border-dark-700"
                    >
                      <div class="font-medium text-gray-900 dark:text-white">
                        {{ win.window_seconds === 3600 ? '1h' : win.window_seconds === 21600 ? '6h' : '24h' }}
                      </div>
                      <div class="mt-1 text-gray-500 dark:text-dark-400">
                        {{ formatPercent(win.success_rate) }} · P95 {{ formatMS(win.ttft_p95_ms) }}
                      </div>
                    </div>
                  </div>
                  <div v-if="healthDashboard?.recent_error_reason" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200">
                    {{ healthDashboard.recent_error_reason }}
                  </div>
                </section>

                <section v-if="upstreamAlerts.length > 0" class="mt-5 space-y-3">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ tr('admin.upstreams.alerts', '告警') }}
                  </h3>
                  <div class="space-y-2">
                    <div
                      v-for="alert in upstreamAlerts"
                      :key="alert.id"
                      class="rounded-lg border border-gray-100 px-3 py-2 dark:border-dark-700"
                    >
                      <div class="flex items-start justify-between gap-2">
                        <div class="min-w-0">
                          <div class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ alert.title }}</div>
                          <div class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-dark-400">{{ alert.message }}</div>
                        </div>
                        <span :class="['badge', alertBadgeClass(alert.severity)]">{{ alertSeverityLabel(alert.severity) }}</span>
                      </div>
                      <div class="mt-2 flex items-center justify-between gap-2">
                        <span class="text-xs text-gray-400">{{ formatOptionalDate(alert.created_at) }}</span>
                        <button type="button" class="btn btn-secondary btn-sm" @click="resolveUpstreamAlert(alert)">
                          {{ tr('admin.upstreams.resolve', '解除') }}
                        </button>
                      </div>
                    </div>
                  </div>
                </section>

                <section class="mt-5 space-y-3">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <div>
                      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                        {{ tr('admin.upstreams.apiKeyMappings', 'API key 配置') }}
                      </h3>
                      <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                        {{ tr('admin.upstreams.apiKeyMappingsHint', '逐个配置上游 API key，并选择它映射到哪些本地 OpenAI 分组；调度会选择具体 key。') }}
                      </p>
                    </div>
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm"
                      :disabled="detailLoading"
                      @click="selectedUpstream && loadDetail(selectedUpstream.id)"
                    >
                      <Icon name="refresh" size="sm" />
                    </button>
                  </div>
                  <div v-if="remoteAPIKeys.length === 0" class="rounded-lg border border-dashed border-gray-200 p-4 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
                    {{ tr('admin.upstreams.noAPIKeys', '暂无已同步 API 密钥') }}
                  </div>
                  <div v-else class="space-y-3">
                    <div
                      v-for="key in remoteAPIKeys"
                      :key="key.remote_api_key_id"
                      class="rounded-lg border border-gray-100 p-3 dark:border-dark-700"
                    >
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="min-w-0">
                          <div class="flex flex-wrap items-center gap-2">
                            <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                              {{ key.remote_api_key_name || key.remote_api_key_id }}
                            </span>
                            <span :class="['badge', remoteAPIKeyConfigured(key) ? 'badge-success' : 'badge-warning']">
                              {{ remoteAPIKeyConfigured(key) ? tr('admin.upstreams.keyConfigured', '已配置') : tr('admin.upstreams.keyNotConfigured', '未配置') }}
                            </span>
                            <span :class="['badge', remoteAPIKeySchedulable(key) ? 'badge-success' : 'badge-gray']">
                              {{ remoteAPIKeySchedulable(key) ? tr('admin.upstreams.keySchedulable', '可调度') : tr('admin.upstreams.keyNotSchedulable', '不可调度') }}
                            </span>
                          </div>
                          <div class="mt-1 flex flex-wrap gap-1 text-[11px] text-gray-500 dark:text-dark-400">
                            <span class="font-mono">{{ key.remote_api_key_id }}</span>
                            <span v-if="key.masked_key">· {{ key.masked_key }}</span>
                          </div>
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                          <button
                            type="button"
                            class="btn btn-primary btn-sm"
                            :disabled="savingRemoteKeyId === key.remote_api_key_id"
                            @click="saveRemoteAPIKeyConfig(key)"
                          >
                            <Icon name="check" size="sm" />
                            {{ savingRemoteKeyId === key.remote_api_key_id ? tr('common.saving', '保存中') : tr('common.save', '保存') }}
                          </button>
                        </div>
                      </div>

                      <div class="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-[minmax(220px,0.8fr)_minmax(0,1.2fr)]">
                        <label class="space-y-1">
                          <span class="input-label">{{ tr('admin.upstreams.upstreamAPIKeySecret', '上游 API key') }}</span>
                          <input
                            :value="remoteKeyDraft(key.remote_api_key_id).api_key"
                            type="password"
                            class="input"
                            :placeholder="key.api_key_configured ? tr('admin.upstreams.keepExistingKey', '留空则保留已配置密钥') : 'sk-...'"
                            autocomplete="new-password"
                            @input="remoteKeyDraft(key.remote_api_key_id).api_key = ($event.target as HTMLInputElement).value"
                          />
                        </label>
                        <div class="space-y-2">
                          <span class="input-label">{{ tr('admin.upstreams.mappedLocalGroups', '映射本地分组') }}</span>
                          <div v-if="localGroups.length === 0" class="rounded-lg border border-dashed border-gray-200 px-3 py-2 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
                            {{ localGroupsLoading ? tr('common.loading', '加载中') : tr('admin.upstreams.noLocalRuntimeGroups', '暂无本地 OpenAI / Anthropic 分组') }}
                          </div>
                          <div v-else class="flex max-h-28 flex-wrap gap-2 overflow-y-auto rounded-lg border border-gray-100 p-2 dark:border-dark-700">
                            <label
                              v-for="group in localGroups"
                              :key="group.id"
                              class="inline-flex items-center gap-2 rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-700 dark:border-dark-600 dark:text-gray-200"
                            >
                              <input
                                type="checkbox"
                                class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                                :checked="remoteKeyDraft(key.remote_api_key_id).local_group_ids.includes(group.id)"
                                @change="toggleRemoteKeyLocalGroup(key.remote_api_key_id, group.id)"
                              />
                              <span>{{ localGroupLabel(group) }}</span>
                            </label>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </section>

                <section class="mt-5 space-y-3">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <div>
                      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                        {{ tr('admin.upstreams.availableModels', '可用模型') }}
                      </h3>
                      <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                        {{ tr('admin.upstreams.availableModelsHint', '选择允许参与调度的模型；为空表示全部模型。支持自定义通配符。') }}
                      </p>
                    </div>
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm"
                      :disabled="savingModels || !selectedUpstream"
                      @click="saveSupportedModels"
                    >
                      {{ savingModels ? tr('common.saving', '保存中') : tr('common.save', '保存') }}
                    </button>
                  </div>
                  <ModelWhitelistSelector
                    v-model="supportedModelsList"
                    :platforms="upstreamModelPlatforms"
                  />
                  <p class="text-xs text-gray-500 dark:text-dark-400">
                    {{ t('admin.accounts.selectedModels', { count: supportedModelsList.length }) }}
                    <span v-if="supportedModelsList.length === 0">{{ t('admin.accounts.supportsAllModels') }}</span>
                  </p>
                </section>

                <section v-if="syncPreviewResult" class="mt-5 space-y-3">
                  <div class="flex items-center justify-between">
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                      {{ tr('admin.upstreams.syncDiffPreview', '同步差异预览') }}
                    </h3>
                    <button
                      type="button"
                      class="btn btn-primary btn-sm"
                      :disabled="applyingPreview"
                      @click="applySyncPreview"
                    >
                      {{ applyingPreview ? tr('common.saving', '保存中') : tr('admin.upstreams.applyPreview', '确认应用') }}
                    </button>
                  </div>
                  <div class="rounded-lg border border-gray-100 p-3 text-xs dark:border-dark-700">
                    <div class="grid grid-cols-3 gap-2 text-center">
                      <div>
                        <div class="font-semibold text-gray-900 dark:text-white">{{ syncDiffTotal(syncPreviewResult.diff) }}</div>
                        <div class="text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.changes', '变更') }}</div>
                      </div>
                      <div>
                        <div class="font-semibold text-gray-900 dark:text-white">{{ syncPreviewResult.diff.unschedulable_api_key_ids.length }}</div>
                        <div class="text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.unschedulable', '不可调度') }}</div>
                      </div>
                      <div>
                        <div class="font-semibold text-gray-900 dark:text-white">{{ syncPreviewResult.diff.cost_multiplier_change_count }}</div>
                        <div class="text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.rateChanges', '倍率变化') }}</div>
                      </div>
                    </div>
                    <div class="mt-3 space-y-1 text-gray-600 dark:text-dark-300">
                      <div v-if="syncPreviewResult.diff.added_groups.length">+ {{ syncPreviewResult.diff.added_groups.length }} {{ tr('admin.upstreams.groups', '分组') }}</div>
                      <div v-if="syncPreviewResult.diff.removed_groups.length">- {{ syncPreviewResult.diff.removed_groups.length }} {{ tr('admin.upstreams.groups', '分组') }}</div>
                      <div v-if="syncPreviewResult.diff.changed_groups.length">~ {{ syncPreviewResult.diff.changed_groups.length }} {{ tr('admin.upstreams.groups', '分组') }}</div>
                      <div v-if="syncPreviewResult.diff.added_api_keys.length">+ {{ syncPreviewResult.diff.added_api_keys.length }} {{ tr('admin.upstreams.apiKeys', 'API 密钥') }}</div>
                      <div v-if="syncPreviewResult.diff.removed_api_keys.length">- {{ syncPreviewResult.diff.removed_api_keys.length }} {{ tr('admin.upstreams.apiKeys', 'API 密钥') }}</div>
                      <div v-if="syncPreviewResult.diff.changed_api_keys.length">~ {{ syncPreviewResult.diff.changed_api_keys.length }} {{ tr('admin.upstreams.apiKeys', 'API 密钥') }}</div>
                    </div>
                  </div>
                </section>

                <section class="mt-5 space-y-3">
                  <div class="flex items-center justify-between">
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                      {{ tr('admin.upstreams.schedulePreview', '调度预览') }}
                    </h3>
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm"
                      :disabled="previewLoading"
                      @click="runSchedulePreview"
                    >
                      <Icon name="bolt" size="sm" />
                      {{ tr('admin.upstreams.preview', '预览') }}
                    </button>
                  </div>
                  <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
                    <input
                      v-model.trim="previewForm.model"
                      type="text"
                      class="input"
                      :placeholder="tr('admin.upstreams.modelPlaceholder', '模型，可选')"
                    />
                    <Select
                      v-model="previewForm.mode"
                      :options="routingModeOptions"
                    />
                    <Select
                      v-model="previewForm.local_group_id"
                      :options="previewLocalGroupOptions"
                      clearable
                      class="sm:col-span-2"
                    />
                  </div>

                  <div v-if="scheduleDecision" class="space-y-3 rounded-lg border border-gray-100 p-3 dark:border-dark-700">
                    <div class="flex items-start justify-between gap-3">
                      <div>
                        <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.selected', '选中') }}</div>
                        <div class="font-semibold text-gray-900 dark:text-white">
                          {{ selectedScheduleKeyLabel }}
                        </div>
                      </div>
                      <span class="badge badge-success">{{ routingModeLabel(scheduleDecision.mode) }}</span>
                    </div>
                    <p class="text-xs text-gray-500 dark:text-dark-400">{{ scheduleDecision.reason }}</p>
                    <div class="space-y-2">
                      <div
                        v-for="score in scheduleDecision.candidate_scores"
                        :key="score.upstream_id"
                        class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900"
                      >
                        <div class="flex items-center justify-between text-sm">
                          <span class="font-medium text-gray-900 dark:text-white">{{ score.name }}</span>
                          <span class="font-mono text-gray-700 dark:text-gray-200">{{ formatNumber(score.score, 4) }}</span>
                        </div>
                        <div class="mt-2 grid grid-cols-5 gap-1 text-center text-[11px] text-gray-500 dark:text-dark-400">
                          <span>{{ tr('admin.upstreams.scoreHealth', '健康') }} {{ formatNumber(score.health_score, 2) }}</span>
                          <span>{{ tr('admin.upstreams.scorePerformance', '性能') }} {{ formatNumber(score.performance_score, 2) }}</span>
                          <span>{{ tr('admin.upstreams.scoreCost', '成本') }} {{ formatNumber(score.cost_score, 2) }}</span>
                          <span>{{ tr('admin.upstreams.scoreCapacity', '容量') }} {{ formatNumber(score.capacity_score, 2) }}</span>
                          <span>{{ tr('admin.upstreams.scorePriority', '优先') }} {{ formatNumber(score.priority_score, 2) }}</span>
                        </div>
                      </div>
                    </div>
                    <div v-if="scheduleDecision.filtered.length > 0" class="text-xs text-gray-500 dark:text-dark-400">
                      {{ tr('admin.upstreams.filtered', '已过滤') }}:
                      {{ scheduleDecision.filtered.map(item => `${item.name || item.upstream_id}: ${item.filter_reason}`).join('; ') }}
                    </div>
                    <div v-if="(scheduleDecision.candidate_api_keys || []).length > 0" class="space-y-2">
                      <div class="text-xs font-medium text-gray-500 dark:text-dark-400">
                        {{ tr('admin.upstreams.candidateAPIKeys', '候选 API 密钥') }}
                      </div>
                      <div
                        v-for="key in scheduleDecision.candidate_api_keys"
                        :key="`${key.upstream_id}-${key.remote_api_key_id}`"
                        class="rounded-lg bg-gray-50 px-3 py-2 text-xs dark:bg-dark-900"
                      >
                        <div class="flex items-center justify-between gap-2">
                          <span class="truncate font-medium text-gray-900 dark:text-white">
                            {{ key.remote_api_key_name || key.remote_api_key_id }}
                          </span>
                          <span class="badge badge-success">{{ localGroupNames(key.local_group_ids) }}</span>
                        </div>
                      </div>
                    </div>
                    <div v-if="(scheduleDecision.filtered_api_keys || []).length > 0" class="text-xs text-gray-500 dark:text-dark-400">
                      {{ tr('admin.upstreams.filteredAPIKeys', '已过滤 API 密钥') }}:
                      {{ (scheduleDecision.filtered_api_keys || []).map(item => `${item.remote_api_key_name || item.remote_api_key_id}: ${item.filter_reason}`).join('; ') }}
                    </div>
                  </div>
                </section>

                <section class="mt-5 space-y-3">
                  <div class="flex items-center justify-between">
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                      {{ tr('admin.upstreams.policyConfig', '治理策略') }}
                    </h3>
                    <button type="button" class="btn btn-secondary btn-sm" :disabled="policySaving" @click="savePolicy">
                      {{ policySaving ? tr('common.saving', '保存中') : tr('common.save', '保存') }}
                    </button>
                  </div>
                  <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.failuresToCircuit', '连续失败熔断阈值') }}</span>
                      <input v-model.number="policyForm.consecutive_failures_to_circuit_open" type="number" min="1" class="input" />
                    </label>
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.ttftThreshold', '首 token 降权阈值（毫秒）') }}</span>
                      <input v-model.number="policyForm.first_token_degrade_threshold_ms" type="number" min="1" class="input" />
                    </label>
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.errorRateThreshold', '错误率降权阈值') }}</span>
                      <input v-model.number="policyForm.error_rate_degrade_threshold" type="number" min="0" max="1" step="0.01" class="input" />
                    </label>
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.recoveryInterval', '恢复探测间隔（秒）') }}</span>
                      <input v-model.number="policyForm.recovery_probe_interval_seconds" type="number" min="1" class="input" />
                    </label>
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.recoverySuccesses', '恢复所需成功次数') }}</span>
                      <input v-model.number="policyForm.recovery_successes_required" type="number" min="1" class="input" />
                    </label>
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.ignoredCodes', '忽略状态码') }}</span>
                      <input v-model.trim="policyForm.ignored_status_codes" type="text" class="input" :placeholder="tr('admin.upstreams.statusCodesPlaceholder', '例如 400,404')" />
                    </label>
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.circuitCodes', '立即熔断状态码') }}</span>
                      <input v-model.trim="policyForm.immediate_circuit_status_codes" type="text" class="input" :placeholder="tr('admin.upstreams.statusCodesPlaceholder', '例如 429,500,502')" />
                    </label>
                    <label class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700 dark:border-dark-700 dark:text-gray-200">
                      <input v-model="policyForm.alert_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                      {{ tr('admin.upstreams.alertEnabled', '启用告警') }}
                    </label>
                  </div>
                </section>

                <section class="mt-5 space-y-3">
                  <div class="flex items-center justify-between">
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                      {{ tr('admin.upstreams.costReconciliation', '成本对账') }}
                    </h3>
                    <div class="flex items-center gap-2">
                      <button
                        type="button"
                        class="btn btn-secondary btn-sm"
                        :disabled="costLoading || resettingCost"
                        @click="requestResetCostReport"
                      >
                        <Icon name="trash" size="sm" />
                        {{ tr('admin.upstreams.resetCostReport', '重置') }}
                      </button>
                      <button type="button" class="btn btn-secondary btn-sm" :disabled="costLoading" @click="refreshCostReport">
                        <Icon name="refresh" size="sm" :class="costLoading ? 'animate-spin' : ''" />
                      </button>
                    </div>
                  </div>
                  <div v-if="costReport?.reset_at" class="text-xs text-gray-500 dark:text-dark-400">
                    {{ tr('admin.upstreams.costReportResetAt', '统计起点') }}: {{ formatOptionalDate(costReport.reset_at) }}
                  </div>
                  <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.costDimension', '对账维度') }}</span>
                      <Select v-model="costForm.dimension" :options="costDimensionOptions" @change="refreshCostReport" />
                    </label>
                    <label class="space-y-1">
                      <span class="input-label">{{ tr('admin.upstreams.costWindowHours', '统计窗口（小时）') }}</span>
                      <input v-model.number="costForm.hours" type="number" min="1" max="720" class="input" @change="refreshCostReport" />
                    </label>
                  </div>
                  <div class="grid grid-cols-3 gap-2 text-center">
                    <div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
                      <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.localBilled', '本地计费') }}</div>
                      <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ formatCurrency(costReport?.totals.local_billed_cost) }}</div>
                    </div>
                    <div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
                      <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.upstreamCost', '上游成本') }}</div>
                      <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ formatCurrency(costReport?.totals.upstream_cost) }}</div>
                    </div>
                    <div class="rounded-lg bg-gray-50 px-2 py-2 dark:bg-dark-900">
                      <div class="text-xs text-gray-500 dark:text-dark-400">{{ tr('admin.upstreams.grossProfit', '毛利') }}</div>
                      <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ formatCurrency(costReport?.totals.gross_profit) }}</div>
                    </div>
                  </div>
                  <div v-if="costReport?.items.length" class="max-h-52 space-y-2 overflow-y-auto pr-1">
                    <div
                      v-for="item in costReport.items"
                      :key="`${costReport.dimension}-${costDimensionLabel(item)}`"
                      class="rounded-lg border border-gray-100 px-3 py-2 text-xs dark:border-dark-700"
                    >
                      <div class="flex items-center justify-between gap-2">
                        <span class="truncate font-medium text-gray-900 dark:text-white">{{ costDimensionLabel(item) }}</span>
                        <span class="text-gray-500 dark:text-dark-400">{{ item.request_count }}</span>
                      </div>
                      <div class="mt-1 text-gray-500 dark:text-dark-400">
                        {{ formatCurrency(item.local_billed_cost) }} / {{ formatCurrency(item.upstream_cost) }} · {{ formatCurrency(item.gross_profit) }}
                      </div>
                    </div>
                  </div>
                </section>

                <section class="mt-5 space-y-3">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ tr('admin.upstreams.eventTimeline', '事件时间线') }}
                  </h3>
                  <div v-if="upstreamEvents.length === 0" class="rounded-lg border border-dashed border-gray-200 p-4 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
                    {{ tr('admin.upstreams.noEvents', '暂无事件') }}
                  </div>
                  <div v-else class="max-h-80 space-y-2 overflow-y-auto pr-1">
                    <div
                      v-for="event in upstreamEvents"
                      :key="event.id"
                      class="rounded-lg border border-gray-100 px-3 py-2 dark:border-dark-700"
                    >
                      <div class="flex items-center justify-between gap-2">
                        <span class="truncate text-sm font-medium text-gray-900 dark:text-white">
                          {{ eventTypeLabel(event.event_type) }}
                        </span>
                        <span class="text-xs text-gray-400">{{ formatOptionalDate(event.created_at) }}</span>
                      </div>
                      <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                        {{ event.reason || '-' }}
                      </div>
                      <div class="mt-2 flex flex-wrap gap-1 text-[11px]">
                        <span v-if="event.remote_api_key_id" class="badge badge-gray">{{ event.remote_api_key_id }}</span>
                        <span v-if="event.remote_group_id" class="badge badge-primary">{{ event.remote_group_id }}</span>
                        <span v-if="event.status_code" class="badge badge-gray">HTTP {{ event.status_code }}</span>
                        <span v-if="event.first_token_ms" class="badge badge-gray">{{ formatMS(event.first_token_ms) }}</span>
                      </div>
                    </div>
                  </div>
                </section>
              </template>
            </div>
          </div>

      </BaseDialog>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>

    <BaseDialog
      :show="showEditDialog"
      :title="editingUpstream ? tr('admin.upstreams.edit', 'Edit upstream') : tr('admin.upstreams.create', 'Create upstream')"
      width="extra-wide"
      @close="closeEditDialog"
    >
      <form id="upstream-form" class="space-y-6" @submit.prevent="saveUpstream">
        <section>
          <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
            {{ tr('admin.upstreams.basicConfig', 'Basic config') }}
          </h3>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label class="input-label">{{ tr('admin.upstreams.name', 'Name') }}</label>
              <input v-model.trim="form.name" type="text" class="input" required />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.baseURL', 'Base URL') }}</label>
              <input v-model.trim="form.base_url" type="url" class="input" required placeholder="https://upstream.example.com" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.openaiAPIBaseURL', 'OpenAI API Base URL') }}</label>
              <input v-model.trim="form.openai_api_base_url" type="url" class="input" placeholder="https://upstream.example.com/v1" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.type', 'Type') }}</label>
              <Select v-model="form.type" :options="typeOptions" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.status', 'Status') }}</label>
              <Select v-model="form.status" :options="statusOptions" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.routingMode', 'Routing mode') }}</label>
              <Select v-model="form.routing_mode" :options="routingModeOptions" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.probeModel', 'Probe model') }}</label>
              <input v-model.trim="form.probe_model" type="text" class="input" placeholder="gpt-4o-mini" />
            </div>
          </div>
        </section>

        <section>
          <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
            {{ tr('admin.upstreams.routingConfig', 'Routing and timeout') }}
          </h3>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div>
              <label class="input-label">{{ tr('admin.upstreams.priority', 'Priority') }}</label>
              <input v-model.number="form.priority" type="number" min="1" class="input" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.weight', 'Weight') }}</label>
              <input v-model.number="form.weight" type="number" min="0" class="input" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.costMultiplier', '调度成本权重') }}</label>
              <input v-model.number="form.cost_multiplier" type="number" min="0" step="0.0001" class="input" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.timeoutMS', 'Timeout ms') }}</label>
              <input v-model.number="form.timeout_ms" type="number" min="1000" class="input" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.connectTimeoutMS', 'Connect timeout ms') }}</label>
              <input v-model.number="form.connect_timeout_ms" type="number" min="500" class="input" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.retryMax', 'Retries') }}</label>
              <input v-model.number="form.retry_max" type="number" min="0" class="input" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.probeInterval', 'Probe interval seconds') }}</label>
              <input v-model.number="form.probe_interval_seconds" type="number" min="10" class="input" />
            </div>
            <label class="flex items-center gap-2 pt-7 text-sm font-medium text-gray-700 dark:text-gray-300">
              <input v-model="form.probe_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ tr('admin.upstreams.probeEnabled', 'Enable probes') }}
            </label>
            <label class="flex items-center gap-2 pt-7 text-sm font-medium text-gray-700 dark:text-gray-300">
              <input v-model="form.auto_sync_enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ tr('admin.upstreams.autoSyncEnabled', '启用定时同步') }}
            </label>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.autoSyncIntervalMinutes', '同步间隔（分钟）') }}</label>
              <input v-model.number="form.auto_sync_interval_minutes" type="number" min="1" max="1440" class="input" />
            </div>
          </div>
        </section>

        <section>
          <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
            {{ tr('admin.upstreams.adminAuth', 'Upstream admin auth') }}
          </h3>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label class="input-label">{{ tr('admin.upstreams.authMode', 'Auth mode') }}</label>
              <Select v-model="form.admin_auth_mode" :options="adminAuthModeOptions" />
            </div>
            <div>
              <label class="input-label">{{ tr('admin.upstreams.loginURL', 'Login URL') }}</label>
              <input v-model.trim="form.admin_login_url" type="url" class="input" placeholder="https://upstream.example.com/api/user/login" />
            </div>
            <template v-if="form.admin_auth_mode === 'password'">
              <div>
                <label class="input-label">{{ tr('admin.upstreams.username', 'Username') }}</label>
                <input v-model.trim="form.admin_username" type="text" class="input" :placeholder="form.admin_username_placeholder" autocomplete="username" />
              </div>
              <div>
                <label class="input-label">{{ tr('admin.upstreams.password', 'Password') }}</label>
                <input v-model.trim="form.admin_password" type="password" class="input" :placeholder="form.admin_password_placeholder" autocomplete="new-password" />
              </div>
            </template>
            <template v-if="form.admin_auth_mode === 'token'">
              <div>
                <label class="input-label">{{ tr('admin.upstreams.accessToken', 'Access token') }}</label>
                <input v-model.trim="form.admin_access_token" type="password" class="input" :placeholder="form.admin_access_token_placeholder" autocomplete="new-password" />
              </div>
              <div>
                <label class="input-label">{{ tr('admin.upstreams.refreshToken', 'Refresh token') }}</label>
                <input v-model.trim="form.admin_refresh_token" type="password" class="input" :placeholder="form.admin_refresh_token_placeholder" autocomplete="new-password" />
              </div>
            </template>
          </div>
        </section>

        <section>
          <label class="input-label">{{ tr('admin.upstreams.notes', 'Notes') }}</label>
          <textarea v-model.trim="form.notes" rows="3" class="input"></textarea>
        </section>
      </form>

      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeEditDialog">
          {{ tr('common.cancel', 'Cancel') }}
        </button>
        <button type="submit" form="upstream-form" class="btn btn-primary" :disabled="saving">
          {{ saving ? tr('common.saving', 'Saving') : tr('common.save', 'Save') }}
        </button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="tr('admin.upstreams.deleteTitle', 'Delete upstream')"
      :message="deleteMessage"
      :confirm-text="tr('common.delete', 'Delete')"
      :cancel-text="tr('common.cancel', 'Cancel')"
      danger
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />

    <ConfirmDialog
      :show="showResetCostDialog"
      :title="tr('admin.upstreams.resetCostReportTitle', '重置成本统计')"
      :message="resetCostMessage"
      :confirm-text="tr('admin.upstreams.resetCostReport', '重置')"
      :cancel-text="tr('common.cancel', 'Cancel')"
      danger
      @confirm="confirmResetCostReport"
      @cancel="showResetCostDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  Upstream,
  UpstreamAccountBalanceResult,
  UpstreamAdminAuthMode,
  UpstreamAlert,
  UpstreamCostReport,
  UpstreamEvent,
  UpstreamGovernancePolicy,
  UpstreamHealthDashboard,
  UpstreamPayload,
  UpstreamRemoteAPIKey,
  UpstreamRoutingMode,
  UpstreamScheduleDecision,
  UpstreamSyncPreview,
  UpstreamStatus,
  UpstreamType
} from '@/api/admin/upstreams'
import type { Column } from '@/components/common/types'
import type { SelectOption } from '@/components/common/Select.vue'
import type { AdminGroup, GroupPlatform } from '@/types'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatDateTime } from '@/utils/format'

import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelWhitelistSelector from '@/components/account/ModelWhitelistSelector.vue'

const { t } = useI18n()
const appStore = useAppStore()

const tr = (key: string, fallback: string) => t(key, fallback)

type UpstreamForm = {
  name: string
  type: UpstreamType
  base_url: string
  openai_api_base_url: string
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
  auto_sync_enabled: boolean
  auto_sync_interval_minutes: number
  routing_mode: UpstreamRoutingMode
  notes: string
  admin_auth_mode: UpstreamAdminAuthMode
  admin_login_url: string
  admin_username: string
  admin_username_placeholder: string
  admin_password: string
  admin_password_placeholder: string
  admin_access_token: string
  admin_access_token_placeholder: string
  admin_refresh_token: string
  admin_refresh_token_placeholder: string
  metadata: Record<string, unknown>
}

type RemoteAPIKeyDraft = {
  api_key: string
  local_group_ids: number[]
}

const upstreams = ref<Upstream[]>([])
const loading = ref(false)
const saving = ref(false)
const savingModels = ref(false)
const localGroupsLoading = ref(false)
const testingId = ref<number | null>(null)
const probeId = ref<number | null>(null)
const syncingIds = ref<Set<number>>(new Set())
const selectedUpstream = ref<Upstream | null>(null)
const showDetailDialog = ref(false)
const detailLoading = ref(false)
const remoteAPIKeys = ref<UpstreamRemoteAPIKey[]>([])
const scheduleDecision = ref<UpstreamScheduleDecision | null>(null)
const healthDashboard = ref<UpstreamHealthDashboard | null>(null)
const upstreamEvents = ref<UpstreamEvent[]>([])
const syncPreviewResult = ref<UpstreamSyncPreview | null>(null)
const governancePolicy = ref<UpstreamGovernancePolicy | null>(null)
const upstreamAlerts = ref<UpstreamAlert[]>([])
const costReport = ref<UpstreamCostReport | null>(null)
const previewLoading = ref(false)
const syncPreviewLoading = ref(false)
const applyingPreview = ref(false)
const policySaving = ref(false)
const costLoading = ref(false)
const savingRemoteKeyId = ref<string | null>(null)
const refreshingBalance = ref(false)
const resettingCost = ref(false)
const showEditDialog = ref(false)
const showDeleteDialog = ref(false)
const showResetCostDialog = ref(false)
const editingUpstream = ref<Upstream | null>(null)
const deletingUpstream = ref<Upstream | null>(null)
const searchQuery = ref('')
const localGroups = ref<AdminGroup[]>([])
const remoteKeyDrafts = reactive<Record<string, RemoteAPIKeyDraft>>({})
const supportedModelsText = ref('')
const upstreamLocalGroupPlatforms: GroupPlatform[] = ['openai', 'anthropic']
const upstreamModelPlatforms: GroupPlatform[] = ['openai', 'anthropic']

const filters = reactive({
  type: '',
  status: ''
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const previewForm = reactive({
  model: '',
  local_group_id: null as number | null,
  mode: 'balanced' as UpstreamRoutingMode,
  random_seed: 0
})

const policyForm = reactive({
  consecutive_failures_to_circuit_open: 5,
  first_token_degrade_threshold_ms: 8000,
  error_rate_degrade_threshold: 0.25,
  recovery_probe_interval_seconds: 60,
  recovery_successes_required: 3,
  ignored_status_codes: '',
  immediate_circuit_status_codes: '',
  probe_failure_weight: 1,
  runtime_failure_weight: 2,
  alert_enabled: true
})

const costForm = reactive({
  dimension: 'upstream',
  hours: 24
})

const form = reactive<UpstreamForm>({
  name: '',
  type: 'sub2api',
  base_url: '',
  openai_api_base_url: '',
  status: 'active',
  priority: 100,
  weight: 100,
  cost_multiplier: 1,
  timeout_ms: 60000,
  connect_timeout_ms: 10000,
  retry_max: 0,
  probe_enabled: true,
  probe_model: '',
  probe_interval_seconds: 60,
  auto_sync_enabled: false,
  auto_sync_interval_minutes: 30,
  routing_mode: 'balanced',
  notes: '',
  admin_auth_mode: 'password',
  admin_login_url: '',
  admin_username: '',
  admin_username_placeholder: '',
  admin_password: '',
  admin_password_placeholder: '',
  admin_access_token: '',
  admin_access_token_placeholder: '',
  admin_refresh_token: '',
  admin_refresh_token_placeholder: '',
  metadata: {}
})

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null
let listController: AbortController | null = null

const columns = computed<Column[]>(() => [
  { key: 'name', label: tr('admin.upstreams.name', 'Name'), sortable: true },
  { key: 'type', label: tr('admin.upstreams.type', 'Type'), sortable: true },
  { key: 'status', label: tr('admin.upstreams.status', 'Status'), sortable: true },
  { key: 'routing_mode', label: tr('admin.upstreams.routingMode', 'Mode'), sortable: true },
  { key: 'weight', label: tr('admin.upstreams.weightPriority', 'Weight / Priority'), sortable: true },
  { key: 'cost_multiplier', label: tr('admin.upstreams.costMultiplier', '调度成本'), sortable: true },
  { key: 'groups_count', label: tr('admin.upstreams.groupsKeys', 'Groups / Keys'), sortable: true },
  { key: 'last_synced_at', label: tr('admin.upstreams.lastSync', 'Last sync'), sortable: true },
  { key: 'actions', label: tr('common.actions', 'Actions') }
])

const typeOptions = computed<SelectOption[]>(() => [
  { value: 'sub2api', label: 'sub2api' },
  { value: 'newapi', label: 'newapi' },
  { value: 'openai_compatible', label: 'OpenAI compatible' },
  { value: 'custom', label: tr('admin.upstreams.typeCustom', 'Custom') }
])

const typeFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: tr('admin.upstreams.allTypes', 'All types') },
  ...typeOptions.value
])

const statusOptions = computed<SelectOption[]>(() => [
  { value: 'active', label: tr('admin.upstreams.statusActive', 'Active') },
  { value: 'degraded', label: tr('admin.upstreams.statusDegraded', 'Degraded') },
  { value: 'half_open', label: tr('admin.upstreams.statusHalfOpen', 'Half open') },
  { value: 'circuit_open', label: tr('admin.upstreams.statusCircuitOpen', 'Circuit open') },
  { value: 'disabled', label: tr('admin.upstreams.statusDisabled', 'Disabled') }
])

const statusFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: tr('admin.upstreams.allStatuses', 'All statuses') },
  ...statusOptions.value
])

const routingModeOptions = computed<SelectOption[]>(() => [
  { value: 'balanced', label: tr('admin.upstreams.modeBalanced', 'Balanced') },
  { value: 'stability', label: tr('admin.upstreams.modeStability', 'Stability') },
  { value: 'cost', label: tr('admin.upstreams.modeCost', 'Cost') },
  { value: 'speed', label: tr('admin.upstreams.modeSpeed', 'Speed') },
  { value: 'manual', label: tr('admin.upstreams.modeManual', 'Manual') }
])

const adminAuthModeOptions = computed<SelectOption[]>(() => [
  { value: 'password', label: tr('admin.upstreams.authPassword', 'Username/password') },
  { value: 'token', label: tr('admin.upstreams.authToken', 'Token') },
  { value: 'none', label: tr('admin.upstreams.authNone', 'None') }
])

const previewLocalGroupOptions = computed<SelectOption[]>(() => [
  { value: null, label: tr('admin.upstreams.anyLocalGroup', 'Any local group') },
  ...localGroups.value.map((group) => ({
    value: group.id,
    label: localGroupLabel(group)
  }))
])

const costDimensionOptions = computed<SelectOption[]>(() => [
  { value: 'upstream', label: tr('admin.upstreams.costByUpstream', 'Upstream') },
  { value: 'remote_group', label: tr('admin.upstreams.costByRemoteGroup', 'Remote group') },
  { value: 'api_key', label: tr('admin.upstreams.costByAPIKey', 'API key') },
  { value: 'local_group', label: tr('admin.upstreams.costByLocalGroup', 'Local group') },
  { value: 'user', label: tr('admin.upstreams.costByUser', 'User') },
  { value: 'model', label: tr('admin.upstreams.costByModel', 'Model') }
])

const deleteMessage = computed(() => {
  const name = deletingUpstream.value?.name || tr('admin.upstreams.thisUpstream', 'this upstream')
  return `${tr('admin.upstreams.deleteConfirmPrefix', 'Delete')} ${name}?`
})

const resetCostMessage = computed(() => {
  const name = selectedUpstream.value?.name || tr('admin.upstreams.thisUpstream', 'this upstream')
  return `${tr('admin.upstreams.resetCostConfirmPrefix', '重置')} ${name} ${tr('admin.upstreams.resetCostConfirmSuffix', '的成本统计？历史调用记录会保留，成本对账将从当前时间重新累计。')}`
})

const supportedModels = computed(() => parseStringList(supportedModelsText.value))

const supportedModelsList = computed<string[]>({
  get: () => parseStringList(supportedModelsText.value),
  set: (models) => {
    supportedModelsText.value = normalizeStringList(models).join('\n')
  }
})

const selectedScheduleKeyLabel = computed(() => {
  if (!scheduleDecision.value) return '-'
  const selectedKeyID = scheduleDecision.value.selected_remote_api_key_id
  if (!selectedKeyID) return scheduleDecision.value.selected_name || '-'
  const key = remoteAPIKeys.value.find((item) => item.remote_api_key_id === selectedKeyID)
  const keyName = key?.remote_api_key_name || selectedKeyID
  const upstreamName = scheduleDecision.value.selected_name || key?.remote_api_key_id || '-'
  return `${upstreamName} / ${keyName}`
})

const upstreamAccountBalance = computed(() => metadataNumber(selectedUpstream.value?.metadata?.account_balance))
const upstreamAccountUsedQuota = computed(() => metadataNumber(selectedUpstream.value?.metadata?.account_used_quota))
const upstreamAccountBalanceCheckedAt = computed(() => metadataString(selectedUpstream.value?.metadata?.account_balance_checked_at))

async function loadLocalGroups() {
  if (localGroups.value.length > 0 || localGroupsLoading.value) return
  localGroupsLoading.value = true
  try {
    const groups = await adminAPI.groups.getAll()
    localGroups.value = groups.filter((group) => upstreamLocalGroupPlatforms.includes(group.platform))
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.localGroupsLoadFailed', 'Failed to load local OpenAI / Anthropic groups')))
  } finally {
    localGroupsLoading.value = false
  }
}

async function loadUpstreams() {
  listController?.abort()
  const controller = new AbortController()
  listController = controller
  loading.value = true
  try {
    const response = await adminAPI.upstreams.list(
      pagination.page,
      pagination.page_size,
      {
        type: filters.type || undefined,
        status: filters.status || undefined,
        search: searchQuery.value.trim() || undefined
      },
      { signal: controller.signal }
    )
    if (controller.signal.aborted) return
    upstreams.value = response.items
    pagination.total = response.total
    pagination.pages = response.pages
    pagination.page = response.page
    pagination.page_size = response.page_size

    if (selectedUpstream.value) {
      const refreshed = response.items.find((item) => item.id === selectedUpstream.value?.id)
      if (refreshed) {
        selectedUpstream.value = { ...selectedUpstream.value, ...refreshed }
      } else {
        clearSelection()
      }
    }
  } catch (error: any) {
    if (controller.signal.aborted || error?.code === 'ERR_CANCELED') return
    appStore.showError(errorMessage(error, tr('admin.upstreams.loadFailed', 'Failed to load upstreams')))
  } finally {
    if (listController === controller) {
      loading.value = false
      listController = null
    }
  }
}

function handleSearch() {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    pagination.page = 1
    loadUpstreams()
  }, 300)
}

function handleFilterChange() {
  pagination.page = 1
  loadUpstreams()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadUpstreams()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadUpstreams()
}

async function selectUpstream(row: Upstream) {
  const isNewSelection = selectedUpstream.value?.id !== row.id
  selectedUpstream.value = row
  showDetailDialog.value = true
  if (isNewSelection) {
    resetDetailState()
  }
  await loadDetail(row.id)
}

async function loadDetail(id: number) {
  detailLoading.value = true
  scheduleDecision.value = null
  syncPreviewResult.value = null
  try {
    const [upstream, keys, health, events, policy, alerts, cost] = await Promise.all([
      adminAPI.upstreams.get(id),
      adminAPI.upstreams.listRemoteAPIKeys(id),
      adminAPI.upstreams.health(id),
      adminAPI.upstreams.events(id, { limit: 30 }),
      adminAPI.upstreams.getPolicy(id),
      adminAPI.upstreams.listAlerts(id, true),
      loadCostReport(id)
    ])
    selectedUpstream.value = upstream
    remoteAPIKeys.value = keys
    healthDashboard.value = health
    upstreamEvents.value = events
    governancePolicy.value = policy
    upstreamAlerts.value = alerts
    costReport.value = cost
    syncRemoteKeyDrafts(keys)
    supportedModelsText.value = parseMetadataStringList(upstream.metadata?.supported_models).join('\n')
    fillPolicyForm(policy)
    previewForm.mode = upstream.routing_mode || 'balanced'
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.detailFailed', 'Failed to load upstream details')))
  } finally {
    detailLoading.value = false
  }
}

function clearSelection() {
  selectedUpstream.value = null
  showDetailDialog.value = false
  resetDetailState()
}

function closeDetailDialog() {
  showDetailDialog.value = false
}

function resetDetailState() {
  remoteAPIKeys.value = []
  healthDashboard.value = null
  upstreamEvents.value = []
  upstreamAlerts.value = []
  governancePolicy.value = null
  syncPreviewResult.value = null
  costReport.value = null
  scheduleDecision.value = null
  clearRemoteKeyDrafts()
  supportedModelsText.value = ''
}

function openCreateDialog() {
  editingUpstream.value = null
  resetForm()
  showEditDialog.value = true
}

async function openEditDialog(row: Upstream) {
  editingUpstream.value = row
  resetForm()
  try {
    const detail = await adminAPI.upstreams.get(row.id)
    fillForm(detail)
  } catch {
    fillForm(row)
  }
  showEditDialog.value = true
}

function closeEditDialog() {
  showEditDialog.value = false
  editingUpstream.value = null
}

function resetForm() {
  form.name = ''
  form.type = 'sub2api'
  form.base_url = ''
  form.openai_api_base_url = ''
  form.status = 'active'
  form.priority = 100
  form.weight = 100
  form.cost_multiplier = 1
  form.timeout_ms = 60000
  form.connect_timeout_ms = 10000
  form.retry_max = 0
  form.probe_enabled = true
  form.probe_model = ''
  form.probe_interval_seconds = 60
  form.auto_sync_enabled = false
  form.auto_sync_interval_minutes = 30
  form.routing_mode = 'balanced'
  form.notes = ''
  form.admin_auth_mode = 'password'
  form.admin_login_url = ''
  form.admin_username = ''
  form.admin_username_placeholder = ''
  form.admin_password = ''
  form.admin_password_placeholder = ''
  form.admin_access_token = ''
  form.admin_access_token_placeholder = ''
  form.admin_refresh_token = ''
  form.admin_refresh_token_placeholder = ''
  form.metadata = {}
}

function fillForm(row: Upstream) {
  form.name = row.name
  form.type = row.type
  form.base_url = row.base_url
  form.openai_api_base_url = metadataString(row.metadata?.openai_api_base_url)
  form.status = row.status
  form.priority = row.priority
  form.weight = row.weight
  form.cost_multiplier = row.cost_multiplier
  form.timeout_ms = row.timeout_ms
  form.connect_timeout_ms = row.connect_timeout_ms
  form.retry_max = row.retry_max
  form.probe_enabled = row.probe_enabled
  form.probe_model = row.probe_model || ''
  form.probe_interval_seconds = row.probe_interval_seconds
  form.auto_sync_enabled = metadataBool(row.metadata?.auto_sync_enabled)
  form.auto_sync_interval_minutes = metadataIntervalMinutes(row.metadata)
  form.routing_mode = row.routing_mode
  form.notes = row.notes || ''
  form.admin_auth_mode = row.admin_auth?.auth_mode || 'password'
  form.admin_login_url = row.admin_auth?.login_url || ''
  form.admin_username = ''
  form.admin_username_placeholder = row.admin_auth?.username_masked || ''
  form.admin_password = ''
  form.admin_password_placeholder = row.admin_auth?.password_configured ? tr('admin.upstreams.configuredPassword', 'Configured') : ''
  form.admin_access_token = ''
  form.admin_access_token_placeholder = row.admin_auth?.access_token_masked || ''
  form.admin_refresh_token = ''
  form.admin_refresh_token_placeholder = row.admin_auth?.refresh_token_masked || ''
  form.metadata = { ...(row.metadata || {}) }
}

function fillPolicyForm(policy: UpstreamGovernancePolicy | null) {
  if (!policy) return
  policyForm.consecutive_failures_to_circuit_open = policy.consecutive_failures_to_circuit_open
  policyForm.first_token_degrade_threshold_ms = policy.first_token_degrade_threshold_ms
  policyForm.error_rate_degrade_threshold = policy.error_rate_degrade_threshold
  policyForm.recovery_probe_interval_seconds = policy.recovery_probe_interval_seconds
  policyForm.recovery_successes_required = policy.recovery_successes_required
  policyForm.ignored_status_codes = (policy.ignored_status_codes || []).join(',')
  policyForm.immediate_circuit_status_codes = (policy.immediate_circuit_status_codes || []).join(',')
  policyForm.probe_failure_weight = policy.probe_failure_weight
  policyForm.runtime_failure_weight = policy.runtime_failure_weight
  policyForm.alert_enabled = policy.alert_enabled
}

function policyPayload(): UpstreamGovernancePolicy {
  return {
    consecutive_failures_to_circuit_open: Number(policyForm.consecutive_failures_to_circuit_open) || 5,
    first_token_degrade_threshold_ms: Number(policyForm.first_token_degrade_threshold_ms) || 8000,
    error_rate_degrade_threshold: Number(policyForm.error_rate_degrade_threshold) || 0.25,
    recovery_probe_interval_seconds: Number(policyForm.recovery_probe_interval_seconds) || 60,
    recovery_successes_required: Number(policyForm.recovery_successes_required) || 3,
    ignored_status_codes: parseStatusCodes(policyForm.ignored_status_codes),
    immediate_circuit_status_codes: parseStatusCodes(policyForm.immediate_circuit_status_codes),
    probe_failure_weight: Number(policyForm.probe_failure_weight) || 1,
    runtime_failure_weight: Number(policyForm.runtime_failure_weight) || 2,
    alert_enabled: policyForm.alert_enabled
  }
}

function parseStatusCodes(raw: string) {
  return raw
    .split(',')
    .map((item) => Number(item.trim()))
    .filter((item, index, arr) => Number.isSafeInteger(item) && item > 0 && arr.indexOf(item) === index)
}

function buildPayload(): UpstreamPayload {
  const adminAuth: UpstreamPayload['admin_auth'] = {
    auth_mode: form.admin_auth_mode,
    login_url: form.admin_login_url.trim()
  }
  if (form.admin_username.trim()) adminAuth.username = form.admin_username.trim()
  if (form.admin_password.trim()) adminAuth.password = form.admin_password.trim()
  if (form.admin_access_token.trim()) adminAuth.access_token = form.admin_access_token.trim()
  if (form.admin_refresh_token.trim()) adminAuth.refresh_token = form.admin_refresh_token.trim()

  const metadata = buildPayloadMetadata()

  return {
    name: form.name.trim(),
    type: form.type,
    base_url: form.base_url.trim(),
    status: form.status,
    priority: Number(form.priority) || 100,
    weight: Number(form.weight) || 0,
    cost_multiplier: Number(form.cost_multiplier) || 1,
    timeout_ms: Number(form.timeout_ms) || 60000,
    connect_timeout_ms: Number(form.connect_timeout_ms) || 10000,
    retry_max: Number(form.retry_max) || 0,
    probe_enabled: form.probe_enabled,
    probe_model: form.probe_model.trim(),
    probe_interval_seconds: Number(form.probe_interval_seconds) || 60,
    routing_mode: form.routing_mode,
    notes: form.notes.trim(),
    metadata,
    admin_auth: adminAuth
  }
}

function buildPayloadMetadata(): Record<string, unknown> {
  const metadata: Record<string, unknown> = { ...form.metadata }
  delete metadata.local_group_ids
  delete metadata.local_group_remote_group_ids
  const apiBaseURL = form.openai_api_base_url.trim()
  if (apiBaseURL) {
    metadata.openai_api_base_url = apiBaseURL
  } else {
    delete metadata.openai_api_base_url
  }
  const models = parseMetadataStringList(metadata.supported_models)
  if (models.length > 0) {
    metadata.supported_models = models
  } else {
    delete metadata.supported_models
  }
  if (form.auto_sync_enabled) {
    metadata.auto_sync_enabled = true
    metadata.auto_sync_interval_seconds = Math.max(60, Math.min(86400, Math.round((Number(form.auto_sync_interval_minutes) || 30) * 60)))
  } else {
    delete metadata.auto_sync_enabled
    delete metadata.auto_sync_interval_seconds
  }
  delete metadata.auto_sync_interval_minutes
  return metadata
}

function metadataString(raw: unknown): string {
  return typeof raw === 'string' ? raw.trim() : ''
}

function metadataBool(raw: unknown): boolean {
  if (typeof raw === 'boolean') return raw
  if (typeof raw === 'number') return raw !== 0
  if (typeof raw === 'string') {
    return ['1', 'true', 'yes', 'on', 'enabled'].includes(raw.trim().toLowerCase())
  }
  return false
}

function metadataIntervalMinutes(metadata?: Record<string, unknown>): number {
  const seconds = metadataNumber(metadata?.auto_sync_interval_seconds)
  if (seconds && seconds > 0) {
    return Math.max(1, Math.round(seconds / 60))
  }
  const minutes = metadataNumber(metadata?.auto_sync_interval_minutes)
  if (minutes && minutes > 0) {
    return Math.max(1, Math.round(minutes))
  }
  return 30
}

function upstreamVisitURL(row?: Upstream | null): string {
  const raw = String(row?.base_url || '').trim()
  if (!raw) return ''
  try {
    const url = new URL(raw)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return ''
    return url.toString()
  } catch {
    return ''
  }
}

function visitUpstream(row?: Upstream | null) {
  const url = upstreamVisitURL(row)
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}

async function saveUpstream() {
  saving.value = true
  try {
    const payload = buildPayload()
    const saved = editingUpstream.value
      ? await adminAPI.upstreams.update(editingUpstream.value.id, payload)
      : await adminAPI.upstreams.create(payload)
    appStore.showSuccess(tr('admin.upstreams.saved', 'Upstream saved'))
    closeEditDialog()
    await loadUpstreams()
    await loadDetail(saved.id)
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.saveFailed', 'Failed to save upstream')))
  } finally {
    saving.value = false
  }
}

async function syncUpstream(row: Upstream) {
  setSyncing(row.id, true)
  try {
    const result = await adminAPI.upstreams.sync(row.id)
    appStore.showSuccess(
      `${tr('admin.upstreams.syncCompleted', 'Sync completed')}: ${result.groups.length} ${tr('admin.upstreams.groups', 'groups')}, ${result.api_keys.length} ${tr('admin.upstreams.apiKeys', 'API keys')}`
    )
    await loadUpstreams()
    await loadDetail(row.id)
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.syncFailed', 'Sync failed')))
    if (selectedUpstream.value?.id === row.id) {
      await loadDetail(row.id)
    }
  } finally {
    setSyncing(row.id, false)
  }
}

async function previewSync(row: Upstream) {
  syncPreviewLoading.value = true
  try {
    syncPreviewResult.value = await adminAPI.upstreams.syncPreview(row.id)
    appStore.showSuccess(tr('admin.upstreams.syncPreviewReady', 'Sync preview ready'))
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.syncPreviewFailed', 'Failed to preview sync')))
  } finally {
    syncPreviewLoading.value = false
  }
}

async function applySyncPreview() {
  if (!selectedUpstream.value || !syncPreviewResult.value) return
  applyingPreview.value = true
  try {
    const result = await adminAPI.upstreams.applySyncPreview(selectedUpstream.value.id, syncPreviewResult.value.preview_token)
    appStore.showSuccess(
      `${tr('admin.upstreams.syncApplied', 'Sync applied')}: ${result.groups.length} ${tr('admin.upstreams.groups', 'groups')}, ${result.api_keys.length} ${tr('admin.upstreams.apiKeys', 'API keys')}`
    )
    syncPreviewResult.value = null
    await loadUpstreams()
    await loadDetail(selectedUpstream.value.id)
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.syncApplyFailed', 'Failed to apply sync preview')))
  } finally {
    applyingPreview.value = false
  }
}

function remoteKeyDraft(remoteAPIKeyID: string): RemoteAPIKeyDraft {
  const id = String(remoteAPIKeyID)
  if (!remoteKeyDrafts[id]) {
    const key = remoteAPIKeys.value.find((item) => item.remote_api_key_id === id)
    remoteKeyDrafts[id] = {
      api_key: '',
      local_group_ids: [...(key?.local_group_ids || [])]
    }
  }
  return remoteKeyDrafts[id]
}

function syncRemoteKeyDrafts(keys: UpstreamRemoteAPIKey[]) {
  const keep = new Set(keys.map((item) => item.remote_api_key_id))
  Object.keys(remoteKeyDrafts).forEach((id) => {
    if (!keep.has(id)) {
      delete remoteKeyDrafts[id]
    }
  })
  keys.forEach((key) => {
    remoteKeyDrafts[key.remote_api_key_id] = {
      api_key: '',
      local_group_ids: [...(key.local_group_ids || [])]
    }
  })
}

function clearRemoteKeyDrafts() {
  Object.keys(remoteKeyDrafts).forEach((id) => {
    delete remoteKeyDrafts[id]
  })
}

function toggleRemoteKeyLocalGroup(remoteAPIKeyID: string, groupID: number) {
  const draft = remoteKeyDraft(remoteAPIKeyID)
  const index = draft.local_group_ids.indexOf(groupID)
  if (index >= 0) {
    draft.local_group_ids.splice(index, 1)
  } else {
    draft.local_group_ids.push(groupID)
  }
}

async function saveRemoteAPIKeyConfig(key: UpstreamRemoteAPIKey) {
  if (!selectedUpstream.value) return
  const draft = remoteKeyDraft(key.remote_api_key_id)
  savingRemoteKeyId.value = key.remote_api_key_id
  try {
    const saved = await adminAPI.upstreams.updateRemoteAPIKeyConfig(
      selectedUpstream.value.id,
      key.remote_api_key_id,
      {
        local_group_ids: [...draft.local_group_ids],
        api_key: draft.api_key.trim() || undefined
      }
    )
    const index = remoteAPIKeys.value.findIndex((item) => item.remote_api_key_id === saved.remote_api_key_id)
    if (index >= 0) {
      remoteAPIKeys.value[index] = saved
    }
    remoteKeyDrafts[saved.remote_api_key_id] = {
      api_key: '',
      local_group_ids: [...(saved.local_group_ids || [])]
    }
    appStore.showSuccess(tr('admin.upstreams.keyConfigSaved', 'API key 配置已保存'))
    await refreshEventsAndHealth()
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.keyConfigSaveFailed', '保存 API key 配置失败')))
  } finally {
    savingRemoteKeyId.value = null
  }
}

async function refreshUpstreamBalance() {
  if (!selectedUpstream.value || refreshingBalance.value) return
  refreshingBalance.value = true
  try {
    const result = await adminAPI.upstreams.refreshBalance(selectedUpstream.value.id)
    applyUpstreamAccountBalance(result)
    if (result.has_balance) {
      appStore.showSuccess(tr('admin.upstreams.balanceRefreshed', '余额已刷新'))
    } else {
      appStore.showInfo(result.message || tr('admin.upstreams.balanceUnavailable', '上游账户未返回余额字段'))
    }
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.balanceRefreshFailed', '查询余额失败')))
  } finally {
    refreshingBalance.value = false
  }
}

function applyUpstreamAccountBalance(result: UpstreamAccountBalanceResult) {
  if (!selectedUpstream.value) return
  const metadata = { ...(selectedUpstream.value.metadata || {}) }
  setOptionalMetadataNumber(metadata, 'account_balance', result.balance)
  setOptionalMetadataNumber(metadata, 'account_quota', result.quota)
  setOptionalMetadataNumber(metadata, 'account_used_quota', result.used_quota)
  setOptionalMetadataNumber(metadata, 'account_remaining_quota', result.remaining_quota)
  delete metadata.account_balance_currency
  if (result.source) {
    metadata.account_balance_source = result.source
  }
  if (result.checked_at) {
    metadata.account_balance_checked_at = result.checked_at
  }
  if (result.has_balance) {
    delete metadata.account_balance_error
  } else if (result.message) {
    metadata.account_balance_error = result.message
  }
  selectedUpstream.value = {
    ...selectedUpstream.value,
    metadata
  }
}

async function saveSupportedModels() {
  if (!selectedUpstream.value) return
  savingModels.value = true
  try {
    const metadata = { ...(selectedUpstream.value.metadata || {}) }
    const models = parseStringList(supportedModelsText.value)
    if (models.length > 0) {
      metadata.supported_models = models
    } else {
      delete metadata.supported_models
    }
    const saved = await adminAPI.upstreams.update(selectedUpstream.value.id, { metadata })
    selectedUpstream.value = saved
    supportedModelsText.value = parseMetadataStringList(saved.metadata?.supported_models).join('\n')
    appStore.showSuccess(tr('admin.upstreams.modelsSaved', '可用模型已保存'))
    await loadUpstreams()
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.modelsSaveFailed', '保存可用模型失败')))
  } finally {
    savingModels.value = false
  }
}

async function testLogin(row: Upstream) {
  testingId.value = row.id
  try {
    const result = await adminAPI.upstreams.testLogin(row.id)
    appStore.showSuccess(result.message || tr('admin.upstreams.loginSucceeded', 'Login succeeded'))
    if (selectedUpstream.value?.id === row.id) {
      await loadDetail(row.id)
    }
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.loginFailed', 'Login failed')))
  } finally {
    testingId.value = null
  }
}

async function probeUpstream(row: Upstream) {
  probeId.value = row.id
  try {
    const result = await adminAPI.upstreams.probe(row.id)
    const ttft = result.first_token_ms != null ? `${result.first_token_ms}ms` : '-'
    const remoteKey = result.remote_api_key_name || result.remote_api_key_id || tr('admin.upstreams.unknownAPIKey', 'unknown key')
    if (result.ignored) {
      appStore.showInfo(`${tr('admin.upstreams.probeSkipped', 'Probe skipped')}: ${result.error_message || result.reason}`)
    } else if (result.success) {
      appStore.showSuccess(`${tr('admin.upstreams.probeSucceeded', 'Probe succeeded')}: ${remoteKey}, TTFT ${ttft}`)
    } else {
      appStore.showWarning(`${tr('admin.upstreams.probeFailed', 'Probe failed')}: ${result.error_message || result.reason}`)
    }
    await loadUpstreams()
    if (selectedUpstream.value?.id === row.id) {
      await loadDetail(row.id)
    }
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.probeFailed', 'Probe failed')))
  } finally {
    probeId.value = null
  }
}

async function runSchedulePreview() {
  previewLoading.value = true
  try {
    previewForm.random_seed = Date.now()
    scheduleDecision.value = await adminAPI.upstreams.schedulePreview({
      model: previewForm.model.trim() || undefined,
      local_group_id: previewForm.local_group_id || undefined,
      mode: previewForm.mode,
      random_seed: previewForm.random_seed
    })
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.previewFailed', 'Failed to preview schedule')))
  } finally {
    previewLoading.value = false
  }
}

async function savePolicy() {
  if (!selectedUpstream.value) return
  policySaving.value = true
  try {
    const payload = policyPayload()
    const saved = await adminAPI.upstreams.updatePolicy(selectedUpstream.value.id, payload)
    governancePolicy.value = saved
    fillPolicyForm(saved)
    appStore.showSuccess(tr('admin.upstreams.policySaved', 'Policy saved'))
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.policySaveFailed', 'Failed to save policy')))
  } finally {
    policySaving.value = false
  }
}

async function refreshEventsAndHealth() {
  if (!selectedUpstream.value) return
  try {
    const [health, events, alerts] = await Promise.all([
      adminAPI.upstreams.health(selectedUpstream.value.id),
      adminAPI.upstreams.events(selectedUpstream.value.id, { limit: 30 }),
      adminAPI.upstreams.listAlerts(selectedUpstream.value.id, true)
    ])
    healthDashboard.value = health
    upstreamEvents.value = events
    upstreamAlerts.value = alerts
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.healthLoadFailed', 'Failed to load upstream health')))
  }
}

async function resolveUpstreamAlert(alert: UpstreamAlert) {
  if (!selectedUpstream.value) return
  try {
    await adminAPI.upstreams.resolveAlert(selectedUpstream.value.id, alert.alert_type)
    await refreshEventsAndHealth()
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.alertResolveFailed', 'Failed to resolve alert')))
  }
}

async function refreshCostReport() {
  if (!selectedUpstream.value) return
  costLoading.value = true
  try {
    costReport.value = await loadCostReport(selectedUpstream.value.id)
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.costLoadFailed', 'Failed to load cost report')))
  } finally {
    costLoading.value = false
  }
}

function requestResetCostReport() {
  if (!selectedUpstream.value) return
  showResetCostDialog.value = true
}

async function confirmResetCostReport() {
  if (!selectedUpstream.value || resettingCost.value) return
  resettingCost.value = true
  try {
    const result = await adminAPI.upstreams.resetCostReport(selectedUpstream.value.id)
    const metadata = { ...(selectedUpstream.value.metadata || {}) }
    metadata.cost_report_reset_at = result.reset_at
    selectedUpstream.value = {
      ...selectedUpstream.value,
      metadata
    }
    showResetCostDialog.value = false
    appStore.showSuccess(tr('admin.upstreams.costReportReset', '成本统计已重置'))
    await refreshCostReport()
    await loadUpstreams()
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.costResetFailed', '重置成本统计失败')))
  } finally {
    resettingCost.value = false
  }
}

async function loadCostReport(id: number) {
  const end = new Date()
  const start = new Date(end.getTime() - (Number(costForm.hours) || 24) * 60 * 60 * 1000)
  return adminAPI.upstreams.costReport(id, {
    start: start.toISOString(),
    end: end.toISOString(),
    dimension: costForm.dimension
  })
}

function requestDelete(row: Upstream) {
  deletingUpstream.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingUpstream.value) return
  try {
    await adminAPI.upstreams.remove(deletingUpstream.value.id)
    appStore.showSuccess(tr('admin.upstreams.deleted', 'Upstream deleted'))
    showDeleteDialog.value = false
    if (selectedUpstream.value?.id === deletingUpstream.value.id) {
      clearSelection()
    }
    deletingUpstream.value = null
    await loadUpstreams()
  } catch (error: any) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.deleteFailed', 'Failed to delete upstream')))
  }
}

function remoteAPIKeyConfigured(key: UpstreamRemoteAPIKey) {
  return !!key.api_key_configured || !!key.masked_key
}

function remoteAPIKeySchedulable(key: UpstreamRemoteAPIKey) {
  return remoteAPIKeyConfigured(key) &&
    remoteAPIKeyStatusActive(key.status) &&
    (key.local_group_ids || []).length > 0
}

function remoteAPIKeyStatusActive(status?: string) {
  const normalized = String(status || '').trim().toLowerCase()
  return ['', 'active', 'enabled', 'enable', '1', 'true'].includes(normalized)
}

function localGroupLabel(group: AdminGroup) {
  const platform = group.platform === 'anthropic' ? 'Anthropic' : group.platform === 'openai' ? 'OpenAI' : group.platform
  return `${group.name} · ${platform}`
}

function localGroupNames(ids?: number[]) {
  const uniqueIDs = Array.from(new Set((ids || []).filter((id) => Number.isFinite(Number(id)) && Number(id) > 0)))
  if (uniqueIDs.length === 0) return tr('admin.upstreams.noMappedLocalGroups', '未映射本地分组')
  return uniqueIDs
    .map((id) => {
      const group = localGroups.value.find((item) => item.id === id)
      return group ? localGroupLabel(group) : String(id)
    })
    .join(', ')
}

function setSyncing(id: number, syncing: boolean) {
  const next = new Set(syncingIds.value)
  if (syncing) {
    next.add(id)
  } else {
    next.delete(id)
  }
  syncingIds.value = next
}

function isSyncing(id: number) {
  return syncingIds.value.has(id)
}

function typeLabel(value: string) {
  return typeOptions.value.find((option) => option.value === value)?.label || value
}

function statusLabel(value: string) {
  return statusOptions.value.find((option) => option.value === value)?.label || value
}

function routingModeLabel(value: string) {
  return routingModeOptions.value.find((option) => option.value === value)?.label || value
}

function statusBadgeClass(value: string) {
  switch (value) {
    case 'active':
      return 'badge-success'
    case 'degraded':
    case 'half_open':
      return 'badge-warning'
    case 'circuit_open':
      return 'badge-danger'
    case 'disabled':
      return 'badge-gray'
    default:
      return 'badge-gray'
  }
}

function syncStatusClass(value: string) {
  return value === 'success'
    ? 'text-emerald-600 dark:text-emerald-400'
    : value === 'failed'
      ? 'text-red-600 dark:text-red-400'
      : 'text-gray-500 dark:text-dark-400'
}

function syncStatusLabel(value: string) {
  if (value === 'success') return tr('admin.upstreams.syncSuccess', 'Sync success')
  if (value === 'failed') return tr('admin.upstreams.syncFailedShort', 'Sync failed')
  return value
}

function eventTypeLabel(value: string) {
  const labels: Record<string, string> = {
    probe_success: tr('admin.upstreams.eventProbeSuccess', 'Probe success'),
    probe_failure: tr('admin.upstreams.eventProbeFailure', 'Probe failed'),
    runtime_success: tr('admin.upstreams.eventRuntimeSuccess', 'Request success'),
    runtime_error: tr('admin.upstreams.eventRuntimeError', 'Request error'),
    stream_interrupted: tr('admin.upstreams.eventStreamInterrupted', 'Stream interrupted'),
    runtime_ignored: tr('admin.upstreams.eventIgnored', 'Ignored')
  }
  return labels[value] || value
}

function alertBadgeClass(severity: string) {
  switch (severity) {
    case 'critical':
      return 'badge-danger'
    case 'warning':
      return 'badge-warning'
    default:
      return 'badge-gray'
  }
}

function alertSeverityLabel(severity: string) {
  switch (severity) {
    case 'critical':
      return tr('admin.upstreams.alertCritical', '严重')
    case 'warning':
      return tr('admin.upstreams.alertWarning', '警告')
    case 'info':
      return tr('admin.upstreams.alertInfo', '提示')
    default:
      return severity
  }
}

function formatPercent(value: unknown, digits = 1) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  return `${(n * 100).toFixed(digits).replace(/\.?0+$/, '')}%`
}

function formatMS(value?: number | null) {
  if (value == null || !Number.isFinite(Number(value))) return '-'
  return `${formatNumber(value, 0)}ms`
}

function formatCurrency(value: unknown) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '$0'
  return `$${n.toFixed(6).replace(/\.?0+$/, '')}`
}

function syncDiffTotal(diff?: UpstreamSyncPreview['diff'] | null) {
  if (!diff) return 0
  return diff.added_groups.length + diff.removed_groups.length + diff.changed_groups.length +
    diff.added_api_keys.length + diff.removed_api_keys.length + diff.changed_api_keys.length
}

function costDimensionLabel(item: any) {
  return item.remote_api_key_id || item.remote_group_id || item.model || item.local_group_id || item.user_id || item.upstream_name || '-'
}

function parseStringList(raw: string) {
  return normalizeStringList(raw.split(/[\n,\t]+/))
}

function parseMetadataStringList(raw: unknown): string[] {
  if (Array.isArray(raw)) {
    return normalizeStringList(raw)
  }
  if (typeof raw === 'string') {
    return parseStringList(raw)
  }
  return []
}

function normalizeStringList(items: unknown[]) {
  return Array.from(
    new Set(
      items
        .map((item) => String(item).trim())
        .filter(Boolean)
    )
  )
}

function formatOptionalDate(value?: string | null) {
  if (!value) return '-'
  return formatDateTime(value)
}

function formatNumber(value: unknown, digits = 2) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '0'
  return n.toFixed(digits).replace(/\.?0+$/, '')
}

function formatQuotaValue(value?: number | null) {
  if (value == null || !Number.isFinite(Number(value))) return '-'
  return formatNumber(value, 6)
}

function metadataNumber(value: unknown): number | null {
  if (value == null) return null
  const n = Number(value)
  return Number.isFinite(n) ? n : null
}

function setOptionalMetadataNumber(metadata: Record<string, unknown>, key: string, value?: number | null) {
  if (value == null || !Number.isFinite(Number(value))) {
    delete metadata[key]
    return
  }
  metadata[key] = Number(value)
}

function errorMessage(error: any, fallback: string) {
  return error?.response?.data?.detail || error?.message || error?.error || fallback
}

onMounted(() => {
  loadLocalGroups()
  loadUpstreams()
})

onUnmounted(() => {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  listController?.abort()
})
</script>
