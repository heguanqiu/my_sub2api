<template>
  <AppLayout>
    <div class="mx-auto flex max-w-7xl flex-col gap-5">
      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900/70">
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div class="relative w-full lg:max-w-xl">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('availableChannels.searchPlaceholder')"
                class="input pl-10"
              />
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <div class="inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200">
                <Icon name="cube" size="sm" class="text-primary-500" />
                <span>{{ t('availableChannels.modelCount', { count: displayModelCards.length }) }}</span>
              </div>
              <button
                type="button"
                class="btn btn-secondary btn-icon"
                :disabled="loading"
                :title="t('common.refresh')"
                @click="loadChannels"
              >
                <Icon name="refresh" size="md" :class="{ 'animate-spin': loading }" />
              </button>
            </div>
          </div>

          <div class="flex flex-col gap-3">
            <div class="flex flex-wrap items-center gap-2">
              <span class="w-16 flex-shrink-0 text-xs font-medium uppercase text-gray-500 dark:text-dark-400">
                {{ t('availableChannels.filters.groups') }}
              </span>
              <button
                type="button"
                :class="filterButtonClasses(selectedGroupId === null)"
                @click="selectedGroupId = null"
              >
                {{ t('availableChannels.filters.allGroups') }}
              </button>
              <button
                v-for="group in groupFilters"
                :key="group.id"
                type="button"
                :class="filterButtonClasses(selectedGroupId === group.id)"
                @click="selectedGroupId = group.id"
              >
                <span class="truncate">{{ group.name }}</span>
                <span class="text-[11px] opacity-75">x{{ formatMultiplier(group.multiplier) }}</span>
              </button>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <span class="w-16 flex-shrink-0 text-xs font-medium uppercase text-gray-500 dark:text-dark-400">
                {{ t('availableChannels.filters.vendors') }}
              </span>
              <button
                type="button"
                :class="filterButtonClasses(selectedPlatform === '')"
                @click="selectedPlatform = ''"
              >
                {{ t('availableChannels.filters.allVendors') }}
              </button>
              <button
                v-for="platform in platformFilters"
                :key="platform"
                type="button"
                :class="filterButtonClasses(selectedPlatform === platform)"
                @click="selectedPlatform = platform"
              >
                <PlatformIcon :platform="platform as GroupPlatform" size="xs" />
                {{ platformLabel(platform) }}
              </button>
            </div>
          </div>

          <div class="flex justify-end text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('availableChannels.lastUpdated', { time: lastUpdatedLabel }) }}</span>
          </div>
        </div>
      </section>

      <section v-if="loading && modelCards.length === 0" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        <div
          v-for="idx in 8"
          :key="idx"
          class="h-80 animate-pulse rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900/70"
        >
          <div class="mb-4 h-5 w-32 rounded bg-gray-200 dark:bg-dark-700"></div>
          <div class="mb-6 h-4 w-5/6 rounded bg-gray-100 dark:bg-dark-800"></div>
          <div class="space-y-3">
            <div class="h-10 rounded bg-gray-100 dark:bg-dark-800"></div>
            <div class="h-10 rounded bg-gray-100 dark:bg-dark-800"></div>
            <div class="h-10 rounded bg-gray-100 dark:bg-dark-800"></div>
          </div>
        </div>
      </section>

      <section
        v-else-if="displayModelCards.length === 0"
        class="rounded-lg border border-dashed border-gray-300 bg-white p-10 text-center dark:border-dark-700 dark:bg-dark-900/60"
      >
        <Icon name="cube" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-dark-600" />
        <div class="text-sm font-medium text-gray-700 dark:text-gray-200">
          {{ t('availableChannels.empty') }}
        </div>
      </section>

      <section v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        <article
          v-for="card in displayModelCards"
          :key="card.id"
          class="flex min-h-[22rem] flex-col rounded-lg border bg-white p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:bg-dark-900/70"
          :class="platformBorderClass(card.platform)"
        >
          <div class="mb-3 flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span
                  class="inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium"
                  :class="platformBadgeClass(card.platform)"
                >
                  <PlatformIcon :platform="card.platform as GroupPlatform" size="xs" />
                  {{ platformLabel(card.platform) }}
                </span>
                <span class="inline-flex items-center rounded-md bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                  {{ t('availableChannels.statusAvailable') }}
                </span>
              </div>
              <h2 class="mt-3 break-all text-sm font-semibold leading-5 text-gray-950 dark:text-gray-50">
                {{ card.name }}
              </h2>
            </div>
            <div class="flex flex-shrink-0 items-center gap-2">
              <button
                type="button"
                class="btn btn-secondary btn-icon !rounded-lg !p-2"
                :title="t('availableChannels.details')"
                @click="openModelDetails(card)"
              >
                <Icon name="infoCircle" size="sm" />
              </button>
              <button
                type="button"
                class="btn btn-secondary btn-icon !rounded-lg !p-2"
                :title="t('availableChannels.copyModel')"
                @click="copyModelName(card.name)"
              >
                <Icon :name="copiedModelName === card.name ? 'check' : 'copy'" size="sm" />
              </button>
            </div>
          </div>

          <div class="mb-3 min-h-[1.5rem] text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('availableChannels.source') }}: {{ visibleChannelNames(card).join(', ') }}</span>
            <span v-if="remainingChannelCount(card) > 0"> +{{ remainingChannelCount(card) }}</span>
          </div>

          <div class="mb-3 rounded-lg bg-gray-50 p-3 dark:bg-dark-800/70">
            <div class="mb-3 flex items-center justify-between gap-3 text-xs">
              <span class="text-gray-500 dark:text-dark-400">{{ t('availableChannels.priceScope') }}</span>
              <span class="truncate font-medium text-gray-700 dark:text-gray-200">
                {{ displayAccessLabel(card.displayAccess) }}
              </span>
            </div>

            <div v-if="!card.displayAccess?.pricing" class="rounded-lg border border-dashed border-gray-200 p-4 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
              {{ t('availableChannels.noPricing') }}
            </div>

            <div v-else class="space-y-2">
              <div class="flex items-center justify-between text-xs">
                <span class="text-gray-500 dark:text-dark-400">{{ t('availableChannels.pricing.billingMode') }}</span>
                <span class="font-medium text-gray-700 dark:text-gray-200">
                  {{ billingModeLabel(card.displayAccess.pricing) }}
                </span>
              </div>

              <PriceLine
                v-for="row in pricingRows(card.displayAccess)"
                :key="row.label"
                :label="row.label"
                :value="row.value"
                :unit="row.unit"
              />

              <div
                v-if="card.displayAccess.pricing.intervals?.length"
                class="mt-3 border-t border-gray-200 pt-2 dark:border-dark-700"
              >
                <div class="mb-1 text-xs font-medium text-gray-500 dark:text-dark-400">
                  {{ t('availableChannels.pricing.intervals') }}
                </div>
                <div class="space-y-1">
                  <div
                    v-for="(interval, idx) in card.displayAccess.pricing.intervals"
                    :key="idx"
                    class="flex items-center justify-between gap-2 text-[11px] text-gray-600 dark:text-dark-300"
                  >
                    <span>{{ interval.tier_label || formatIntervalRange(interval.min_tokens, interval.max_tokens) }}</span>
                    <span class="font-medium">
                      {{ formatIntervalPrice(card.displayAccess, interval) }}
                      <span v-if="formatIntervalPrice(card.displayAccess, interval) !== '-'" class="font-normal text-gray-400 dark:text-dark-500">
                        {{ intervalPriceUnit(card.displayAccess) }}
                      </span>
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="mt-auto flex flex-wrap gap-2 border-t border-gray-100 pt-3 dark:border-dark-800">
            <span
              v-for="access in card.accesses"
              :key="access.groupId"
              class="inline-flex max-w-full items-center gap-1 rounded-md bg-primary-500/10 px-2 py-1 text-xs font-medium text-primary-700 dark:text-primary-300"
              :title="`${access.groupName} x${formatMultiplier(access.multiplier)}`"
            >
              <span class="truncate">{{ access.groupName }}</span>
              <span class="flex-shrink-0">x{{ formatMultiplier(access.multiplier) }}</span>
            </span>
          </div>
        </article>
      </section>

      <div
        v-if="detailsCard"
        class="fixed inset-0 z-50 flex items-end bg-black/45 p-0 backdrop-blur-sm sm:items-center sm:p-6"
        @click.self="closeModelDetails"
      >
        <section class="flex max-h-[92vh] w-full flex-col overflow-hidden rounded-t-xl bg-white shadow-2xl dark:bg-dark-900 sm:mx-auto sm:max-w-4xl sm:rounded-xl">
          <header class="flex items-start justify-between gap-4 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <div class="min-w-0">
              <div class="mb-2 flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-medium" :class="platformBadgeClass(detailsCard.platform)">
                  <PlatformIcon :platform="detailsCard.platform as GroupPlatform" size="xs" />
                  {{ platformLabel(detailsCard.platform) }}
                </span>
                <span class="inline-flex items-center rounded-md bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                  {{ t('availableChannels.statusAvailable') }}
                </span>
              </div>
              <h2 class="break-all text-base font-semibold text-gray-950 dark:text-gray-50">
                {{ detailsCard.name }}
              </h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ t('availableChannels.modelId') }}
              </p>
            </div>
            <div class="flex flex-shrink-0 items-center gap-2">
              <button
                type="button"
                class="btn btn-secondary btn-icon !rounded-lg !p-2"
                :title="t('availableChannels.copyModel')"
                @click="copyModelName(detailsCard.name)"
              >
                <Icon :name="copiedModelName === detailsCard.name ? 'check' : 'copy'" size="sm" />
              </button>
              <button
                type="button"
                class="btn btn-secondary btn-icon !rounded-lg !p-2"
                :title="t('availableChannels.closeDetails')"
                @click="closeModelDetails"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </header>

          <div class="overflow-y-auto px-5 py-5">
            <div class="grid gap-4 lg:grid-cols-[1fr_0.9fr]">
              <div class="space-y-4">
                <section class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                  <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                    {{ t('availableChannels.groupAccess') }}
                  </h3>
                  <div class="space-y-3">
                    <div
                      v-for="access in detailsCard.accesses"
                      :key="access.groupId"
                      class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800/70"
                    >
                      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                        <span class="inline-flex max-w-full items-center gap-1 rounded-md bg-primary-500/10 px-2 py-1 text-xs font-medium text-primary-700 dark:text-primary-300">
                          <span class="truncate">{{ access.groupName }}</span>
                          <span class="flex-shrink-0">x{{ formatMultiplier(access.multiplier) }}</span>
                        </span>
                        <span v-if="detailsCard.displayAccess?.groupId === access.groupId" class="rounded-md bg-emerald-500/10 px-2 py-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                          {{ t('availableChannels.selected') }}
                        </span>
                      </div>
                      <div class="mb-3 text-xs text-gray-500 dark:text-dark-400">
                        {{ t('availableChannels.channels') }}: {{ channelNamesLabel(access.channelNames) }}
                      </div>

                      <div v-if="!access.pricing" class="rounded-lg border border-dashed border-gray-200 p-3 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
                        {{ t('availableChannels.noPricing') }}
                      </div>
                      <div v-else class="space-y-2">
                        <div class="flex items-center justify-between text-xs">
                          <span class="text-gray-500 dark:text-dark-400">{{ t('availableChannels.pricing.billingMode') }}</span>
                          <span class="font-medium text-gray-700 dark:text-gray-200">{{ billingModeLabel(access.pricing) }}</span>
                        </div>
                        <PriceLine
                          v-for="row in pricingRows(access)"
                          :key="row.label"
                          :label="row.label"
                          :value="row.value"
                          :unit="row.unit"
                        />

                        <div
                          v-if="access.pricing.intervals?.length"
                          class="mt-3 border-t border-gray-200 pt-2 dark:border-dark-700"
                        >
                          <div class="mb-2 text-xs font-medium text-gray-500 dark:text-dark-400">
                            {{ t('availableChannels.pricing.intervals') }}
                          </div>
                          <div class="space-y-1">
                            <div
                              v-for="(interval, idx) in access.pricing.intervals"
                              :key="idx"
                              class="flex items-center justify-between gap-3 rounded-md bg-white px-3 py-2 text-xs dark:bg-dark-900/70"
                            >
                              <span class="min-w-0 truncate text-gray-500 dark:text-dark-400">
                                {{ interval.tier_label || formatIntervalRange(interval.min_tokens, interval.max_tokens) }}
                              </span>
                              <span class="flex-shrink-0 font-semibold text-gray-900 dark:text-gray-100">
                                {{ formatIntervalPrice(access, interval) }}
                                <span v-if="formatIntervalPrice(access, interval) !== '-'" class="ml-1 font-normal text-gray-400 dark:text-dark-500">
                                  {{ intervalPriceUnit(access) }}
                                </span>
                              </span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </section>
              </div>

              <aside class="space-y-4">
                <section class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                  <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
                    {{ t('availableChannels.source') }}
                  </h3>
                  <div class="flex flex-wrap gap-2">
                    <span
                      v-for="channelName in detailsCard.channelNames"
                      :key="channelName"
                      class="rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-800 dark:text-dark-300"
                    >
                      {{ channelName }}
                    </span>
                    <span v-if="detailsCard.channelNames.length === 0" class="text-xs text-gray-500 dark:text-dark-400">
                      {{ t('availableChannels.noChannels') }}
                    </span>
                  </div>
                </section>

                <section class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                  <div class="mb-3 flex items-center justify-between gap-3">
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                      {{ t('availableChannels.apiExample') }}
                    </h3>
                    <button
                      type="button"
                      class="btn btn-secondary btn-icon !rounded-lg !p-2"
                      :title="t('availableChannels.copyExample')"
                      @click="copyModelExample(detailsCard)"
                    >
                      <Icon name="copy" size="sm" />
                    </button>
                  </div>
                  <pre class="max-h-72 overflow-auto rounded-lg bg-gray-950 p-3 text-xs leading-5 text-gray-100"><code>{{ apiExample(detailsCard) }}</code></pre>
                  <router-link
                    :to="{ path: '/playground', query: { model: detailsCard.name } }"
                    class="mt-3 inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:underline dark:text-primary-400"
                  >
                    <Icon name="terminal" size="xs" />
                    {{ t('availableChannels.openPlayground') }}
                  </router-link>
                </section>
              </aside>
            </div>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import userChannelsAPI, {
  type UserMarketplaceAccess,
  type UserMarketplaceModel,
  type UserMarketplacePricing,
  type UserMarketplacePricingInterval,
} from '@/api/channels'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatScaled } from '@/utils/pricing'
import { platformBadgeClass, platformBorderClass, platformLabel } from '@/utils/platformColors'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_TOKEN,
  type BillingMode,
} from '@/constants/channel'
import type { GroupPlatform } from '@/types'

