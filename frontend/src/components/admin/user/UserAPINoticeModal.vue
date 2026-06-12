<template>
  <BaseDialog :show="show" title="API 请求提醒" width="wide" @close="handleClose">
    <div v-if="user" class="space-y-5">
      <div class="flex items-center gap-3 rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
          <span class="text-lg font-medium text-primary-700 dark:text-primary-300">
            {{ user.email.charAt(0).toUpperCase() }}
          </span>
        </div>
        <div class="min-w-0">
          <p class="truncate font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            #{{ user.id }}<span v-if="user.username"> · {{ user.username }}</span>
          </p>
        </div>
      </div>

      <form class="space-y-3 rounded-lg border border-gray-200 p-4 dark:border-dark-600" @submit.prevent="submitNotice">
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">提醒内容</label>
          <textarea
            v-model="message"
            class="input min-h-[96px] w-full resize-y"
            maxlength="2000"
            placeholder="这条消息会在该用户下一次 API 请求时返回，并终止本次请求。"
          ></textarea>
          <div class="mt-1 flex justify-between text-xs text-gray-500 dark:text-dark-400">
            <span>仅消费一次；消费前可取消。</span>
            <span>{{ message.length }}/2000</span>
          </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">过期时间</label>
            <input v-model="expiresAtLocal" type="datetime-local" class="input w-full" />
          </div>
          <button type="submit" class="btn btn-primary" :disabled="submitting || !message.trim()">
            {{ submitting ? '发送中...' : '发送提醒' }}
          </button>
        </div>
      </form>

      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">提醒记录</h4>
          <div class="flex items-center gap-2">
            <select v-model="statusFilter" class="input h-9 w-32 text-sm" @change="loadNotices(1)">
              <option value="">全部状态</option>
              <option value="pending">待消费</option>
              <option value="consumed">已消费</option>
              <option value="cancelled">已取消</option>
            </select>
            <button class="btn btn-secondary px-3 py-2 text-sm" :disabled="loading" @click="loadNotices(1)">
              刷新
            </button>
          </div>
        </div>

        <div v-if="loading" class="flex justify-center py-8">
          <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        </div>

        <div v-else-if="notices.length === 0" class="rounded-lg border border-dashed border-gray-200 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">
          暂无提醒记录
        </div>

        <div v-else class="max-h-[26rem] space-y-3 overflow-y-auto">
          <div
            v-for="notice in notices"
            :key="notice.id"
            class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <div class="mb-2 flex flex-wrap items-center gap-2">
                  <span :class="statusClass(notice.status)">{{ statusLabel(notice.status) }}</span>
                  <span class="text-xs text-gray-400 dark:text-dark-500">#{{ notice.id }}</span>
                  <span v-if="notice.expires_at" class="text-xs text-gray-500 dark:text-dark-400">
                    过期：{{ formatDateTime(notice.expires_at) }}
                  </span>
                </div>
                <p class="whitespace-pre-wrap break-words text-sm text-gray-800 dark:text-gray-200">{{ notice.message }}</p>
                <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
                  创建：{{ formatDateTime(notice.created_at) }}
                  <span v-if="notice.consumed_at"> · 消费：{{ formatDateTime(notice.consumed_at) }}</span>
                  <span v-if="notice.consumed_request_id"> · 请求：{{ notice.consumed_request_id }}</span>
                  <span v-if="notice.cancelled_at"> · 取消：{{ formatDateTime(notice.cancelled_at) }}</span>
                </p>
              </div>
              <button
                v-if="notice.status === 'pending'"
                class="btn btn-secondary shrink-0 px-3 py-1.5 text-sm"
                :disabled="cancellingIds.has(notice.id)"
                @click="cancelNotice(notice.id)"
              >
                {{ cancellingIds.has(notice.id) ? '取消中...' : '取消' }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="totalPages > 1" class="flex items-center justify-center gap-2 pt-1">
          <button class="btn btn-secondary px-3 py-1 text-sm" :disabled="page <= 1" @click="loadNotices(page - 1)">上一页</button>
          <span class="text-sm text-gray-500 dark:text-dark-400">{{ page }} / {{ totalPages }}</span>
          <button class="btn btn-secondary px-3 py-1 text-sm" :disabled="page >= totalPages" @click="loadNotices(page + 1)">下一页</button>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import type { UserAPINotice, UserAPINoticeStatus } from '@/api/admin/users'
import type { AdminUser } from '@/types'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits(['close'])
const appStore = useAppStore()

const message = ref('')
const expiresAtLocal = ref('')
const submitting = ref(false)
const loading = ref(false)
const notices = ref<UserAPINotice[]>([])
const page = ref(1)
const pageSize = 10
const total = ref(0)
const statusFilter = ref<UserAPINoticeStatus | ''>('')
const cancellingIds = ref(new Set<number>())

const totalPages = computed(() => Math.ceil(total.value / pageSize) || 1)

watch(() => props.show, (visible) => {
  if (visible && props.user) {
    message.value = ''
    expiresAtLocal.value = ''
    statusFilter.value = ''
    loadNotices(1)
  }
})

const handleClose = () => {
  emit('close')
}

const submitNotice = async () => {
  if (!props.user) return
  const trimmed = message.value.trim()
  if (!trimmed) {
    appStore.showError('请输入提醒内容')
    return
  }

  let expiresAt: number | null = null
  if (expiresAtLocal.value) {
    const time = new Date(expiresAtLocal.value)
    if (Number.isNaN(time.getTime()) || time.getTime() <= Date.now()) {
      appStore.showError('过期时间必须晚于当前时间')
      return
    }
    expiresAt = Math.floor(time.getTime() / 1000)
  }

  submitting.value = true
  try {
    await adminAPI.users.createApiNotice(props.user.id, {
      message: trimmed,
      expires_at: expiresAt
    })
    appStore.showSuccess('提醒已发送')
    message.value = ''
    expiresAtLocal.value = ''
    await loadNotices(1)
  } catch (error: any) {
    appStore.showError(error?.message || error?.response?.data?.message || '发送提醒失败')
  } finally {
    submitting.value = false
  }
}

const loadNotices = async (nextPage: number) => {
  if (!props.user) return
  page.value = nextPage
  loading.value = true
  try {
    const res = await adminAPI.users.listApiNotices(props.user.id, page.value, pageSize, statusFilter.value)
    notices.value = res.items || []
    total.value = res.total || 0
  } catch (error: any) {
    appStore.showError(error?.message || error?.response?.data?.message || '加载提醒记录失败')
  } finally {
    loading.value = false
  }
}

const cancelNotice = async (noticeId: number) => {
  cancellingIds.value.add(noticeId)
  try {
    await adminAPI.users.cancelApiNotice(noticeId)
    appStore.showSuccess('提醒已取消')
    await loadNotices(page.value)
  } catch (error: any) {
    appStore.showError(error?.message || error?.response?.data?.message || '取消提醒失败')
  } finally {
    cancellingIds.value.delete(noticeId)
  }
}

const statusLabel = (status: UserAPINoticeStatus) => {
  switch (status) {
    case 'pending':
      return '待消费'
    case 'consumed':
      return '已消费'
    case 'cancelled':
      return '已取消'
    default:
      return status
  }
}

const statusClass = (status: UserAPINoticeStatus) => {
  const base = 'inline-flex items-center rounded px-2 py-0.5 text-xs font-medium'
  switch (status) {
    case 'pending':
      return `${base} bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300`
    case 'consumed':
      return `${base} bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300`
    case 'cancelled':
      return `${base} bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300`
    default:
      return `${base} bg-gray-100 text-gray-600`
  }
}
</script>
