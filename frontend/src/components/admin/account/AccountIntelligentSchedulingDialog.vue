<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.intelligentScheduling.title')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800">
        <div class="grid gap-4 lg:grid-cols-2">
          <div class="space-y-1.5">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.intelligentScheduling.scope') }}
            </label>
            <div class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200">
              {{ t('admin.accounts.intelligentScheduling.selectedScope', { count: targetAccountIDs.length }) }}
            </div>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.intelligentScheduling.scopeHint') }}
            </p>
          </div>

          <div class="space-y-1.5">
            <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.intelligentScheduling.model') }}
            </label>
            <Select
              v-model="selectedModelId"
              :options="modelOptions"
              :disabled="running || loadingModels"
              searchable
              creatable
              :creatable-prefix="t('admin.accounts.batchHealthCheckCustomModel')"
              :placeholder="loadingModels ? t('common.loading') : t('admin.accounts.intelligentScheduling.useDefaultModel')"
            />
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.intelligentScheduling.modelHint') }}
            </p>
          </div>
        </div>

        <div
          v-if="hasMixedPlatforms"
          class="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-200"
        >
          {{ t('admin.accounts.intelligentScheduling.mixedPlatformWarning') }}
        </div>

        <div class="mt-4 rounded-lg border border-primary-200 bg-primary-50 px-3 py-3 dark:border-primary-900/40 dark:bg-primary-900/10">
          <div class="text-sm font-medium text-primary-900 dark:text-primary-100">
            {{ t('admin.accounts.intelligentScheduling.strategyTitle') }}
          </div>
          <p class="mt-1 text-xs leading-5 text-primary-800 dark:text-primary-200">
            {{ t('admin.accounts.intelligentScheduling.strategyHint') }}
          </p>
        </div>
      </div>

      <div class="grid gap-3 md:grid-cols-4">
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
          <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.intelligentScheduling.targetCount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ summary.total }}
          </div>
        </div>
        <div class="rounded-xl border border-sky-200 bg-sky-50 p-4 dark:border-sky-900/40 dark:bg-sky-900/10">
          <div class="text-xs uppercase tracking-wide text-sky-700 dark:text-sky-300">
            {{ t('admin.accounts.intelligentScheduling.testSuccessCount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-sky-700 dark:text-sky-300">
            {{ summary.test_success_count }}
          </div>
        </div>
        <div class="rounded-xl border border-green-200 bg-green-50 p-4 dark:border-green-900/40 dark:bg-green-900/10">
          <div class="text-xs uppercase tracking-wide text-green-700 dark:text-green-300">
            {{ t('admin.accounts.intelligentScheduling.appliedCount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-green-700 dark:text-green-300">
            {{ summary.applied_count }}
          </div>
        </div>
        <div class="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-900/40 dark:bg-amber-900/10">
          <div class="text-xs uppercase tracking-wide text-amber-700 dark:text-amber-300">
            {{ t('admin.accounts.intelligentScheduling.unchangedCount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-amber-700 dark:text-amber-300">
            {{ unchangedCount }}
          </div>
        </div>
      </div>

      <div class="rounded-xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700">
          <div>
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.accounts.intelligentScheduling.results') }}
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.intelligentScheduling.resultsHint') }}
            </div>
          </div>
        </div>

        <div v-if="running" class="flex items-center gap-2 px-4 py-6 text-sm text-gray-500 dark:text-gray-400">
          <Icon name="refresh" size="sm" class="animate-spin" />
          <span>{{ t('admin.accounts.intelligentScheduling.running') }}</span>
        </div>

        <div v-else-if="results.length === 0" class="px-4 py-8 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.intelligentScheduling.empty') }}
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 text-left text-gray-500 dark:border-dark-700">
                <th class="px-4 py-3">{{ t('admin.accounts.columns.name') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.columns.platformType') }}</th>
                <th class="px-4 py-3">{{ t('common.status') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.testModel') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.intelligentScheduling.latency') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.intelligentScheduling.priorityChange') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.intelligentScheduling.loadFactorChange') }}</th>
                <th class="px-4 py-3">{{ t('admin.accounts.intelligentScheduling.message') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="item in results"
                :key="item.account_id"
                class="border-b border-gray-50 align-top dark:border-dark-800"
              >
                <td class="px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ item.name }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">#{{ item.account_id }}</div>
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  {{ item.platform }} / {{ item.type }}
                </td>
                <td class="px-4 py-3">
                  <span :class="statusClass(item.status)">
                    {{ statusLabel(item.status) }}
                  </span>
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  {{ item.model_id || t('admin.accounts.intelligentScheduling.useDefaultModel') }}
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  {{ item.latency_ms != null ? `${item.latency_ms} ms` : '-' }}
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  {{ formatPriorityChange(item) }}
                </td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">
                  {{ formatLoadFactorChange(item) }}
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
        <button type="button" class="btn btn-secondary" :disabled="running" @click="emit('close')">
          {{ t('common.close') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="running || targetAccountIDs.length === 0"
          @click="runIntelligentScheduling"
        >
          <Icon v-if="running" name="refresh" size="sm" class="mr-2 animate-spin" />
          <Icon v-else name="play" size="sm" class="mr-2" />
          {{
            running
              ? t('common.processing')
              : t('admin.accounts.intelligentScheduling.start')
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
import type { IntelligentSchedulingItem, IntelligentSchedulingResponse } from '@/api/admin/accounts'
import type { Account } from '@/types'
import { useAppStore } from '@/stores'

const props = defineProps<{
  show: boolean
  visibleAccounts: Account[]
  selectedAccountIDs: number[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'applied', accountIds: number[]): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const selectedModelId = ref<string | number | boolean | null>('')
const modelOptions = ref<SelectOption[]>([])
const loadingModels = ref(false)
const running = ref(false)
const results = ref<IntelligentSchedulingItem[]>([])
const summary = ref<IntelligentSchedulingResponse>({
  items: [],
  total: 0,
  test_success_count: 0,
  test_failed_count: 0,
  applied_count: 0
})

const targetAccountIDs = computed(() => normalizeIDs(props.selectedAccountIDs))
const selectedAccounts = computed(() => {
  const selected = new Set(targetAccountIDs.value)
  return props.visibleAccounts.filter((account) => selected.has(account.id))
})
const hasMixedPlatforms = computed(() => {
  const platforms = new Set(selectedAccounts.value.map((account) => account.platform))
  return platforms.size > 1
})
const unchangedCount = computed(() => Math.max(0, summary.value.total - summary.value.applied_count))

watch(
  () => props.show,
  async (visible) => {
    if (!visible) return
    selectedModelId.value = ''
    results.value = []
    summary.value = {
      items: [],
      total: targetAccountIDs.value.length,
      test_success_count: 0,
      test_failed_count: 0,
      applied_count: 0
    }
    await loadModelOptions()
  },
  { immediate: true }
)

watch(
  () => [props.show, props.selectedAccountIDs.join(','), props.visibleAccounts.map((account) => account.id).join(',')],
  async () => {
    if (!props.show) return
    selectedModelId.value = ''
    await loadModelOptions()
  }
)

async function loadModelOptions() {
  const sourceAccountID = targetAccountIDs.value[0]
  modelOptions.value = [{ value: '', label: t('admin.accounts.intelligentScheduling.useDefaultModel') }]
  if (!sourceAccountID) return

  loadingModels.value = true
  try {
    const models = await adminAPI.accounts.getAvailableModels(sourceAccountID)
    modelOptions.value = [
      { value: '', label: t('admin.accounts.intelligentScheduling.useDefaultModel') },
      ...models.map((model) => ({
        value: model.id,
        label: model.display_name || model.id
      }))
    ]
  } catch (error) {
    console.error('Failed to load intelligent scheduling model options:', error)
  } finally {
    loadingModels.value = false
  }
}

async function runIntelligentScheduling() {
  if (targetAccountIDs.value.length === 0) {
    appStore.showError(t('admin.accounts.intelligentScheduling.emptyScope'))
    return
  }

  running.value = true
  try {
    const response = await adminAPI.accounts.intelligentScheduling({
      account_ids: targetAccountIDs.value,
      model_id: String(selectedModelId.value || '').trim() || undefined
    })
    summary.value = response
    results.value = response.items

    const appliedIDs = response.items
      .filter((item) => item.updated)
      .map((item) => item.account_id)

    if (appliedIDs.length > 0) {
      emit('applied', appliedIDs)
    }

    if (response.applied_count === response.total) {
      appStore.showSuccess(
        t('admin.accounts.intelligentScheduling.success', { count: response.applied_count })
      )
    } else if (response.applied_count > 0) {
      appStore.showInfo(
        t('admin.accounts.intelligentScheduling.partial', {
          applied: response.applied_count,
          unchanged: response.total - response.applied_count
        })
      )
    } else {
      appStore.showError(t('admin.accounts.intelligentScheduling.failed'))
    }
  } catch (error) {
    console.error('Failed to run intelligent scheduling:', error)
    appStore.showError(t('admin.accounts.intelligentScheduling.failed'))
  } finally {
    running.value = false
  }
}

function formatPriorityChange(item: IntelligentSchedulingItem) {
  if (item.updated && item.new_priority != null) {
    return `${item.previous_priority} -> ${item.new_priority}`
  }
  return String(item.previous_priority)
}

function formatLoadFactorChange(item: IntelligentSchedulingItem) {
  const previous = item.previous_effective_load_factor
  if (item.updated && item.new_load_factor != null) {
    return `${previous} -> ${item.new_load_factor}`
  }
  return String(previous)
}

function statusLabel(status: IntelligentSchedulingItem['status']) {
  switch (status) {
    case 'applied':
      return t('admin.accounts.intelligentScheduling.statusApplied')
    case 'update_failed':
      return t('admin.accounts.intelligentScheduling.statusUpdateFailed')
    case 'tested':
      return t('admin.accounts.intelligentScheduling.statusTested')
    default:
      return t('admin.accounts.intelligentScheduling.statusTestFailed')
  }
}

function statusClass(status: IntelligentSchedulingItem['status']) {
  switch (status) {
    case 'applied':
      return 'inline-flex items-center rounded-full bg-green-100 px-2.5 py-1 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-300'
    case 'update_failed':
      return 'inline-flex items-center rounded-full bg-amber-100 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'tested':
      return 'inline-flex items-center rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
    default:
      return 'inline-flex items-center rounded-full bg-red-100 px-2.5 py-1 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-300'
  }
}

function normalizeIDs(ids: number[]) {
  return [...new Set(ids.filter((id) => Number.isFinite(id) && id > 0))]
}
</script>