const PriceLine = defineComponent({
  name: 'PriceLine',
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
    unit: { type: String, required: true },
  },
  setup(props) {
    return () =>
      h('div', { class: 'flex items-center justify-between gap-3 rounded-md bg-white px-3 py-2 text-xs dark:bg-dark-900/70' }, [
        h('span', { class: 'text-gray-500 dark:text-dark-400' }, props.label),
        h('span', { class: 'font-semibold text-gray-900 dark:text-gray-100' }, [
          props.value,
          props.value === '-' ? '' : h('span', { class: 'ml-1 font-normal text-gray-400 dark:text-dark-500' }, props.unit),
        ]),
      ])
  },
})

type PricingNumberKey =
  | 'input_price'
  | 'output_price'
  | 'cache_write_price'
  | 'cache_read_price'
  | 'image_output_price'
  | 'per_request_price'

interface ModelAccess {
  groupId: number
  groupName: string
  platform: string
  multiplier: number
  isExclusive: boolean
  pricing: UserMarketplacePricing | null
  channelNames: string[]
}

interface ModelCard {
  id: string
  name: string
  platform: string
  accesses: ModelAccess[]
  channelNames: string[]
}

interface DisplayModelCard extends ModelCard {
  displayAccess: ModelAccess | null
}

interface GroupFilter {
  id: number
  name: string
  platform: string
  multiplier: number
}

