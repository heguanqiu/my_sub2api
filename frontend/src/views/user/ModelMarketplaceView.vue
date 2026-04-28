<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-96">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('modelMarketplace.searchPlaceholder')"
                class="input pl-10"
              />
            </div>
          </div>

          <button
            class="btn btn-secondary"
            :disabled="loading"
            :title="t('common.refresh', 'Refresh')"
            @click="loadMarketplace"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <div v-if="loading" class="card py-12 text-center">
          <Icon name="refresh" size="lg" class="inline-block animate-spin text-gray-400" />
        </div>

        <div v-else-if="filteredVendors.length === 0" class="card py-12 text-center">
          <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('modelMarketplace.empty') }}
          </p>
        </div>

        <div v-else class="space-y-6">
          <section v-for="vendor in filteredVendors" :key="vendor.name" class="space-y-3">
            <div class="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ vendor.name }}
                </h2>
                <p v-if="vendor.description" class="text-sm text-gray-500 dark:text-gray-400">
                  {{ vendor.description }}
                </p>
              </div>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('modelMarketplace.groupCount', { count: vendor.groups.length }) }}
              </span>
            </div>

            <div
              v-for="groupSection in vendor.groups"
              :key="groupSection.key"
              class="card overflow-hidden"
            >
              <div class="flex flex-col gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700 lg:flex-row lg:items-center lg:justify-between">
                <div class="flex min-w-0 flex-wrap items-center gap-2">
                  <GroupBadge
                    :name="groupSection.group.name"
                    :platform="groupSection.group.platform as GroupPlatform"
                    :subscription-type="groupSection.group.subscription_type as SubscriptionType"
                    :rate-multiplier="groupSection.group.rate_multiplier"
                    :user-rate-multiplier="userGroupRates[groupSection.group.id] ?? null"
                    always-show-rate
                  />
                  <span
                    :class="[
                      'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                      platformBadgeClass(groupSection.platform),
                    ]"
                  >
                    <PlatformIcon :platform="groupSection.platform as GroupPlatform" size="xs" />
                    {{ groupSection.platform }}
                  </span>
                  <span
                    class="rounded-md px-2 py-0.5 text-[11px] font-medium"
                    :class="groupSection.group.is_exclusive ? 'bg-purple-50 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                  >
                    {{ groupSection.group.is_exclusive ? t('modelMarketplace.exclusive') : t('modelMarketplace.public') }}
                  </span>
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('modelMarketplace.effectiveRate') }}:
                  <span class="font-mono text-gray-700 dark:text-gray-200">
                    x{{ formatRate(effectiveRate(groupSection.group)) }}
                  </span>
                </div>
              </div>

              <div class="overflow-x-auto">
                <table class="min-w-full table-fixed text-sm">
                  <thead>
                    <tr class="border-b border-gray-100 bg-gray-50/50 text-left text-xs font-medium uppercase text-gray-500 dark:border-dark-700 dark:bg-dark-800/50 dark:text-gray-400">
                      <th class="w-[30%] px-4 py-3">{{ t('modelMarketplace.columns.model') }}</th>
                      <th class="w-[14%] px-4 py-3">{{ t('modelMarketplace.columns.billingMode') }}</th>
                      <th class="w-[17%] px-4 py-3">{{ t('modelMarketplace.columns.inputOutput') }}</th>
                      <th class="w-[17%] px-4 py-3">{{ t('modelMarketplace.columns.cache') }}</th>
                      <th class="w-[14%] px-4 py-3">{{ t('modelMarketplace.columns.extra') }}</th>
                      <th class="w-[8%] px-4 py-3">{{ t('modelMarketplace.columns.tiers') }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
                    <tr
                      v-for="model in groupSection.models"
                      :key="`${groupSection.key}-${model.name}`"
                      class="transition-colors hover:bg-gray-50/50 dark:hover:bg-dark-800/40"
                    >
                      <td class="px-4 py-3 align-top">
                        <SupportedModelChip
                          :model="model"
                          pricing-key-prefix="modelMarketplace.pricing"
                          :no-pricing-label="t('modelMarketplace.noPricing')"
                          :show-platform="false"
                          :platform-hint="groupSection.platform"
                        />
                      </td>
                      <td class="px-4 py-3 align-top text-gray-700 dark:text-gray-300">
                        {{ billingModeLabel(model) }}
                      </td>
                      <td class="px-4 py-3 align-top font-mono text-xs text-gray-700 dark:text-gray-300">
                        <div>{{ t('modelMarketplace.short.input') }} {{ tokenPrice(model.pricing?.input_price ?? null) }}</div>
                        <div>{{ t('modelMarketplace.short.output') }} {{ tokenPrice(model.pricing?.output_price ?? null) }}</div>
                      </td>
                      <td class="px-4 py-3 align-top font-mono text-xs text-gray-700 dark:text-gray-300">
                        <div>{{ t('modelMarketplace.short.write') }} {{ tokenPrice(model.pricing?.cache_write_price ?? null) }}</div>
                        <div>{{ t('modelMarketplace.short.read') }} {{ tokenPrice(model.pricing?.cache_read_price ?? null) }}</div>
                      </td>
                      <td class="px-4 py-3 align-top font-mono text-xs text-gray-700 dark:text-gray-300">
                        <div v-if="model.pricing?.per_request_price != null">
                          {{ requestPrice(model.pricing.per_request_price) }}
                        </div>
                        <div v-else-if="model.pricing?.image_output_price != null">
                          {{ imagePrice(model) }}
                        </div>
                        <span v-else>-</span>
                      </td>
                      <td class="px-4 py-3 align-top text-xs text-gray-500 dark:text-gray-400">
                        <span v-if="tierCount(model) > 0">
                          {{ tierCount(model) }}
                        </span>
                        <span v-else>-</span>
                      </td>
                    </tr>
                    <tr v-if="groupSection.models.length === 0">
                      <td colspan="6" class="px-4 py-8 text-center text-sm text-gray-500">
                        {{ t('modelMarketplace.noModels') }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import SupportedModelChip from '@/components/channels/SupportedModelChip.vue'
import userChannelsAPI, {
  type UserAvailableChannel,
  type UserAvailableGroup,
  type UserSupportedModel,
} from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { platformBadgeClass } from '@/utils/platformColors'
import { formatScaled } from '@/utils/pricing'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
} from '@/constants/channel'
import type { GroupPlatform, SubscriptionType } from '@/types'

interface MarketplaceGroupSection {
  key: string
  platform: string
  group: UserAvailableGroup
  models: UserSupportedModel[]
}

interface MarketplaceVendor {
  name: string
  description: string
  groups: MarketplaceGroupSection[]
}

const { t } = useI18n()
const appStore = useAppStore()

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')

const perMillionScale = 1_000_000

const vendors = computed<MarketplaceVendor[]>(() =>
  channels.value
    .map((channel) => {
      const groups = channel.platforms.flatMap((section) =>
        section.groups.map((group) => ({
          key: `${channel.name}-${section.platform}-${group.id}`,
          platform: section.platform,
          group,
          models: section.supported_models,
        })),
      )
      return {
        name: channel.name,
        description: channel.description,
        groups,
      }
    })
    .filter((vendor) => vendor.groups.length > 0),
)

const filteredVendors = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return vendors.value

  return vendors.value
    .map((vendor) => {
      const vendorHit =
        vendor.name.toLowerCase().includes(q) ||
        (vendor.description || '').toLowerCase().includes(q)
      if (vendorHit) return vendor

      const groups = vendor.groups
        .map((groupSection) => {
          const groupHit =
            groupSection.platform.toLowerCase().includes(q) ||
            groupSection.group.name.toLowerCase().includes(q)
          if (groupHit) return groupSection
          const models = groupSection.models.filter((model) =>
            model.name.toLowerCase().includes(q),
          )
          return models.length > 0 ? { ...groupSection, models } : null
        })
        .filter((groupSection): groupSection is MarketplaceGroupSection => groupSection !== null)

      return groups.length > 0 ? { ...vendor, groups } : null
    })
    .filter((vendor): vendor is MarketplaceVendor => vendor !== null)
})

