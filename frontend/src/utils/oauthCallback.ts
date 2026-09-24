export interface ParsedOAuthCallbackInput {
  code: string
  state: string
  isCallback: boolean
}

export const DEFAULT_PUBLIC_OAUTH_CALLBACK_URL = 'https://starbridaeai.top/auth/callback'
export const OPENAI_OAUTH_CALLBACK_URL = DEFAULT_PUBLIC_OAUTH_CALLBACK_URL
export const OAUTH_CALLBACK_MESSAGE_TYPE = 'starbridge.oauth.callback'

export interface OAuthCallbackMessage {
  type: typeof OAUTH_CALLBACK_MESSAGE_TYPE
  code: string
  state: string
}

/**
 * Build the callback address that the browser is currently using for Starbridge.
 * This keeps OAuth links valid when the deployment is served from a custom domain
 * while retaining a deterministic value for non-browser rendering and tests.
 */
export function getPublicOAuthCallbackUrl(): string {
  if (typeof window !== 'undefined') {
    const origin = window.location.origin?.trim()
    if (origin && origin !== 'null') return `${origin.replace(/\/+$/, '')}/auth/callback`
  }
  return DEFAULT_PUBLIC_OAUTH_CALLBACK_URL
}

/**
 * Accept a complete OAuth callback URL, a query string, or a bare code.
 * Keeping this parser outside the component makes the mobile copy/paste flow
 * predictable and lets every supported OAuth provider share the same rules.
 */
export function parseOAuthCallbackInput(input: string): ParsedOAuthCallbackInput {
  const trimmed = input.trim()
  if (!trimmed) return { code: '', state: '', isCallback: false }

  const parseParams = (params: URLSearchParams): ParsedOAuthCallbackInput => {
    const code = (params.get('code') || '').trim()
    const state = (params.get('state') || '').trim()
    return {
      code: code || trimmed,
      state,
      isCallback: Boolean(code)
    }
  }

  try {
    if (/^[a-z][a-z\d+.-]*:\/\//i.test(trimmed)) {
      return parseParams(new URL(trimmed).searchParams)
    }
    if (trimmed.startsWith('?') || /(^|&)code=/.test(trimmed)) {
      return parseParams(new URLSearchParams(trimmed.replace(/^\?/, '')))
    }
  } catch {
    // Fall through and preserve the original value as a bare authorization code.
  }

  return { code: trimmed, state: '', isCallback: false }
}
