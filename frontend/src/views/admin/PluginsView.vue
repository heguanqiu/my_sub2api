<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-72">
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('admin.plugins.searchPlaceholder')"
              class="input"
              @input="handleSearch"
            />
          </div>

          <Select
            v-model="filters.status"
            :options="statusFilterOptions"
            class="w-40"
            @change="handleStatusChange"
          />

          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadPlugins"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              data-test="open-create-plugin"
              type="button"
              class="btn btn-primary"
              @click="openCreateDialog"
            >
              <Icon name="plus" size="md" class="mr-1" />
              {{ t('admin.plugins.create') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="plugins"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-name="{ row }">
            <div class="flex min-w-0 items-center gap-3">
              <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300">
                <Icon name="cube" size="md" />
              </div>
              <div class="min-w-0">
                <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
                <div class="mt-1 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                  <span>#{{ row.id }}</span>
                  <span v-if="row.file_name" class="truncate">{{ row.file_name }}</span>
                </div>
              </div>
            </div>
          </template>

          <template #cell-category="{ value }">
            <span class="text-sm text-gray-600 dark:text-gray-300">{{ value || '-' }}</span>
          </template>

          <template #cell-version="{ value }">
            <span class="badge badge-gray">{{ value || '-' }}</span>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', value === 'published' ? 'badge-success' : 'badge-gray']">
              {{ statusLabel(value) }}
            </span>
          </template>

          <template #cell-download_count="{ value }">
            <span class="text-sm tabular-nums text-gray-700 dark:text-gray-200">{{ value }}</span>
          </template>

          <template #cell-sort_weight="{ value }">
            <span class="text-sm tabular-nums text-gray-700 dark:text-gray-200">{{ value }}</span>
          </template>

          <template #cell-updated_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center space-x-1">
              <button
                :data-test="`toggle-plugin-${row.id}`"
                type="button"
                class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                :title="row.status === 'published' ? t('admin.plugins.statusDraft') : t('admin.plugins.statusPublished')"
                :disabled="saving"
                @click="toggleStatus(row)"
              >
                <Icon :name="row.status === 'published' ? 'eyeOff' : 'eye'" size="sm" />
              </button>
              <button
                :data-test="`edit-plugin-${row.id}`"
                type="button"
                class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('common.edit')"
                @click="openEditDialog(row)"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                :data-test="`delete-plugin-${row.id}`"
                type="button"
                class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('common.delete')"
                @click="handleDelete(row)"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('empty.noData')"
              :description="t('admin.plugins.description')"
              :action-text="t('admin.plugins.create')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showEditDialog"
      :title="isEditing ? t('admin.plugins.edit') : t('admin.plugins.create')"
      width="wide"
      @close="closeEdit"
    >
      <form id="plugin-form" data-test="plugin-form" class="space-y-5" @submit.prevent="handleSave">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.plugins.name') }}</label>
            <input v-model="form.name" data-test="plugin-name-input" type="text" class="input" required />
          </div>
          <div>
            <label class="input-label">{{ t('admin.plugins.version') }}</label>
            <input v-model="form.version" data-test="plugin-version-input" type="text" class="input" placeholder="v1.0.0" />
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.plugins.description') }}</label>
          <textarea v-model="form.description" data-test="plugin-description-input" rows="4" class="input"></textarea>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.plugins.category') }}</label>
            <input v-model="form.category" data-test="plugin-category-input" type="text" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.plugins.platform') }}</label>
            <Select v-model="form.platform" :options="platformOptions" />
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.plugins.status') }}</label>
            <Select v-model="form.status" :options="statusOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.plugins.sortWeight') }}</label>
            <input v-model.number="form.sort_weight" type="number" class="input" />
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.plugins.packageFile') }}</label>
            <input
              data-test="package-upload"
              type="file"
              class="input"
              accept=".zip,.vsix,.tar.gz,.tgz"
              :disabled="uploadingPackage"
              @change="handlePackageUpload"
            />
            <p class="input-hint">
              {{ form.file_name || form.file_key || t('admin.plugins.uploadPackage') }}
              <span v-if="form.file_size">({{ formatBytes(form.file_size, 1) }})</span>
            </p>
          </div>

          <div>
            <label class="input-label">{{ t('admin.plugins.icon') }}</label>
            <input
              data-test="icon-upload"
              type="file"
              class="input"
              accept=".png,.jpg,.jpeg,.svg,.webp,image/*"
              :disabled="uploadingIcon"
              @change="handleIconUpload"
            />
            <p class="input-hint">
              {{ form.icon_key || t('admin.plugins.uploadIcon') }}
            </p>
          </div>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeEdit">
            {{ t('common.cancel') }}
          </button>
          <button
            data-test="save-plugin"
            type="submit"
            form="plugin-form"
            class="btn btn-primary"
            :disabled="saving || uploadingPackage || uploadingIcon"
          >
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('common.delete')"
      :message="t('admin.plugins.deleteConfirm')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { adminAPI } from '@/api/admin'
import { formatBytes, formatDateTime } from '@/utils/format'
import type { AdminPlugin, SavePluginRequest } from '@/types'
import type { Column } from '@/components/common/types'

import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const plugins = ref<AdminPlugin[]>([])
const loading = ref(false)
const saving = ref(false)
const uploadingPackage = ref(false)
const uploadingIcon = ref(false)

const filters = reactive({
  status: ''
})
const searchQuery = ref('')

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const statusFilterOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'draft', label: t('admin.plugins.statusDraft') },
  { value: 'published', label: t('admin.plugins.statusPublished') }
])

