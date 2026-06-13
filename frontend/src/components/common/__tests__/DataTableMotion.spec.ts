import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'

const motion = vi.hoisted(() => ({
  animateMountedSurface: vi.fn(),
  clearMotion: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/composables/useGsapMotion', () => motion)

vi.mock('@tanstack/vue-virtual', async () => {
  const { computed } = await vi.importActual<typeof import('vue')>('vue')

  return {
    observeElementRect: vi.fn((_instance, cb: (rect: { width: number; height: number }) => void) => {
      cb({ width: 1024, height: 600 })
      return () => {}
    }),
    useVirtualizer: vi.fn((options) => computed(() => {
      const count = options.value.count
      const rowHeight = 56
      const virtualItems = Array.from({ length: count }, (_, index) => ({
        key: index,
        index,
        start: index * rowHeight,
        end: (index + 1) * rowHeight,
        size: rowHeight,
      }))

      return {
        getVirtualItems: () => virtualItems,
        getTotalSize: () => count * rowHeight,
        measureElement: vi.fn(),
      }
    })),
  }
})

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: 'Actions' },
]

const rows = [
  { id: 'row-1', name: 'Alpha', status: 'active' },
  { id: 'row-2', name: 'Beta', status: 'paused' },
]

const mobileMotionSelector = '.data-table-mobile-card'
const desktopViewportQuery = '(min-width: 768px)'
let desktopViewportMatches = true
let desktopViewportListeners = new Set<(event: MediaQueryListEvent) => void>()

const mockDesktopViewport = (matches: boolean) => {
  desktopViewportMatches = matches
  desktopViewportListeners = new Set()

  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn((query: string) => ({
      matches: query === desktopViewportQuery ? desktopViewportMatches : false,
      media: query,
      onchange: null,
      addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
        if (query === desktopViewportQuery) desktopViewportListeners.add(listener)
      }),
      removeEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
        desktopViewportListeners.delete(listener)
      }),
      addListener: vi.fn((listener: (event: MediaQueryListEvent) => void) => {
        if (query === desktopViewportQuery) desktopViewportListeners.add(listener)
      }),
      removeListener: vi.fn((listener: (event: MediaQueryListEvent) => void) => {
        desktopViewportListeners.delete(listener)
      }),
      dispatchEvent: vi.fn(),
    })),
  })
}

const setDesktopViewport = (matches: boolean) => {
  desktopViewportMatches = matches
  const event = { matches, media: desktopViewportQuery } as MediaQueryListEvent
  for (const listener of desktopViewportListeners) {
    listener(event)
  }
}

const mountDataTable = (props: Partial<InstanceType<typeof DataTable>['$props']> = {}) => mount(DataTable, {
  attachTo: document.body,
  props: {
    columns,
    data: rows,
    ...props,
  },
  global: {
    stubs: {
      Icon: true,
    },
  },
})

const flushMountedMotion = async () => {
  await flushPromises()
  await flushPromises()
}

