import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import PluginCenterView from '../PluginCenterView.vue'

const { listPublic } = vi.hoisted(() => ({
  listPublic: vi.fn()
}))

vi.mock('@/api/plugins', () => ({
  default: {
    listPublic
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (!params) return key
        return `${key}:${JSON.stringify(params)}`
      }
    })
  }
})

describe('PluginCenterView', () => {
  beforeEach(() => {
    listPublic.mockReset()
    listPublic.mockResolvedValue([
      {
        id: 1,
        name: 'Claude Code Helper',
        description: 'Install helper',
        version: 'v1.0.0',
        category: 'Claude Code',
        platform: 'all',
        icon_url: '',
        file_name: 'helper.zip',
        file_size: 1024,
        download_count: 8,
        download_url: '/api/v1/plugins/1/download'
      },
      {
        id: 2,
        name: 'Codex Helper',
        description: 'Codex workflow helper',
        version: 'v2.0.0',
        category: 'Codex',
        platform: 'linux',
        icon_url: '',
        file_name: 'codex.zip',
        file_size: 2048,
        download_count: 3,
        download_url: '/api/v1/plugins/2/download'
      }
    ])
  })

  it('renders public plugin cards and filters by category', async () => {
    const wrapper = mount(PluginCenterView, {
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Claude Code Helper')
    expect(wrapper.text()).toContain('Codex Helper')

    await wrapper.findAll('button').find((button) => button.text() === 'Claude Code')?.trigger('click')

    expect(wrapper.text()).toContain('Claude Code Helper')
    expect(wrapper.text()).not.toContain('Codex Helper')
  })
})
