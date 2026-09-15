import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ApiKeyLimitsForm from '../ApiKeyLimitsForm.vue'
import type { ApiKey } from '@/types'

const { updateApiKeyLimits, showError } = vi.hoisted(() => ({ updateApiKeyLimits: vi.fn(), showError: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { apiKeys: { updateApiKeyLimits } } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const key = { id: 1, quota: 10, quota_used: 3, rate_limit_5h: 2, rate_limit_1d: 5, rate_limit_7d: 20 } as ApiKey

describe('ApiKeyLimitsForm', () => {
  beforeEach(() => { vi.clearAllMocks(); updateApiKeyLimits.mockResolvedValue(key) })

  it('sends only changed limits, including zero, without resetting usage', async () => {
    const wrapper = mount(ApiKeyLimitsForm, { props: { apiKey: key } })
    await wrapper.get('[data-test="rate_limit_1d"]').setValue('0')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(updateApiKeyLimits).toHaveBeenCalledWith(1, { rate_limit_1d: 0 })
    expect(wrapper.emitted('updated')).toEqual([[key]])
  })

  it.each(['', '-1'])('rejects an invalid limit %s', async (value) => {
    const wrapper = mount(ApiKeyLimitsForm, { props: { apiKey: key } })
    await wrapper.get('[data-test="quota"]').setValue(value)
    await wrapper.get('form').trigger('submit')
    expect(updateApiKeyLimits).not.toHaveBeenCalled()
  })

  it('keeps the form open with edits when saving fails', async () => {
    updateApiKeyLimits.mockRejectedValue(new Error('Cannot save'))
    const wrapper = mount(ApiKeyLimitsForm, { props: { apiKey: key } })
    await wrapper.get('[data-test="quota"]').setValue('20')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(showError).toHaveBeenCalledWith('Cannot save')
    expect(wrapper.emitted('updated')).toBeUndefined()
    expect(wrapper.emitted('close')).toBeUndefined()
  })
})
