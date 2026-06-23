<template>
  <div class="min-h-screen bg-white text-[#0f1729]">
    <header class="sticky top-0 z-40 border-b border-[#e6e9ef] bg-white/95 shadow-sm backdrop-blur">
      <nav class="mx-auto flex h-18 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <router-link to="/home" class="flex min-w-0 items-center gap-3">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-[#1d6ff2] text-sm font-black text-white">
            J
          </span>
          <span class="min-w-0">
            <span class="block text-base font-bold text-[#0f1729]">Jlaude</span>
            <span class="hidden text-xs text-[#98a2b3] sm:block">{{ t('pluginCenter.title') }}</span>
          </span>
        </router-link>

        <div class="flex items-center gap-2">
          <router-link
            to="/home"
            class="inline-flex min-h-11 items-center justify-center rounded-lg px-3 py-2 text-sm font-semibold text-[#475467] transition hover:bg-[#f3f5f9] hover:text-[#0f1729]"
          >
            <Icon name="home" size="sm" class="mr-1" />
            Home
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section class="border-b border-[#e6e9ef] bg-[#f7f9fc]">
        <div class="mx-auto max-w-7xl px-4 py-14 sm:px-6 sm:py-20 lg:px-8">
          <div class="max-w-3xl">
            <p class="text-sm font-bold uppercase tracking-normal text-[#1d6ff2]">Plugin Center</p>
            <h1 class="mt-4 text-4xl font-black leading-tight text-[#0f1729] sm:text-6xl">
              {{ t('pluginCenter.title') }}
            </h1>
            <p class="mt-5 max-w-2xl text-lg leading-8 text-[#475467]">
              {{ t('pluginCenter.subtitle') }}
            </p>
          </div>
        </div>
      </section>

      <section class="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        <div class="mb-8 flex flex-wrap items-center gap-2">
          <button
            v-for="category in categoryTabs"
            :key="category.value"
            type="button"
            class="min-h-11 rounded-lg border px-4 py-2 text-sm font-semibold transition"
            :class="selectedCategory === category.value
              ? 'border-[#1d6ff2] bg-[#eaf2ff] text-[#1d6ff2]'
              : 'border-[#e6e9ef] bg-white text-[#475467] hover:border-[#d6e4ff] hover:bg-[#f7f9fc] hover:text-[#0f1729]'"
            @click="selectCategory(category.value)"
          >
            {{ category.label }}
          </button>
        </div>

        <div v-if="loading" class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="i in 6"
            :key="i"
            class="rounded-lg border border-[#e6e9ef] bg-white p-5"
          >
            <div class="h-12 w-12 animate-pulse rounded-lg bg-[#f3f5f9]"></div>
            <div class="mt-5 h-5 w-2/3 animate-pulse rounded bg-[#f3f5f9]"></div>
            <div class="mt-3 h-4 w-full animate-pulse rounded bg-[#f3f5f9]"></div>
            <div class="mt-2 h-4 w-5/6 animate-pulse rounded bg-[#f3f5f9]"></div>
          </div>
        </div>

        <div
          v-else-if="visiblePlugins.length > 0"
          class="grid gap-5 sm:grid-cols-2 lg:grid-cols-3"
        >
          <article
            v-for="plugin in visiblePlugins"
            :key="plugin.id"
            class="flex min-h-[18rem] flex-col rounded-lg border border-[#e6e9ef] bg-white p-5 shadow-sm transition hover:-translate-y-1 hover:border-[#d6e4ff]"
          >
            <div class="flex items-start gap-4">
              <img
                v-if="plugin.icon_url"
                :src="plugin.icon_url"
                :alt="`${plugin.name} icon`"
                class="h-12 w-12 rounded-lg border border-[#e6e9ef] object-cover"
                loading="lazy"
              />
              <div
                v-else
                class="flex h-12 w-12 items-center justify-center rounded-lg bg-[#eaf2ff] text-[#1d6ff2]"
                aria-hidden="true"
              >
                <Icon name="cube" size="md" />
              </div>

              <div class="min-w-0 flex-1">
                <h2 class="truncate text-lg font-bold text-[#0f1729]">{{ plugin.name }}</h2>
                <div class="mt-2 flex flex-wrap gap-2">
                  <span v-if="plugin.version" class="rounded-md bg-[#f3f5f9] px-2 py-1 text-xs font-bold text-[#475467]">
                    {{ plugin.version }}
                  </span>
                  <span class="rounded-md bg-[#eaf2ff] px-2 py-1 text-xs font-bold text-[#1d6ff2]">
                    {{ plugin.platform || 'all' }}
                  </span>
                </div>
              </div>
            </div>

            <p class="mt-5 line-clamp-4 flex-1 text-sm leading-7 text-[#475467]">
              {{ plugin.description }}
            </p>

            <div class="mt-6 flex items-center justify-between gap-3 border-t border-[#e6e9ef] pt-4">
              <span class="text-xs font-semibold text-[#98a2b3]">
                {{ t('pluginCenter.downloads', { count: plugin.download_count }) }}
              </span>
              <a
                :href="plugin.download_url"
                class="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg bg-[#1d6ff2] px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-[#1551c4]"
              >
                <Icon name="download" size="sm" />
                {{ t('pluginCenter.download') }}
              </a>
            </div>
          </article>
        </div>

        <div v-else class="rounded-lg border border-[#e6e9ef] bg-white px-6 py-16 text-center">
          <Icon name="inbox" size="xl" class="mx-auto h-12 w-12 text-[#98a2b3]" />
          <p class="mt-4 text-base font-semibold text-[#475467]">{{ t('pluginCenter.empty') }}</p>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import pluginsAPI from '@/api/plugins'
import type { PublicPlugin } from '@/types'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const plugins = ref<PublicPlugin[]>([])
const loading = ref(false)
const selectedCategory = ref('')

const categoryTabs = computed(() => {
  const categories = Array.from(
    new Set(plugins.value.map((plugin) => plugin.category).filter(Boolean))
  ).sort((a, b) => a.localeCompare(b))
  return [
    { value: '', label: t('pluginCenter.all') },
    ...categories.map((category) => ({ value: category, label: category }))
  ]
})

const visiblePlugins = computed(() => {
  if (!selectedCategory.value) return plugins.value
  return plugins.value.filter((plugin) => plugin.category === selectedCategory.value)
})

async function loadPlugins() {
  loading.value = true
  try {
    plugins.value = await pluginsAPI.listPublic()
  } finally {
    loading.value = false
  }
}

function selectCategory(category: string) {
  selectedCategory.value = category
}

onMounted(() => {
  loadPlugins()
})
</script>
