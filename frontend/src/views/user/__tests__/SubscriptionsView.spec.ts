import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { UserSubscription } from '@/types'
import SubscriptionsView from '../SubscriptionsView.vue'

const { getMySubscriptions, showError } = vi.hoisted(() => ({
  getMySubscriptions: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/subscriptions', () => ({
  default: { getMySubscriptions }
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError })
}))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() })
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

function subscription(id: number, status: UserSubscription['status']): UserSubscription {
  return {
    id,
    user_id: 1,
    group_id: id,
    status,
    starts_at: '2020-01-01T00:00:00Z',
    expires_at: status === 'expired' ? '2021-01-01T00:00:00Z' : '2100-01-01T00:00:00Z',
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    quota_usd: 0,
    quota_used_usd: 0,
    usage_multiplier: 1,
    daily_window_start: null,
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2020-01-01T00:00:00Z',
    updated_at: '2020-01-01T00:00:00Z'
  }
}

describe('SubscriptionsView', () => {
  it('shows expired subscriptions last without changing the order within each group', async () => {
    getMySubscriptions.mockResolvedValue([
      subscription(1, 'expired'),
      subscription(2, 'active'),
      subscription(3, 'expired'),
      subscription(4, 'active')
    ])

    const wrapper = mount(SubscriptionsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          SubscriptionQuotaSummary: true,
          SubscriptionUsageGuide: true
        }
      }
    })
    await flushPromises()

    expect(wrapper.findAll('.grid h3').map(heading => heading.text())).toEqual([
      'Group #2', 'Group #4', 'Group #1', 'Group #3'
    ])
    wrapper.unmount()
  })
})
