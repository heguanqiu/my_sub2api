<template>
  <AppLayout :padded="false">
    <div class="playground-page">
      <div class="playground-toolbar">
        <div class="playground-controls">
          <label class="sr-only" for="playground-key-select">
            {{ t('playground.selectApiKey') }}
          </label>
          <select
            id="playground-key-select"
            v-model="selectedKeyId"
            class="input playground-key-select"
            :disabled="loadingKeys || usableApiKeys.length === 0"
          >
            <option value="" disabled>{{ t('playground.selectApiKey') }}</option>
            <option v-for="apiKey in usableApiKeys" :key="apiKey.id" :value="String(apiKey.id)">
              {{ apiKeyLabel(apiKey) }}
            </option>
          </select>

          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loadingKeys"
            :title="t('common.refresh')"
            @click="loadApiKeys"
          >
            <Icon name="refresh" size="md" :class="loadingKeys ? 'animate-spin' : ''" />
          </button>

          <a
            v-if="selectedApiKey && embedSession"
            class="btn btn-secondary"
            :href="iframeSrc"
            target="_blank"
            rel="noopener noreferrer"
            :title="t('playground.openInNewTab')"
          >
            <Icon name="externalLink" size="md" />
          </a>
          <button
            v-else
            type="button"
            class="btn btn-secondary"
            disabled
            :title="t('playground.openInNewTab')"
          >
            <Icon name="externalLink" size="md" />
          </button>
        </div>
      </div>

      <div v-if="loadingKeys" class="playground-empty">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="usableApiKeys.length === 0" class="playground-empty">
        <div class="max-w-md text-center">
          <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
            <Icon name="key" size="lg" class="text-gray-400" />
          </div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('playground.noKeysTitle') }}
          </h2>
          <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
            {{ t('playground.noKeysDesc') }}
          </p>
        </div>
      </div>

      <div v-else class="playground-frame-shell">
        <div v-if="frameLoading || loadingEmbedSession" class="playground-frame-loading">
          <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        </div>
        <iframe
          v-if="!selectedApiKey || embedSession"
          :key="frameKey"
          :src="iframeSrc"
          :name="iframeName"
          class="playground-frame"
          allow="clipboard-read; clipboard-write; microphone; fullscreen"
          allowfullscreen
          @load="frameLoading = false"
        ></iframe>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import keysAPI from '@/api/keys'
import playgroundAPI, { type PlaygroundEmbedSession } from '@/api/playground'
import type { ApiKey } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const OPEN_WEBUI_URL =
  (import.meta.env.VITE_OPEN_WEBUI_URL as string | undefined)?.trim() ||
  'https://chat.jinlongjiangzhuang.click'
const SELECTED_KEY_STORAGE_KEY = 'sub2api.playground.selectedKeyId'

const { t, locale } = useI18n()
const appStore = useAppStore()

const apiKeys = ref<ApiKey[]>([])
const selectedKeyId = ref('')
const loadingKeys = ref(false)
const loadingEmbedSession = ref(false)
const frameLoading = ref(false)
const frameNonce = ref(0)
const embedSession = ref<PlaygroundEmbedSession | null>(null)

const usableApiKeys = computed(() =>
  apiKeys.value.filter((apiKey) => apiKey.status === 'active' && !!apiKey.key),
)

const selectedApiKey = computed(() => {
  if (!selectedKeyId.value) return null
  return usableApiKeys.value.find((apiKey) => String(apiKey.id) === selectedKeyId.value) ?? null
})

const apiBaseUrl = computed(() => {
  const configured = appStore.cachedPublicSettings?.api_base_url?.trim()
  return ensureV1BaseUrl(configured || window.location.origin)
})

const embedPayload = computed(() => {
  if (!selectedApiKey.value || !embedSession.value) {
    return null
  }

  return {
    source: embedSession.value.source,
    version: embedSession.value.version,
    expires_at: embedSession.value.expires_at,
    signature: embedSession.value.signature,
    apiBaseUrl: embedSession.value.api_base_url,
    selectedKeyId: embedSession.value.selected_key_id,
    selectedKeyName: embedSession.value.selected_key_name,
    user: embedSession.value.user,
    directConnections: embedSession.value.direct_connections,
  }
})

const encodedEmbedPayload = computed(() => {
  if (!embedPayload.value) return ''
  return base64UrlEncode(JSON.stringify(embedPayload.value))
})

const iframeSrc = computed(() => {
  const url = new URL(OPEN_WEBUI_URL)
  url.searchParams.set('sub2api_embed', '1')
  url.searchParams.set('theme', detectTheme())
  url.searchParams.set('lang', openWebUILocale.value)

  if (encodedEmbedPayload.value) {
    url.hash = new URLSearchParams({
      sub2api_direct: encodedEmbedPayload.value,
    }).toString()
  }

  return url.toString()
})

