import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SubscriptionQuotaSummary from '../SubscriptionQuotaSummary.vue'
import { formatSubscriptionQuota } from '@/utils/subscriptionQuota'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('subscription quota visibility', () => {
  it('shows small deductions and their exact remaining amount', () => {
    const wrapper = mount(SubscriptionQuotaSummary, { props: { quota: 50, used: 0.000024 } })
    expect(wrapper.text()).toContain('$50.00')
    expect(wrapper.text()).toContain('$0.000024')
    expect(wrapper.text()).toContain('$49.999976')
    expect(wrapper.text()).not.toContain('quotaNotConfigured')
  })

  it('marks missing quotas and exhausted subscriptions clearly', () => {
    const legacy = mount(SubscriptionQuotaSummary, { props: { quota: 0, used: 0 } })
    expect(legacy.text()).toContain('userSubscriptions.quotaNotConfigured')
    const exhausted = mount(SubscriptionQuotaSummary, { props: { quota: 5, used: 5 } })
    expect(exhausted.text()).toContain('userSubscriptions.quotaExhausted')
    expect(exhausted.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('100')
  })

  it.each([[50, '50.00'], [0, '0.00'], [1.2, '1.20'], [0.00001, '0.00001'], [0.00000001, '<0.000001']])('formats %s without hiding nonzero usage', (value, expected) => {
    expect(formatSubscriptionQuota(Number(value))).toBe(expected)
  })
})
