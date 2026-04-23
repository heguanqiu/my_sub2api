<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card p-6">
        <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('referral.title') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('referral.description') }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-secondary" :disabled="loading" @click="fetchAll">{{ t('common.refresh') }}</button>
            <button class="btn btn-primary" :disabled="loading" @click="regenerateLink">{{ t('referral.regenerate') }}</button>
          </div>
        </div>
        <div v-if="link" class="mt-5 rounded-2xl border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div class="min-w-0">
              <p class="text-xs uppercase tracking-wide text-gray-500">{{ t('referral.myLink') }}</p>
              <p class="mt-1 break-all font-mono text-sm text-gray-900 dark:text-white">{{ link.url }}</p>
              <p class="mt-2 text-xs text-gray-500">{{ t('referral.linkHint') }}</p>
              <p class="mt-2 text-xs text-gray-500">{{ t('referral.status') }}: {{ formatReferralStatus(link.status) }}</p>
            </div>
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-secondary" @click="copyLink">{{ t('common.copy') }}</button>
              <button class="btn btn-secondary" :disabled="link.status !== 'active'" @click="disableLink">{{ t('referral.disable') }}</button>
              <button class="btn btn-danger" :disabled="link.status === 'revoked'" @click="revokeLink">{{ t('referral.revoke') }}</button>
            </div>
          </div>
        </div>
      </div>

      <div class="card p-6">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('referral.invitees') }}</h3>
          <input v-model="inviteeSearch" class="input w-56" :placeholder="t('common.search')" @keyup.enter="fetchInvitees" />
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 text-left text-gray-500 dark:border-dark-700">
                <th class="py-3 pr-4">{{ t('admin.users.email') }}</th>
                <th class="py-3 pr-4">{{ t('admin.users.username') }}</th>
                <th class="py-3 pr-4">{{ t('admin.users.status') }}</th>
                <th class="py-3">{{ t('admin.users.created') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in invitees" :key="user.id" class="border-b border-gray-50 dark:border-dark-800">
                <td class="py-3 pr-4">{{ user.email }}</td>
                <td class="py-3 pr-4">{{ user.username || '-' }}</td>
                <td class="py-3 pr-4">{{ formatUserStatus(user.status) }}</td>
                <td class="py-3">{{ formatDate(user.created_at) }}</td>
              </tr>
              <tr v-if="!invitees.length">
                <td colspan="4" class="py-6 text-center text-gray-500">{{ t('common.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card p-6">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('referral.rewards') }}</h3>
          <Select v-model="rewardStatus" :options="rewardStatusOptions" class="w-48" @change="fetchRewards" />
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 text-left text-gray-500 dark:border-dark-700">
                <th class="py-3 pr-4">{{ t('referral.rewardType') }}</th>
                <th class="py-3 pr-4">{{ t('payment.orders.amount') }}</th>
                <th class="py-3 pr-4">{{ t('common.status') }}</th>
                <th class="py-3">{{ t('admin.users.created') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="reward in rewards" :key="reward.id" class="border-b border-gray-50 dark:border-dark-800">
                <td class="py-3 pr-4">{{ reward.reward_type }}</td>
                <td class="py-3 pr-4">{{ reward.reward_amount.toFixed(2) }}</td>
                <td class="py-3 pr-4">{{ formatRewardStatus(reward.status) }}</td>
                <td class="py-3">{{ formatDate(reward.created_at) }}</td>
              </tr>
              <tr v-if="!rewards.length">
                <td colspan="4" class="py-6 text-center text-gray-500">{{ t('common.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import { referralAPI, type InviteLink, type InviteReward } from '@/api/referral'
import { useAppStore } from '@/stores'
import type { User } from '@/types'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const link = ref<InviteLink | null>(null)
const invitees = ref<User[]>([])
const rewards = ref<InviteReward[]>([])
const inviteeSearch = ref('')
const rewardStatus = ref('')

const rewardStatusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'granted', label: 'granted' },
  { value: 'reversed', label: 'reversed' },
  { value: 'pending', label: 'pending' }
])

function formatDate(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function formatUserStatus(value?: string | null) {
  if (!value) return '-'
  if (value === 'active') return t('common.active')
  if (value === 'disabled') return t('admin.users.disabled')
  return value
}

function formatReferralStatus(value?: string | null) {
  if (!value) return '-'
  return t(`referral.statuses.${value}`, value)
}

function formatRewardStatus(value?: string | null) {
  if (!value) return '-'
  return t(`referral.rewardStatuses.${value}`, value)
}

async function fetchLink() {
  const res = await referralAPI.getMyLink()
  link.value = res.data
}

async function fetchInvitees() {
  const res = await referralAPI.getMyInvitees({ search: inviteeSearch.value || undefined })
  invitees.value = res.data.items ?? []
}

async function fetchRewards() {
  const res = await referralAPI.getMyRewards({ status: rewardStatus.value || undefined })
  rewards.value = res.data.items ?? []
}

async function fetchAll() {
  loading.value = true
  try {
    await Promise.all([fetchLink(), fetchInvitees(), fetchRewards()])
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  } finally {
    loading.value = false
  }
}

async function regenerateLink() {
  try {
    const res = await referralAPI.regenerateMyLink()
    link.value = res.data
    appStore.showSuccess(t('common.success'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

async function disableLink() {
  try {
    const res = await referralAPI.disableMyLink()
    link.value = res.data
    appStore.showSuccess(t('common.success'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

async function revokeLink() {
  try {
    const res = await referralAPI.revokeMyLink()
    link.value = res.data
    appStore.showSuccess(t('common.success'))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'common', t('common.error')))
  }
}

async function copyLink() {
  if (!link.value?.url) return
  await navigator.clipboard.writeText(link.value.url)
  appStore.showSuccess(t('common.success'))
}

onMounted(fetchAll)
</script>
