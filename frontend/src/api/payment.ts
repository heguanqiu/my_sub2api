/**
 * User Payment API endpoints
 * Handles payment operations for regular users
 */

import { apiClient } from './client'
import type {
  PaymentConfig,
  SubscriptionPlan,
  PaymentChannel,
  MethodLimitsResponse,
  CheckoutInfoResponse,
  CreateOrderRequest,
  CreateOrderResult,
  PaymentOrder,
  InvoiceSummary,
  InvoiceProfile,
  InvoiceRequest,
  InvoiceProfilePayload,
  CreateInvoiceRequestPayload
} from '@/types/payment'
import type { BasePaginationResponse } from '@/types'

export const paymentAPI = {
  /** Get payment configuration (enabled types, limits, etc.) */
  getConfig() {
    return apiClient.get<PaymentConfig>('/payment/config')
  },

  /** Get available subscription plans */
  getPlans() {
    return apiClient.get<SubscriptionPlan[]>('/payment/plans')
  },

  /** Get available payment channels */
  getChannels() {
    return apiClient.get<PaymentChannel[]>('/payment/channels')
  },

  /** Get all checkout page data in a single call */
  getCheckoutInfo() {
    return apiClient.get<CheckoutInfoResponse>('/payment/checkout-info')
  },

  /** Get payment method limits and fee rates */
  getLimits() {
    return apiClient.get<MethodLimitsResponse>('/payment/limits')
  },

  /** Create a new payment order */
  createOrder(data: CreateOrderRequest) {
    return apiClient.post<CreateOrderResult>('/payment/orders', data)
  },

  /** Get current user's orders */
  getMyOrders(params?: { page?: number; page_size?: number; status?: string }) {
    return apiClient.get<BasePaginationResponse<PaymentOrder>>('/payment/orders/my', { params })
  },

  /** Get a specific order by ID */
  getOrder(id: number) {
    return apiClient.get<PaymentOrder>(`/payment/orders/${id}`)
  },

  /** Cancel a pending order */
  cancelOrder(id: number) {
    return apiClient.post(`/payment/orders/${id}/cancel`)
  },

  /** Verify order payment status with upstream provider */
  verifyOrder(outTradeNo: string) {
    return apiClient.post<PaymentOrder>('/payment/orders/verify', { out_trade_no: outTradeNo })
  },

  /** Legacy-compatible public order lookup by out_trade_no */
  verifyOrderPublic(outTradeNo: string) {
    return apiClient.post<PaymentOrder>('/payment/public/orders/verify', { out_trade_no: outTradeNo })
  },

  /** Resolve an order from a signed resume token without auth */
  resolveOrderPublicByResumeToken(resumeToken: string) {
    return apiClient.post<PaymentOrder>('/payment/public/orders/resolve', { resume_token: resumeToken })
  },

  /** Request a refund for a completed order */
  requestRefund(id: number, data: { reason: string }) {
    return apiClient.post(`/payment/orders/${id}/refund-request`, data)
  },

  /** Get provider instance IDs that allow user refund */
  getRefundEligibleProviders() {
    return apiClient.get<{ provider_instance_ids: string[] }>('/payment/orders/refund-eligible-providers')
  },

  /** Get current user's invoice quota summary */
  getInvoiceSummary() {
    return apiClient.get<InvoiceSummary>('/payment/invoices/summary')
  },

  /** Get current user's invoice application history */
  getInvoiceRequests(params?: { page?: number; page_size?: number; status?: string }) {
    return apiClient.get<BasePaginationResponse<InvoiceRequest>>('/payment/invoices/requests', { params })
  },

  /** Submit a new invoice application */
  createInvoiceRequest(data: CreateInvoiceRequestPayload) {
    return apiClient.post<InvoiceRequest>('/payment/invoices/requests', data)
  },

  /** Get current user's invoice titles */
  getInvoiceProfiles() {
    return apiClient.get<InvoiceProfile[]>('/payment/invoices/profiles')
  },

  /** Create an invoice title */
  createInvoiceProfile(data: InvoiceProfilePayload) {
    return apiClient.post<InvoiceProfile>('/payment/invoices/profiles', data)
  },

  /** Update an invoice title */
  updateInvoiceProfile(id: number, data: InvoiceProfilePayload) {
    return apiClient.put<InvoiceProfile>(`/payment/invoices/profiles/${id}`, data)
  },

  /** Delete an invoice title */
  deleteInvoiceProfile(id: number) {
    return apiClient.delete(`/payment/invoices/profiles/${id}`)
  }
}
