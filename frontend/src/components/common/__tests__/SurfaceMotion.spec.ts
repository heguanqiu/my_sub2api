import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import EmptyState from '@/components/common/EmptyState.vue'
import StatCard from '@/components/common/StatCard.vue'

const motion = vi.hoisted(() => ({
  animateMountedSurface: vi.fn(),
  animateHoverLift: vi.fn(),
  clearMotion: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/composables/useGsapMotion', () => motion)

describe('common surface motion', () => {
  afterEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = ''
  })

  it('animates EmptyState motion items on mount and clears them on unmount', () => {
    const wrapper = mount(EmptyState, {
      attachTo: document.body,
      props: {
        title: 'Nothing here',
        description: 'Create something to get started.',
      },
      global: {
        stubs: {
          Icon: true,
          RouterLink: true,
        },
      },
    })

    expect(motion.animateMountedSurface).toHaveBeenCalledWith(
      expect.any(HTMLDivElement),
      '.empty-state-motion-item'
    )

    const root = wrapper.get('.empty-state').element
    const motionItems = Array.from(root.querySelectorAll('.empty-state-motion-item'))

    wrapper.unmount()

    expect(motion.clearMotion).toHaveBeenCalledWith([root, ...motionItems])
  })

  it('animates StatCard mount and hover lift, then clears motion targets on unmount', async () => {
    const wrapper = mount(StatCard, {
      attachTo: document.body,
      props: {
        title: 'Requests',
        value: 42,
        change: 8,
        changeType: 'up',
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const element = wrapper.get('.stat-card').element

    expect(motion.animateMountedSurface).toHaveBeenCalledWith(
      expect.any(HTMLDivElement),
      '.stat-card-motion-item'
    )

    await wrapper.get('.stat-card').trigger('mouseenter')
    expect(motion.animateHoverLift).toHaveBeenCalledWith(element, true)

    await wrapper.get('.stat-card').trigger('mouseleave')
    expect(motion.animateHoverLift).toHaveBeenCalledWith(element, false)

    const motionItems = Array.from(element.querySelectorAll('.stat-card-motion-item'))

    wrapper.unmount()

    expect(motion.clearMotion).toHaveBeenCalledWith([element, ...motionItems])
  })
})