describe('DataTable mobile motion', () => {
  afterEach(() => {
    vi.clearAllMocks()
    document.body.innerHTML = ''
  })

  it('animates loading mobile skeleton cards and clears them on unmount', async () => {
    mockDesktopViewport(false)

    const wrapper = mountDataTable({ loading: true })
    await flushMountedMotion()

    const root = wrapper.element
    const cards = Array.from(root.querySelectorAll(mobileMotionSelector))

    expect(cards).toHaveLength(5)
    expect(motion.animateMountedSurface).toHaveBeenCalledTimes(1)
    expect(motion.animateMountedSurface).toHaveBeenCalledWith(root, mobileMotionSelector)

    wrapper.unmount()

    expect(motion.clearMotion).toHaveBeenCalledWith([root, ...cards])
  })

  it('animates mobile empty state and clears it on unmount', async () => {
    mockDesktopViewport(false)

    const wrapper = mountDataTable({ data: [] })
    await flushMountedMotion()

    const root = wrapper.element
    const cards = Array.from(root.querySelectorAll(mobileMotionSelector))

    expect(cards).toHaveLength(1)
    expect(motion.animateMountedSurface).toHaveBeenCalledTimes(1)
    expect(motion.animateMountedSurface).toHaveBeenCalledWith(root, mobileMotionSelector)

    wrapper.unmount()

    expect(motion.clearMotion).toHaveBeenCalledWith([root, ...cards])
  })

  it('animates mobile data row cards and clears them on unmount', async () => {
    mockDesktopViewport(false)

    const wrapper = mountDataTable()
    await flushMountedMotion()

    const root = wrapper.element
    const cards = Array.from(root.querySelectorAll(mobileMotionSelector))

    expect(cards).toHaveLength(rows.length)
    expect(motion.animateMountedSurface).toHaveBeenCalledTimes(1)
    expect(motion.animateMountedSurface).toHaveBeenCalledWith(root, mobileMotionSelector)

    wrapper.unmount()

    expect(motion.clearMotion).toHaveBeenCalledWith([root, ...cards])
  })

  it('does not run mobile motion for desktop virtual table rows', async () => {
    mockDesktopViewport(true)

    const wrapper = mountDataTable()
    await flushMountedMotion()

    expect(wrapper.find('table').exists()).toBe(true)
    expect(wrapper.find('.table-wrapper').exists()).toBe(true)
    expect(wrapper.findAll('tbody tr[data-row-id]')).toHaveLength(rows.length)
    expect(wrapper.findAll(mobileMotionSelector)).toHaveLength(0)
    expect(motion.animateMountedSurface).not.toHaveBeenCalled()

    for (const row of wrapper.findAll('tbody tr[data-row-id]')) {
      expect(row.classes()).not.toContain('data-table-mobile-card')
    }

    wrapper.unmount()
  })

  it('animates same-length mobile row replacement when row keys change', async () => {
    mockDesktopViewport(false)

    const wrapper = mountDataTable()
    await flushMountedMotion()

    const root = wrapper.element
    expect(motion.animateMountedSurface).toHaveBeenCalledTimes(1)

    await wrapper.setProps({
      data: [
        { id: 'row-3', name: 'Gamma', status: 'active' },
        { id: 'row-4', name: 'Delta', status: 'paused' },
      ],
    })
    await flushMountedMotion()

    const cards = Array.from(root.querySelectorAll(mobileMotionSelector))
    expect(cards).toHaveLength(2)
    expect(cards[0].textContent).toContain('Gamma')
    expect(motion.animateMountedSurface).toHaveBeenCalledTimes(2)
    expect(motion.animateMountedSurface).toHaveBeenLastCalledWith(root, mobileMotionSelector)

    wrapper.unmount()
  })

  it('does not run stale mobile animations for rapid loading and data changes', async () => {
    mockDesktopViewport(false)

    const wrapper = mountDataTable()
    await flushMountedMotion()
    motion.animateMountedSurface.mockClear()

    void wrapper.setProps({ loading: true })
    void wrapper.setProps({ loading: false, data: [] })
    void wrapper.setProps({
      loading: false,
      data: [
        { id: 'row-final-1', name: 'Final Alpha', status: 'active' },
        { id: 'row-final-2', name: 'Final Beta', status: 'paused' },
      ],
    })
    await flushMountedMotion()

    expect(wrapper.findAll(mobileMotionSelector)).toHaveLength(2)
    expect(wrapper.text()).toContain('Final Alpha')
    expect(motion.animateMountedSurface).toHaveBeenCalledTimes(1)
    expect(motion.animateMountedSurface).toHaveBeenLastCalledWith(wrapper.element, mobileMotionSelector)

    wrapper.unmount()
  })

  it('clears active mobile card motion when switching from mobile to desktop', async () => {
    mockDesktopViewport(false)

    const wrapper = mountDataTable()
    await flushMountedMotion()

    const root = wrapper.element
    const cards = Array.from(root.querySelectorAll(mobileMotionSelector))
    motion.clearMotion.mockClear()

    setDesktopViewport(true)
    await flushMountedMotion()

    expect(wrapper.find('.table-wrapper').exists()).toBe(true)
    expect(wrapper.findAll(mobileMotionSelector)).toHaveLength(0)
    expect(motion.clearMotion).toHaveBeenCalledWith([root, ...cards])

    wrapper.unmount()
  })
})