const statusOptions = computed(() => [
  { value: 'draft', label: t('admin.plugins.statusDraft') },
  { value: 'published', label: t('admin.plugins.statusPublished') }
])

const platformOptions = [
  { value: 'all', label: 'All' },
  { value: 'windows', label: 'Windows' },
  { value: 'macos', label: 'macOS' },
  { value: 'linux', label: 'Linux' }
]

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.plugins.name'), sortable: true },
  { key: 'category', label: t('admin.plugins.category'), sortable: true },
  { key: 'version', label: t('admin.plugins.version') },
  { key: 'status', label: t('admin.plugins.status'), sortable: true },
  { key: 'download_count', label: t('admin.plugins.downloadCount'), sortable: true },
  { key: 'sort_weight', label: t('admin.plugins.sortWeight'), sortable: true },
  { key: 'updated_at', label: 'Updated', sortable: true },
  { key: 'actions', label: '' }
])

const statusLabel = (status: string) => {
  if (status === 'published') return t('admin.plugins.statusPublished')
  return t('admin.plugins.statusDraft')
}

let currentController: AbortController | null = null

async function loadPlugins() {
  currentController?.abort()
  const requestController = new AbortController()
  currentController = requestController
  const { signal } = requestController

  try {
    loading.value = true
    const res = await adminAPI.plugins.list(pagination.page, pagination.page_size, {
      status: filters.status || undefined,
      search: searchQuery.value || undefined,
      sort_by: sortState.sort_by,
      sort_order: sortState.sort_order
    }, { signal })

    if (signal.aborted || currentController !== requestController) return

    plugins.value = res.items
    pagination.total = res.total
    pagination.pages = res.pages
    pagination.page = res.page
    pagination.page_size = res.page_size
  } catch (error: any) {
    if (
      signal.aborted ||
      currentController !== requestController ||
      error?.name === 'AbortError' ||
      error?.code === 'ERR_CANCELED'
    ) {
      return
    }
    appStore.showError(error.response?.data?.detail || error?.message || t('admin.plugins.description'))
  } finally {
    if (currentController === requestController) {
      loading.value = false
      currentController = null
    }
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadPlugins()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadPlugins()
}

function handleStatusChange() {
  pagination.page = 1
  loadPlugins()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadPlugins()
}

let searchDebounceTimer: number | null = null
function handleSearch() {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  searchDebounceTimer = window.setTimeout(() => {
    pagination.page = 1
    loadPlugins()
  }, 300)
}

const showEditDialog = ref(false)
const editingPlugin = ref<AdminPlugin | null>(null)
const isEditing = computed(() => !!editingPlugin.value)

const form = reactive({
  name: '',
  description: '',
  version: '',
  category: '',
  platform: 'all',
  icon_key: '',
  file_key: '',
  file_name: '',
  file_size: 0,
  status: 'draft' as 'draft' | 'published',
  sort_weight: 0
})

function resetForm() {
  form.name = ''
  form.description = ''
  form.version = ''
  form.category = ''
  form.platform = 'all'
  form.icon_key = ''
  form.file_key = ''
  form.file_name = ''
  form.file_size = 0
  form.status = 'draft'
  form.sort_weight = 0
}

function fillForm(row: AdminPlugin) {
  form.name = row.name
  form.description = row.description
  form.version = row.version
  form.category = row.category
  form.platform = row.platform || 'all'
  form.icon_key = row.icon_key
  form.file_key = row.file_key
  form.file_name = row.file_name
  form.file_size = row.file_size
  form.status = row.status
  form.sort_weight = row.sort_weight
}

function openCreateDialog() {
  editingPlugin.value = null
  resetForm()
  showEditDialog.value = true
}

function openEditDialog(row: AdminPlugin) {
  editingPlugin.value = row
  fillForm(row)
  showEditDialog.value = true
}

function closeEdit() {
  showEditDialog.value = false
  editingPlugin.value = null
}

function buildPayload(): SavePluginRequest {
  return {
    name: form.name,
    description: form.description,
    version: form.version,
    category: form.category,
    platform: form.platform,
    icon_key: form.icon_key,
    file_key: form.file_key,
    file_name: form.file_name,
    file_size: Number(form.file_size) || 0,
    status: form.status,
    sort_weight: Number(form.sort_weight) || 0
  }
}

async function handleSave() {
  saving.value = true
  try {
    const payload = buildPayload()
    if (editingPlugin.value) {
      await adminAPI.plugins.update(editingPlugin.value.id, payload)
    } else {
      await adminAPI.plugins.create(payload)
    }
    appStore.showSuccess(t('admin.plugins.saved'))
    closeEdit()
    await loadPlugins()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || error?.message || t('admin.plugins.description'))
  } finally {
    saving.value = false
  }
}

