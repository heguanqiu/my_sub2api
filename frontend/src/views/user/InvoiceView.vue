<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('payment.invoice.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('payment.invoice.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            <span>{{ t('common.refresh') }}</span>
          </button>
          <button type="button" class="btn btn-secondary" @click="openProfileDialog">
            <Icon name="user" size="md" />
            <span>{{ t('payment.invoice.manageProfiles') }}</span>
          </button>
          <button type="button" class="btn btn-primary" :disabled="!canOpenApplication" @click="openApplicationDialog">
            <Icon name="plus" size="md" />
            <span>{{ t('payment.invoice.apply') }}</span>
          </button>
        </div>
      </div>

      <div
        v-if="summary && !summary.enabled"
        class="rounded-lg border border-yellow-200 bg-yellow-50 p-4 text-sm text-yellow-800 dark:border-yellow-900/50 dark:bg-yellow-900/20 dark:text-yellow-200"
      >
        {{ t('payment.invoice.disabledHint') }}
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div v-for="stat in summaryStats" :key="stat.key" class="card p-5">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ stat.label }}</p>
              <p class="mt-2 font-mono text-2xl font-semibold text-gray-900 dark:text-white">{{ stat.value }}</p>
            </div>
            <div :class="['rounded-lg p-2', stat.iconClass]">
              <Icon :name="stat.icon" size="lg" />
            </div>
          </div>
          <p v-if="stat.hint" class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ stat.hint }}</p>
        </div>
      </div>

      <div class="card">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-4 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between sm:px-6">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('payment.invoice.history') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('payment.invoice.historyHint') }}</p>
          </div>
          <Select v-model="currentStatus" :options="statusFilters" class="w-full sm:w-44" @change="handleStatusChange" />
        </div>

        <div v-if="requestsLoading" class="flex items-center justify-center py-12 text-gray-500 dark:text-gray-400">
          <Icon name="refresh" size="lg" class="mr-2 animate-spin" />
          <span>{{ t('common.loading') }}</span>
        </div>
        <div v-else-if="requests.length === 0" class="py-12">
          <EmptyState :title="t('payment.invoice.emptyHistory')" :description="t('payment.invoice.emptyHistoryHint')">
            <template #icon>
              <Icon name="document" class="empty-state-icon h-10 w-10" aria-hidden="true" />
            </template>
          </EmptyState>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('payment.invoice.table.createdAt') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('payment.invoice.table.amount') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('payment.invoice.table.title') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('payment.invoice.table.content') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('payment.invoice.table.status') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('payment.invoice.table.files') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-for="request in requests" :key="request.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-600 dark:text-gray-300">
                  {{ formatDateTime(request.created_at) }}
                </td>
                <td class="whitespace-nowrap px-4 py-3 font-mono text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatCurrency(request.amount, request.currency) }}
                </td>
                <td class="min-w-48 px-4 py-3">
                  <div class="text-sm font-medium text-gray-900 dark:text-white">{{ request.title_name }}</div>
                  <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ request.email }}</div>
                </td>
                <td class="min-w-56 px-4 py-3">
                  <div class="text-sm text-gray-700 dark:text-gray-300">{{ request.content }}</div>
                  <div v-if="request.remark" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ request.remark }}</div>
                </td>
                <td class="min-w-48 px-4 py-3">
                  <span :class="statusBadgeClass(request.status)">{{ invoiceStatusLabel(request.status) }}</span>
                  <div v-if="request.invoice_no" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('payment.invoice.invoiceNo') }}: {{ request.invoice_no }}
                  </div>
                  <div v-if="request.sdk_message && request.status !== 'ISSUED'" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ request.sdk_message }}
                  </div>
                </td>
                <td class="whitespace-nowrap px-4 py-3">
                  <div class="flex flex-wrap gap-2">
                    <a v-if="request.pdf_url" :href="request.pdf_url" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20">
                      <Icon name="download" size="xs" />
                      PDF
                    </a>
                    <a v-if="request.ofd_url" :href="request.ofd_url" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20">
                      <Icon name="download" size="xs" />
                      OFD
                    </a>
                    <a v-if="request.xml_url" :href="request.xml_url" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20">
                      <Icon name="download" size="xs" />
                      XML
                    </a>
                    <span v-if="!request.pdf_url && !request.ofd_url && !request.xml_url" class="text-xs text-gray-400 dark:text-gray-500">-</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </div>
    </div>

    <BaseDialog :show="profileDialogOpen" :title="t('payment.invoice.profileDialogTitle')" width="wide" :z-index="invoiceDialogZIndex" @close="closeProfileDialog">
      <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
        <div class="space-y-3">
          <div v-if="profiles.length === 0" class="rounded-lg border border-dashed border-gray-200 p-6 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
            {{ t('payment.invoice.noProfiles') }}
          </div>
          <div
            v-for="profile in profiles"
            :key="profile.id"
            class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ profile.name }}</span>
                  <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ titleTypeLabel(profile.title_type) }}</span>
                  <span v-if="profile.is_default" class="rounded-full bg-primary-50 px-2 py-0.5 text-xs text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">{{ t('payment.invoice.defaultProfile') }}</span>
                </div>
                <div class="mt-2 space-y-1 text-xs text-gray-500 dark:text-gray-400">
                  <p v-if="profile.tax_no">{{ t('payment.invoice.taxNo') }}: {{ profile.tax_no }}</p>
                  <p v-if="profile.email">{{ t('payment.invoice.email') }}: {{ profile.email }}</p>
                  <p v-if="profile.address_phone">{{ t('payment.invoice.addressPhone') }}: {{ profile.address_phone }}</p>
                  <p v-if="profile.bank_account">{{ t('payment.invoice.bankAccount') }}: {{ profile.bank_account }}</p>
                </div>
              </div>
              <div class="flex flex-shrink-0 items-center gap-1">
                <button type="button" class="rounded-md p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-gray-200" :title="t('common.edit')" @click="editProfile(profile)">
                  <Icon name="edit" size="sm" />
                </button>
                <button type="button" class="rounded-md p-2 text-red-500 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20" :title="t('common.delete')" @click="askDeleteProfile(profile)">
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <form class="space-y-4" @submit.prevent="submitProfile">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ editingProfileId ? t('payment.invoice.editProfile') : t('payment.invoice.addProfile') }}
          </h3>
          <div>
            <label class="input-label">{{ t('payment.invoice.titleType') }} <span class="text-red-500">*</span></label>
            <Select v-model="profileForm.title_type" :options="titleTypeOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.titleName') }} <span class="text-red-500">*</span></label>
            <input v-model.trim="profileForm.name" type="text" class="input" :placeholder="t('payment.invoice.titleNamePlaceholder')" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.taxNo') }}</label>
            <input v-model.trim="profileForm.tax_no" type="text" class="input" :placeholder="t('payment.invoice.taxNoPlaceholder')" />
            <p v-if="profileForm.title_type === 'company'" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.companyTaxNoHint') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.email') }}</label>
            <input v-model.trim="profileForm.email" type="email" class="input" autocomplete="email" :placeholder="t('payment.invoice.emailPlaceholder')" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.addressPhone') }}</label>
            <input v-model.trim="profileForm.address_phone" type="text" class="input" :placeholder="t('payment.invoice.addressPhonePlaceholder')" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.bankAccount') }}</label>
            <input v-model.trim="profileForm.bank_account" type="text" class="input" :placeholder="t('payment.invoice.bankAccountPlaceholder')" />
          </div>
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="profileForm.is_default" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('payment.invoice.setDefault') }}</span>
          </label>
          <div class="flex justify-end gap-2">
            <button v-if="editingProfileId" type="button" class="btn btn-secondary" @click="resetProfileForm">{{ t('common.cancel') }}</button>
            <button type="submit" class="btn btn-primary" :disabled="profileSaving || !profileForm.name.trim()">
              {{ profileSaving ? t('common.processing') : t('common.save') }}
            </button>
          </div>
        </form>
      </div>
    </BaseDialog>

    <BaseDialog :show="applicationDialogOpen" :title="t('payment.invoice.applicationDialogTitle')" width="wide" :z-index="invoiceDialogZIndex" @close="closeApplicationDialog">
      <form class="space-y-5" @submit.prevent="submitApplication">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.totalPaid') }}</p>
            <p class="mt-1 font-mono text-lg font-semibold text-gray-900 dark:text-white">{{ formatCurrency(summary?.total_paid || 0) }}</p>
          </div>
          <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.invoicedAmount') }}</p>
            <p class="mt-1 font-mono text-lg font-semibold text-gray-900 dark:text-white">{{ formatCurrency(summary?.invoiced_amount || 0) }}</p>
          </div>
          <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.invoice.availableAmount') }}</p>
            <p class="mt-1 font-mono text-lg font-semibold text-primary-600 dark:text-primary-400">{{ formatCurrency(summary?.available_amount || 0) }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('payment.invoice.selectProfile') }} <span class="text-red-500">*</span></label>
            <Select v-model="applicationForm.profile_id" :options="profileOptions" :placeholder="t('payment.invoice.selectProfilePlaceholder')" @change="handleProfileSelected" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.applicationEmail') }} <span class="text-red-500">*</span></label>
            <input v-model.trim="applicationForm.email" type="email" class="input" autocomplete="email" :placeholder="t('payment.invoice.emailPlaceholder')" />
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.applicationAmount') }} <span class="text-red-500">*</span></label>
            <input v-model.number="applicationForm.amount" type="number" min="0" step="0.01" class="input" :placeholder="formatCurrency(summary?.min_amount || 0)" />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('payment.invoice.minAmountHint', { amount: formatCurrency(summary?.min_amount || 0) }) }}
            </p>
          </div>
          <div>
            <label class="input-label">{{ t('payment.invoice.content') }} <span class="text-red-500">*</span></label>
            <input v-model.trim="applicationForm.content" type="text" class="input" :placeholder="t('payment.invoice.contentPlaceholder')" />
          </div>
          <div class="md:col-span-2">
            <label class="input-label">{{ t('payment.invoice.remark') }}</label>
            <textarea v-model.trim="applicationForm.remark" rows="3" class="input" :placeholder="t('payment.invoice.remarkPlaceholder')" />
          </div>
        </div>

        <p v-if="applicationError" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
          {{ applicationError }}
        </p>

        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeApplicationDialog">{{ t('common.cancel') }}</button>
          <button type="submit" class="btn btn-primary" :disabled="applicationSaving || !!validateApplication()">
            {{ applicationSaving ? t('payment.invoice.submitting') : t('payment.invoice.submitApplication') }}
          </button>
        </div>
      </form>
    </BaseDialog>

    <BaseDialog :show="!!deleteProfileTarget" :title="t('payment.invoice.deleteProfile')" width="narrow" :z-index="invoiceDialogZIndex" @close="deleteProfileTarget = null">
      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('payment.invoice.deleteProfileConfirm', { name: deleteProfileTarget?.name || '' }) }}
      </p>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="deleteProfileTarget = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="profileSaving" @click="confirmDeleteProfile">
            {{ profileSaving ? t('common.processing') : t('common.delete') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentAPI } from '@/api/payment'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { BasePaginationResponse } from '@/types'
import type { InvoiceProfile, InvoiceRequest, InvoiceSummary } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

type IconName = InstanceType<typeof Icon>['$props']['name']

const { t, locale } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const requestsLoading = ref(false)
const profileSaving = ref(false)
const applicationSaving = ref(false)
const summary = ref<InvoiceSummary | null>(null)
const profiles = ref<InvoiceProfile[]>([])
const requests = ref<InvoiceRequest[]>([])
const currentStatus = ref('')
const profileDialogOpen = ref(false)
const applicationDialogOpen = ref(false)
const editingProfileId = ref<number | null>(null)
const deleteProfileTarget = ref<InvoiceProfile | null>(null)
const applicationError = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const invoiceDialogZIndex = 100000010

const profileForm = reactive({
  title_type: 'personal',
  name: '',
  tax_no: '',
  address_phone: '',
  bank_account: '',
  email: '',
  is_default: false,
})

const applicationForm = reactive({
  profile_id: null as number | null,
  amount: 0,
  content: '',
  email: '',
  remark: '',
})

const titleTypeOptions = computed(() => [
  { value: 'personal', label: t('payment.invoice.titleTypes.personal') },
  { value: 'company', label: t('payment.invoice.titleTypes.company') },
])

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'ISSUING', label: t('payment.invoice.status.issuing') },
  { value: 'ISSUED', label: t('payment.invoice.status.issued') },
  { value: 'REQUIRES_AUTH', label: t('payment.invoice.status.requires_auth') },
  { value: 'FAILED', label: t('payment.invoice.status.failed') },
])

