import { apiClient } from './client'
import type { BasePaginationResponse } from '@/types'

export interface UserImageGeneration {
  id: number
  prompt: string
  revised_prompt?: string | null
  model: string
  mime_type: string
  image_base64: string
  created_at: string
}

export interface UserImageHistoryItem {
  id: number
  prompt: string
  revised_prompt?: string | null
  model: string
  mime_type: string
  image_sha256: string
  created_at: string
}

export const imageAPI = {
  async generate(prompt: string): Promise<UserImageGeneration> {
    const response = await apiClient.post<UserImageGeneration>(
      '/user/images/generations',
      { prompt },
      { timeout: 180000 }
    )
    return response.data
  },

  async getHistory(page = 1, pageSize = 20): Promise<BasePaginationResponse<UserImageHistoryItem>> {
    const response = await apiClient.get<BasePaginationResponse<UserImageHistoryItem>>('/user/images/history', {
      params: { page, page_size: pageSize }
    })
    return response.data
  },

  async getHistoryFile(id: number): Promise<Blob> {
    const response = await apiClient.get<Blob>(`/user/images/history/${id}/file`, {
      responseType: 'blob'
    })
    return response.data
  }
}
