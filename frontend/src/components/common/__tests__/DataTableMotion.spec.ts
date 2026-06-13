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

const mockDesktopViewport = (matches: boolean) => {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn((query: string) => ({
      matches: query === '(min-width: 768px)' ? matches : false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
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
    const cards = Array.from(root.querySelectorAll('.data-table-mobile-card'))

    expect(cards).toHaveLength(5)
    expect(motion.animateMountedSurface).toHaveBeenCalledWith(root, '.data-table-mobile-card')

    wrapper.unmount()

    expect(motion.clearMotion).toHaveBeenCalledWith([root, ...cards])
  })

  it('animates mobile empty state and clears it on unmount', async () => {
    mockDesktopViewport(false)

    const wrapper = mountDataTable({ data: [] })
    await flushMountedMotion()

    const root = wrapper.element
    const cards = Array.from(root.querySelectorAll('.data-table-mobile-card'))

    expect(cards).toHaveLength(1)
    expect(motion.animateMountedSurface).toHaveBeenCalledWith(root, '.data-table-mobile-card')

    wrapper.unmount()

    expect(motion.clearMotion).toHaveBeenCalledWith([root, ...cards])
  })

  it('animates mobile data row cards and clears them on unmount', async () => {
    mockDesktopViewport(false)

    const wrapper = mountDataTable()
    await flushMountedMotion()

    const root = wrapper.element
    const cards = Array.from(root.querySelectorAll('.data-table-mobile-card'))

    expect(cards).toHaveLength(rows.length)
    expect(motion.animateMountedSurface).toHaveBeenCalledWith(root, '.data-table-mobile-card')

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
    expect(wrapper.findAll('.data-table-mobile-card')).toHaveLength(0)
    expect(motion.animateMountedSurface).not.toHaveBeenCalled()

    for (const row of wrapper.findAll('tbody tr[data-row-id]')) {
      expect(row.classes()).not.toContain('data-table-mobile-card')
    }

    wrapper.unmount()
  })
})
