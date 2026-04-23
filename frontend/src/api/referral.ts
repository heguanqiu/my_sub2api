import { apiClient } from './client'
import type { BasePaginationResponse, User } from '@/types'

export interface InviteLink {
  id: number
  code: string
  url: string
  status: string
  creator_role: string
  owner_sales_id?: number | null
}

export interface InviteReward {
  id: number
  inviter_user_id: number
  invitee_user_id: number
  trigger_order_id: number
  reward_type: string
  reward_amount: number
  status: string
  reason?: string
  created_at: string
  confirmed_at?: string | null
  reversed_at?: string | null
}

export const referralAPI = {
  getMyLink() {
    return apiClient.get<InviteLink>('/referral/my-link')
  },
  regenerateMyLink() {
    return apiClient.post<InviteLink>('/referral/my-link/regenerate')
  },
  disableMyLink() {
    return apiClient.post<InviteLink>('/referral/my-link/disable')
  },
  revokeMyLink() {
    return apiClient.post<InviteLink>('/referral/my-link/revoke')
  },
  getMyInvitees(params?: { page?: number; page_size?: number; search?: string }) {
    return apiClient.get<BasePaginationResponse<User>>('/referral/my-invitees', { params })
  },
  getMyRewards(params?: { page?: number; page_size?: number; status?: string }) {
    return apiClient.get<BasePaginationResponse<InviteReward>>('/referral/my-rewards', { params })
  }
}
