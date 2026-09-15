export interface ParsedOAuthCallbackInput {
  code: string
  state: string
  isCallback: boolean
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
