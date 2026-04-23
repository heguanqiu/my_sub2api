import { apiClient } from './client'
import type { BasePaginationResponse } from '@/types'
import type { PaymentOrder } from '@/types/payment'

export interface InvoiceProfile {
  id: number
  user_id: number
  title: string
  tax_no?: string | null
  email?: string | null
  phone?: string | null
  address?: string | null
  bank_name?: string | null
  bank_account?: string | null
  invoice_type: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface InvoiceRequest {
  id: number
  user_id: number
  order_id: number
  profile_id: number
  status: string
  provider: string
  provider_request_id?: string | null
  provider_invoice_id?: string | null
  fail_reason?: string | null
  retry_count: number
  requested_at: string
  issued_at?: string | null
  created_at: string
  updated_at: string
}

export interface InvoiceDocument {
  id: number
  invoice_request_id: number
  invoice_no?: string | null
  invoice_code?: string | null
  file_url?: string | null
  file_type?: string | null
  raw_payload_summary?: Record<string, unknown> | null
  created_at: string
}

export interface InvoiceDetailResponse {
  invoice: InvoiceRequest
  order: PaymentOrder
  profile: InvoiceProfile
  documents: InvoiceDocument[]
}

export const invoiceAPI = {
  listProfiles() {
    return apiClient.get<InvoiceProfile[]>('/invoice-profiles')
  },
  createProfile(payload: Partial<InvoiceProfile>) {
    return apiClient.post<InvoiceProfile>('/invoice-profiles', payload)
  },
  updateProfile(id: number, payload: Partial<InvoiceProfile>) {
    return apiClient.put<InvoiceProfile>(`/invoice-profiles/${id}`, payload)
  },
  deleteProfile(id: number) {
    return apiClient.delete<{ message: string }>(`/invoice-profiles/${id}`)
  },
  setDefaultProfile(id: number) {
    return apiClient.post<InvoiceProfile>(`/invoice-profiles/${id}/set-default`)
  },
  createInvoice(payload: { order_id: number; profile_id: number }) {
    return apiClient.post<InvoiceRequest>('/invoices', payload)
  },
  listMyInvoices(params?: { page?: number; page_size?: number }) {
    return apiClient.get<BasePaginationResponse<InvoiceRequest>>('/invoices/my', { params })
  },
  getMyInvoice(id: number) {
    return apiClient.get<InvoiceDetailResponse>(`/invoices/${id}`)
  }
}