const iframeName = computed(() =>
  encodedEmbedPayload.value ? `sub2api_direct:${encodedEmbedPayload.value}` : 'sub2api_playground',
)

const frameKey = computed(() => `${selectedKeyId.value || 'none'}-${frameNonce.value}`)
const openWebUILocale = computed(() =>
  String(locale.value || 'zh').toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US',
)

function ensureV1BaseUrl(value: string): string {
  try {
    const url = new URL(value || window.location.origin, window.location.origin)
    const withoutV1 = url.pathname.replace(/\/v1\/?$/, '').replace(/\/+$/, '')
    url.pathname = `${withoutV1}/v1`.replace(/\/{2,}/g, '/')
    url.search = ''
    url.hash = ''
    return url.toString().replace(/\/$/, '')
  } catch {
    return `${window.location.origin.replace(/\/+$/, '')}/v1`
  }
}

function base64UrlEncode(value: string): string {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

function detectTheme(): 'light' | 'dark' {
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}

function apiKeyLabel(apiKey: ApiKey): string {
  const group = apiKey.group?.name || t('keys.noGroup')
  const platform = apiKey.group?.platform ? ` / ${apiKey.group.platform}` : ''
  return `${apiKey.name} · ${group}${platform}`
}

async function loadApiKeys() {
  loadingKeys.value = true
  try {
    const response = await keysAPI.list(1, 200, {
      status: 'active',
      sort_by: 'created_at',
      sort_order: 'desc',
    })
    apiKeys.value = response.items ?? []

    const storedKeyId = localStorage.getItem(SELECTED_KEY_STORAGE_KEY)
    if (storedKeyId && usableApiKeys.value.some((apiKey) => String(apiKey.id) === storedKeyId)) {
      selectedKeyId.value = storedKeyId
    } else if (!selectedApiKey.value && usableApiKeys.value.length > 0) {
      selectedKeyId.value = String(usableApiKeys.value[0].id)
    }
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('playground.loadFailed')))
  } finally {
    loadingKeys.value = false
  }
}

async function loadEmbedSession() {
  const apiKey = selectedApiKey.value
  embedSession.value = null

  if (!apiKey) {
    frameLoading.value = false
    return
  }

  loadingEmbedSession.value = true
  frameLoading.value = true
  try {
    embedSession.value = await playgroundAPI.createEmbedSession(apiKey.id, apiBaseUrl.value)
  } catch (err: unknown) {
    frameLoading.value = false
    appStore.showError(extractApiErrorMessage(err, t('playground.loadFailed')))
  } finally {
    loadingEmbedSession.value = false
  }
}

watch(selectedKeyId, (value) => {
  if (value) {
    localStorage.setItem(SELECTED_KEY_STORAGE_KEY, value)
    frameLoading.value = true
    frameNonce.value += 1
    loadEmbedSession()
  }
})

watch(apiBaseUrl, () => {
  if (selectedApiKey.value) {
    frameNonce.value += 1
    loadEmbedSession()
  }
})

onMounted(async () => {
  await appStore.fetchPublicSettings().catch((err: unknown) => {
    console.error('Failed to load public settings:', err)
  })
  await loadApiKeys()
})
</script>

<style scoped>
.playground-page {
  display: flex;
  height: calc(100vh - 4rem);
  min-height: 0;
  flex-direction: column;
}

.playground-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  flex-shrink: 0;
  border-bottom: 1px solid rgb(229 231 235);
  padding: 0.75rem 1rem;
}

.dark .playground-toolbar {
  border-bottom-color: rgb(55 65 81);
}

.playground-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
}

.playground-key-select {
  width: min(28rem, 56vw);
}

.playground-frame-shell,
.playground-empty {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border: 1px solid rgb(229 231 235);
  border-width: 0;
  background: white;
}

.dark .playground-frame-shell,
.dark .playground-empty {
  border-color: rgb(55 65 81);
  background: rgb(17 24 39);
}

.playground-empty,
.playground-frame-loading {
  display: flex;
  align-items: center;
  justify-content: center;
}

.playground-frame-loading {
  position: absolute;
  inset: 0;
  z-index: 1;
  background: rgb(255 255 255 / 0.84);
}

.dark .playground-frame-loading {
  background: rgb(17 24 39 / 0.84);
}

.playground-frame {
  height: 100%;
  min-height: 0;
  width: 100%;
  border: 0;
}

@media (max-width: 768px) {
  .playground-page {
    height: calc(100vh - 4rem);
  }

  .playground-toolbar {
    flex-direction: row;
    align-items: stretch;
    padding: 0.75rem;
  }

  .playground-controls {
    justify-content: flex-start;
  }

  .playground-key-select {
    width: 100%;
  }
}
</style>
