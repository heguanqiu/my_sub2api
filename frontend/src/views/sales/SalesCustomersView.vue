<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-4">
        <div class="flex gap-3">
          <input
            v-model="search"
            class="input flex-1"
            :placeholder="t('common.search')"
            @keyup.enter="handleSearch"
          />
          <button class="btn btn-secondary" :disabled="loading" @click="handleSearch">
            {{ t('common.refresh') }}
          </button>
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
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import { salesAPI, type SalesCustomerSummary } from '@/api/sales'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const loading = ref(false)
const customers = ref<SalesCustomerSummary[]>([])
const search = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

async function fetchCustomers() {
  loading.value = true
  try {
    const res = await salesAPI.listCustomers({
      page: pagination.page,
      page_size: pagination.page_size,
      search: search.value || undefined
    })
    customers.value = res.data.items ?? []
    pagination.total = res.data.total ?? 0
    pagination.page = res.data.page ?? pagination.page
    pagination.page_size = res.data.page_size ?? pagination.page_size
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  fetchCustomers()
}

function handlePageChange(page: number) {
  pagination.page = page
  fetchCustomers()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  fetchCustomers()
}

onMounted(fetchCustomers)
</script>
