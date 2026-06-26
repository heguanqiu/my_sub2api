<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
          <div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('sales.rechargeRecords') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('sales.ordersDescription') }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <label class="block w-40">
              <span class="sr-only">{{ t('dates.startDate') }}</span>
              <input
                v-model="startDate"
                type="date"
                class="input w-full"
                :aria-label="t('dates.startDate')"
                @change="handleFilterChange"
              />
            </label>
            <label class="block w-40">
              <span class="sr-only">{{ t('dates.endDate') }}</span>
              <input
                v-model="endDate"
                type="date"
                class="input w-full"
                :aria-label="t('dates.endDate')"
                @change="handleFilterChange"
              />
            </label>
            <Select
              v-model="statusFilter"
              :options="statusFilters"
              class="w-40"
              @change="handleFilterChange"
            />
            <Select
              v-model="paymentTypeFilter"
              :options="paymentTypeFilters"
              class="w-44"
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
          </div>
        </div>
      </div>

      <OrderTable :orders="orders" :loading="loading">
        <template #actions="{ row }">
          <button
            class="inline-flex items-center rounded-md px-2 py-1 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/20"
            @click="router.push(`/sales/customers/${row.user_id}/orders`)"
          >
            {{ t('sales.viewCustomerOrders') }}
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
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'
import { salesAPI } from '@/api/sales'
import type { PaymentOrder } from '@/types/payment'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const orders = ref<PaymentOrder[]>([])
const statusFilter = ref('')
const paymentTypeFilter = ref('')
const startDate = ref('')
const endDate = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') }
])

const paymentTypeFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'alipay', label: t('payment.methods.alipay') },
  { value: 'alipay_direct', label: t('payment.methods.alipay_direct') },
  { value: 'wxpay', label: t('payment.methods.wxpay') },
  { value: 'wxpay_direct', label: t('payment.methods.wxpay_direct') },
  { value: 'stripe', label: t('payment.methods.stripe') },
  { value: 'easypay', label: t('payment.methods.easypay') }
])

async function fetchOrders() {
  loading.value = true
  try {
    const res = await salesAPI.getOrders({
      page: pagination.page,
      page_size: pagination.page_size,
      status: statusFilter.value || undefined,
      payment_type: paymentTypeFilter.value || undefined,
      start_date: startDate.value || undefined,
      end_date: endDate.value || undefined
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

onMounted(fetchOrders)
</script>
