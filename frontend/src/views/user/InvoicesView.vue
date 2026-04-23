<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-6">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('invoice.title') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('invoice.description') }}</p>
          </div>
          <button class="btn btn-secondary" @click="fetchAll">{{ t('common.refresh') }}</button>
        </div>
      </div>

      <div class="card p-6">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('invoice.profiles') }}</h3>
          <button class="btn btn-primary" @click="openCreateProfile">{{ t('common.create') }}</button>
        </div>
        <div class="space-y-3">
          <div v-for="profile in profiles" :key="profile.id" class="rounded-2xl border border-gray-100 p-4 dark:border-dark-700">
            <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
              <div>
                <div class="flex items-center gap-2">
                  <p class="font-medium text-gray-900 dark:text-white">{{ profile.title }}</p>
                  <span v-if="profile.is_default" class="badge badge-success">{{ t('common.default') }}</span>
                </div>
                <p class="mt-1 text-sm text-gray-500">{{ formatInvoiceProfileType(profile.invoice_type) }}</p>
              </div>
              <div class="flex gap-2">
                <button class="btn btn-secondary" @click="editProfile(profile)">{{ t('common.edit') }}</button>
                <button class="btn btn-secondary" :disabled="profile.is_default" @click="setDefault(profile.id)">{{ t('invoice.setDefault') }}</button>
                <button class="btn btn-danger" @click="removeProfile(profile.id)">{{ t('common.delete') }}</button>
              </div>
            </div>
          </div>
          <div v-if="!profiles.length" class="py-6 text-center text-gray-500">{{ t('common.noData') }}</div>
        </div>
      </div>

      <div class="card p-6">
        <h3 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">{{ t('invoice.requestableOrders') }}</h3>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 text-left text-gray-500 dark:border-dark-700">
                <th class="py-3 pr-4">#</th>
                <th class="py-3 pr-4">{{ t('payment.orders.amount') }}</th>
                <th class="py-3 pr-4">{{ t('common.status') }}</th>
                <th class="py-3 pr-4">{{ t('invoice.invoiceStatus') }}</th>
                <th class="py-3">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="order in requestableOrders" :key="order.id" class="border-b border-gray-50 dark:border-dark-800">
                <td class="py-3 pr-4">#{{ order.id }}</td>
                <td class="py-3 pr-4">{{ order.amount.toFixed(2) }}</td>
                <td class="py-3 pr-4">{{ order.status }}</td>
                <td class="py-3 pr-4">{{ formatInvoiceStatus(order.invoice_status || 'not_requested') }}</td>
                <td class="py-3">
                  <button class="btn btn-primary" :disabled="!profiles.length" @click="requestInvoice(order.id)">{{ t('invoice.requestInvoice') }}</button>
                </td>
              </tr>
              <tr v-if="!requestableOrders.length">
                <td colspan="5" class="py-6 text-center text-gray-500">{{ t('common.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card p-6">
        <h3 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">{{ t('invoice.records') }}</h3>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 text-left text-gray-500 dark:border-dark-700">
                <th class="py-3 pr-4">#</th>
                <th class="py-3 pr-4">{{ t('common.status') }}</th>
                <th class="py-3 pr-4">{{ t('invoice.provider') }}</th>
                <th class="py-3">{{ t('admin.users.created') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="invoice in invoices" :key="invoice.id" class="border-b border-gray-50 dark:border-dark-800">
                <td class="py-3 pr-4">#{{ invoice.id }}</td>
                <td class="py-3 pr-4">{{ formatInvoiceStatus(invoice.status) }}</td>
                <td class="py-3 pr-4">{{ formatInvoiceProvider(invoice.provider) }}</td>
                <td class="py-3">{{ formatDate(invoice.created_at) }}</td>
              </tr>
              <tr v-if="!invoices.length">
                <td colspan="4" class="py-6 text-center text-gray-500">{{ t('common.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <BaseDialog :show="showProfileDialog" :title="editingProfile ? t('common.edit') : t('common.create')" @close="showProfileDialog = false">
        <div class="grid gap-3">
          <input v-model="profileForm.title" class="input" :placeholder="t('invoice.profileTitle')" />
          <input v-model="profileForm.tax_no" class="input" :placeholder="t('invoice.taxNo')" />
          <input v-model="profileForm.email" class="input" :placeholder="t('admin.users.email')" />
          <input v-model="profileForm.phone" class="input" :placeholder="t('invoice.phone')" />
          <input v-model="profileForm.address" class="input" :placeholder="t('invoice.address')" />
          <input v-model="profileForm.bank_name" class="input" :placeholder="t('invoice.bankName')" />
          <input v-model="profileForm.bank_account" class="input" :placeholder="t('invoice.bankAccount')" />
          <Select v-model="profileForm.invoice_type" :options="invoiceTypeOptions" />
          <label class="flex items-center gap-2 text-sm">
            <input v-model="profileForm.is_default" type="checkbox" />
            {{ t('invoice.setDefault') }}
          </label>
        </div>
        <template #footer>
          <div class="flex justify-end gap-3">
            <button class="btn btn-secondary" @click="showProfileDialog = false">{{ t('common.cancel') }}</button>
            <button class="btn btn-primary" @click="saveProfile">{{ t('common.save') }}</button>
          </div>
        </template>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import { invoiceAPI, type InvoiceProfile, type InvoiceRequest } from '@/api/invoice'
import { paymentAPI } from '@/api/payment'
import type { PaymentOrder } from '@/types/payment'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const profiles = ref<InvoiceProfile[]>([])
const invoices = ref<InvoiceRequest[]>([])
const orders = ref<PaymentOrder[]>([])
const showProfileDialog = ref(false)
const editingProfile = ref<InvoiceProfile | null>(null)
const profileForm = reactive({
  title: '',
  tax_no: '',
  email: '',
  phone: '',
  address: '',
  bank_name: '',
  bank_account: '',
  invoice_type: 'personal_electronic',
  is_default: false
})

const invoiceTypeOptions = computed(() => [
  { value: 'personal_electronic', label: t('invoice.profileTypes.personal_electronic') },
  { value: 'enterprise_electronic', label: t('invoice.profileTypes.enterprise_electronic') }
])

const requestableOrders = computed(() =>
  orders.value.filter((order) => order.status === 'COMPLETED' && order.invoice_status !== 'issued' && order.invoice_status !== 'requested' && order.invoice_status !== 'processing')
)

function formatDate(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function formatInvoiceStatus(value?: string | null) {
  if (!value) return '-'
  return t(`invoice.statuses.${value}`, value)
}

function formatInvoiceProvider(value?: string | null) {
  if (!value) return '-'
  return t(`invoice.providers.${value}`, value)
}

function formatInvoiceProfileType(value?: string | null) {
  if (!value) return '-'
  return t(`invoice.profileTypes.${value}`, value)
}

function resetProfileForm() {
  Object.assign(profileForm, {
    title: '',
    tax_no: '',
    email: '',
    phone: '',
    address: '',
    bank_name: '',
    bank_account: '',
    invoice_type: 'personal_electronic',
    is_default: false
  })
}

function openCreateProfile() {
  editingProfile.value = null
  resetProfileForm()
  showProfileDialog.value = true
}

function editProfile(profile: InvoiceProfile) {
  editingProfile.value = profile
  Object.assign(profileForm, profile)
  showProfileDialog.value = true
}

async function fetchAll() {
  try {
    const [profilesRes, invoicesRes, ordersRes] = await Promise.all([
      invoiceAPI.listProfiles(),
      invoiceAPI.listMyInvoices(),
      paymentAPI.getMyOrders()
    ])
    profiles.value = profilesRes.data
    invoices.value = invoicesRes.data.items ?? []
    orders.value = ordersRes.data.items ?? []
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

async function saveProfile() {
  try {
    if (editingProfile.value) {
      await invoiceAPI.updateProfile(editingProfile.value.id, profileForm)
    } else {
      await invoiceAPI.createProfile(profileForm)
    }
    showProfileDialog.value = false
    await fetchAll()
    appStore.showSuccess(t('common.success'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

async function removeProfile(id: number) {
  try {
    await invoiceAPI.deleteProfile(id)
    await fetchAll()
    appStore.showSuccess(t('common.success'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

async function setDefault(id: number) {
  try {
    await invoiceAPI.setDefaultProfile(id)
    await fetchAll()
    appStore.showSuccess(t('common.success'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

async function requestInvoice(orderId: number) {
  const profile = profiles.value.find((item) => item.is_default) ?? profiles.value[0]
  if (!profile) {
    appStore.showError(t('invoice.profileRequired'))
    return
  }
  try {
    await invoiceAPI.createInvoice({ order_id: orderId, profile_id: profile.id })
    await fetchAll()
    appStore.showSuccess(t('common.success'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

onMounted(fetchAll)
</script>
