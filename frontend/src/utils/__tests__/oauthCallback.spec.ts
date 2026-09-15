import { describe, expect, it } from 'vitest'
import { parseOAuthCallbackInput } from '@/utils/oauthCallback'

describe('parseOAuthCallbackInput', () => {
  it('extracts code and state from the localhost callback URL used by ChatGPT OAuth', () => {
    expect(
      parseOAuthCallbackInput(
        'http://localhost:1455/auth/callback?code=codex-code&state=session-state'
      )
    ).toEqual({ code: 'codex-code', state: 'session-state', isCallback: true })
  })

  it('accepts a copied callback query string', () => {
    expect(parseOAuthCallbackInput('?code=query-code&state=query-state')).toEqual({
      code: 'query-code',
      state: 'query-state',
      isCallback: true
    })
  })

  it('keeps a bare authorization code unchanged', () => {
    expect(parseOAuthCallbackInput('  bare-code  ')).toEqual({
      code: 'bare-code',
      state: '',
      isCallback: false
    })
  })
})
