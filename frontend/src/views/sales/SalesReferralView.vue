<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 p-6 dark:border-dark-700">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.24em] text-primary-600 dark:text-primary-400">
                {{ t('nav.salesReferral') }}
              </p>
              <h2 class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                {{ t('sales.referralTitle') }}
              </h2>
              <p class="mt-2 max-w-3xl text-sm text-gray-500 dark:text-gray-400">
                {{ t('sales.referralDescription') }}
              </p>
            </div>
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-secondary" :disabled="loading" @click="fetchAll">
                {{ t('common.refresh') }}
              </button>
              <button class="btn btn-primary" :disabled="loading" @click="regenerateLink">
                {{ t('referral.regenerate') }}
              </button>
            </div>
          </div>

          <div class="mt-6 grid gap-3 md:grid-cols-2">
            <div class="rounded-2xl border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
              <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                {{ t('referral.status') }}
              </p>
              <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                {{ formatReferralStatus(link?.status) }}
              </p>
            </div>
            <div class="rounded-2xl border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
              <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                {{ t('referral.invitees') }}
              </p>
              <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">
                {{ inviteeTotal }}
              </p>
            </div>
          </div>
        </div>

        <div class="p-6">
          <div
            v-if="link"
            class="rounded-2xl border border-primary-100 bg-primary-50/70 p-5 dark:border-primary-900/40 dark:bg-primary-900/10"
          >
            <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
              <div class="min-w-0">
                <p class="text-xs font-semibold uppercase tracking-wide text-primary-700 dark:text-primary-300">
                  {{ t('sales.exclusiveRegisterLink') }}
                </p>
                <p class="mt-3 break-all rounded-xl bg-white/80 px-4 py-3 font-mono text-sm text-gray-900 shadow-sm dark:bg-dark-900/60 dark:text-white">
                  {{ link.url }}
                </p>
                <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">
                  {{ t('sales.referralLinkHint') }}
                </p>
              </div>
              <div class="flex flex-wrap gap-2">
                <button class="btn btn-primary" @click="copyLink">{{ t('common.copy') }}</button>
                <button class="btn btn-secondary" @click="openRegisterPage">
                  {{ t('sales.openRegisterPage') }}
                </button>
                <button class="btn btn-secondary" :disabled="link.status !== 'active'" @click="disableLink">
                  {{ t('referral.disable') }}
                </button>
                <button class="btn btn-danger" :disabled="link.status === 'revoked'" @click="revokeLink">
                  {{ t('referral.revoke') }}
                </button>
              </div>
            </div>
          </div>
          <div
            v-else
            class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 p-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
          >
            {{ t('common.noData') }}
          </div>
        </div>
      </div>

      <div class="card p-6">
        <div class="mb-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('referral.invitees') }}
            </h3>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('sales.inviteesDescription') }}
            </p>
          </div>
          <input
            v-model="inviteeSearch"
            class="input w-full md:w-72"
            :placeholder="t('common.search')"
            @keyup.enter="fetchInvitees"
          />
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
              <tr
                v-for="user in invitees"
                :key="user.id"
                class="border-b border-gray-50 dark:border-dark-800"
              >
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
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { referralAPI, type InviteLink } from '@/api/referral'
import { useAppStore } from '@/stores'
import type { User } from '@/types'
import { extractI18nErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const link = ref<InviteLink | null>(null)
const invitees = ref<User[]>([])
const inviteeTotal = ref(0)
const inviteeSearch = ref('')

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

async function fetchLink() {
  const res = await referralAPI.getMyLink()
  link.value = res.data
}

async function fetchInvitees() {
  const res = await referralAPI.getMyInvitees({ search: inviteeSearch.value || undefined })
  invitees.value = res.data.items ?? []
  inviteeTotal.value = res.data.total ?? invitees.value.length
}

async function fetchAll() {
  loading.value = true
  try {
    await Promise.all([fetchLink(), fetchInvitees()])
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
  try {
    await navigator.clipboard.writeText(link.value.url)
    appStore.showSuccess(t('common.success'))
  } catch {
    appStore.showError(t('common.error'))
  }
}

function openRegisterPage() {
  if (!link.value?.url) return
  window.open(link.value.url, '_blank', 'noopener,noreferrer')
}

onMounted(fetchAll)
</script>
