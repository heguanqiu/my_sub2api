import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { AdminUser } from '@/types'
import UsersView from '../UsersView.vue'

const {
  listUsers,
  changeInviter,
  previewSalesOwnerMigration,
  migrateSalesOwner,
  getAllGroups,
  getBatchUsersUsage,
  listEnabledDefinitions,
  getBatchUserAttributes
} = vi.hoisted(() => ({
  listUsers: vi.fn(),
  changeInviter: vi.fn(),
  previewSalesOwnerMigration: vi.fn(),
  migrateSalesOwner: vi.fn(),
  getAllGroups: vi.fn(),
  getBatchUsersUsage: vi.fn(),
  listEnabledDefinitions: vi.fn(),
  getBatchUserAttributes: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      list: listUsers,
      changeInviter,
      previewSalesOwnerMigration,
      migrateSalesOwner,
      toggleStatus: vi.fn(),
      delete: vi.fn()
    },
    groups: {
      getAll: getAllGroups
    },
    dashboard: {
      getBatchUsersUsage
    },
    userAttributes: {
      listEnabledDefinitions,
      getBatchUserAttributes
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const createAdminUser = (): AdminUser => ({
  id: 42,
  username: 'scoped-user',
  email: 'scoped@example.com',
  role: 'user',
  balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-04-17T00:00:00Z',
  updated_at: '2026-04-17T00:00:00Z',
  notes: '',
  last_active_at: '2026-04-16T02:00:00Z',
  last_used_at: '2026-04-17T02:00:00Z',
  current_concurrency: 0
})

const createSearchResultUser = (overrides: Partial<AdminUser> = {}): AdminUser => ({
  ...createAdminUser(),
  id: overrides.id ?? 9,
  email: overrides.email ?? 'target@example.com',
  username: overrides.username ?? 'target-user',
  role: overrides.role ?? 'user',
  invited_by_user_id: overrides.invited_by_user_id ?? null,
  owner_sales_id: overrides.owner_sales_id ?? null,
  ...overrides
})

const DataTableStub = {
  props: ['columns', 'data'],
  emits: ['sort'],
  template: `
    <div>
      <div data-test="columns">{{ columns.map(col => col.key).join(',') }}</div>
      <button data-test="sort-last-used" @click="$emit('sort', 'last_used_at', 'desc')">sort</button>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-last_used_at" :value="row.last_used_at" :row="row" />
      </div>
    </div>
  `
}

describe('admin UsersView', () => {
  beforeEach(() => {
    localStorage.clear()

    listUsers.mockReset()
    changeInviter.mockReset()
    previewSalesOwnerMigration.mockReset()
    migrateSalesOwner.mockReset()
    getAllGroups.mockReset()
    getBatchUsersUsage.mockReset()
    listEnabledDefinitions.mockReset()
    getBatchUserAttributes.mockReset()

    listUsers.mockImplementation((_page, pageSize, filters) => {
      if (filters?.search === 'invite-target') {
        return Promise.resolve({
          items: [createSearchResultUser({ id: 9, email: 'invite-target@example.com', username: 'invite-target', role: 'user' })],
          total: 1,
          page: 1,
          page_size: pageSize,
          pages: 1
        })
      }

      if (filters?.search === 'sales-target' && filters?.role === 'sales') {
        return Promise.resolve({
          items: [createSearchResultUser({ id: 19, email: 'sales-target@example.com', username: 'sales-target', role: 'sales' })],
          total: 1,
          page: 1,
          page_size: pageSize,
          pages: 1
        })
      }

      return Promise.resolve({
        items: [createAdminUser()],
        total: 1,
        page: 1,
        page_size: pageSize,
        pages: 1
      })
    })
    getAllGroups.mockResolvedValue([])
    getBatchUsersUsage.mockResolvedValue({ stats: {} })
    listEnabledDefinitions.mockResolvedValue([])
    getBatchUserAttributes.mockResolvedValue({ values: {} })
    changeInviter.mockResolvedValue({ root_user_id: 42, affected_user_count: 1, affected_user_ids: [42] })
    previewSalesOwnerMigration.mockResolvedValue({ root_user_id: 42, affected_user_count: 3, affected_user_ids: [42, 43, 44], target_sales_user_id: 9 })
    migrateSalesOwner.mockResolvedValue({ root_user_id: 42, affected_user_count: 3, affected_user_ids: [42, 43, 44], target_sales_user_id: 9 })
  })

  it('shows active, used, and created activity columns in order and requests last_used_at sort', async () => {
    const wrapper = mount(UsersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          EmptyState: true,
          GroupBadge: true,
          Select: true,
          UserAttributesConfigModal: true,
          UserConcurrencyCell: true,
          UserCreateModal: true,
          UserEditModal: true,
          UserApiKeysModal: true,
          UserAllowedGroupsModal: true,
          UserBalanceModal: true,
          UserBalanceHistoryModal: true,
          GroupReplaceModal: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    await flushPromises()

    const columns = wrapper.get('[data-test="columns"]').text()
    const visibleColumns = columns.split(',')
    expect(visibleColumns.slice(-4, -1)).toEqual(['last_active_at', 'last_used_at', 'created_at'])
    expect(visibleColumns).not.toContain('last_login_at')

    await wrapper.get('[data-test="sort-last-used"]').trigger('click')
    await flushPromises()

    expect(listUsers).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        sort_by: 'last_used_at',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )
  })

  it('submits inviter migration from the inline dialog', async () => {
    const wrapper = mount(UsersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          EmptyState: true,
          GroupBadge: true,
          Select: true,
          UserAttributesConfigModal: true,
          UserConcurrencyCell: true,
          UserCreateModal: true,
          UserEditModal: true,
          UserApiKeysModal: true,
          UserAllowedGroupsModal: true,
          UserBalanceModal: true,
          UserBalanceHistoryModal: true,
          GroupReplaceModal: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    await flushPromises()

    ;(wrapper.vm as any).openChangeInviterDialog(createAdminUser())
    await flushPromises()

    const input = wrapper.get('[data-test="change-inviter-search-input"]')
    await input.setValue('invite-target')
    await wrapper.get('[data-test="change-inviter-search"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="change-inviter-result-9"]').trigger('click')
    await wrapper.get('[data-test="change-inviter-submit"]').trigger('click')
    await flushPromises()

    expect(changeInviter).toHaveBeenCalledWith(42, 9)
  })

  it('previews and submits sales owner migration from the inline dialog', async () => {
    const wrapper = mount(UsersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          EmptyState: true,
          GroupBadge: true,
          Select: true,
          UserAttributesConfigModal: true,
          UserConcurrencyCell: true,
          UserCreateModal: true,
          UserEditModal: true,
          UserApiKeysModal: true,
          UserAllowedGroupsModal: true,
          UserBalanceModal: true,
          UserBalanceHistoryModal: true,
          GroupReplaceModal: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    await flushPromises()

    ;(wrapper.vm as any).openSalesMigrationDialog(createAdminUser())
    await flushPromises()

    const input = wrapper.get('[data-test="sales-migration-search-input"]')
    await input.setValue('sales-target')
    await wrapper.get('[data-test="sales-migration-search"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="sales-migration-result-19"]').trigger('click')
    await wrapper.get('[data-test="sales-migration-preview"]').trigger('click')
    await flushPromises()
    expect(previewSalesOwnerMigration).toHaveBeenCalledWith(42, 19)
    expect(wrapper.text()).toContain('预计影响 3 个用户')

    await wrapper.get('[data-test="sales-migration-submit"]').trigger('click')
    await flushPromises()
    expect(migrateSalesOwner).toHaveBeenCalledWith(42, 19)
  })
})
