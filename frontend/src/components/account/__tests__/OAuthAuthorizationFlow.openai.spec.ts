import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copied: { value: false }, copyToClipboard: vi.fn() })
}))

import OAuthAuthorizationFlow from '../OAuthAuthorizationFlow.vue'

describe('OAuthAuthorizationFlow OpenAI subscription sign-in', () => {
  it('provides a direct ChatGPT authorization link and recognizes the public callback', async () => {
    const authUrl = 'https://auth.openai.com/oauth/authorize?state=generated-state'
    const wrapper = mount(OAuthAuthorizationFlow, {
      props: {
        addMethod: 'oauth',
        platform: 'openai',
        authUrl,
        showCookieOption: false,
        showRefreshTokenOption: true
      },
      global: { stubs: { Icon: true } }
    })

    const authorizationLink = wrapper.get('[data-testid="oauth-open-authorization-page"]')
    expect(authorizationLink.attributes('href')).toBe(authUrl)
    expect(authorizationLink.attributes('target')).toBe('_blank')

    await wrapper.get('textarea[placeholder]').setValue(
      'https://starbridaeai.top/auth/callback?code=subscription-code&state=callback-state'
    )
    await nextTick()

    expect(wrapper.get('[data-testid="oauth-callback-recognized"]').exists()).toBe(true)
    expect((wrapper.vm as unknown as { authCode: string }).authCode).toBe('subscription-code')
    expect((wrapper.vm as unknown as { oauthState: string }).oauthState).toBe('callback-state')
  })
})
