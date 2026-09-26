import { flushPromises, shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { MonitorConfig } from '@/api/channelMonitorV2'

const { getConfig, updateConfig } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
}))

vi.mock('@/api/channelMonitorV2', async (loadOriginal) => ({
  ...await loadOriginal<typeof import('@/api/channelMonitorV2')>(),
  getConfig,
  updateConfig,
}))
vi.mock('@/api/admin', () => ({ adminAPI: { groups: { getAllIncludingInactive: vi.fn().mockResolvedValue([]) } } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: { channel_monitor_enabled: true }, showError: vi.fn(), showSuccess: vi.fn() }) }))
vi.mock('@/utils/featureFlags', () => ({ isChannelMonitorV2Mode: () => true, getChannelMonitorMode: () => 'v2' }))
vi.mock('vue-i18n', async (loadOriginal) => ({
  ...await loadOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key, te: () => true }),
}))

import MonitorSettingsPanel from '../MonitorSettingsPanel.vue'

describe('MonitorSettingsPanel model inventory', () => {
  it('switches a manually pinned list to actual-traffic discovery only after save', async () => {
    const config = {
      version: 1,
      enabled: true,
      refresh_interval_seconds: 300,
      group_ids: [],
      platforms: [{ platform: 'openai', enabled: true, models: ['gpt-4', 'gpt-6'] }],
      health_thresholds: {},
      ignored_error_categories: [],
    } as MonitorConfig
    getConfig.mockResolvedValue(structuredClone(config))
    updateConfig.mockImplementation(async (next: MonitorConfig) => next)

    const wrapper = shallowMount(MonitorSettingsPanel, {
      global: { stubs: { 'router-link': true } },
    })
    await flushPromises()
    const discover = wrapper.findAll('button').find((button) => button.text().includes('useObservedModels'))
    expect(discover).toBeDefined()
    expect(discover!.attributes('disabled')).toBeUndefined()
    expect((wrapper.find('input[type="text"]').element as HTMLInputElement).value).toBe('gpt-4, gpt-6')

    await discover!.trigger('click')
    expect((wrapper.find('input[type="text"]').element as HTMLInputElement).value).toBe('')
    expect(updateConfig).not.toHaveBeenCalled()
    await wrapper.findAll('button').find((button) => button.text().includes('settings.save'))!.trigger('click')
    await flushPromises()
    expect(updateConfig).toHaveBeenCalledWith(expect.objectContaining({
      platforms: [{ platform: 'openai', enabled: true, models: [] }],
    }))
    wrapper.unmount()
  })
})