async function uploadFile(kind: 'package' | 'icon', file: File) {
  if (kind === 'package') {
    uploadingPackage.value = true
  } else {
    uploadingIcon.value = true
  }
  try {
    const result = await adminAPI.plugins.upload(kind, file)
    if (kind === 'package') {
      form.file_key = result.key
      form.file_name = result.file_name
      form.file_size = result.size
    } else {
      form.icon_key = result.key
    }
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || error?.message || t('admin.plugins.description'))
  } finally {
    if (kind === 'package') {
      uploadingPackage.value = false
    } else {
      uploadingIcon.value = false
    }
  }
}

function handlePackageUpload(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (file) {
    uploadFile('package', file)
  }
}

function handleIconUpload(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (file) {
    uploadFile('icon', file)
  }
}

async function toggleStatus(row: AdminPlugin) {
  saving.value = true
  try {
    await adminAPI.plugins.update(row.id, {
      ...rowToPayload(row),
      status: row.status === 'published' ? 'draft' : 'published'
    })
    appStore.showSuccess(t('admin.plugins.saved'))
    await loadPlugins()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || error?.message || t('admin.plugins.description'))
  } finally {
    saving.value = false
  }
}

function rowToPayload(row: AdminPlugin): SavePluginRequest {
  return {
    name: row.name,
    description: row.description,
    version: row.version,
    category: row.category,
    platform: row.platform,
    icon_key: row.icon_key,
    file_key: row.file_key,
    file_name: row.file_name,
    file_size: row.file_size,
    status: row.status,
    sort_weight: row.sort_weight
  }
}

const showDeleteDialog = ref(false)
const deletingPlugin = ref<AdminPlugin | null>(null)

function handleDelete(row: AdminPlugin) {
  deletingPlugin.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingPlugin.value) return

  try {
    await adminAPI.plugins.deletePlugin(deletingPlugin.value.id)
    appStore.showSuccess(t('admin.plugins.deleted'))
    showDeleteDialog.value = false
    deletingPlugin.value = null
    await loadPlugins()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || error?.message || t('admin.plugins.description'))
  }
}

onMounted(() => {
  loadPlugins()
})

onUnmounted(() => {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  currentController?.abort()
})
</script>
