/**
 * Public Plugin Center API
 */

import { apiClient } from './client'
import type { PublicPlugin } from '@/types'

export async function listPublic(category?: string): Promise<PublicPlugin[]> {
  const { data } = await apiClient.get<PublicPlugin[]>('/plugins', {
    params: category ? { category } : undefined
  })
  return data
}

const pluginsAPI = {
  listPublic
}

export default pluginsAPI