const { t, locale } = useI18n()
const appStore = useAppStore()

const perMillionScale = 1_000_000

const marketplaceModels = ref<UserMarketplaceModel[]>([])
const loading = ref(false)
const searchQuery = ref('')
const selectedGroupId = ref<number | null>(null)
const selectedPlatform = ref('')
const copiedModelName = ref('')
const lastUpdatedAt = ref<number | null>(null)
const detailsCard = ref<DisplayModelCard | null>(null)

const modelCards = computed<ModelCard[]>(() => marketplaceModels.value.map(toModelCard))

const groupFilters = computed<GroupFilter[]>(() => {
  const seen = new Map<number, GroupFilter>()
  for (const card of modelCards.value) {
    for (const access of card.accesses) {
      if (!seen.has(access.groupId)) {
        seen.set(access.groupId, {
          id: access.groupId,
          name: access.groupName,
          platform: access.platform,
          multiplier: access.multiplier,
        })
      }
    }
  }
  return Array.from(seen.values()).sort((a, b) => a.name.localeCompare(b.name))
})

const platformFilters = computed<string[]>(() => {
  const set = new Set<string>()
  for (const card of modelCards.value) {
    if (card.platform) set.add(card.platform)
  }
  return Array.from(set).sort((a, b) => platformLabel(a).localeCompare(platformLabel(b)))
})

