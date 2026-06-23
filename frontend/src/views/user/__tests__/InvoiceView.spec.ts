import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

import InvoiceView from '../InvoiceView.vue'

const { paymentAPI, showError, showSuccess } = vi.hoisted(() => ({
  paymentAPI: {
    getInvoiceSummary: vi.fn(),
    getInvoiceProfiles: vi.fn(),
    getInvoiceRequests: vi.fn(),
    createInvoiceProfile: vi.fn(),
    updateInvoiceProfile: vi.fn(),
    deleteInvoiceProfile: vi.fn(),
    createInvoiceRequest: vi.fn(),
  },
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

const messages: Record<string, string> = {
  'common.all': '全部',
  'common.cancel': '取消',
  'common.delete': '删除',
  'common.edit': '编辑',
  'common.error': '错误',
  'common.loading': '加载中',
  'common.processing': '处理中',
  'common.refresh': '刷新',
  'common.save': '保存',
  'common.selectOption': '请选择',
  'pagination.next': '下一页',
  'pagination.of': '共',
  'pagination.pageOf': '第 {page} / {total} 页',
  'pagination.perPage': '每页',
  'pagination.previous': '上一页',
  'pagination.results': '条',
  'pagination.showing': '显示',
  'pagination.to': '至',
  'payment.invoice.addProfile': '新增抬头',
  'payment.invoice.addressPhone': '地址及电话',
  'payment.invoice.addressPhonePlaceholder': '注册地址及联系电话',
  'payment.invoice.applicationAmount': '开票金额',
  'payment.invoice.applicationDialogTitle': '申请开票',
  'payment.invoice.applicationEmail': '收票邮箱',
  'payment.invoice.apply': '申请开票',
  'payment.invoice.availableAmount': '可开票额度',
  'payment.invoice.bankAccount': '开户行及账号',
  'payment.invoice.bankAccountPlaceholder': '开户银行及银行账号',
  'payment.invoice.companyTaxNoHint': '企业抬头必须填写纳税人识别号。',
  'payment.invoice.content': '开票内容',
  'payment.invoice.contentPlaceholder': '例如：信息技术服务',
  'payment.invoice.defaultProfile': '默认',
  'payment.invoice.deleteProfile': '删除抬头',
  'payment.invoice.deleteProfileConfirm': '确定要删除“{name}”吗？',
  'payment.invoice.description': '查看可开票金额，管理发票抬头，并提交开票申请',
  'payment.invoice.disabledHint': '自助开票暂未开放，请联系管理员。',
  'payment.invoice.editProfile': '编辑抬头',
  'payment.invoice.email': '收票邮箱',
  'payment.invoice.emailPlaceholder': 'name@example.com',
  'payment.invoice.emptyHistory': '暂无开票申请',
  'payment.invoice.emptyHistoryHint': '满足最低开票金额后，可以在此提交开票申请。',
  'payment.invoice.history': '申请历史',
  'payment.invoice.historyHint': '开票申请提交后会记录处理状态和发票文件。',
  'payment.invoice.invoiceNo': '发票号码',
  'payment.invoice.invoicedAmount': '已开票额度',
  'payment.invoice.manageProfiles': '发票抬头',
  'payment.invoice.minAmountHint': '单次最低开票金额：{amount}',
  'payment.invoice.noProfiles': '暂无发票抬头，请先添加一个抬头。',
  'payment.invoice.profileDialogTitle': '发票抬头管理',
  'payment.invoice.remark': '留言/备注',
  'payment.invoice.remarkPlaceholder': '如有特殊要求可填写在这里',
  'payment.invoice.reservedAmount': '处理中额度',
  'payment.invoice.selectProfile': '选择发票抬头',
  'payment.invoice.selectProfilePlaceholder': '请选择抬头',
  'payment.invoice.setDefault': '设为默认抬头',
  'payment.invoice.status.failed': '失败',
  'payment.invoice.status.issued': '已开票',
  'payment.invoice.status.issuing': '开票中',
  'payment.invoice.status.pending': '待处理',
  'payment.invoice.status.requires_auth': '需认证',
  'payment.invoice.submitApplication': '提交申请',
  'payment.invoice.submitting': '提交中...',
  'payment.invoice.table.amount': '金额',
  'payment.invoice.table.content': '内容/备注',
  'payment.invoice.table.createdAt': '提交时间',
  'payment.invoice.table.files': '文件',
  'payment.invoice.table.status': '状态',
  'payment.invoice.table.title': '抬头',
  'payment.invoice.taxNo': '纳税人识别号',
  'payment.invoice.taxNoPlaceholder': '企业抬头需填写税号',
  'payment.invoice.title': '发票管理',
  'payment.invoice.titleName': '发票抬头',
  'payment.invoice.titleNamePlaceholder': '请输入个人姓名或公司名称',
  'payment.invoice.titleType': '抬头类型',
  'payment.invoice.titleTypes.company': '企业',
  'payment.invoice.titleTypes.personal': '个人',
  'payment.invoice.totalPaid': '累计已支付',
}

vi.mock('@/api/payment', () => ({ paymentAPI }))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string, params?: Record<string, string | number>) => {
        const message = messages[key] ?? key
        return Object.entries(params ?? {}).reduce(
          (acc, [name, value]) => acc.replace(`{${name}}`, String(value)),
          message,
        )
      },
    }),
  }
})

