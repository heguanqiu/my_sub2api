<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex gap-3">
          <input v-model="search" class="input flex-1" :placeholder="t('common.search')" @keyup.enter="fetchCustomers" />
          <button class="btn btn-secondary" @click="fetchCustomers">{{ t('common.refresh') }}</button>
        </div>
      </div>
      <div class="card p-6">
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 text-left text-gray-500 dark:border-dark-700">
                <th class="py-3 pr-4">{{ t('admin.users.email') }}</th>
                <th class="py-3 pr-4">{{ t('admin.users.username') }}</th>
                <th class="py-3 pr-4">{{ t('sales.totalOrders') }}</th>
                <th class="py-3 pr-4">{{ t('sales.completedOrderAmount') }}</th>
                <th class="py-3">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in customers" :key="item.user.id" class="border-b border-gray-50 dark:border-dark-800">
                <td class="py-3 pr-4">{{ item.user.email }}</td>
                <td class="py-3 pr-4">{{ item.user.username || '-' }}</td>
                <td class="py-3 pr-4">{{ item.total_orders }}</td>
                <td class="py-3 pr-4">{{ item.completed_order_amount.toFixed(2) }}</td>
                <td class="py-3">
                  <div class="flex gap-2">
                    <button class="btn btn-secondary" @click="router.push(`/sales/customers/${item.user.id}/orders`)">{{ t('sales.orders') }}</button>
                  </div>
                </td>
              </tr>
              <tr v-if="!customers.length">
                <td colspan="5" class="py-6 text-center text-gray-500">{{ t('common.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { salesAPI, type SalesCustomerSummary } from '@/api/sales'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const customers = ref<SalesCustomerSummary[]>([])
const search = ref('')

async function fetchCustomers() {
  try {
    const res = await salesAPI.listCustomers({ search: search.value || undefined })
    customers.value = res.data.items ?? []
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

onMounted(fetchCustomers)
</script>
