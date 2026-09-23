import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SupportContact from '../SupportContact.vue'

const { copyToClipboard } = vi.hoisted(() => ({ copyToClipboard: vi.fn() }))
vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copied: false, copyToClipboard }),
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('SupportContact', () => {
  beforeEach(() => copyToClipboard.mockReset())

  it('copies only the WeChat ID from a prefixed value', async () => {
    const wrapper = mount(SupportContact, { props: { contact: ' 微信： starbridge_support ' } })
    expect(wrapper.text()).toContain('common.contactSupport')
    expect(wrapper.text()).toContain('common.contactWeChat')
    expect(wrapper.get('[data-testid="wechat-contact-detail"]').classes()).toContain('text-xs')
    expect(wrapper.get('[data-testid="wechat-contact-detail"]').text()).not.toContain('微信：')
    expect(wrapper.find('a').exists()).toBe(false)
    await wrapper.get('button').trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith('starbridge_support', 'common.copiedToClipboard')
  })

  it('preserves email contact semantics instead of labelling an email as WeChat', () => {
    const wrapper = mount(SupportContact, { props: { contact: 'support@example.com', showValue: true } })
    expect(wrapper.get('a').attributes('href')).toBe('mailto:support@example.com')
    expect(wrapper.text()).toContain('support@example.com')
    expect(wrapper.text()).toContain('common.contactSupport')
    expect(wrapper.text()).not.toContain('common.contactWeChat')
  })

  it('opens safe contact URLs and keeps unsupported schemes as copyable text', () => {
    const safe = mount(SupportContact, { props: { contact: 'https://example.com/support' } })
    expect(safe.get('a').attributes('rel')).toBe('noopener noreferrer')
    const unsafe = mount(SupportContact, { props: { contact: 'javascript:alert(1)' } })
    expect(unsafe.find('a').exists()).toBe(false)
    expect(unsafe.get('button').exists()).toBe(true)
  })

  it('renders nothing when the configured contact is blank', () => {
    const wrapper = mount(SupportContact, { props: { contact: '  ' } })
    expect(wrapper.text()).toBe('')
    expect(wrapper.find('button').exists()).toBe(false)
  })
})
