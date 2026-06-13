import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import Select from '@/components/common/Select.vue'

const selectHooks = vi.hoisted(() => ({
  onEnter: vi.fn((_el: Element, done: () => void) => done()),
  onLeave: vi.fn((_el: Element, done: () => void) => done()),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/composables/useGsapMotion', () => ({
  createSelectTransitionHooks: () => selectHooks,
}))

describe('Select motion', () => {
  afterEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = ''
  })

  it('uses select transition hooks while preserving option selection', async () => {
    const Wrapper = defineComponent({
      components: { Select },
      data() {
        return {
          value: null as string | null,
          options: [
            { value: 'a', label: 'Alpha' },
            { value: 'b', label: 'Beta' },
          ],
        }
      },
      template: '<Select v-model="value" :options="options" />',
    })

    const wrapper = mount(Wrapper, {
      attachTo: document.body,
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    await wrapper.get('.select-trigger').trigger('click')
    await nextTick()

    expect(selectHooks.onEnter).toHaveBeenCalled()
    expect(document.body.textContent).toContain('Alpha')

    const alphaOption = Array.from(document.body.querySelectorAll('.select-option')).find(
      (option) => option.textContent?.includes('Alpha')
    )
    if (!(alphaOption instanceof HTMLElement)) {
      throw new Error('Alpha option not found')
    }

    alphaOption.click()
    await nextTick()

    expect(wrapper.vm.value).toBe('a')

    wrapper.unmount()
  })
})
