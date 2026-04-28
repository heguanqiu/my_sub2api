<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-6">
        <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('nav.salesDashboard') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('sales.ordersDescription') }}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in rangeOptions"
              :key="option.value"
              type="button"
              :class="selectedRange === option.value ? 'btn btn-primary' : 'btn btn-secondary'"
              :disabled="loading"
              @click="setRange(option.value)"
            >
              {{ option.label }}
            </button>
            <button class="btn btn-secondary" @click="router.push('/sales/customers')">
              {{ t('nav.salesCustomers') }}
            </button>
            <button class="btn btn-secondary" @click="router.push('/sales/orders')">
              {{ t('nav.salesOrders') }}
            </button>
            <button class="btn btn-primary" @click="router.push('/sales/referral')">
              {{ t('nav.salesReferral') }}
            </button>
          </div>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div class="card p-6" v-for="item in cards" :key="item.label">
          <p class="text-sm text-gray-500">{{ item.label }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { salesAPI, type SalesDashboardRange, type SalesDashboardSummary } from '@/api/sales'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const stats = ref<SalesDashboardSummary | null>(null)
const selectedRange = ref<SalesDashboardRange>('today')
const loading = ref(false)

const rangeOptions = computed<Array<{ value: SalesDashboardRange; label: string }>>(() => [
  { value: 'today', label: t('sales.rangeToday') },
  { value: '7d', label: t('sales.range7d') },
  { value: '30d', label: t('sales.range30d') }
])

const cards = computed(() => [
  { label: t('sales.totalCustomers'), value: stats.value?.total_customers ?? 0 },
  { label: t('sales.totalOrders'), value: stats.value?.total_orders ?? 0 },
  { label: t('sales.completedOrders'), value: stats.value?.completed_orders ?? 0 },
  {
    label: t('sales.totalOrderAmount'),
    value: `¥${(stats.value?.total_order_amount ?? 0).toFixed(2)}`
  }
])

async function loadDashboard() {
  loading.value = true
  try {
    const res = await salesAPI.getDashboard({ range: selectedRange.value })
    stats.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  } finally {
    loading.value = false
  }
}

function setRange(range: SalesDashboardRange) {
  if (selectedRange.value === range) return
  selectedRange.value = range
  void loadDashboard()
}

onMounted(loadDashboard)
</script>