const filteredModelCards = computed<ModelCard[]>(() => {
  const query = searchQuery.value.trim().toLowerCase()
  return modelCards.value.filter((card) => {
    if (selectedPlatform.value && card.platform !== selectedPlatform.value) return false
    if (selectedGroupId.value != null && !card.accesses.some((access) => access.groupId === selectedGroupId.value)) {
      return false
    }
    if (!query) return true

    const searchable = [
      card.name,
      card.platform,
      platformLabel(card.platform),
      ...card.channelNames,
      ...card.accesses.map((access) => access.groupName),
    ]
    return searchable.some((item) => item.toLowerCase().includes(query))
  })
})

const displayModelCards = computed<DisplayModelCard[]>(() =>
  filteredModelCards.value.map((card) => ({
    ...card,
    displayAccess: resolveDisplayAccess(card),
  })),
)

const lastUpdatedLabel = computed(() => {
  if (!lastUpdatedAt.value) return '-'
  const date = new Date(lastUpdatedAt.value)
  return new Intl.DateTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
})

function toModelCard(model: UserMarketplaceModel): ModelCard {
  return {
    id: model.id,
    name: model.name,
    platform: normalizePlatform(model.platform),
    accesses: (model.accesses || []).map(toModelAccess),
    channelNames: [...(model.channel_names || [])].sort((a, b) => a.localeCompare(b)),
  }
}