async function loadMarketplace() {
  loading.value = true
  try {
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

function effectiveRate(group: UserAvailableGroup): number {
  return userGroupRates.value[group.id] ?? group.rate_multiplier ?? 1
}

function formatRate(value: number): string {
  return Number(value || 1)
    .toFixed(4)
    .replace(/\.?0+$/, '')
}

function tokenPrice(value: number | null): string {
  return value == null
    ? '-'
    : `${formatScaled(value, perMillionScale)} ${t('modelMarketplace.pricing.unitPerMillion')}`
}

function requestPrice(value: number | null): string {
  return value == null
    ? '-'
    : `${formatScaled(value, 1)} ${t('modelMarketplace.pricing.unitPerRequest')}`
}

function imagePrice(model: UserSupportedModel): string {
  const value = model.pricing?.image_output_price ?? null
  if (model.pricing?.billing_mode === BILLING_MODE_TOKEN) {
    return tokenPrice(value)
  }
  return requestPrice(value)
}

function billingModeLabel(model: UserSupportedModel): string {
  switch (model.pricing?.billing_mode) {
    case BILLING_MODE_TOKEN:
      return t('modelMarketplace.pricing.billingModeToken')
    case BILLING_MODE_PER_REQUEST:
      return t('modelMarketplace.pricing.billingModePerRequest')
    case BILLING_MODE_IMAGE:
      return t('modelMarketplace.pricing.billingModeImage')
    default:
      return t('modelMarketplace.noPricing')
  }
}

function tierCount(model: UserSupportedModel): number {
  return model.pricing?.intervals?.length ?? 0
}

onMounted(loadMarketplace)
</script>