const profileOptions = computed(() =>
  profiles.value.map((profile) => ({
    value: profile.id,
    label: profile.is_default
      ? `${profile.name} (${t('payment.invoice.defaultProfile')})`
      : profile.name,
  })),
)

const canOpenApplication = computed(() => {
  if (!summary.value?.enabled) return false
  if (profiles.value.length === 0) return false
  if ((summary.value.available_amount || 0) <= 0) return false
  if ((summary.value.min_amount || 0) > 0 && summary.value.available_amount < summary.value.min_amount) return false
  return true
})

const summaryStats = computed<Array<{ key: string; label: string; value: string; hint?: string; icon: IconName; iconClass: string }>>(() => [
  {
    key: 'total_paid',
    label: t('payment.invoice.totalPaid'),
    value: formatCurrency(summary.value?.total_paid || 0),
    icon: 'creditCard',
    iconClass: 'bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-300',
  },
  {
    key: 'invoiced_amount',
    label: t('payment.invoice.invoicedAmount'),
    value: formatCurrency(summary.value?.invoiced_amount || 0),
    icon: 'checkCircle',
    iconClass: 'bg-green-50 text-green-600 dark:bg-green-900/20 dark:text-green-300',
  },
  {
    key: 'reserved_amount',
    label: t('payment.invoice.reservedAmount'),
    value: formatCurrency(summary.value?.reserved_amount || 0),
    icon: 'clock',
    iconClass: 'bg-yellow-50 text-yellow-600 dark:bg-yellow-900/20 dark:text-yellow-300',
  },
  {
    key: 'available_amount',
    label: t('payment.invoice.availableAmount'),
    value: formatCurrency(summary.value?.available_amount || 0),
    hint: t('payment.invoice.minAmountHint', { amount: formatCurrency(summary.value?.min_amount || 0) }),
    icon: 'document',
    iconClass: 'bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300',
  },
])

