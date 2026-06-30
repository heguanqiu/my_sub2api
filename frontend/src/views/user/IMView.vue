<template>
  <AppLayout :padded="false">
    <div class="im-page">
      <div v-if="errorMessage" class="im-state">
        <div class="im-state-icon">
          <Icon name="exclamationTriangle" size="lg" />
        </div>
        <p class="im-state-text">{{ errorMessage }}</p>
        <button type="button" class="btn btn-primary" :disabled="loading" @click="loadIM">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span>{{ t('common.retry') }}</span>
        </button>
      </div>

      <div v-else class="im-frame-shell">
        <div v-if="loading || frameLoading" class="im-frame-loading">
          <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        </div>
        <iframe
          v-if="iframeSrc"
          :key="iframeKey"
          :src="iframeSrc"
          class="im-frame"
          allow="clipboard-read; clipboard-write; microphone; camera; fullscreen; notifications"
          allowfullscreen
          @load="frameLoading = false"
        ></iframe>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import imAPI, { type IMSsoTokenResponse } from '@/api/im'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()

const iframeSrc = ref('')
const iframeKey = ref(0)
const loading = ref(false)
const frameLoading = ref(false)
const errorMessage = ref('')

function buildIframeURL(session: IMSsoTokenResponse): string {
  const url = new URL(session.web_url)
  const hashParams = new URLSearchParams({
    mySub2apiSso: session.token,
  })
  if (session.service_url?.trim()) {
    hashParams.set('serviceUrl', session.service_url.trim())
  }
  url.hash = hashParams.toString()
  return url.toString()
}

async function loadIM() {
  loading.value = true
  frameLoading.value = true
  errorMessage.value = ''

  try {
    const session = await imAPI.issueSSOToken()
    iframeSrc.value = buildIframeURL(session)
    iframeKey.value += 1
  } catch (error) {
    frameLoading.value = false
    iframeSrc.value = ''
    errorMessage.value = extractApiErrorMessage(error, t('im.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadIM()
})
</script>

<style scoped>
.im-page {
  position: relative;
  min-height: calc(100vh - 64px);
  background: rgb(249 250 251);
}

.dark .im-page {
  background: rgb(2 6 23);
}

.im-frame-shell {
  position: relative;
  min-height: calc(100vh - 64px);
}

.im-frame {
  display: block;
  width: 100%;
  height: calc(100vh - 64px);
  border: 0;
  background: white;
}

.im-frame-loading,
.im-state {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
}

.im-frame-loading {
  background: rgb(249 250 251 / 0.85);
}

.dark .im-frame-loading {
  background: rgb(2 6 23 / 0.85);
}

.im-state {
  flex-direction: column;
  gap: 1rem;
  padding: 1.5rem;
  text-align: center;
}

.im-state-icon {
  display: flex;
  height: 3rem;
  width: 3rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background: rgb(254 242 242);
  color: rgb(220 38 38);
}

.dark .im-state-icon {
  background: rgb(127 29 29 / 0.3);
  color: rgb(248 113 113);
}

.im-state-text {
  max-width: 28rem;
  color: rgb(75 85 99);
  font-size: 0.875rem;
  line-height: 1.5rem;
}

.dark .im-state-text {
  color: rgb(148 163 184);
}
</style>
