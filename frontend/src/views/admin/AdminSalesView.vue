<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
          <div class="min-w-0 flex-1">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('admin.sales.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.sales.description') }}
            </p>
          </div>
          <div class="flex flex-col gap-3 lg:flex-row lg:items-end">
            <label class="block w-full lg:w-80">
              <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.sales.salesAccount') }}
              </span>
              <Select
                v-model="selectedSalesId"
                :options="salesOptions"
                :disabled="salesLoading"
                @change="handleSalesChange"
              />
            </label>
            <label class="block w-full lg:w-52">
              <span class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.sales.range') }}
              </span>
              <Select
                v-model="selectedRange"
                :options="rangeOptions"
                :disabled="dashboardLoading || !selectedSalesId"
                @change="loadDashboard"
              />
            </label>
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="refreshing || !selectedSalesId"
              :title="t('common.refresh')"
              @click="refreshAll"
            >
              <Icon name="refresh" size="md" :class="refreshing ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </div>

      <div v-if="!selectedSalesId" class="card p-12 text-center">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ salesLoading ? t('common.loading') : t('admin.sales.emptySales') }}
        </p>
      </div>

      <template v-else>
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div v-for="item in summaryCards" :key="item.label" class="card p-6">
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ item.label }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
          </div>
        </div>

        <div class="card p-4">
          <div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
            <div class="flex flex-wrap gap-2">
              <button
                v-for="tab in tabs"
                :key="tab.value"
                type="button"
                :class="activeTab === tab.value ? 'btn btn-primary' : 'btn btn-secondary'"
                @click="setActiveTab(tab.value)"
              >
                {{ tab.label }}
              </button>
            </div>

            <div v-if="activeTab === 'customers'" class="flex flex-col gap-3 sm:flex-row sm:items-center">
              <input
                v-model="customerSearch"
                class="input w-full sm:w-72"
                :placeholder="t('admin.sales.searchCustomers')"
                @keyup.enter="loadCustomers"
              />
              <Select v-model="customerStatus" :options="userStatusOptions" class="w-full sm:w-40" @change="handleCustomerFilterChange" />
              <button type="button" class="btn btn-secondary" :disabled="customersLoading" @click="loadCustomers">
                {{ t('common.search') }}
              </button>
            </div>

            <div v-else class="flex flex-col gap-3 sm:flex-row sm:items-center">
              <Select v-model="orderStatus" :options="orderStatusOptions" class="w-full sm:w-44" @change="handleOrderFilterChange" />
              <Select
                v-if="activeTab === 'orders'"
                v-model="paymentType"
                :options="paymentTypeOptions"
                class="w-full sm:w-44"
                @change="handleOrderFilterChange"
              />
              <button type="button" class="btn btn-secondary" :disabled="ordersLoading" @click="loadCurrentOrders">
                {{ t('common.refresh') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="activeTab === 'customers'" class="space-y-4">
          <div class="card overflow-hidden">
            <DataTable :columns="customerColumns" :data="customers" :loading="customersLoading">
              <template #cell-user="{ value }">
                <div class="text-sm">
                  <div class="font-medium text-gray-900 dark:text-white">{{ value.email }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ value.username || '-' }} #{{ value.id }}</div>
                </div>
              </template>
              <template #cell-completed_order_amount="{ value }">
                <span class="font-medium text-gray-900 dark:text-white">¥{{ Number(value || 0).toFixed(2) }}</span>
              </template>
              <template #cell-actions="{ row }">
                <button
                  type="button"
                  class="inline-flex items-center rounded-md px-2 py-1 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
                  @click="openCustomerOrders(row.user)"
                >
                  {{ t('sales.viewCustomerOrders') }}
                </button>
              </template>
            </DataTable>
          </div>
          <Pagination
            v-if="customerPagination.total > 0"
            :page="customerPagination.page"
            :total="customerPagination.total"
            :page-size="customerPagination.page_size"
            @update:page="handleCustomerPageChange"
            @update:pageSize="handleCustomerPageSizeChange"
          />
        </div>

        <div v-else class="space-y-4">
          <div v-if="activeTab === 'customerOrders' && selectedCustomer" class="card p-4">
            <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div class="text-sm text-gray-600 dark:text-gray-300">
                {{ t('admin.sales.currentCustomer') }}:
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ selectedCustomer.email }}
                </span>
                <span class="text-gray-500 dark:text-gray-400">#{{ selectedCustomer.id }}</span>
              </div>
              <button type="button" class="btn btn-secondary" @click="setActiveTab('customers')">
                {{ t('sales.backToCustomers') }}
              </button>
            </div>
          </div>

          <OrderTable :orders="visibleOrders" :loading="ordersLoading" show-user>
            <template #actions="{ row }">
              <button
                v-if="activeTab === 'orders'"
                type="button"
                class="inline-flex items-center rounded-md px-2 py-1 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
                @click="openCustomerOrdersByID(row.user_id)"
              >
                {{ t('sales.viewCustomerOrders') }}
              </button>
            </template>
          </OrderTable>
          <Pagination
            v-if="orderPagination.total > 0"
            :page="orderPagination.page"
            :total="orderPagination.total"
            :page-size="orderPagination.page_size"
            @update:page="handleOrderPageChange"
            @update:pageSize="handleOrderPageSizeChange"
          />
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'
import { adminAPI } from '@/api/admin'
import { adminPaymentAPI } from '@/api/admin/payment'
import type { SalesCustomerSummary, SalesDashboardRange, SalesDashboardSummary } from '@/api/sales'
import type { AdminUser, User } from '@/types'
import type { Column } from '@/components/common/types'
import type { PaymentOrder } from '@/types/payment'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

