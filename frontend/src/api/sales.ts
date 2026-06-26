import { apiClient } from './client'
import type { BasePaginationResponse, User } from '@/types'
import type { PaymentOrder } from '@/types/payment'

export interface SalesDashboardSummary {
  total_customers: number
  total_orders: number
  completed_orders: number
  total_order_amount: number
  range: string
  month: string
  start_date: string
  end_date: string
}

export interface SalesCustomerSummary {
  user: User
  total_orders: number
  completed_order_amount: number
}

export const salesAPI = {
  getDashboard(params?: { month?: string }) {
    return apiClient.get<SalesDashboardSummary>('/sales/dashboard', { params })
  },
  listCustomers(params?: { page?: number; page_size?: number; search?: string; status?: string }) {
    return apiClient.get<BasePaginationResponse<SalesCustomerSummary>>('/sales/customers', { params })
  },
  getOrders(params?: {
    page?: number
    page_size?: number
    status?: string
    payment_type?: string
    start_date?: string
    end_date?: string
  }) {
    return apiClient.get<BasePaginationResponse<PaymentOrder>>('/sales/orders', { params })
  },
  getCustomer(id: number) {
    return apiClient.get<User>(`/sales/customers/${id}`)
  },
  getCustomerOrders(id: number, params?: { page?: number; page_size?: number; status?: string; start_date?: string; end_date?: string }) {
    return apiClient.get<BasePaginationResponse<PaymentOrder>>(`/sales/customers/${id}/orders`, { params })
  }
}