vi.mock('@/composables/useGsapMotion', async () => {
  const actual = await vi.importActual<typeof import('@/composables/useGsapMotion')>(
    '@/composables/useGsapMotion',
  )
  return {
    ...actual,
    animateMountedSurface: vi.fn(),
    clearMotion: vi.fn(),
  }
})

const profile = {
  id: 7,
  title_type: 'company',
  name: '测试公司',
  tax_no: '91300000000000000X',
  address_phone: '',
  bank_account: '',
  email: 'invoice@example.com',
  is_default: true,
  created_at: '2026-06-23T08:00:00Z',
  updated_at: '2026-06-23T08:00:00Z',
}

const request = {
  id: 11,
  profile_id: 7,
  status: 'ISSUED',
  amount: 128,
  paid_total: 500,
  invoiced_total: 128,
  reserved_total: 0,
  available_amount: 372,
  currency: 'CNY',
  title_type: 'company',
  title_name: '测试公司',
  tax_no: '91300000000000000X',
  address_phone: '',
  bank_account: '',
  email: 'invoice@example.com',
  content: '信息技术服务',
  remark: '请尽快开票',
  sdk_message: '',
  invoice_no: 'FP202606230001',
  invoice_date: '2026-06-23',
  pdf_url: '',
  ofd_url: '',
  xml_url: '',
  issued_at: '2026-06-23T08:10:00Z',
  created_at: '2026-06-23T08:00:00Z',
  updated_at: '2026-06-23T08:10:00Z',
}

describe('InvoiceView', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    vi.clearAllMocks()

    paymentAPI.getInvoiceSummary.mockResolvedValue({
      data: {
        total_paid: 500,
        invoiced_amount: 128,
        reserved_amount: 0,
        available_amount: 372,
        min_amount: 100,
        currency: 'CNY',
        enabled: true,
      },
    })
    paymentAPI.getInvoiceProfiles.mockResolvedValue({ data: [profile] })
    paymentAPI.getInvoiceRequests.mockResolvedValue({
      data: {
        items: [request],
        total: 1,
        page: 1,
        page_size: 20,
        pages: 1,
      },
    })
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders invoice history and opens the invoice title dialog', async () => {
    const wrapper = mount(InvoiceView, {
      attachTo: document.body,
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('测试公司')
    expect(wrapper.text()).toContain('信息技术服务')
    expect(showError).not.toHaveBeenCalled()

    const profileButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('发票抬头'))

    expect(profileButton).toBeTruthy()
    await profileButton!.trigger('click')
    await nextTick()

    expect(document.body.textContent).toContain('发票抬头管理')
    expect(document.body.textContent).toContain('新增抬头')
  })

  it('also accepts already-unwrapped invoice API data', async () => {
    paymentAPI.getInvoiceSummary.mockResolvedValue({
      total_paid: 500,
      invoiced_amount: 128,
      reserved_amount: 0,
      available_amount: 372,
      min_amount: 100,
      currency: 'CNY',
      enabled: true,
    })
    paymentAPI.getInvoiceProfiles.mockResolvedValue([profile])
    paymentAPI.getInvoiceRequests.mockResolvedValue({
      items: [request],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mount(InvoiceView, {
      attachTo: document.body,
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('测试公司')
    expect(wrapper.text()).toContain('信息技术服务')
    expect(showError).not.toHaveBeenCalled()
  })
})
