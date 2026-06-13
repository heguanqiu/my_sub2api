import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import Toast from '@/components/common/Toast.vue'
import { useAppStore } from '@/stores/app'

const toastHooks = vi.hoisted(() => ({
  onEnter: vi.fn((_el: Element, done: () => void) => done()),
  onLeave: vi.fn((_el: Element, done: () => void) => done()),
}))

vi.mock('@/composables/useGsapMotion', () => ({
  createToastTransitionHooks: () => toastHooks,
}))

describe('Toast motion', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = ''
  })

  it('uses toast transition hooks while preserving close behavior', async () => {
    const appStore = useAppStore()
    const wrapper = mount(Toast, {
      attachTo: document.body,
    })

    appStore.showSuccess('Motion toast', undefined)
    await wrapper.vm.$nextTick()

    expect(toastHooks.onEnter).toHaveBeenCalled()
    expect(document.body.textContent).toContain('Motion toast')

    const closeButton = document.body.querySelector('button[aria-label="Close notification"]')
    if (!(closeButton instanceof HTMLButtonElement)) {
      throw new Error('close notification button not found')
    }

    closeButton.click()
    await wrapper.vm.$nextTick()

    expect(appStore.toasts).toHaveLength(0)

    wrapper.unmount()
  })

  it('declares the toast stack move class for TransitionGroup movement', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/components/common/Toast.vue'), 'utf8')

    expect(source).toContain('move-class="toast-move"')
    expect(source).toContain('.toast-move')
  })
})