function toModelAccess(access: UserMarketplaceAccess): ModelAccess {
  return {
    groupId: access.group_id,
    groupName: access.group_name,
    platform: normalizePlatform(access.platform),
    multiplier: access.rate_multiplier,
    isExclusive: access.is_exclusive,
    pricing: access.effective_pricing,
    channelNames: [...(access.channel_names || [])].sort((a, b) => a.localeCompare(b)),
  }
}

function normalizePlatform(platform: string | undefined): string {
  return (platform || '').trim().toLowerCase()
}

function resolveDisplayAccess(card: ModelCard): ModelAccess | null {
  const candidates =
    selectedGroupId.value == null
      ? card.accesses
      : card.accesses.filter((access) => access.groupId === selectedGroupId.value)
  if (candidates.length === 0) return null
  return candidates.reduce((best, access) => {
    const bestScore = pricingScore(best.pricing)
    const accessScore = pricingScore(access.pricing)
    if (accessScore !== bestScore) return accessScore < bestScore ? access : best
    if (access.multiplier !== best.multiplier) return access.multiplier < best.multiplier ? access : best
    return access.groupName.localeCompare(best.groupName) < 0 ? access : best
  }, candidates[0])
}

function pricingScore(pricing: UserMarketplacePricing | null): number {
  if (!pricing) return Number.POSITIVE_INFINITY
  const values = priceValues(pricing)
  if (values.length === 0) return Number.POSITIVE_INFINITY
  return values.reduce((sum, value) => sum + value, 0)
}

