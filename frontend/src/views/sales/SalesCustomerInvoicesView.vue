<template>
  <AppLayout>
    <div class="card p-6">
      <h2 class="mb-4 text-xl font-semibold text-gray-900 dark:text-white">{{ t('sales.invoices') }}</h2>
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
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { salesAPI } from '@/api/sales'
import type { InvoiceRequest } from '@/api/invoice'
import { useAppStore } from '@/stores'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const invoices = ref<InvoiceRequest[]>([])

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

onMounted(async () => {
  try {
    const res = await salesAPI.getCustomerInvoices(Number(route.params.id))
    invoices.value = res.data.items ?? []
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
})
</script>
