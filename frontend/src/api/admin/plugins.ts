/**
 * Admin Plugins API endpoints
 */

import { apiClient } from '../client'
import type {
  AdminPlugin,
  BasePaginationResponse,
  PluginUploadResult,
  SavePluginRequest
} from '@/types'

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: string
    category?: string
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<BasePaginationResponse<AdminPlugin>> {
  const { data } = await apiClient.get<BasePaginationResponse<AdminPlugin>>('/admin/plugins', {
    params: { page, page_size: pageSize, ...filters },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<AdminPlugin> {
  const { data } = await apiClient.get<AdminPlugin>(`/admin/plugins/${id}`)
  return data
}

export async function create(request: SavePluginRequest): Promise<AdminPlugin> {
  const { data } = await apiClient.post<AdminPlugin>('/admin/plugins', request)
  return data
}

export async function update(id: number, request: SavePluginRequest): Promise<AdminPlugin> {
  const { data } = await apiClient.put<AdminPlugin>(`/admin/plugins/${id}`, request)
  return data
}

export async function deletePlugin(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/plugins/${id}`)
  return data
}

export async function upload(kind: 'package' | 'icon', file: File): Promise<PluginUploadResult> {
  const form = new FormData()
  form.append('kind', kind)
  form.append('file', file)
  const { data } = await apiClient.post<PluginUploadResult>('/admin/plugins/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000
  })
  return data
}

const pluginsAPI = {
  list,
  getById,
  create,
  update,
  delete: deletePlugin,
  deletePlugin,
  upload
}

export default pluginsAPI