function priceValues(pricing: UserMarketplacePricing): number[] {
  if (pricing.billing_mode === BILLING_MODE_PER_REQUEST || pricing.billing_mode === BILLING_MODE_IMAGE) {
    return numberList(
      requestPriceValue(pricing),
      ...pricing.intervals.map((interval) => interval.per_request_price),
    )
  }
  return numberList(
    pricing.input_price,
    pricing.output_price,
    pricing.cache_write_price,
    pricing.cache_read_price,
    pricing.image_output_price,
  )
}

function numberList(...values: Array<number | null | undefined>): number[] {
  return values.filter((value): value is number => typeof value === 'number' && Number.isFinite(value) && value >= 0)
}

function filterButtonClasses(active: boolean): string[] {
  return [
    'inline-flex max-w-full items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium transition',
    active
      ? 'border-primary-500 bg-primary-500 text-white shadow-sm'
      : 'border-gray-200 bg-white text-gray-600 hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300 dark:hover:border-primary-500/60 dark:hover:text-primary-300',
  ]
}

function formatMultiplier(value: number): string {
  if (!Number.isFinite(value)) return '1'
  return value.toLocaleString('en-US', { maximumFractionDigits: 4 })
}

function formatAccessPrice(access: ModelAccess | null, key: PricingNumberKey, scale: number): string {
  if (!access) return '-'
  const value = access?.pricing?.[key]
  if (value == null || !Number.isFinite(value)) return '-'
  return formatScaled(value, scale)
}

function hasPositivePricingValue(access: ModelAccess | null, key: PricingNumberKey): boolean {
  const value = access?.pricing?.[key]
  return typeof value === 'number' && value > 0
}

function requestPriceValue(pricing: UserMarketplacePricing): number | null {
  return pricing.per_request_price ?? pricing.image_output_price ?? null
}

function formatRequestPrice(access: ModelAccess | null): string {
  if (!access?.pricing) return '-'
  const value = requestPriceValue(access.pricing)
  if (value == null || !Number.isFinite(value)) return '-'
  return formatScaled(value, 1)
}

function billingModeLabel(pricing: UserMarketplacePricing): string {
  const mode: BillingMode = pricing.billing_mode
  switch (mode) {
    case BILLING_MODE_TOKEN:
      return t('availableChannels.pricing.billingModeToken')
    case BILLING_MODE_PER_REQUEST:
      return t('availableChannels.pricing.billingModePerRequest')
    case BILLING_MODE_IMAGE:
      return t('availableChannels.pricing.billingModeImage')
    default:
      return '-'
  }
}

function displayAccessLabel(access: ModelAccess | null): string {
  if (!access) return '-'
  const label = selectedGroupId.value == null ? t('availableChannels.lowestAvailablePrice') : t('availableChannels.selectedGroupPrice')
  return `${label}: ${access.groupName} x${formatMultiplier(access.multiplier)}`
}

interface PricingRow {
  label: string
  value: string
  unit: string
}

function pricingRows(access: ModelAccess | null): PricingRow[] {
  const pricing = access?.pricing
  if (!pricing) return []

  if (pricing.billing_mode === BILLING_MODE_PER_REQUEST) {
    return [
      {
        label: t('availableChannels.pricing.perRequestPrice'),
        value: formatRequestPrice(access),
        unit: t('availableChannels.pricing.unitPerRequest'),
      },
    ]
  }

  if (pricing.billing_mode === BILLING_MODE_IMAGE) {
    return [
      {
        label: t('availableChannels.pricing.imageRequestPrice'),
        value: formatRequestPrice(access),
        unit: t('availableChannels.pricing.unitPerRequest'),
      },
    ]
  }

  const rows: PricingRow[] = [
    {
      label: t('availableChannels.pricing.inputPrice'),
      value: formatAccessPrice(access, 'input_price', perMillionScale),
      unit: t('availableChannels.pricing.unitPerMillion'),
    },
    {
      label: t('availableChannels.pricing.outputPrice'),
      value: formatAccessPrice(access, 'output_price', perMillionScale),
      unit: t('availableChannels.pricing.unitPerMillion'),
    },
    {
      label: t('availableChannels.pricing.cacheWritePrice'),
      value: formatAccessPrice(access, 'cache_write_price', perMillionScale),
      unit: t('availableChannels.pricing.unitPerMillion'),
    },
    {
      label: t('availableChannels.pricing.cacheReadPrice'),
      value: formatAccessPrice(access, 'cache_read_price', perMillionScale),
      unit: t('availableChannels.pricing.unitPerMillion'),
    },
  ]

  if (hasPositivePricingValue(access, 'image_output_price')) {
    rows.push({
      label: t('availableChannels.pricing.imageOutputPrice'),
      value: formatAccessPrice(access, 'image_output_price', perMillionScale),
      unit: t('availableChannels.pricing.unitPerMillion'),
    })
  }

  return rows
}

