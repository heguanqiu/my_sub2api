<template>
  <div class="rounded-lg border border-gray-100 p-3 dark:border-dark-700">
    <!-- 头部：key名称、状态徽章、保存按钮 -->
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">
            {{ remoteKey.remote_api_key_name || remoteKey.remote_api_key_id }}
          </span>
          <span :class="['badge', keyConfigured ? 'badge-success' : 'badge-warning']">
            {{ keyConfigured ? tr('admin.upstreams.keyConfigured', '已配置') : tr('admin.upstreams.keyNotConfigured', '未配置') }}
          </span>
          <span :class="['badge', keySchedulable ? 'badge-success' : 'badge-gray']">
            {{ keySchedulable ? tr('admin.upstreams.keySchedulable', '可调度') : tr('admin.upstreams.keyNotSchedulable', '不可调度') }}
          </span>
        </div>
        <div class="mt-1 flex flex-wrap gap-1 text-[11px] text-gray-500 dark:text-dark-400">
          <span class="font-mono">{{ remoteKey.remote_api_key_id }}</span>
          <span v-if="remoteKey.masked_key">· {{ remoteKey.masked_key }}</span>
        </div>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="btn btn-primary btn-sm"
          :disabled="saving"
          @click="save"
        >
          <Icon name="check" size="sm" />
          {{ saving ? tr('common.saving', '保存中') : tr('common.save', '保存') }}
        </button>
      </div>
    </div>

    <!-- 表单主体：API key 输入 + 分组勾选 -->
    <div class="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-[minmax(220px,0.8fr)_minmax(0,1.2fr)]">
      <!-- 左列：API key 密文 + 调度开关 -->
      <label class="space-y-1">
        <span class="input-label">{{ tr('admin.upstreams.upstreamAPIKeySecret', '上游 API key') }}</span>
        <input
          v-model="draft.apiKey"
          type="password"
          class="input"
          :placeholder="remoteKey.api_key_configured
            ? tr('admin.upstreams.keepExistingKey', '留空则保留已配置密钥')
            : 'sk-...'"
          autocomplete="new-password"
        />
        <div class="mt-2 inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
          <input
            v-model="draft.schedulingEnabled"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span>{{ tr('admin.upstreams.keySchedulingEnabled', '参与调度') }}</span>
        </div>
      </label>

      <!-- 右列：本地映射分组 -->
      <div class="space-y-2">
        <span class="input-label">{{ tr('admin.upstreams.mappedLocalGroups', '映射本地分组') }}</span>

        <!-- 孤立 ID 警告条：已删除分组 -->
        <div
          v-if="orphanedGroupIds.length > 0"
          class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 dark:border-amber-800 dark:bg-amber-900/20"
        >
          <div class="mb-1 text-xs font-medium text-amber-700 dark:text-amber-400">
            {{ tr('admin.upstreams.deletedGroupsWarning', '{count} 个分组已被删除，保存后将自动清除').replace('{count}', String(orphanedGroupIds.length)) }}
          </div>
          <div class="flex flex-wrap gap-1">
            <span
              v-for="id in orphanedGroupIds"
              :key="id"
              class="inline-flex items-center gap-1 rounded-md bg-amber-100 px-2 py-0.5 text-xs text-amber-700 dark:bg-amber-800/40 dark:text-amber-300"
            >
              {{ tr('admin.upstreams.deletedGroupId', 'ID: {id}（已删除）').replace('{id}', String(id)) }}
              <button
                type="button"
                class="ml-0.5 text-amber-500 hover:text-amber-700 dark:text-amber-400 dark:hover:text-amber-200"
                @click="removeGroupId(id)"
              >
                ×
              </button>
            </span>
          </div>
        </div>

        <!-- 可用分组列表 -->
        <div
          v-if="localGroups.length === 0"
          class="rounded-lg border border-dashed border-gray-200 px-3 py-2 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400"
        >
          {{ tr('admin.upstreams.noLocalRuntimeGroups', '暂无本地 OpenAI / Anthropic 分组') }}
        </div>
        <div
          v-else
          class="flex max-h-28 flex-wrap gap-2 overflow-y-auto rounded-lg border border-gray-100 p-2 dark:border-dark-700"
        >
          <label
            v-for="group in localGroups"
            :key="group.id"
            class="inline-flex items-center gap-2 rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-700 dark:border-dark-600 dark:text-gray-200"
          >
            <input
              type="checkbox"
              class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              :checked="draft.localGroupIds.includes(group.id)"
              @change="toggleGroup(group.id)"
            />
            <span>{{ groupLabel(group) }}</span>
          </label>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminGroup } from '@/types'
