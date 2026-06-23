import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import PluginsView from '../PluginsView.vue'

const { listPlugins, createPlugin, updatePlugin, deletePlugin, uploadPlugin, showSuccess, showError } =
  vi.hoisted(() => ({
    listPlugins: vi.fn(),
    createPlugin: vi.fn(),
    updatePlugin: vi.fn(),
    deletePlugin: vi.fn(),
    uploadPlugin: vi.fn(),
    showSuccess: vi.fn(),
    showError: vi.fn()
  }))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    plugins: {
      list: listPlugins,
      create: createPlugin,
      update: updatePlugin,
      deletePlugin,
      upload: uploadPlugin
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError
  })
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

const DataTableStub = {
  props: ['columns', 'data'],
  emits: ['sort'],
  template: `
    <table>
      <thead>
        <tr>
          <th v-for="column in columns" :key="column.key">{{ column.label }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in data" :key="row.id">
          <td v-for="column in columns" :key="column.key">
            <slot :name="'cell-' + column.key" :row="row" :value="row[column.key]">
              {{ row[column.key] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  `
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  setup(props: { options: Array<{ value: unknown; label: string }> }, { emit }: { emit: (event: string, ...args: unknown[]) => void }) {
    const onChange = (event: Event) => {
      const raw = (event.target as HTMLSelectElement).value
      const option = props.options.find((item) => String(item.value ?? '') === raw)
      const value = option ? option.value : raw
      emit('update:modelValue', value)
      emit('change', value, option ?? null)
    }
    return { onChange }
  },
  template: `
    <select :value="modelValue ?? ''" @change="onChange">
      <option v-for="option in options" :key="String(option.value ?? '')" :value="option.value ?? ''">
        {{ option.label }}
      </option>
    </select>
  `
}

function mountView() {
  return mount(PluginsView, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: {
          props: ['show', 'title'],
          template: '<section v-if="show"><h2>{{ title }}</h2><slot /><slot name="footer" /></section>'
        },
        ConfirmDialog: {
          props: ['show'],
          emits: ['confirm', 'cancel'],
          template: '<button v-if="show" data-test="confirm-delete" @click="$emit(\'confirm\')">confirm</button>'
        },
        EmptyState: true,
        Select: SelectStub,
        Icon: true,
        Teleport: true
      }
    }
  })
}

describe('admin PluginsView', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    listPlugins.mockReset()
    createPlugin.mockReset()
    updatePlugin.mockReset()
    deletePlugin.mockReset()
    uploadPlugin.mockReset()
    showSuccess.mockReset()
    showError.mockReset()

    listPlugins.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'Claude Code Helper',
          description: 'IDE helper',
          version: 'v1.0.0',
          category: 'Claude Code',
          platform: 'all',
          icon_key: 'plugins/icons/icon.png',
          file_key: 'plugins/files/helper.zip',
          file_name: 'helper.zip',
          file_size: 1024,
          download_count: 3,
          status: 'published',
          sort_weight: 10,
          created_at: '2026-06-18T00:00:00Z',
          updated_at: '2026-06-18T00:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    createPlugin.mockResolvedValue({ id: 2 })
    updatePlugin.mockResolvedValue({ id: 1 })
    deletePlugin.mockResolvedValue({ message: 'deleted' })
    uploadPlugin.mockResolvedValue({
      key: 'plugins/files/uploaded.zip',
      file_name: 'uploaded.zip',
      size: 2048
    })
  })

  it('renders list rows', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Claude Code Helper')
  })

  it('submits create payload', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="open-create-plugin"]').trigger('click')
    await wrapper.get('[data-test="plugin-name-input"]').setValue('Codex Tool')
    await wrapper.get('[data-test="plugin-version-input"]').setValue('v2.0.0')
    await wrapper.get('[data-test="plugin-description-input"]').setValue('Command helper')
    await wrapper.get('[data-test="plugin-category-input"]').setValue('Codex')
    await wrapper.get('[data-test="plugin-form"]').trigger('submit')
    await flushPromises()

    expect(createPlugin).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Codex Tool',
      version: 'v2.0.0',
      description: 'Command helper',
      category: 'Codex',
      platform: 'all',
      status: 'draft'
    }))
  })

  it('upload sets package metadata before save', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="open-create-plugin"]').trigger('click')
    await wrapper.get('[data-test="plugin-name-input"]').setValue('Uploaded Tool')
    const file = new File(['zip'], 'uploaded.zip', { type: 'application/zip' })
    const input = wrapper.get<HTMLInputElement>('[data-test="package-upload"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [file]
    })
    await input.trigger('change')
    await flushPromises()
    await wrapper.get('[data-test="plugin-form"]').trigger('submit')
    await flushPromises()

    expect(uploadPlugin).toHaveBeenCalledWith('package', file)
    expect(createPlugin).toHaveBeenCalledWith(expect.objectContaining({
      file_key: 'plugins/files/uploaded.zip',
      file_name: 'uploaded.zip',
      file_size: 2048
    }))
  })

  it('submits remote resource URL without uploading a package', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="open-create-plugin"]').trigger('click')
    await wrapper.get('[data-test="plugin-name-input"]').setValue('Remote Tool')
    await wrapper.get('[data-test="plugin-source-remote"]').trigger('click')
    await wrapper.get('[data-test="plugin-resource-url-input"]').setValue('https://downloads.example.com/tools/remote-tool.zip?token=abc')
    await wrapper.get('[data-test="plugin-form"]').trigger('submit')
    await flushPromises()

    expect(uploadPlugin).not.toHaveBeenCalled()
    expect(createPlugin).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Remote Tool',
      file_key: 'https://downloads.example.com/tools/remote-tool.zip?token=abc',
      file_name: 'remote-tool.zip',
      file_size: 0
    }))
  })

  it('rejects non-http remote resource URLs', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="open-create-plugin"]').trigger('click')
    await wrapper.get('[data-test="plugin-name-input"]').setValue('Remote Tool')
    await wrapper.get('[data-test="plugin-source-remote"]').trigger('click')
    await wrapper.get('[data-test="plugin-resource-url-input"]').setValue('ftp://downloads.example.com/tool.zip')
    await wrapper.get('[data-test="plugin-form"]').trigger('submit')
    await flushPromises()

    expect(createPlugin).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.plugins.resourceURLInvalid')
  })

  it('delete confirmation calls api', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="delete-plugin-1"]').trigger('click')
    await wrapper.get('[data-test="confirm-delete"]').trigger('click')
    await flushPromises()

    expect(deletePlugin).toHaveBeenCalledWith(1)
    expect(showSuccess).toHaveBeenCalledWith('admin.plugins.deleted')
  })
})
