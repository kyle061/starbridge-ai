/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey } from '@/types'

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

export type ApiKeyLimits = Pick<ApiKey, 'quota' | 'rate_limit_5h' | 'rate_limit_1d' | 'rate_limit_7d'>

export async function updateApiKeyLimits(id: number, limits: Partial<ApiKeyLimits>): Promise<ApiKey> {
  const { data } = await apiClient.put<ApiKey>(`/admin/api-keys/${id}/limits`, limits)
  return data
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupId: number | null): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    group_id: groupId === null ? 0 : groupId
  })
  return data
}

export const apiKeysAPI = {
  updateApiKeyGroup,
  updateApiKeyLimits
}

export default apiKeysAPI
