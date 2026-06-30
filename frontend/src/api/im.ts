import { apiClient } from './client'

export interface IMSsoTokenResponse {
  token: string
  token_type: 'Bearer'
  expires_in: number
  web_url: string
  service_url?: string
}

export async function issueSSOToken(): Promise<IMSsoTokenResponse> {
  const { data } = await apiClient.post<IMSsoTokenResponse>('/im/sso-token')
  return data
}

export const imAPI = {
  issueSSOToken,
}

export default imAPI