type AdminSalesTab = 'customers' | 'orders' | 'customerOrders'

const { t } = useI18n()
const appStore = useAppStore()

const salesUsers = ref<AdminUser[]>([])
const selectedSalesId = ref<number | ''>('')
const selectedRange = ref<SalesDashboardRange>('today')
const dashboard = ref<SalesDashboardSummary | null>(null)
const customers = ref<SalesCustomerSummary[]>([])
const visibleOrders = ref<PaymentOrder[]>([])
const selectedCustomer = ref<User | null>(null)
const activeTab = ref<AdminSalesTab>('customers')

const salesLoading = ref(false)
const dashboardLoading = ref(false)
const customersLoading = ref(false)
const ordersLoading = ref(false)

const customerSearch = ref('')
const customerStatus = ref('')
const orderStatus = ref('')
const paymentType = ref('')

const customerPagination = reactive({ page: 1, page_size: 20, total: 0 })
const orderPagination = reactive({ page: 1, page_size: 20, total: 0 })

const currentSalesId = computed(() => (typeof selectedSalesId.value === 'number' ? selectedSalesId.value : 0))
const refreshing = computed(() => salesLoading.value || dashboardLoading.value || customersLoading.value || ordersLoading.value)

const salesOptions = computed(() => {
  const options = salesUsers.value.map((user) => ({
    value: user.id,
    label: `${user.email}${user.username ? ` (${user.username})` : ''} #${user.id}`
  }))
  return options.length > 0 ? options : [{ value: '', label: t('admin.sales.noSalesOptions') }]
})

const rangeOptions = computed(() => [
  { value: 'today', label: t('sales.rangeToday') },
  { value: '7d', label: t('sales.range7d') },
  { value: '30d', label: t('sales.range30d') }
])

const userStatusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'active', label: t('admin.users.active') },
  { value: 'disabled', label: t('admin.users.disabled') }
])

const orderStatusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'PAID', label: t('payment.status.paid') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
  { value: 'REFUND_REQUESTED', label: t('payment.status.refund_requested') }
])

const paymentTypeOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'alipay', label: t('payment.methods.alipay') },
  { value: 'alipay_direct', label: t('payment.methods.alipay_direct') },
  { value: 'wxpay', label: t('payment.methods.wxpay') },
  { value: 'wxpay_direct', label: t('payment.methods.wxpay_direct') },
  { value: 'stripe', label: t('payment.methods.stripe') },
  { value: 'easypay', label: t('payment.methods.easypay') },
  { value: 'airwallex', label: t('payment.methods.airwallex') }
])

const tabs = computed(() => [
  { value: 'customers' as const, label: t('admin.sales.customersTab') },
  { value: 'orders' as const, label: t('admin.sales.ordersTab') },
  ...(selectedCustomer.value ? [{ value: 'customerOrders' as const, label: t('admin.sales.customerOrdersTab') }] : [])
])

const summaryCards = computed(() => [
  { label: t('sales.totalCustomers'), value: dashboard.value?.total_customers ?? 0 },
  { label: t('sales.totalOrders'), value: dashboard.value?.total_orders ?? 0 },
  { label: t('sales.completedOrders'), value: dashboard.value?.completed_orders ?? 0 },
  { label: t('sales.totalOrderAmount'), value: `¥${(dashboard.value?.total_order_amount ?? 0).toFixed(2)}` }
])

const customerColumns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.sales.customer') },
  { key: 'total_orders', label: t('sales.totalOrders') },
  { key: 'completed_order_amount', label: t('sales.completedOrderAmount') },
  { key: 'actions', label: t('common.actions') }
])