import type { UpstreamRemoteAPIKey } from '@/api/admin/upstreams'
import { updateRemoteAPIKeyConfig } from '@/api/admin/upstreams'
import { useAppStore } from '@/stores/app'
import Icon from '@/components/icons/Icon.vue'

function errorMessage(error: unknown, fallback: string): string {
  const e = error as any
  return e?.response?.data?.detail || e?.message || e?.error || fallback
}

// ─── props & emits ──────────────────────────────────────────────────────────

const props = defineProps<{
  upstreamId: number
  remoteKey: UpstreamRemoteAPIKey
  localGroups: AdminGroup[]
}>()

const emit = defineEmits<{
  saved: [key: UpstreamRemoteAPIKey]
}>()

// ─── i18n & store ───────────────────────────────────────────────────────────

const { t } = useI18n()
const appStore = useAppStore()

function tr(key: string, fallback: string): string {
  return t(key, fallback)
}

// ─── draft state ─────────────────────────────────────────────────────────────

const draft = reactive({
  apiKey: '',
  schedulingEnabled: true,
  localGroupIds: [] as number[]
})

const saving = ref(false)

function syncDraft() {
  draft.apiKey = ''
  draft.schedulingEnabled = props.remoteKey.scheduling_enabled !== false
  draft.localGroupIds = [...(props.remoteKey.local_group_ids || [])]
}

// remoteKey 变化时同步草稿（切换上游或后端返回新数据后刷新）
watch(() => props.remoteKey, syncDraft, { immediate: true })

// ─── computed ────────────────────────────────────────────────────────────────

/** 已删除的分组 ID（在草稿里但不在可用列表里） */
const orphanedGroupIds = computed(() =>
  draft.localGroupIds.filter((id) => !props.localGroups.find((g) => g.id === id))
)

/** 调度相关的只读派生属性 */
const keyConfigured = computed(() => !!props.remoteKey.api_key_configured || !!props.remoteKey.masked_key)

const keySchedulable = computed(() => {
  const status = String(props.remoteKey.status || '').trim().toLowerCase()
  const active = ['', 'active', 'enabled', 'enable', '1', 'true'].includes(status)
  return draft.schedulingEnabled && keyConfigured.value && active && draft.localGroupIds.length > 0
})

// ─── actions ─────────────────────────────────────────────────────────────────

function groupLabel(group: AdminGroup): string {
  const platform =
    group.platform === 'anthropic' ? 'Anthropic' :
    group.platform === 'openai' ? 'OpenAI' :
    group.platform
  return `${group.name} · ${platform}`
}

function toggleGroup(id: number) {
  const idx = draft.localGroupIds.indexOf(id)
  if (idx >= 0) {
    draft.localGroupIds.splice(idx, 1)
  } else {
    draft.localGroupIds.push(id)
  }
}

/** 手动移除一个孤立 ID */
function removeGroupId(id: number) {
  const idx = draft.localGroupIds.indexOf(id)
  if (idx >= 0) draft.localGroupIds.splice(idx, 1)
}

async function save() {
  saving.value = true
  try {
    // 只提交仍然存在的分组 ID，孤立 ID 在此处静默丢弃
    const validGroupIds = draft.localGroupIds.filter((id) =>
      props.localGroups.find((g) => g.id === id)
    )
    const saved = await updateRemoteAPIKeyConfig(
      props.upstreamId,
      props.remoteKey.remote_api_key_id,
      {
        local_group_ids: validGroupIds,
        scheduling_enabled: draft.schedulingEnabled,
        api_key: draft.apiKey.trim() || undefined
      }
    )
    // 用后端返回值刷新草稿（后端可能已清理了其他悬空 ID）
    draft.apiKey = ''
    draft.localGroupIds = [...(saved.local_group_ids || [])]
    draft.schedulingEnabled = saved.scheduling_enabled !== false
    appStore.showSuccess(tr('admin.upstreams.keyConfigSaved', 'API key 配置已保存'))
    emit('saved', saved)
  } catch (error: unknown) {
    appStore.showError(errorMessage(error, tr('admin.upstreams.keyConfigSaveFailed', '保存 API key 配置失败')))
  } finally {
    saving.value = false
  }
}
</script>
