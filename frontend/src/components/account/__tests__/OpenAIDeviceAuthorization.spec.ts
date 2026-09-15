import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const post = vi.hoisted(() => vi.fn())
vi.mock('@/api/client', () => ({ apiClient: { post } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copied: false, copyToClipboard: vi.fn() }) }))
import OpenAIDeviceAuthorization from '../OpenAIDeviceAuthorization.vue'

const device = { session_id: 'local-session', user_code: 'ABCD-EFGH', interval: 5, expires_at: Math.floor(Date.now() / 1000) + 900 }
const prefix = 'admin.accounts.oauth.openai.device'
const wrappers: ReturnType<typeof mount>[] = []
function render() {
  const wrapper = mount(OpenAIDeviceAuthorization, { props: { proxyId: 42 } })
  wrappers.push(wrapper)
  return wrapper
}
beforeEach(() => { vi.useFakeTimers(); post.mockReset(); post.mockResolvedValue({ data: {} }) })
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); vi.useRealTimers() })

describe('OpenAI device authorization', () => {
  it('uses the current site address and only emits credentials after approval', async () => {
    post.mockResolvedValueOnce({ data: device })
      .mockResolvedValueOnce({ data: { status: 'pending', interval: 5 } })
      .mockResolvedValueOnce({ data: { status: 'authorized', token_info: { access_token: 'access', refresh_token: 'refresh' } } })
    const wrapper = render()
    expect(wrapper.get('[data-testid="device-return-address"]').text()).toBe(window.location.origin + window.location.pathname)
    await wrapper.get('button').trigger('click'); await flushPromises()
    expect(post).toHaveBeenCalledWith('/admin/openai/device/start', { proxy_id: 42 }, expect.any(Object))
    expect(wrapper.get('a').attributes('href')).toBe('https://auth.openai.com/codex/device')
    expect(wrapper.get('a').attributes('target')).toBe('_blank')
    await vi.advanceTimersByTimeAsync(5000)
    expect(wrapper.emitted('authorized')).toBeUndefined()
    await vi.advanceTimersByTimeAsync(5000)
    expect(wrapper.emitted('authorized')).toEqual([[{ access_token: 'access', refresh_token: 'refresh' }]])
    await vi.advanceTimersByTimeAsync(30000)
    expect(wrapper.emitted('authorized')).toHaveLength(1)
  })
  it('stops polling and cancels the session when the dialog closes', async () => {
    post.mockResolvedValueOnce({ data: device })
    const wrapper = render()
    await wrapper.get('button').trigger('click'); await flushPromises()
    wrapper.unmount()
    expect(post).toHaveBeenCalledWith('/admin/openai/device/cancel', { session_id: 'local-session' })
    await vi.advanceTimersByTimeAsync(30000)
    expect(post.mock.calls.filter(([url]) => url.endsWith('/poll'))).toHaveLength(0)
  })
  it('ignores late authorization responses after changing the selected proxy', async () => {
    let resolvePoll: (value: unknown) => void = () => {}
    post.mockResolvedValueOnce({ data: device }).mockImplementationOnce(() => new Promise(resolve => { resolvePoll = resolve }))
    const wrapper = render()
    await wrapper.get('button').trigger('click'); await flushPromises()
    await vi.advanceTimersByTimeAsync(5000)
    await wrapper.setProps({ proxyId: 43 })
    resolvePoll({ data: { status: 'authorized', token_info: { access_token: 'late' } } }); await flushPromises()
    expect(wrapper.emitted('authorized')).toBeUndefined()
    expect(wrapper.find('[data-testid="device-code"]').exists()).toBe(false)
  })
  it('retries transient network failures and expires without saving credentials', async () => {
    post.mockResolvedValueOnce({ data: { ...device, expires_at: Math.floor(Date.now() / 1000) + 12 } })
      .mockRejectedValueOnce(new Error('offline'))
    const wrapper = render()
    await wrapper.get('button').trigger('click'); await flushPromises()
    await vi.advanceTimersByTimeAsync(5000)
    expect(wrapper.text()).toContain(`${prefix}.retrying`)
    await vi.advanceTimersByTimeAsync(10000)
    expect(wrapper.text()).toContain(`${prefix}.expired`)
    expect(wrapper.emitted('authorized')).toBeUndefined()
  })
})
