import { apiClient } from './client'

export interface PlaygroundEmbedSession {
  source: 'sub2api'
  version: number
  expires_at: number
  signature: string
  api_base_url: string
  selected_key_id: number
  selected_key_name: string
  user: {
    id: string
    email: string
    username: string
    name: string
  }
  direct_connections: {
    OPENAI_API_BASE_URLS: string[]
    OPENAI_API_KEYS: string[]
    OPENAI_API_CONFIGS: Record<
      string,
      {
        enable: boolean
        prefix_id: string
        model_ids: string[]
      }
    >
  }
}

export async function createEmbedSession(apiKeyId: number | string, apiBaseUrl: string): Promise<PlaygroundEmbedSession> {
  const { data } = await apiClient.post<PlaygroundEmbedSession>('/playground/embed-session', {
    api_key_id: apiKeyId,
    api_base_url: apiBaseUrl,
  })
  return data
}

export const playgroundAPI = {
  createEmbedSession,
}

export default playgroundAPI