function formatCurrency(value: number, currency?: string): string {
  return new Intl.NumberFormat(locale.value, {
    style: 'currency',
    currency: currency || summary.value?.currency || 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(Number(value) || 0)
}

function formatDateTime(value: string): string {
  if (!value) return '-'
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function invoiceStatusLabel(status: string): string {
  const key = status.toLowerCase()
  return t(`payment.invoice.status.${key}`)
}

function statusBadgeClass(status: string): string {
  const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-medium'
  switch (status) {
    case 'ISSUED':
      return `${base} bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300`
    case 'ISSUING':
    case 'PENDING':
      return `${base} bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300`
    case 'REQUIRES_AUTH':
      return `${base} bg-yellow-50 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300`
    case 'FAILED':
      return `${base} bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300`
    default:
      return `${base} bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300`
  }
}

function titleTypeLabel(type: string): string {
  return type === 'company' ? t('payment.invoice.titleTypes.company') : t('payment.invoice.titleTypes.personal')
}

function unwrapApiData<T>(response: unknown): T {
  if (response && typeof response === 'object' && 'data' in response) {
    return (response as { data: T }).data
  }
  return response as T
}

async function loadSummary() {
  const res = await paymentAPI.getInvoiceSummary()
  summary.value = unwrapApiData<InvoiceSummary>(res)
}

async function loadProfiles() {
  const res = await paymentAPI.getInvoiceProfiles()
  const data = unwrapApiData<InvoiceProfile[]>(res)
  profiles.value = Array.isArray(data) ? data : []
}

async function loadRequests() {
  requestsLoading.value = true
  try {
    const res = await paymentAPI.getInvoiceRequests({
      page: pagination.page,
      page_size: pagination.page_size,
      status: currentStatus.value || undefined,
    })
    const data = unwrapApiData<BasePaginationResponse<InvoiceRequest>>(res)
    requests.value = Array.isArray(data?.items) ? data.items : []
    pagination.total = Number(data?.total || 0)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.invoice.errors', t('common.error')))
  } finally {
    requestsLoading.value = false
  }
}

async function loadAll() {
  loading.value = true
  try {
    await Promise.all([loadSummary(), loadProfiles(), loadRequests()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.invoice.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handleStatusChange() {
  pagination.page = 1
  loadRequests()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadRequests()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  loadRequests()
}

function openProfileDialog() {
  profileDialogOpen.value = true
  resetProfileForm()
}

function closeProfileDialog() {
  profileDialogOpen.value = false
  resetProfileForm()
}

function resetProfileForm() {
  editingProfileId.value = null
  Object.assign(profileForm, {
    title_type: 'personal',
    name: '',
    tax_no: '',
    address_phone: '',
    bank_account: '',
    email: '',
    is_default: profiles.value.length === 0,
  })
}

function editProfile(profile: InvoiceProfile) {
  editingProfileId.value = profile.id
  Object.assign(profileForm, {
    title_type: profile.title_type || 'personal',
    name: profile.name || '',
    tax_no: profile.tax_no || '',
    address_phone: profile.address_phone || '',
    bank_account: profile.bank_account || '',
    email: profile.email || '',
    is_default: profile.is_default,
  })
}

async function submitProfile() {
  if (!profileForm.name.trim()) {
    appStore.showError(t('payment.invoice.validation.titleNameRequired'))
    return
  }
  if (profileForm.email.trim() && !isValidEmail(profileForm.email.trim())) {
    appStore.showError(t('payment.invoice.validation.emailInvalid'))
    return
  }
  profileSaving.value = true
  try {
    const payload = {
      title_type: profileForm.title_type,
      name: profileForm.name.trim(),
      tax_no: profileForm.tax_no.trim(),
      address_phone: profileForm.address_phone.trim(),
      bank_account: profileForm.bank_account.trim(),
      email: profileForm.email.trim(),
      is_default: profileForm.is_default,
    }
    if (editingProfileId.value) {
      await paymentAPI.updateInvoiceProfile(editingProfileId.value, payload)
    } else {
      await paymentAPI.createInvoiceProfile(payload)
    }
    appStore.showSuccess(t('common.success'))
    await loadProfiles()
    resetProfileForm()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.invoice.errors', t('common.error')))
  } finally {
    profileSaving.value = false
  }
}

function askDeleteProfile(profile: InvoiceProfile) {
  deleteProfileTarget.value = profile
}

async function confirmDeleteProfile() {
  if (!deleteProfileTarget.value) return
  profileSaving.value = true
  try {
    await paymentAPI.deleteInvoiceProfile(deleteProfileTarget.value.id)
    appStore.showSuccess(t('common.success'))
    deleteProfileTarget.value = null
    await loadProfiles()
    resetProfileForm()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.invoice.errors', t('common.error')))
  } finally {
    profileSaving.value = false
  }
}

function openApplicationDialog() {
  const defaultProfile = profiles.value.find((profile) => profile.is_default) || profiles.value[0]
  Object.assign(applicationForm, {
    profile_id: defaultProfile?.id || null,
    amount: Math.max(Number(summary.value?.min_amount || 0), Math.min(Number(summary.value?.available_amount || 0), Number(summary.value?.available_amount || 0))),
    content: '',
    email: defaultProfile?.email || '',
    remark: '',
  })
  applicationError.value = ''
  applicationDialogOpen.value = true
}

function closeApplicationDialog() {
  applicationDialogOpen.value = false
  applicationError.value = ''
}

function handleProfileSelected(value: string | number | boolean | null) {
  const profileId = Number(value)
  const profile = profiles.value.find((item) => item.id === profileId)
  if (profile?.email) {
    applicationForm.email = profile.email
  }
}

function validateApplication(): string {
  if (!applicationForm.profile_id) return t('payment.invoice.validation.profileRequired')
  if (!applicationForm.email.trim() || !isValidEmail(applicationForm.email.trim())) return t('payment.invoice.validation.emailInvalid')
  const amount = Number(applicationForm.amount) || 0
  const available = Number(summary.value?.available_amount || 0)
  const minAmount = Number(summary.value?.min_amount || 0)
  if (amount <= 0) return t('payment.invoice.validation.amountRequired')
  if (minAmount > 0 && amount < minAmount) return t('payment.invoice.validation.amountBelowMin', { amount: formatCurrency(minAmount) })
  if (amount > available) return t('payment.invoice.validation.amountExceedsAvailable', { amount: formatCurrency(available) })
  if (!applicationForm.content.trim()) return t('payment.invoice.validation.contentRequired')
  return ''
}

async function submitApplication() {
  const validation = validateApplication()
  if (validation) {
    applicationError.value = validation
    return
  }
  applicationSaving.value = true
  applicationError.value = ''
  try {
    const res = await paymentAPI.createInvoiceRequest({
      profile_id: Number(applicationForm.profile_id),
      amount: Number(applicationForm.amount),
      content: applicationForm.content.trim(),
      email: applicationForm.email.trim(),
      remark: applicationForm.remark.trim(),
    })
    const created = unwrapApiData<InvoiceRequest>(res)
    appStore.showSuccess(
      created.status === 'ISSUED'
        ? t('payment.invoice.submitSuccessIssued')
        : t('payment.invoice.submitSuccessPending'),
    )
    closeApplicationDialog()
    await Promise.all([loadSummary(), loadRequests()])
  } catch (err: unknown) {
    applicationError.value = extractI18nErrorMessage(err, t, 'payment.invoice.errors', t('common.error'))
  } finally {
    applicationSaving.value = false
  }
}

function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
}

onMounted(loadAll)
</script>
