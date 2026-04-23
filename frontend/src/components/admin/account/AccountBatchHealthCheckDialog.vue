<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.batchHealthCheckTitle')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800">
        <div class="grid gap-4 lg:grid-cols-2">
          <div class="space-y-1.5">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.batchHealthCheckScope') }}
            </label>
            <Select
              v-model="scope"
              :options="scopeOptions"
              :disabled="running"
            />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.batchHealthCheckScopeHint', { count: scopeAccountIDs.length }) }}
            </p>
          </div>

          <div class="space-y-1.5">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.batchHealthCheckModel') }}
            </label>
            <Select
              v-model="selectedModelId"
              :options="modelOptions"
              :disabled="running || loadingModels"
              searchable
              creatable
              :creatable-prefix="t('admin.accounts.batchHealthCheckCustomModel')"
              :placeholder="loadingModels ? t('common.loading') : t('admin.accounts.batchHealthCheckUseDefaultModel')"
            />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.batchHealthCheckModelHint') }}
            </p>
          </div>
        </div>

        <div
          v-if="hasMixedPlatforms"
          class="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-200"
        >
          {{ t('admin.accounts.batchHealthCheckMixedPlatformWarning') }}
        </div>
      </div>

      <div class="grid gap-3 md:grid-cols-4">
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
          <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.batchHealthCheckTargetCount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ lastRunSummary.total }}
          </div>
        </div>
        <div class="rounded-xl border border-green-200 bg-green-50 p-4 dark:border-green-900/40 dark:bg-green-900/10">
          <div class="text-xs uppercase tracking-wide text-green-700 dark:text-green-300">
            {{ t('admin.accounts.batchHealthCheckSuccessCount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-green-700 dark:text-green-300">
            {{ lastRunSummary.success_count }}
          </div>
        </div>
        <div class="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-900/40 dark:bg-red-900/10">
          <div class="text-xs uppercase tracking-wide text-red-700 dark:text-red-300">
            {{ t('admin.accounts.batchHealthCheckFailedCount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-red-700 dark:text-red-300">
            {{ lastRunSummary.failed_count }}
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
          <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.batchHealthCheckCleanupSelected') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ selectedFailedAccountIDs.length }}
          </div>
        </div>
      </div>

      <div class="rounded-xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <div>
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.accounts.batchHealthCheckResults') }}
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.batchHealthCheckResultsHint') }}
            </div>
          </div>
          <button
            v-if="failedRows.length > 0"
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="cleanupRunning"
            @click="toggleSelectAllFailed"
          >
            {{
              allFailedSelected
                ? t('admin.accounts.batchHealthCheckClearFailedSelection')
                : t('admin.accounts.batchHealthCheckSelectAllFailed')
            }}
          </button>
        </div>

        <div v-if="running" class="flex items-center gap-2 px-4 py-6 text-sm text-gray-500 dark:text-gray-400">
          <Icon name="refresh" size="sm" class="animate-spin" />
          <span>{{ t('admin.accounts.batchHealthCheckRunning') }}</span>
        </div>

        <div v-else-if="results.length === 0" class="px-4 py-8 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.batchHealthCheckEmpty') }}
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 text-left text-gray-500 dark:border-dark-700">
                <th class="w-12 px-4 py-3"></th>
                <th class="px-4 py-3">{{ t('admin.accounts.columns.name') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.columns.platformType') }}</th>
                <th class="px-4 py-3">{{ t('common.status') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.testModel') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.batchHealthCheckLatency') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.batchHealthCheckMessage') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in results"
                :key="item.account_id"
                class="border-b border-gray-50 align-top dark:border-dark-800"
              >
                <td class="px-4 py-3">
                  <input
                    v-if="!item.success && !item.deleted"
                    :checked="selectedFailedAccountIDs.includes(item.account_id)"
                    type="checkbox"
                    class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                    @change="toggleFailedSelection(item.account_id)"
                  />
                </td>
                <td class="px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ item.name }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">#{{ item.account_id }}</div>
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  {{ item.platform }} / {{ item.type }}
                </td>
                <td class="px-4 py-3">
                  <span
                    :class="[
                      'inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium',
                      item.deleted
                        ? 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
                        : item.success
                          ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                          : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
                    ]"
                  >
                    {{
                      item.deleted
                        ? t('admin.accounts.batchHealthCheckDeleted')
                        : item.success
                          ? t('common.success')
                          : t('common.error')
                    }}
                  </span>
                  <span
                    v-if="item.recovered && !item.deleted"
                    class="mt-1 inline-flex items-center rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                  >
                    {{ t('admin.accounts.batchHealthCheckRecovered') }}
                  </span>
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  {{ item.model_id || t('admin.accounts.batchHealthCheckUseDefaultModel') }}
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  {{ item.latency_ms != null ? `${item.latency_ms} ms` : '-' }}
                </td>
                <td class="px-4 py-3">
                  <div v-if="item.error_message" class="max-w-md whitespace-pre-wrap break-words text-red-600 dark:text-red-300">
                    {{ item.error_message }}
                  </div>
                  <div v-else class="max-w-md whitespace-pre-wrap break-words text-gray-600 dark:text-gray-300">
                    {{ item.response_text || '-' }}
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex flex-wrap justify-end gap-3">
        <button
          type="button"
          class="btn btn-danger"
          :disabled="cleanupRunning || selectedFailedAccountIDs.length === 0"
          @click="cleanupFailedAccounts"
        >
          {{
            cleanupRunning
              ? t('common.processing')
              : t('admin.accounts.batchHealthCheckCleanupFailed')
          }}
        </button>
        <button type="button" class="btn btn-secondary" :disabled="running" @click="emit('close')">
          {{ t('common.close') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="running || scopeAccountIDs.length === 0"
          @click="runBatchHealthCheck"
        >
          <Icon v-if="running" name="refresh" size="sm" class="mr-2 animate-spin" />
          <Icon v-else name="play" size="sm" class="mr-2" />
          {{
            running
              ? t('admin.accounts.testing')
              : t('admin.accounts.batchHealthCheckStart')
          }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { BatchHealthCheckItem, BatchHealthCheckResponse } from '@/api/admin/accounts'
import type { Account } from '@/types'
import { useAppStore } from '@/stores'

type HealthCheckScope = 'current_page' | 'selected'
type LocalBatchHealthCheckItem = BatchHealthCheckItem & { deleted?: boolean }

const props = defineProps<{
  show: boolean
  visibleAccounts: Account[]
  selectedAccountIDs: number[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'deleted', accountIds: number[]): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const scope = ref<HealthCheckScope>('current_page')
const selectedModelId = ref<string | number | boolean | null>('')
const modelOptions = ref<SelectOption[]>([])
const loadingModels = ref(false)
const running = ref(false)
const cleanupRunning = ref(false)
const results = ref<LocalBatchHealthCheckItem[]>([])
const selectedFailedAccountIDs = ref<number[]>([])
const lastRunSummary = ref<BatchHealthCheckResponse>({
  items: [],
  total: 0,
  success_count: 0,
  failed_count: 0
})

const hasSelection = computed(() => props.selectedAccountIDs.length > 0)
const scopeOptions = computed<SelectOption[]>(() => {
  const options: SelectOption[] = [
    {
      value: 'current_page',
      label: t('admin.accounts.batchHealthCheckCurrentPage', { count: props.visibleAccounts.length })
    }
  ]
  if (hasSelection.value) {
    options.unshift({
      value: 'selected',
      label: t('admin.accounts.batchHealthCheckSelectedScope', { count: props.selectedAccountIDs.length })
    })
  }
  return options
})

const scopeAccountIDs = computed(() => {
  if (scope.value === 'selected' && hasSelection.value) {
    return normalizeIDs(props.selectedAccountIDs)
  }
  return normalizeIDs(props.visibleAccounts.map((account) => account.id))
})

const scopedVisibleAccounts = computed(() => {
  if (scope.value === 'selected' && hasSelection.value) {
    const selected = new Set(scopeAccountIDs.value)
    return props.visibleAccounts.filter((account) => selected.has(account.id))
  }
  return props.visibleAccounts
})

const hasMixedPlatforms = computed(() => {
  const platforms = new Set(scopedVisibleAccounts.value.map((account) => account.platform))
  return platforms.size > 1
})

const failedRows = computed(() => results.value.filter((item) => !item.success && !item.deleted))
const allFailedSelected = computed(() => {
  if (failedRows.value.length === 0) return false
  return failedRows.value.every((item) => selectedFailedAccountIDs.value.includes(item.account_id))
})

watch(
  () => props.show,
  async (visible) => {
    if (!visible) return
    scope.value = hasSelection.value ? 'selected' : 'current_page'
    selectedModelId.value = ''
    results.value = []
    selectedFailedAccountIDs.value = []
    lastRunSummary.value = {
      items: [],
      total: scopeAccountIDs.value.length,
      success_count: 0,
      failed_count: 0
    }
    await loadModelOptions()
  },
  { immediate: true }
)

watch(
  () => [scope.value, props.selectedAccountIDs.join(','), props.visibleAccounts.map((account) => account.id).join(',')],
  async () => {
    if (!props.show) return
    selectedModelId.value = ''
    await loadModelOptions()
  }
)

async function loadModelOptions() {
  const sourceAccountID = scopeAccountIDs.value[0]
  modelOptions.value = [{ value: '', label: t('admin.accounts.batchHealthCheckUseDefaultModel') }]
  if (!sourceAccountID) return

  loadingModels.value = true
  try {
    const models = await adminAPI.accounts.getAvailableModels(sourceAccountID)
    modelOptions.value = [
      { value: '', label: t('admin.accounts.batchHealthCheckUseDefaultModel') },
      ...models.map((model) => ({
        value: model.id,
        label: model.display_name || model.id
      }))
    ]
  } catch (error) {
    console.error('Failed to load batch health check model options:', error)
  } finally {
    loadingModels.value = false
  }
}

async function runBatchHealthCheck() {
  if (scopeAccountIDs.value.length === 0) {
    appStore.showError(t('admin.accounts.batchHealthCheckEmptyScope'))
    return
  }

  running.value = true
  selectedFailedAccountIDs.value = []
  try {
    const response = await adminAPI.accounts.batchHealthCheck({
      account_ids: scopeAccountIDs.value,
      model_id: String(selectedModelId.value || '').trim() || undefined
    })
    lastRunSummary.value = response
    results.value = response.items
    if (response.failed_count > 0) {
      appStore.showError(
        t('admin.accounts.batchHealthCheckPartial', {
          success: response.success_count,
          failed: response.failed_count
        })
      )
    } else {
      appStore.showSuccess(
        t('admin.accounts.batchHealthCheckSuccess', { count: response.success_count })
      )
    }
  } catch (error) {
    console.error('Failed to run batch health check:', error)
    appStore.showError(t('admin.accounts.batchHealthCheckFailed'))
  } finally {
    running.value = false
  }
}

function toggleFailedSelection(accountID: number) {
  if (selectedFailedAccountIDs.value.includes(accountID)) {
    selectedFailedAccountIDs.value = selectedFailedAccountIDs.value.filter((id) => id !== accountID)
    return
  }
  selectedFailedAccountIDs.value = [...selectedFailedAccountIDs.value, accountID]
}

function toggleSelectAllFailed() {
  if (allFailedSelected.value) {
    selectedFailedAccountIDs.value = []
    return
  }
  selectedFailedAccountIDs.value = failedRows.value.map((item) => item.account_id)
}

async function cleanupFailedAccounts() {
  const targetIDs = normalizeIDs(selectedFailedAccountIDs.value)
  if (targetIDs.length === 0) return
  if (!window.confirm(t('admin.accounts.batchHealthCheckCleanupConfirm', { count: targetIDs.length }))) {
    return
  }

  cleanupRunning.value = true
  try {
    const settled = await Promise.allSettled(targetIDs.map((accountID) => adminAPI.accounts.delete(accountID)))
    const deletedIDs: number[] = []
    let failedCount = 0

    settled.forEach((result, index) => {
      if (result.status === 'fulfilled') {
        deletedIDs.push(targetIDs[index])
      } else {
        failedCount++
      }
    })

    if (deletedIDs.length > 0) {
      const deletedSet = new Set(deletedIDs)
      results.value = results.value.map((item) =>
        deletedSet.has(item.account_id) ? { ...item, deleted: true } : item
      )
      selectedFailedAccountIDs.value = selectedFailedAccountIDs.value.filter((id) => !deletedSet.has(id))
      emit('deleted', deletedIDs)
    }

    if (failedCount > 0) {
      appStore.showError(
        t('admin.accounts.batchHealthCheckCleanupPartial', {
          success: deletedIDs.length,
          failed: failedCount
        })
      )
    } else {
      appStore.showSuccess(
        t('admin.accounts.batchHealthCheckCleanupSuccess', { count: deletedIDs.length })
      )
    }
  } catch (error) {
    console.error('Failed to cleanup abnormal accounts:', error)
    appStore.showError(t('admin.accounts.batchHealthCheckCleanupFailedMessage'))
  } finally {
    cleanupRunning.value = false
  }
}

function normalizeIDs(ids: number[]) {
  return [...new Set(ids.filter((id) => Number.isFinite(id) && id > 0))]
}
</script>
