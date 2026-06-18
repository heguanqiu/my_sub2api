<template>
  <div v-if="homeContent" class="home-custom-shell min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="home-custom-frame h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else class="home-custom-content" v-html="homeContent"></div>
  </div>

  <div v-else class="landing-shell min-h-screen bg-white text-[#0f1729]">
    <header
      class="fixed inset-x-0 top-0 z-40 border-b transition-all duration-300"
      :class="isScrolled
        ? 'border-[#e6e9ef] bg-white/95 shadow-sm backdrop-blur'
        : 'border-transparent bg-white/80 backdrop-blur'"
    >
      <nav class="mx-auto flex h-18 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <router-link to="/home" class="flex min-w-0 items-center gap-3">
          <span class="brand-mark flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-lg">
            <img
              v-if="siteLogo"
              :src="siteLogo"
              :alt="`${brandName} logo`"
              class="h-full w-full object-contain"
            />
            <span v-else class="text-base font-black text-white">J</span>
          </span>
          <span class="min-w-0">
            <span class="block text-base font-bold tracking-normal text-[#0f1729]">{{ brandName }}</span>
            <span class="hidden text-xs text-[#98a2b3] sm:block">AI API Gateway</span>
          </span>
        </router-link>

        <div class="hidden flex-1 items-center justify-center gap-1 lg:flex">
          <router-link
            to="/token-merchant"
            data-test="home-partner-entry"
            class="rounded-lg px-3 py-2 text-sm font-semibold text-[#1d6ff2] transition hover:bg-[#eaf2ff]"
          >
            成为合伙人
          </router-link>
          <button
            v-for="item in navItems"
            :key="item.id"
            type="button"
            class="rounded-lg px-3 py-2 text-sm font-semibold text-[#475467] transition hover:bg-[#f3f5f9] hover:text-[#0f1729]"
            @click="scrollToSection(item.id)"
          >
            {{ item.label }}
          </button>
          <a
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-lg px-3 py-2 text-sm font-semibold text-[#475467] transition hover:bg-[#f3f5f9] hover:text-[#0f1729]"
          >
            文档
          </a>
        </div>

        <div class="flex justify-end gap-2">
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="hidden items-center justify-center gap-2 rounded-lg bg-[#1d6ff2] px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-[#1551c4] sm:inline-flex"
          >
            {{ isAuthenticated ? '进入控制台' : '登入' }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
          <button
            type="button"
            class="inline-flex rounded-lg p-2 text-[#475467] transition hover:bg-[#f3f5f9] hover:text-[#0f1729] lg:hidden"
            @click="mobileMenuOpen = !mobileMenuOpen"
          >
            <Icon v-if="mobileMenuOpen" name="x" size="md" />
            <Icon v-else name="menu" size="md" />
          </button>
        </div>
      </nav>

      <div
        v-if="mobileMenuOpen"
        class="border-t border-[#e6e9ef] bg-white px-4 py-4 lg:hidden"
      >
        <div class="mx-auto grid max-w-7xl gap-2">
          <button
            v-for="item in navItems"
            :key="item.id"
            type="button"
            class="rounded-lg px-3 py-2 text-left text-sm font-semibold text-[#475467] hover:bg-[#f3f5f9]"
            @click="scrollToSection(item.id)"
          >
            {{ item.label }}
          </button>
          <a
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded-lg px-3 py-2 text-sm font-semibold text-[#475467] hover:bg-[#f3f5f9]"
          >
            文档
          </a>
          <button
            type="button"
            class="rounded-lg px-3 py-2 text-left text-sm font-semibold text-[#475467] hover:bg-[#f3f5f9]"
            @click="openContact"
          >
            联系支持
          </button>
          <router-link
            to="/token-merchant"
            data-test="home-partner-entry-mobile"
            class="rounded-lg px-3 py-2 text-sm font-semibold text-[#1d6ff2] hover:bg-[#eaf2ff]"
            @click="mobileMenuOpen = false"
          >
            成为合伙人
          </router-link>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="mt-2 inline-flex items-center justify-center gap-2 rounded-lg bg-[#1d6ff2] px-4 py-2.5 text-sm font-semibold text-white"
          >
            {{ isAuthenticated ? '进入控制台' : '登入' }}
            <Icon name="arrowRight" size="sm" />
          </router-link>
        </div>
      </div>
    </header>

    <main>
      <section class="hero-stage relative min-h-screen overflow-hidden pt-28">
        <div class="relative mx-auto grid min-h-[calc(100vh-7rem)] max-w-7xl items-center gap-12 px-4 pb-10 sm:px-6 lg:grid-cols-[1.1fr_0.9fr] lg:px-8">
          <div class="max-w-2xl">
            <div class="mb-6 inline-flex items-center gap-2 rounded-full border border-[#d6e4ff] bg-[#eaf2ff] px-3 py-1.5 text-sm font-semibold text-[#1d6ff2]">
              <span class="h-2 w-2 rounded-full bg-[#1d6ff2]"></span>
              Claude Code / Codex / OpenAI 统一接入
            </div>

            <h1 class="text-[clamp(2.6rem,6vw,5.4rem)] font-bold leading-[1.02] tracking-tight text-[#0f1729]">
              把 Claude Code 与 Codex
              <span class="block text-[#1d6ff2]">接入一条稳定高速的线路</span>
            </h1>

            <p class="mt-6 max-w-xl text-lg leading-8 text-[#475467]">
              Jlaude 是面向开发者工作流的 AI API 网关，用一个 Base URL 连接主流模型、编码工具与团队交付流程。
            </p>

            <div class="mt-8 flex flex-col gap-3 sm:flex-row">
              <router-link
                :to="isAuthenticated ? dashboardPath : '/login'"
                class="inline-flex items-center justify-center gap-2 rounded-lg bg-[#1d6ff2] px-6 py-3 text-base font-semibold text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-[#1551c4]"
              >
                {{ isAuthenticated ? '进入控制台' : '开始接入' }}
                <Icon name="arrowRight" size="md" />
              </router-link>
              <a
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center justify-center gap-2 rounded-lg border border-[#e6e9ef] bg-white px-6 py-3 text-base font-semibold text-[#0f1729] transition hover:border-[#d6e4ff] hover:bg-[#f7f9fc]"
              >
                查看文档
                <Icon name="book" size="md" />
              </a>
              <router-link
                to="/token-merchant"
                class="inline-flex items-center justify-center gap-2 rounded-lg border border-[#d6e4ff] bg-[#eaf2ff] px-6 py-3 text-base font-semibold text-[#1d6ff2] transition hover:-translate-y-0.5 hover:bg-[#dbe9ff]"
              >
                成为合伙人
                <Icon name="users" size="md" />
              </router-link>
            </div>

            <button
              type="button"
              class="base-url-bar mt-8 flex w-full max-w-xl items-center justify-between gap-4 rounded-lg border border-[#e6e9ef] bg-white px-4 py-4 text-left shadow-sm transition hover:border-[#d6e4ff]"
              @click="copyBaseUrl"
            >
              <span class="min-w-0">
                <span class="block text-xs font-bold uppercase tracking-normal text-[#98a2b3]">API Base URL</span>
                <span class="mt-1 block truncate font-mono text-base text-[#1d6ff2]">{{ apiBaseUrl }}</span>
              </span>
              <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-[#e6e9ef] bg-[#f7f9fc] text-[#475467]">
                <Icon name="copy" size="md" />
              </span>
            </button>
          </div>

          <div class="hero-geo-stage relative h-[26rem]">
            <div class="hero-geo-grid absolute inset-6 rounded-2xl opacity-70"></div>
            <div class="hero-geo-ring absolute right-10 top-6 h-56 w-56"></div>
            <div class="hero-geo-ring absolute right-24 top-20 h-32 w-32"></div>
            <div class="absolute left-4 top-16 h-28 w-44 rounded-2xl border border-[#e6e9ef] bg-white shadow-sm"></div>
            <div class="absolute left-16 top-44 h-24 w-52 rounded-2xl border border-[#d6e4ff] bg-[#eaf2ff]"></div>
            <div class="absolute bottom-8 right-16 h-20 w-20 rounded-2xl bg-[#1d6ff2] shadow-sm"></div>
            <div class="absolute right-40 bottom-24 h-10 w-10 rounded-full bg-[#3a8bff]"></div>
          </div>
        </div>
      </section>

      <section class="border-y border-[#e6e9ef] bg-white py-10">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div
              v-for="stat in heroStats"
              :key="stat.label"
              class="metric-tile flex items-center gap-3 rounded-lg border border-[#e6e9ef] bg-white p-5"
            >
              <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-[#eaf2ff] text-[#1d6ff2]">
                <Icon :name="stat.icon" size="md" />
              </span>
              <div class="min-w-0">
                <p class="text-base font-bold text-[#0f1729]">{{ stat.value }}</p>
                <p class="mt-0.5 text-xs font-semibold text-[#98a2b3]">{{ stat.label }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="workflow" class="border-y border-[#e6e9ef] bg-[#f7f9fc] py-10">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <p class="text-center text-sm font-semibold text-[#98a2b3]">支持的开发者工作流</p>
          <div class="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
            <div
              v-for="tool in workflowBadges"
              :key="tool.name"
              class="flex h-16 items-center justify-center gap-3 rounded-lg border border-[#e6e9ef] bg-white px-4 text-sm font-semibold text-[#0f1729] transition hover:border-[#d6e4ff] hover:bg-[#eaf2ff]"
            >
              <Icon :name="tool.icon" size="sm" class="text-[#1d6ff2]" />
              {{ tool.name }}
            </div>
          </div>
        </div>
      </section>

      <section id="routing" class="bg-white py-24">
        <div class="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="grid gap-12 lg:grid-cols-[0.72fr_1.28fr] lg:items-center">
            <div>
              <p class="section-eyebrow">Routing Engine</p>
              <h2 class="mt-4 text-3xl font-bold leading-tight text-[#0f1729] sm:text-5xl">
                自动选择更稳的模型线路
              </h2>
              <p class="mt-5 text-base leading-8 text-[#475467]">
                Jlaude 把 API 接入、模型路由、用量计费和故障切换收敛到一个入口。开发者只需要维护一套配置，后面的稳定性由网关持续处理。
              </p>
            </div>

            <div class="routing-board rounded-lg border border-[#e6e9ef] bg-[#f7f9fc] p-4 sm:p-6">
              <div class="grid gap-3 md:grid-cols-5">
                <article
                  v-for="(step, index) in routingSteps"
                  :key="step.title"
                  class="routing-step relative rounded-lg border border-[#e6e9ef] bg-white p-4 shadow-sm"
                >
                  <span class="text-xs font-bold text-[#1d6ff2]">0{{ index + 1 }}</span>
                  <div class="mt-4 flex h-11 w-11 items-center justify-center rounded-lg bg-[#eaf2ff] text-[#1d6ff2]">
                    <Icon :name="step.icon" size="md" />
                  </div>
                  <h3 class="mt-5 text-sm font-semibold text-[#0f1729]">{{ step.title }}</h3>
                  <p class="mt-2 text-xs leading-6 text-[#475467]">{{ step.description }}</p>
                </article>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="guarantees" class="bg-[#f7f9fc] py-24">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="grid gap-12 lg:grid-cols-[1fr_0.9fr] lg:items-start">
            <div>
              <p class="section-eyebrow">Service Layer</p>
              <h2 class="mt-4 max-w-3xl text-3xl font-bold leading-tight text-[#0f1729] sm:text-5xl">
                为高频 AI 编码请求准备的服务层
              </h2>
            </div>
            <p class="text-base leading-8 text-[#475467]">
              首页不需要解释所有细节，但用户要一眼看懂：Jlaude 不是普通转发地址，而是围绕研发链路稳定性做的一层 API 基础设施。
            </p>
          </div>

          <div class="mt-12 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <article
              v-for="item in assuranceItems"
              :key="item.title"
              class="assurance-card rounded-lg border border-[#e6e9ef] bg-white p-6 shadow-sm transition hover:-translate-y-1 hover:border-[#d6e4ff]"
            >
              <div class="flex h-11 w-11 items-center justify-center rounded-lg bg-[#eaf2ff] text-[#1d6ff2]">
                <Icon :name="item.icon" size="md" />
              </div>
              <h3 class="mt-5 text-lg font-semibold text-[#0f1729]">{{ item.title }}</h3>
              <p class="mt-3 text-sm leading-7 text-[#475467]">{{ item.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section id="faq" class="bg-white py-24">
        <div class="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8">
          <div class="text-center">
            <p class="section-eyebrow mx-auto">FAQ</p>
            <h2 class="mt-4 text-3xl font-bold text-[#0f1729] sm:text-5xl">
              常见问题
            </h2>
          </div>

          <div class="mt-10 divide-y divide-[#e6e9ef] rounded-lg border border-[#e6e9ef] bg-white">
            <article v-for="(item, index) in faqs" :key="item.question">
              <button
                type="button"
                class="flex w-full items-center justify-between gap-4 px-5 py-5 text-left"
                @click="activeFaq = activeFaq === index ? -1 : index"
              >
                <span class="text-base font-semibold text-[#0f1729]">{{ item.question }}</span>
                <Icon
                  name="chevronDown"
                  size="sm"
                  class="shrink-0 text-[#98a2b3] transition"
                  :class="activeFaq === index ? 'rotate-180' : ''"
                />
              </button>
              <div v-if="activeFaq === index" class="px-5 pb-5 text-sm leading-7 text-[#475467]">
                {{ item.answer }}
              </div>
            </article>
          </div>
        </div>
      </section>

      <section class="bg-white pb-24">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="final-cta overflow-hidden rounded-lg border border-[#d6e4ff] bg-[#eaf2ff] px-6 py-12 sm:px-10 lg:px-14">
            <div class="grid gap-8 lg:grid-cols-[1fr_auto] lg:items-center">
              <div>
                <p class="section-eyebrow">Start</p>
                <h2 class="mt-4 text-3xl font-bold text-[#0f1729] sm:text-5xl">
                  用一条线路接住你的 AI 开发工作流
                </h2>
                <p class="mt-4 max-w-2xl text-base leading-8 text-[#475467]">
                  个人开发、团队协作、企业交付，都从统一 Base URL 开始。
                </p>
              </div>
              <div class="flex flex-col gap-3 sm:flex-row">
                <router-link
                  :to="isAuthenticated ? dashboardPath : '/login'"
                  class="inline-flex items-center justify-center gap-2 rounded-lg bg-[#1d6ff2] px-6 py-3 text-sm font-semibold text-white shadow-sm transition hover:bg-[#1551c4]"
                >
                  {{ isAuthenticated ? '进入控制台' : '开始接入' }}
                  <Icon name="arrowRight" size="sm" />
                </router-link>
                <button
                  type="button"
                  class="inline-flex items-center justify-center gap-2 rounded-lg border border-[#d6e4ff] bg-white px-6 py-3 text-sm font-semibold text-[#0f1729] transition hover:bg-[#f7f9fc]"
                  @click="openContact"
                >
                  联系支持
                  <Icon name="chat" size="sm" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="border-t border-[#e6e9ef] bg-white py-12">
      <div class="mx-auto grid max-w-7xl gap-10 px-4 sm:px-6 lg:grid-cols-[1.1fr_1fr] lg:px-8">
        <div>
          <div class="flex items-center gap-3">
            <span class="brand-mark flex h-10 w-10 items-center justify-center overflow-hidden rounded-lg">
              <img
                v-if="siteLogo"
                :src="siteLogo"
                :alt="`${brandName} logo`"
                class="h-full w-full object-contain"
              />
              <span v-else class="font-black text-white">J</span>
            </span>
            <span class="text-lg font-bold text-[#0f1729]">{{ brandName }}</span>
          </div>
          <p class="mt-5 max-w-xl text-sm leading-7 text-[#475467]">
            面向开发者与团队的 AI API 网关，连接 Claude Code、Codex、OpenAI 兼容接口与常见 AI 客户端。
          </p>
          <p class="mt-6 text-sm text-[#98a2b3]">
            © {{ currentYear }} {{ brandName }}. 保留所有权利。
          </p>
        </div>

        <div class="grid gap-8 sm:grid-cols-3">
          <div>
            <h3 class="text-sm font-bold text-[#0f1729]">产品</h3>
            <div class="mt-4 grid gap-3 text-sm text-[#475467]">
              <button type="button" class="text-left hover:text-[#0f1729]" @click="scrollToSection('routing')">智能路由</button>
              <button type="button" class="text-left hover:text-[#0f1729]" @click="scrollToSection('guarantees')">服务保障</button>
              <button type="button" class="text-left hover:text-[#0f1729]" @click="scrollToSection('faq')">常见问题</button>
            </div>
          </div>
          <div>
            <h3 class="text-sm font-bold text-[#0f1729]">资源</h3>
            <div class="mt-4 grid gap-3 text-sm text-[#475467]">
              <a :href="docUrl" target="_blank" rel="noopener noreferrer" class="hover:text-[#0f1729]">使用文档</a>
              <router-link to="/legal/privacy" class="hover:text-[#0f1729]">隐私政策</router-link>
              <router-link to="/legal/terms" class="hover:text-[#0f1729]">用户协议</router-link>
            </div>
          </div>
          <div>
            <h3 class="text-sm font-bold text-[#0f1729]">服务</h3>
            <div class="mt-4 grid gap-3 text-sm text-[#475467]">
              <router-link :to="dashboardPath" class="hover:text-[#0f1729]">控制台</router-link>
              <router-link to="/token-merchant" class="hover:text-[#0f1729]">成为合伙人</router-link>
              <button type="button" class="text-left hover:text-[#0f1729]" @click="openContact">联系支持</button>
              <button type="button" class="text-left hover:text-[#0f1729]" @click="copyBaseUrl">复制 Base URL</button>
            </div>
          </div>
        </div>
      </div>
    </footer>

    <div
      v-if="contactModalOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 backdrop-blur-sm"
      @click.self="contactModalOpen = false"
    >
      <div class="w-full max-w-md rounded-lg border border-[#e6e9ef] bg-white p-6 shadow-xl">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="text-xl font-bold text-[#0f1729]">联系 Jlaude</h2>
            <p class="mt-2 text-sm leading-6 text-[#475467]">
              咨询额度、团队接入或企业合作。
            </p>
          </div>
          <button
            type="button"
            class="rounded-lg p-2 text-[#98a2b3] hover:bg-[#f3f5f9] hover:text-[#0f1729]"
            @click="contactModalOpen = false"
          >
            <Icon name="x" size="md" />
          </button>
        </div>

        <div v-if="contactImageUrl" class="mt-6 flex justify-center">
          <img
            :src="contactImageUrl"
            alt="Jlaude 客服二维码"
            class="max-h-72 rounded-lg border border-[#e6e9ef] object-contain"
          />
        </div>

        <div class="mt-6 whitespace-pre-wrap rounded-lg border border-[#e6e9ef] bg-[#f7f9fc] p-4 text-sm leading-7 text-[#475467]">
          {{ contactInfo || '加入用户群或咨询企业方案，请联系 Jlaude 客服。管理员可以在后台配置 QQ 群、入群说明或客服二维码。' }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAppStore, useAuthStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'

type IconName =
  | 'arrowRight'
  | 'book'
  | 'chart'
  | 'chat'
  | 'checkCircle'
  | 'chevronDown'
  | 'clock'
  | 'copy'
  | 'cpu'
  | 'database'
  | 'globe'
  | 'key'
  | 'lock'
  | 'server'
  | 'shield'
  | 'sparkles'
  | 'sync'
  | 'terminal'
  | 'users'
  | 'x'

interface IconTextItem {
  icon: IconName
  title: string
  description: string
}

interface MetricItem {
  icon: IconName
  value: string
  label: string
}

interface WorkflowBadge {
  icon: IconName
  name: string
}

const authStore = useAuthStore()
const appStore = useAppStore()

const brandName = 'Jlaude'
const apiBaseUrl = 'https://jlaudeapi.com'

const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || 'https://docs.jlaudeapi.com')
const contactInfo = computed(() => appStore.cachedPublicSettings?.contact_info || appStore.contactInfo || '')
const contactImageUrl = computed(() => appStore.cachedPublicSettings?.contact_image_url || appStore.contactImageUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isScrolled = ref(false)
const mobileMenuOpen = ref(false)
const contactModalOpen = ref(false)
const activeFaq = ref(0)

const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => {
  if (authStore.isAdmin) return '/admin/dashboard'
  if (authStore.isSales) return '/sales/dashboard'
  return '/dashboard'
})
const currentYear = computed(() => new Date().getFullYear())

const navItems = [
  { id: 'workflow', label: '工作流' },
  { id: 'routing', label: '路由' },
  { id: 'guarantees', label: '保障' },
  { id: 'faq', label: 'FAQ' }
]

const heroStats: MetricItem[] = [
  { icon: 'globe', value: '统一 Base URL', label: '一处配置接入所有工具' },
  { icon: 'sync', value: 'Claude / OpenAI', label: '双协议兼容接入' },
  { icon: 'shield', value: '故障自动切换', label: '异常线路即时降级' },
  { icon: 'chart', value: '透明计费', label: '按模型与用量回执' }
]

const workflowBadges: WorkflowBadge[] = [
  { icon: 'terminal', name: 'Claude Code' },
  { icon: 'cpu', name: 'Codex' },
  { icon: 'sparkles', name: 'Open code' },
  { icon: 'database', name: 'Open claw' },
  { icon: 'sync', name: 'Hermes' },
  { icon: 'server', name: 'Cursor' }
]

const routingSteps: IconTextItem[] = [
  {
    icon: 'terminal',
    title: '客户端请求',
    description: 'Claude Code、Codex 与 OpenAI 兼容工具使用统一入口。'
  },
  {
    icon: 'key',
    title: '权限校验',
    description: '统一识别用户、密钥、额度、分组与访问策略。'
  },
  {
    icon: 'sync',
    title: '模型路由',
    description: '根据模型、线路状态与策略选择更合适的上游。'
  },
  {
    icon: 'shield',
    title: '故障切换',
    description: '异常线路自动降级，减少开发流程被打断的概率。'
  },
  {
    icon: 'chart',
    title: '透明回执',
    description: '请求状态、消耗、延迟与扣费明细回到控制台。'
  }
]

const assuranceItems: IconTextItem[] = [
  {
    icon: 'server',
    title: '多源冗余',
    description: '接入多路可用资源，围绕高频 AI 编码请求做连续性保障。'
  },
  {
    icon: 'globe',
    title: '低延迟线路',
    description: '面向国内开发环境优化访问体验，减少工具等待和上下文中断。'
  },
  {
    icon: 'lock',
    title: '最小化留存',
    description: '平台聚焦认证、路由与计费，默认不把请求内容作为产品数据沉淀。'
  },
  {
    icon: 'chart',
    title: '用量可追踪',
    description: '按模型、用户、分组和请求维度查看消耗，团队成本更容易复盘。'
  },
  {
    icon: 'users',
    title: '团队协作',
    description: '支持多人接入、额度分配和权限管理，避免团队各自维护零散配置。'
  },
  {
    icon: 'chat',
    title: '人工支持',
    description: '遇到链路、模型、扣费或接入问题，可通过后台配置的客服渠道联系。'
  }
]

const faqs = [
  {
    question: 'Jlaude 是什么？',
    answer: 'Jlaude 是面向开发者与团队的 AI API 网关，提供统一 Base URL、API Key 管理、模型路由、用量计费和常见 AI 编程工具接入支持。'
  },
  {
    question: 'Claude Code 和 OpenAI 兼容客户端怎么配置？',
    answer: 'Claude Code 等 Anthropic 兼容工具通常使用 https://jlaudeapi.com；OpenAI 兼容客户端通常使用 https://jlaudeapi.com/v1。具体以工具要求的协议类型为准。'
  },
  {
    question: '是否支持团队使用？',
    answer: '支持。团队可以通过控制台统一管理用户、额度、分组、密钥和调用记录，降低多人协作时的配置和成本管理压力。'
  },
  {
    question: '首页为什么没有价格方案？',
    answer: 'Jlaude 当前把价格、模型和分组倍率放在控制台与模型广场中展示，首页只负责说明产品定位和接入入口。'
  }
]

function onScroll() {
  isScrolled.value = window.scrollY > 12
}

function scrollToSection(id: string) {
  mobileMenuOpen.value = false
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function openContact() {
  mobileMenuOpen.value = false
  contactModalOpen.value = true
}

async function copyBaseUrl() {
  try {
    await navigator.clipboard.writeText(apiBaseUrl)
    appStore.showSuccess('已复制 Base URL')
  } catch {
    appStore.showError('复制失败')
  }
}

function setLandingMeta() {
  document.title = 'Jlaude - 高速稳定的 AI API 网关'
  setMeta('theme-color', '#1d6ff2')
  setMeta('description', 'Jlaude 是面向开发者与团队的 AI API 网关，支持 Claude Code、Codex、OpenAI 兼容接口等常见 AI 工具接入，提供统一 Base URL、透明计费与持续服务支持。')
  setMeta('keywords', 'AI API网关,AI中转站,Claude Code中转站,Codex API,OpenAI API中转,Jlaude')
}

function setMeta(name: string, content: string) {
  let meta = document.head.querySelector(`meta[name="${name}"]`)
  if (!meta) {
    meta = document.createElement('meta')
    meta.setAttribute('name', name)
    document.head.appendChild(meta)
  }
  meta.setAttribute('content', content)
}

onMounted(() => {
  onScroll()
  setLandingMeta()
  window.addEventListener('scroll', onScroll, { passive: true })
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
})
</script>

<style scoped>
.h-18 {
  height: 4.5rem;
}

.landing-shell {
  font-feature-settings: "rlig" 1, "calt" 1;
}

.brand-mark {
  background: linear-gradient(145deg, #3a8bff, #1551c4);
  box-shadow: 0 6px 16px rgba(29, 111, 242, 0.22);
}

.hero-stage {
  background: linear-gradient(180deg, #ffffff 0%, #f7f9fc 100%);
}

.section-eyebrow {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  border: 1px solid #d6e4ff;
  border-radius: 999px;
  background: #eaf2ff;
  padding: 0.375rem 0.75rem;
  color: #1d6ff2;
  font-size: 0.75rem;
  font-weight: 700;
}

.base-url-bar {
  min-height: 5rem;
}

.metric-tile {
  min-height: 5.875rem;
}

/* 几何插画：纯 CSS 形状，无发光 */
.hero-geo-ring {
  border: 1.5px solid #d6e4ff;
  border-radius: 999px;
}

.hero-geo-grid {
  background-image: radial-gradient(#d6e4ff 1.4px, transparent 1.4px);
  background-size: 22px 22px;
}

@media (max-width: 1023px) {
  .hero-geo-stage {
    display: none;
  }

  .base-url-bar {
    min-height: 4.5rem;
  }
}
</style>