async function loadSalesUsers() {
  salesLoading.value = true
  try {
    const pageSize = 100
    const firstPage = await adminAPI.users.list(1, pageSize, { role: 'sales' })
    const allSales = [...(firstPage.items ?? [])]
    for (let page = 2; page <= (firstPage.pages ?? 1); page += 1) {
      const nextPage = await adminAPI.users.list(page, pageSize, { role: 'sales' })
      allSales.push(...(nextPage.items ?? []))
    }
    salesUsers.value = allSales
    if (!selectedSalesId.value && salesUsers.value.length > 0) {
      selectedSalesId.value = salesUsers.value[0].id
      await refreshAll()
    }
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  } finally {
    salesLoading.value = false
  }
}

async function loadDashboard() {
  if (!currentSalesId.value) return
  dashboardLoading.value = true
  try {
    const res = await adminPaymentAPI.getSalesDashboard(currentSalesId.value, { range: selectedRange.value })
    dashboard.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  } finally {
    dashboardLoading.value = false
  }
}

async function loadCustomers() {
  if (!currentSalesId.value) return
  customersLoading.value = true
  try {
    const res = await adminPaymentAPI.getSalesCustomers(currentSalesId.value, {
      page: customerPagination.page,
      page_size: customerPagination.page_size,
      search: customerSearch.value || undefined,
      status: customerStatus.value || undefined
    })
    customers.value = res.data.items ?? []
    customerPagination.total = res.data.total ?? 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  } finally {
    customersLoading.value = false
  }
}

async function loadOrders() {
  if (!currentSalesId.value) return
  ordersLoading.value = true
  try {
    const res = await adminPaymentAPI.getSalesOrders(currentSalesId.value, {
      page: orderPagination.page,
      page_size: orderPagination.page_size,
      status: orderStatus.value || undefined,
      payment_type: paymentType.value || undefined
    })
    visibleOrders.value = res.data.items ?? []
    orderPagination.total = res.data.total ?? 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    ordersLoading.value = false
  }
}

async function loadCustomerOrders() {
  if (!currentSalesId.value || !selectedCustomer.value) return
  ordersLoading.value = true
  try {
    const res = await adminPaymentAPI.getSalesCustomerOrders(currentSalesId.value, selectedCustomer.value.id, {
      page: orderPagination.page,
      page_size: orderPagination.page_size,
      status: orderStatus.value || undefined
    })
    visibleOrders.value = res.data.items ?? []
    orderPagination.total = res.data.total ?? 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    ordersLoading.value = false
  }
}

async function loadCurrentOrders() {
  if (activeTab.value === 'customerOrders') {
    await loadCustomerOrders()
    return
  }
  await loadOrders()
}

async function refreshAll() {
  if (!currentSalesId.value) return
  await Promise.all([loadDashboard(), loadCustomers()])
  if (activeTab.value !== 'customers') {
    await loadCurrentOrders()
  }
}

async function handleSalesChange() {
  dashboard.value = null
  customers.value = []
  visibleOrders.value = []
  selectedCustomer.value = null
  activeTab.value = 'customers'
  customerPagination.page = 1
  orderPagination.page = 1
  orderPagination.total = 0
  await refreshAll()
}

function setActiveTab(tab: AdminSalesTab) {
  if (activeTab.value === tab) return
  activeTab.value = tab
  orderPagination.page = 1
  orderPagination.total = 0
  if (tab === 'customers') {
    void loadCustomers()
    return
  }
  void loadCurrentOrders()
}

function handleCustomerFilterChange() {
  customerPagination.page = 1
  void loadCustomers()
}

function handleOrderFilterChange() {
  orderPagination.page = 1
  void loadCurrentOrders()
}

function handleCustomerPageChange(page: number) {
  customerPagination.page = page
  void loadCustomers()
}

function handleCustomerPageSizeChange(size: number) {
  customerPagination.page_size = size
  customerPagination.page = 1
  void loadCustomers()
}

function handleOrderPageChange(page: number) {
  orderPagination.page = page
  void loadCurrentOrders()
}

function handleOrderPageSizeChange(size: number) {
  orderPagination.page_size = size
  orderPagination.page = 1
  void loadCurrentOrders()
}

function openCustomerOrders(user: User) {
  selectedCustomer.value = user
  activeTab.value = 'customerOrders'
  orderPagination.page = 1
  orderPagination.total = 0
  void loadCustomerOrders()
}

async function openCustomerOrdersByID(customerID: number) {
  if (!currentSalesId.value) return
  const cached = customers.value.find((item) => item.user.id === customerID)?.user
  if (cached) {
    openCustomerOrders(cached)
    return
  }
  try {
    const res = await adminPaymentAPI.getSalesCustomer(currentSalesId.value, customerID)
    openCustomerOrders(res.data)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

onMounted(() => {
  void loadSalesUsers()
})
</script>
