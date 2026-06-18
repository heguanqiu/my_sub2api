<template>
  <div class="partner-page min-h-screen bg-white text-[#0f1729]">
    <header class="sticky top-0 z-40 border-b border-[#e6e9ef] bg-white/95 shadow-sm backdrop-blur">
      <nav class="mx-auto flex h-18 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <router-link to="/home" class="flex min-w-0 items-center gap-3">
          <span class="brand-mark flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-sm font-black text-white">
            J
          </span>
          <span class="min-w-0">
            <span class="block text-base font-bold tracking-normal text-[#0f1729]">Jlaude</span>
            <span class="hidden text-xs text-[#98a2b3] sm:block">Partner Program</span>
          </span>
        </router-link>

        <div class="hidden items-center gap-1 md:flex">
          <button
            v-for="item in navItems"
            :key="item.id"
            type="button"
            class="rounded-lg px-3 py-2 text-sm font-semibold text-[#475467] transition hover:bg-[#f3f5f9] hover:text-[#0f1729]"
            @click="scrollToSection(item.id)"
          >
            {{ item.label }}
          </button>
        </div>

        <button
          type="button"
          class="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg bg-[#1d6ff2] px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-[#1551c4]"
          @click="startPartner"
        >
          立即加入
          <Icon name="arrowRight" size="sm" />
        </button>
      </nav>
    </header>

    <main>
      <section class="hero-stage border-b border-[#e6e9ef]">
        <div class="mx-auto grid min-h-[calc(100dvh-4.5rem)] max-w-7xl gap-12 px-4 py-16 sm:px-6 lg:grid-cols-[0.92fr_1.08fr] lg:items-center lg:px-8">
          <div class="max-w-3xl">
            <p class="partner-eyebrow" v-reveal.left="0">AI Token Partner</p>
            <h1 class="mt-6 max-w-4xl text-4xl font-black leading-tight text-[#0f1729] sm:text-6xl" v-reveal.left="120">
              不是副业，是事业
              <span class="block text-[#1d6ff2]">零成本成为 AI Token 合伙人</span>
            </h1>
            <p class="mt-6 max-w-2xl text-lg leading-8 text-[#475467]" v-reveal.left="240">
              在 AI 浪潮中建立你的商业版图。不需要资金，不需要技术，不需要囤货，今天注册，今天就能开始推广。
            </p>

            <div class="mt-9 flex flex-col gap-3 sm:flex-row" v-reveal.left="360">
              <button
                type="button"
                class="inline-flex min-h-12 items-center justify-center gap-2 rounded-lg bg-[#1d6ff2] px-6 py-3 text-base font-semibold text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-[#1551c4]"
                @click="startPartner"
              >
                立即开启我的 AI 事业
                <Icon name="arrowRight" size="md" />
              </button>
              <router-link
                to="/home"
                class="inline-flex min-h-12 items-center justify-center gap-2 rounded-lg border border-[#e6e9ef] bg-white px-6 py-3 text-base font-semibold text-[#0f1729] transition hover:-translate-y-0.5 hover:border-[#d6e4ff] hover:bg-[#f7f9fc]"
              >
                返回首页
                <Icon name="home" size="md" />
              </router-link>
            </div>
          </div>

          <div class="partner-hero-visual relative hidden h-[26rem] overflow-hidden rounded-lg border border-[#e6e9ef] bg-white shadow-sm lg:block" v-reveal.right="200">
            <div class="partner-visual-grid partner-visual-motion absolute inset-6 rounded-2xl opacity-75"></div>
            <svg
              viewBox="0 0 560 416"
              class="absolute inset-0 h-full w-full"
              aria-hidden="true"
              role="img"
            >
              <defs>
                <linearGradient id="partnerCardBlue" x1="0" y1="0" x2="1" y2="1">
                  <stop offset="0%" stop-color="#3a8bff" />
                  <stop offset="100%" stop-color="#1551c4" />
                </linearGradient>
                <linearGradient id="partnerCardTeal" x1="0" y1="0" x2="1" y2="1">
                  <stop offset="0%" stop-color="#4fd1c5" />
                  <stop offset="100%" stop-color="#0ea5a5" />
                </linearGradient>
              </defs>
              <circle class="partner-visual-motion partner-visual-ring-large" cx="390" cy="132" r="96" fill="none" stroke="#d6e4ff" stroke-width="2" />
              <circle class="partner-visual-motion partner-visual-ring-small" cx="420" cy="158" r="54" fill="none" stroke="#e9ebfb" stroke-width="2" />
              <rect class="partner-visual-motion partner-visual-card-main" x="74" y="104" width="188" height="112" rx="16" fill="#ffffff" stroke="#e6e9ef" stroke-width="2" />
              <rect x="96" y="128" width="64" height="12" rx="6" fill="#d6e4ff" />
              <rect x="96" y="154" width="126" height="10" rx="5" fill="#f3f5f9" />
              <rect x="96" y="176" width="94" height="10" rx="5" fill="#f3f5f9" />
              <rect class="partner-visual-motion partner-visual-card-alt" x="126" y="246" width="222" height="94" rx="16" fill="#eaf2ff" stroke="#d6e4ff" stroke-width="2" />
              <rect x="150" y="272" width="92" height="12" rx="6" fill="#1d6ff2" opacity="0.8" />
              <rect x="150" y="298" width="160" height="10" rx="5" fill="#ffffff" opacity="0.9" />
              <rect class="partner-visual-motion partner-visual-blue-block" x="386" y="266" width="74" height="74" rx="16" fill="url(#partnerCardBlue)" />
              <circle class="partner-visual-motion partner-visual-teal-dot" cx="356" cy="248" r="21" fill="url(#partnerCardTeal)" />
              <circle class="partner-visual-motion partner-visual-amber-dot" cx="464" cy="94" r="14" fill="#d98300" opacity="0.9" />
              <path class="partner-visual-motion partner-visual-line" d="M244 174 C304 126 354 112 422 142" fill="none" stroke="#1d6ff2" stroke-width="3" stroke-linecap="round" stroke-dasharray="8 10" />
              <path class="partner-visual-motion partner-visual-line-alt" d="M286 292 C330 302 360 294 402 260" fill="none" stroke="#12a150" stroke-width="3" stroke-linecap="round" stroke-dasharray="7 9" />
            </svg>
          </div>

          <div class="grid gap-3 lg:col-span-2 lg:grid-cols-3">
            <article
              v-for="(feature, index) in heroFeatures"
              :key="feature.title"
              v-reveal.scale="index * 90"
              class="metric-tile rounded-lg border border-[#e6e9ef] bg-white p-5 transition hover:border-[#d6e4ff] hover:bg-[#f7f9fc]"
            >
              <span
                class="partner-tone-icon flex h-11 w-11 items-center justify-center rounded-lg"
                :style="{ background: tones[feature.tone].bg, color: tones[feature.tone].fg }"
              >
                <Icon :name="feature.icon" size="md" />
              </span>
              <h2 class="mt-5 text-lg font-bold text-[#0f1729]">{{ feature.title }}</h2>
              <p class="mt-2 text-sm leading-7 text-[#475467]">{{ feature.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section id="cognition" class="border-y border-[#e6e9ef] bg-[#f7f9fc] py-20">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="mx-auto max-w-3xl text-center" v-reveal="0">
            <p class="partner-eyebrow">New Business</p>
            <h2 class="mt-4 text-3xl font-black text-[#0f1729] sm:text-5xl">
              你以为的创业 vs 真实的 AI Token 生意
            </h2>
            <p class="mt-5 text-base leading-8 text-[#475467]">打破旧认知，建立新认知。</p>
          </div>

          <div class="mt-12 grid gap-4 lg:grid-cols-2">
            <article class="rounded-lg border border-[#e6e9ef] bg-white p-6" v-reveal.left="120">
              <h3 class="text-xl font-bold text-[#98a2b3]">传统创业</h3>
              <ul class="mt-6 grid gap-3">
                <li v-for="item in traditionalItems" :key="item" class="flex gap-3 text-sm leading-7 text-[#475467]">
                  <Icon name="xCircle" size="sm" class="mt-1 shrink-0 text-[#e5484d]" />
                  <span>{{ item }}</span>
                </li>
              </ul>
            </article>

            <article class="rounded-lg border border-[#d6e4ff] bg-[#eaf2ff] p-6" v-reveal.right="220">
              <h3 class="text-xl font-bold text-[#1d6ff2]">AI Token 生意</h3>
              <ul class="mt-6 grid gap-3">
                <li v-for="item in aiTokenItems" :key="item" class="flex gap-3 text-sm leading-7 text-[#0f1729]">
                  <Icon name="checkCircle" size="sm" class="mt-1 shrink-0 text-[#12a150]" />
                  <span>{{ item }}</span>
                </li>
              </ul>
            </article>
          </div>

          <p class="mt-10 text-center text-xl font-black text-[#1d6ff2]">
            这不是“推广”，这是一门真正的生意。
          </p>
        </div>
      </section>

      <section id="identity" class="bg-white py-20">
        <div class="mx-auto grid max-w-7xl gap-10 px-4 sm:px-6 lg:grid-cols-[0.9fr_1.1fr] lg:px-8">
          <div v-reveal.left="0">
            <p class="partner-eyebrow">Identity</p>
            <h2 class="mt-4 text-3xl font-black leading-tight text-[#0f1729] sm:text-5xl">
              什么是 AI Token 合伙人？
            </h2>
            <p class="mt-5 text-base leading-8 text-[#475467]">
              AI 正在改变世界，每天都有大量用户在使用 AI 服务。这些人都需要 Token，就像用电需要电费一样，而你可以成为他们的 Token 供应商。
            </p>
          </div>

          <div class="grid gap-4">
            <article
              v-for="(item, index) in identityItems"
              :key="item.title"
              v-reveal.right="index * 90"
              class="rounded-lg border border-[#e6e9ef] bg-white p-5 transition hover:border-[#d6e4ff] hover:bg-[#f7f9fc]"
            >
              <p class="text-sm font-bold text-[#1d6ff2]">{{ item.title }}</p>
              <p class="mt-2 text-sm leading-7 text-[#475467]">{{ item.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section id="revenue" class="border-y border-[#e6e9ef] bg-[#f7f9fc] py-20">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="grid gap-10 lg:grid-cols-[0.75fr_1.25fr] lg:items-start">
            <div>
              <p class="partner-eyebrow" v-reveal.left="0">Revenue</p>
              <h2 class="mt-4 text-3xl font-black text-[#0f1729] sm:text-5xl" v-reveal.left="120">你的收益来源</h2>
              <p class="mt-5 text-base leading-8 text-[#475467]" v-reveal.left="240">
                两种方式，双重收益。客户购买月卡、使用 API 或持续消费时，你都可以获得对应回报。
              </p>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <article
                v-for="(item, index) in revenueItems"
                :key="item.title"
                v-reveal.scale="index * 80"
                class="rounded-lg border border-[#e6e9ef] bg-white p-6 transition hover:border-[#d6e4ff]"
              >
                <span
                  class="partner-tone-icon flex h-11 w-11 items-center justify-center rounded-lg"
                  :style="{ background: tones[item.tone].bg, color: tones[item.tone].fg }"
                >
                  <Icon :name="item.icon" size="md" />
                </span>
                <h3 class="mt-5 text-xl font-bold text-[#0f1729]">{{ item.title }}</h3>
                <p class="mt-3 text-sm leading-7 text-[#475467]">{{ item.description }}</p>
              </article>
            </div>
          </div>
        </div>
      </section>

      <section id="howto" class="bg-white py-20">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="mx-auto max-w-3xl text-center" v-reveal="0">
            <p class="partner-eyebrow">Start</p>
            <h2 class="mt-4 text-3xl font-black text-[#0f1729] sm:text-5xl">3 步开启你的 AI 事业</h2>
            <p class="mt-5 text-base leading-8 text-[#475467]">简单到你不敢相信。</p>
          </div>

          <div class="mt-12 grid gap-4 md:grid-cols-3">
            <article
              v-for="(step, index) in steps"
              :key="step.title"
              v-reveal.scale="index * 90"
              class="rounded-lg border border-[#e6e9ef] bg-white p-6 transition hover:border-[#d6e4ff] hover:bg-[#f7f9fc]"
            >
              <span class="text-sm font-black text-[#1d6ff2]">0{{ index + 1 }}</span>
              <h3 class="mt-4 text-xl font-bold text-[#0f1729]">{{ step.title }}</h3>
              <p class="mt-3 text-sm leading-7 text-[#475467]">{{ step.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section id="faq" class="border-y border-[#e6e9ef] bg-[#f7f9fc] py-20">
        <div class="mx-auto max-w-4xl px-4 sm:px-6 lg:px-8">
          <div class="text-center" v-reveal="0">
            <p class="partner-eyebrow">FAQ</p>
            <h2 class="mt-4 text-3xl font-black text-[#0f1729] sm:text-5xl">你可能想问的问题</h2>
          </div>

          <div class="mt-10 divide-y divide-[#e6e9ef] rounded-lg border border-[#e6e9ef] bg-white" v-reveal.scale="120">
            <article v-for="(item, index) in faqs" :key="item.question">
              <button
                type="button"
                class="flex w-full items-center justify-between gap-4 px-5 py-5 text-left transition hover:bg-[#f7f9fc]"
                @click="activeFaq = activeFaq === index ? -1 : index"
              >
                <span class="text-base font-bold text-[#0f1729]">{{ item.question }}</span>
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

      <section class="bg-[#eaf2ff] py-20">
        <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div class="grid gap-8 lg:grid-cols-[1fr_auto] lg:items-center" v-reveal="0">
            <div>
              <p class="partner-eyebrow bg-white">Join</p>
              <h2 class="mt-4 text-3xl font-black text-[#0f1729] sm:text-5xl">准备好开启你的 AI 事业了吗？</h2>
              <p class="mt-4 max-w-2xl text-base leading-8 text-[#475467]">
                不需要资金，不需要技术，不需要经验。只需要几分钟注册，今天就能开始推广。
              </p>
            </div>
            <button
              type="button"
              class="inline-flex min-h-12 items-center justify-center gap-2 rounded-lg bg-[#1d6ff2] px-6 py-3 text-base font-semibold text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-[#1551c4]"
              @click="startPartner"
            >
              立即注册，开启我的 AI 事业
              <Icon name="arrowRight" size="md" />
            </button>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'

type IconName = 'arrowRight' | 'badge' | 'bolt' | 'calculator' | 'checkCircle' | 'chevronDown' | 'dollar' | 'home' | 'sparkles' | 'users' | 'xCircle'

interface IconCard {
  icon: IconName
  title: string
  description: string
  tone: ToneName
}

type ToneName = 'blue' | 'violet' | 'teal' | 'amber' | 'rose' | 'emerald' | 'sky' | 'indigo'

const tones: Record<ToneName, { fg: string; bg: string }> = {
  blue: { fg: '#1d6ff2', bg: '#eaf2ff' },
  violet: { fg: '#6d4aff', bg: '#ece7ff' },
  teal: { fg: '#0ea5a5', bg: '#e2f7f5' },
  amber: { fg: '#d98300', bg: '#fdf0d9' },
  rose: { fg: '#e5484d', bg: '#fdeaeb' },
  emerald: { fg: '#12a150', bg: '#e3f6ea' },
  sky: { fg: '#0b8fd6', bg: '#e2f3fc' },
  indigo: { fg: '#4f5bd5', bg: '#e9ebfb' }
}

const vReveal = {
  mounted(el: HTMLElement, binding: { value?: number; modifiers: Record<string, boolean> }) {
    if (typeof window !== 'undefined'
      && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) {
      return
    }
    el.classList.add('reveal-init')
    const m = binding.modifiers
    if (m.left) el.classList.add('reveal-from-left')
    else if (m.right) el.classList.add('reveal-from-right')
    else if (m.scale) el.classList.add('reveal-from-scale')
    else if (m.down) el.classList.add('reveal-from-down')
    else el.classList.add('reveal-from-up')
    const delay = typeof binding.value === 'number' ? binding.value : 0
    el.style.transitionDelay = `${delay}ms`
    if (typeof IntersectionObserver === 'undefined') {
      el.classList.add('reveal-in')
      return
    }
    const io = new IntersectionObserver((entries, obs) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          el.classList.add('reveal-in')
          obs.unobserve(el)
        }
      })
    }, { threshold: 0.12, rootMargin: '0px 0px -8% 0px' })
    io.observe(el)
  }
}

const router = useRouter()
const authStore = useAuthStore()
const activeFaq = ref(0)

const targetPath = computed(() => (authStore.isAuthenticated ? '/affiliate' : '/register'))

const navItems = [
  { id: 'cognition', label: '认知' },
  { id: 'identity', label: '定位' },
  { id: 'revenue', label: '收益' },
  { id: 'howto', label: '开始' },
  { id: 'faq', label: 'FAQ' }
]

const heroFeatures: IconCard[] = [
  {
    icon: 'badge',
    title: '真正的事业',
    description: '不是帮别人推广，你是在经营一门面向 AI 时代的生意。',
    tone: 'blue'
  },
  {
    icon: 'bolt',
    title: '零成本启动',
    description: '不需要店面、库存或押金，一个人也可以随时随地开始。',
    tone: 'amber'
  },
  {
    icon: 'dollar',
    title: '持续收益',
    description: '客户持续使用，你就持续获得回报，长期价值比一次性成交更重要。',
    tone: 'emerald'
  }
]

const traditionalItems = [
  '需要启动资金、店面、装修和库存',
  '需要雇人、管理交付和承担经营风险',
  '投入大量时间后，失败仍可能血本无归'
]

const aiTokenItems = [
  '0 元启动，没有任何前置成本',
  '不需要技术背景，专注分享和客户连接',
  '客户越用，你越赚，收益可以长期沉淀'
]

const identityItems = [
  {
    title: 'AI Token 是新时代的基础消耗',
    description: '就像用电、用流量一样，AI 使用会持续产生 Token 消耗，需求天然高频。'
  },
  {
    title: '你连接需求与服务',
    description: '你不用自建模型和技术系统，只需要把可信的 AI 服务介绍给有需求的人。'
  },
  {
    title: '越早开始，越早积累客户资产',
    description: '客户、社群、内容和信任会随时间复利，这是合伙人计划最核心的价值。'
  }
]

const revenueItems: IconCard[] = [
  {
    icon: 'dollar',
    title: '购买收益',
    description: '客户购买套餐、充值或月卡时，你获得对应收益，后台可追踪记录。',
    tone: 'emerald'
  },
  {
    icon: 'calculator',
    title: '使用收益',
    description: '客户持续使用 API 或 AI 工具时，你可以持续获得分润。',
    tone: 'violet'
  },
  {
    icon: 'users',
    title: '口碑网络效应',
    description: '好产品会被继续推荐，你的客户网络会随口碑自然扩大。',
    tone: 'teal'
  },
  {
    icon: 'sparkles',
    title: '长期复利',
    description: '老客户持续消费，新客户不断加入，收益曲线会随着客户资产增长。',
    tone: 'sky'
  }
]

const steps = [
  {
    title: '免费注册',
    description: '创建账号，获得进入控制台和推广体系的入口。'
  },
  {
    title: '分享链接',
    description: '把服务介绍给朋友、社群、客户或内容平台上的目标用户。'
  },
  {
    title: '自动记录收益',
    description: '客户注册、购买和使用后的收益记录会在后台沉淀，方便你复盘。'
  }
]

const faqs = [
  {
    question: '真的不需要任何成本吗？',
    answer: '是的，成为合伙人不需要囤货、押金或技术投入。你只需要注册账号并开始分享。'
  },
  {
    question: '我不懂技术，能做吗？',
    answer: '可以。你不需要写代码，也不需要懂模型部署。你要做的是把 AI 服务介绍给真正需要的人。'
  },
  {
    question: '收益什么时候能看到？',
    answer: '推广相关数据和收益以后台记录为准。客户越稳定使用，你越容易看到持续收益。'
  },
  {
    question: '这是合法的吗？',
    answer: '这是常见的推广合作模式，收益来自客户真实购买和使用。具体规则以平台后台展示为准。'
  }
]

function scrollToSection(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function startPartner() {
  router.push(targetPath.value)
}

function setPartnerMeta() {
  document.title = '成为合伙人 - Jlaude'
  setMeta('description', '成为 Jlaude AI Token 合伙人，零成本启动，分享 AI API 服务并积累长期收益。')
  setMeta('keywords', 'AI Token合伙人,AI API推广,AI副业,Jlaude合伙人')
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
  setPartnerMeta()
})
</script>

<style scoped>
.h-18 {
  height: 4.5rem;
}

.partner-page {
  font-feature-settings: "rlig" 1, "calt" 1;
}

.brand-mark {
  background: linear-gradient(145deg, #3a8bff, #1551c4);
  box-shadow: 0 6px 16px rgba(29, 111, 242, 0.22);
}

.hero-stage {
  background: linear-gradient(180deg, #ffffff 0%, #f7f9fc 100%);
}

.partner-visual-grid {
  background-image: radial-gradient(#d6e4ff 1.4px, transparent 1.4px);
  background-size: 22px 22px;
}

.partner-visual-motion {
  transform-box: fill-box;
  transform-origin: center;
  will-change: transform, opacity, background-position, stroke-dashoffset;
}

@media (prefers-reduced-motion: no-preference) {
  .reveal-in .partner-visual-grid {
    animation: partner-visual-grid-drift 16s linear infinite;
  }

  .reveal-in .partner-visual-ring-large {
    animation: partner-visual-ring-float 9s ease-in-out infinite;
  }

  .reveal-in .partner-visual-ring-small {
    animation: partner-visual-ring-float 7s ease-in-out infinite reverse;
    animation-delay: -2s;
  }

  .reveal-in .partner-visual-card-main {
    animation: partner-visual-card-float 8s ease-in-out infinite;
  }

  .reveal-in .partner-visual-card-alt {
    animation: partner-visual-card-float 10s ease-in-out infinite reverse;
    animation-delay: -1.4s;
  }

  .reveal-in .partner-visual-blue-block {
    animation: partner-visual-block-bob 6s ease-in-out infinite;
  }

  .reveal-in .partner-visual-teal-dot {
    animation: partner-visual-dot-orbit 5.5s ease-in-out infinite;
  }

  .reveal-in .partner-visual-amber-dot {
    animation: partner-visual-dot-orbit 6.5s ease-in-out infinite reverse;
  }

  .reveal-in .partner-visual-line,
  .reveal-in .partner-visual-line-alt {
    animation: partner-visual-line-flow 3.4s linear infinite;
  }
}

.metric-tile {
  min-height: 10.5rem;
}

.partner-eyebrow {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  border: 1px solid #d6e4ff;
  border-radius: 999px;
  background: #eaf2ff;
  padding: 0.375rem 0.75rem;
  color: #1d6ff2;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0;
  text-transform: uppercase;
}

@keyframes partner-visual-grid-drift {
  0% {
    background-position: 0 0;
    opacity: 0.75;
  }

  50% {
    background-position: 14px 10px;
    opacity: 0.9;
  }

  100% {
    background-position: 0 0;
    opacity: 0.75;
  }
}

@keyframes partner-visual-ring-float {
  0% {
    transform: translate3d(0, 0, 0) scale(1);
  }

  50% {
    transform: translate3d(-10px, 8px, 0) scale(1.03);
  }

  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
}

@keyframes partner-visual-card-float {
  0% {
    transform: translate3d(0, 0, 0) rotate(0deg);
  }

  50% {
    transform: translate3d(0, -10px, 0) rotate(-1deg);
  }

  100% {
    transform: translate3d(0, 0, 0) rotate(0deg);
  }
}

@keyframes partner-visual-block-bob {
  0% {
    transform: translate3d(0, 0, 0) scale(1);
  }

  50% {
    transform: translate3d(8px, -12px, 0) scale(1.08);
  }

  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
}

@keyframes partner-visual-dot-orbit {
  0% {
    transform: translate3d(0, 0, 0) scale(1);
  }

  33% {
    transform: translate3d(-8px, 6px, 0) scale(0.95);
  }

  66% {
    transform: translate3d(10px, -4px, 0) scale(1.05);
  }

  100% {
    transform: translate3d(0, 0, 0) scale(1);
  }
}

@keyframes partner-visual-line-flow {
  0% {
    stroke-dashoffset: 0;
  }

  100% {
    stroke-dashoffset: -36;
  }
}

/* 渐入动画：从首页复用，进入视口时按方向滑入淡入 */
.reveal-init {
  opacity: 0;
  transition: opacity 0.65s cubic-bezier(0.22, 1, 0.36, 1),
    transform 0.65s cubic-bezier(0.22, 1, 0.36, 1);
  will-change: opacity, transform;
}

.reveal-from-up {
  transform: translateY(28px);
}

.reveal-from-down {
  transform: translateY(-28px);
}

.reveal-from-left {
  transform: translateX(-40px);
}

.reveal-from-right {
  transform: translateX(40px);
}

.reveal-from-scale {
  transform: scale(0.92);
}

.reveal-in {
  opacity: 1;
  transform: none;
}

@media (prefers-reduced-motion: reduce) {
  .reveal-init {
    opacity: 1;
    transform: none;
    transition: none;
  }

  .partner-visual-motion {
    animation: none !important;
  }

  * {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
</style>
