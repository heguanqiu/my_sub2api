import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseDialog from '@/components/common/BaseDialog.vue'

const dialogHooks = vi.hoisted(() => ({
  onEnter: vi.fn((_el: Element, done: () => void) => done()),
  onLeave: vi.fn((_el: Element, done: () => void) => done()),
}))

vi.mock('@/composables/useGsapMotion', () => ({
  createDialogTransitionHooks: () => dialogHooks,
}))

describe('BaseDialog motion', () => {
  afterEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = ''
    document.body.classList.remove('modal-open')
  })

  it('uses dialog transition hooks while preserving Escape close behavior', async () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: {
        show: true,
        title: 'Motion dialog',
      },
      slots: {
        default: '<button type="button">Focusable action</button>',
      },
    })

    expect(dialogHooks.onEnter).toHaveBeenCalled()

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('close')).toHaveLength(1)

    wrapper.unmount()
  })
})