function visibleChannelNames(card: ModelCard): string[] {
  return card.channelNames.slice(0, 2)
}

function remainingChannelCount(card: ModelCard): number {
  return Math.max(0, card.channelNames.length - 2)
}

function channelNamesLabel(names: string[]): string {
  return names.length > 0 ? names.join(', ') : t('availableChannels.noChannels')
}

function formatIntervalRange(min: number, max: number | null): string {
  return max == null ? `${min}+` : `${min}-${max}`
}

function formatIntervalPrice(access: ModelAccess, interval: UserMarketplacePricingInterval): string {
  if (access.pricing?.billing_mode === BILLING_MODE_PER_REQUEST || access.pricing?.billing_mode === BILLING_MODE_IMAGE) {
    return interval.per_request_price == null ? '-' : formatScaled(interval.per_request_price, 1)
  }
  const input = interval.input_price == null ? '-' : formatScaled(interval.input_price, perMillionScale)
  const output = interval.output_price == null ? '-' : formatScaled(interval.output_price, perMillionScale)
  return `${input} / ${output}`
}

function intervalPriceUnit(access: ModelAccess): string {
  if (access.pricing?.billing_mode === BILLING_MODE_PER_REQUEST || access.pricing?.billing_mode === BILLING_MODE_IMAGE) {
    return t('availableChannels.pricing.unitPerRequest')
  }
  return t('availableChannels.pricing.unitPerMillion')
}

async function copyModelName(modelName: string) {
  try {
    await navigator.clipboard.writeText(modelName)
    copiedModelName.value = modelName
    window.setTimeout(() => {
      if (copiedModelName.value === modelName) copiedModelName.value = ''
    }, 1200)
    appStore.showSuccess(t('availableChannels.modelCopied'))
  } catch (err) {
    console.error('Failed to copy model name:', err)
    appStore.showError(t('common.copyFailed'))
  }
}

function openModelDetails(card: DisplayModelCard) {
  detailsCard.value = card
}

function closeModelDetails() {
  detailsCard.value = null
}

function apiExample(card: ModelCard): string {
  const apiBase = `${window.location.origin.replace(/\/+$/, '')}/v1`
  return [
    `curl ${apiBase}/chat/completions \\`,
    `  -H "Authorization: Bearer $SUB2API_KEY" \\`,
    `  -H "Content-Type: application/json" \\`,
    `  -d '{`,
    `    "model": "${card.name}",`,
    `    "messages": [`,
    `      {"role": "user", "content": "Hello"}`,
    `    ]`,
    `  }'`,
  ].join('\n')
}

async function copyModelExample(card: ModelCard) {
  try {
    await navigator.clipboard.writeText(apiExample(card))
    appStore.showSuccess(t('availableChannels.exampleCopied'))
  } catch (err) {
    console.error('Failed to copy model example:', err)
    appStore.showError(t('common.copyFailed'))
  }
}

async function loadChannels() {
  loading.value = true
  try {
    const response = await userChannelsAPI.getMarketplace()
    marketplaceModels.value = response.models || []
    lastUpdatedAt.value = response.updated_at ? new Date(response.updated_at).getTime() : Date.now()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadChannels)
</script>
