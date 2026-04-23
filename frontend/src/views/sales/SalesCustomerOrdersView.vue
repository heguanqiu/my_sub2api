<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
          <div>
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
                {{ t('sales.customerRechargeRecords') }}
              </h2>
              <span
                v-if="customer"
                class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
              >
                {{ customer.email }}
              </span>
            </div>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('sales.customerOrdersDescription') }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <Select
              v-model="statusFilter"
              :options="statusFilters"
              class="w-40"
              @change="handleFilterChange"
            />
            <button
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="fetchOrders"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button class="btn btn-secondary" @click="router.push('/sales/customers')">
              {{ t('sales.backToCustomers') }}
            </button>
          </div>
        </div>
      </div>

      <OrderTable :orders="orders" :loading="loading">
        <template #actions>
          <button
            class="inline-flex items-center rounded-md px-2 py-1 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
            @click="router.push(`/sales/customers/${customerId}/invoices`)"
          >
            {{ t('sales.invoices') }}
          </button>
        </template>
      </OrderTable>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'
import { salesAPI } from '@/api/sales'
import type { User } from '@/types'
import type { PaymentOrder } from '@/types/payment'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const customerId = computed(() => Number(route.params.id))
const loading = ref(false)
const customer = ref<User | null>(null)
const orders = ref<PaymentOrder[]>([])
const statusFilter = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') }
])

async function fetchCustomer() {
  try {
    const res = await salesAPI.getCustomer(customerId.value)
    customer.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

async function fetchOrders() {
  loading.value = true
  try {
    const res = await salesAPI.getCustomerOrders(customerId.value, {
      page: pagination.page,
      page_size: pagination.page_size,
      status: statusFilter.value || undefined
    })
    orders.value = res.data.items ?? []
    pagination.total = res.data.total ?? 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handleFilterChange() {
  pagination.page = 1
  fetchOrders()
}

function handlePageChange(page: number) {
  pagination.page = page
  fetchOrders()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  fetchOrders()
}

onMounted(() => {
  fetchCustomer()
  fetchOrders()
})

watch(
  () => route.params.id,
  () => {
    pagination.page = 1
    fetchCustomer()
    fetchOrders()
  }
)
</script>
