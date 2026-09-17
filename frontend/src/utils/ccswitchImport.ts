import type { GroupPlatform } from '@/types'

export const OPENAI_CC_SWITCH_CODEX_MODEL = 'gpt-6-astra'
export const GROK_CC_SWITCH_MODEL = 'grok-4.5'
export const DEEPSEEK_CC_SWITCH_CODEX_MODEL = 'deepseek-v4-pro'

export type CcSwitchClientType = 'claude' | 'gemini'

export interface CcSwitchImportConfig {
  app: string
  endpoint: string
  usageBaseUrl: string
  model?: string
}

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  platform?: GroupPlatform | null
  clientType: CcSwitchClientType
  providerName: string
  apiKey: string
  usageScript: string
}

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.trim().replace(/\/+$/, '')
}

function stripGatewayVersion(baseUrl: string): string {
  return normalizeBaseUrl(baseUrl).replace(/\/(?:v1beta|v1)$/i, '')
}

function joinUrl(baseUrl: string, path: string): string {
  return `${normalizeBaseUrl(baseUrl)}/${path.replace(/^\/+/, '')}`
}

function withV1Endpoint(baseUrl: string): string {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl)
  return normalizedBaseUrl.endsWith('/v1') ? normalizedBaseUrl : `${normalizedBaseUrl}/v1`
}

export function resolveCcSwitchImportConfig(
  platform: GroupPlatform | undefined | null,
  clientType: CcSwitchClientType,
  baseUrl: string
): CcSwitchImportConfig {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl)
  const usageBaseUrl = stripGatewayVersion(normalizedBaseUrl)

  switch (platform || 'anthropic') {
    case 'antigravity': {
      const antigravityBaseUrl = joinUrl(usageBaseUrl, 'antigravity')
      return {
        app: clientType === 'gemini' ? 'gemini' : 'claude',
        endpoint: antigravityBaseUrl,
        usageBaseUrl: antigravityBaseUrl
      }
    }
    case 'openai':
    case 'composite':
      return {
        app: 'codex',
        endpoint: withV1Endpoint(normalizedBaseUrl),
        usageBaseUrl,
        model: OPENAI_CC_SWITCH_CODEX_MODEL
      }
    case 'gemini':
      return {
        app: 'gemini',
        endpoint: normalizedBaseUrl,
        usageBaseUrl
      }
    case 'deepseek':
      return {
        app: 'codex',
        endpoint: withV1Endpoint(normalizedBaseUrl),
        usageBaseUrl,
        model: DEEPSEEK_CC_SWITCH_CODEX_MODEL
      }
    case 'grok':
      return {
        app: 'grokbuild',
        endpoint: withV1Endpoint(normalizedBaseUrl),
        usageBaseUrl,
        model: GROK_CC_SWITCH_MODEL
      }
    default:
      return {
        app: 'claude',
        endpoint: normalizedBaseUrl,
        usageBaseUrl
      }
  }
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const config = resolveCcSwitchImportConfig(input.platform, input.clientType, input.baseUrl)
  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', config.app],
    ['name', input.providerName],
    ['homepage', stripGatewayVersion(input.baseUrl)],
    ['endpoint', config.endpoint],
    ['apiKey', input.apiKey],
    ['configFormat', 'json'],
    ['usageEnabled', 'true'],
    ['usageScript', btoa(input.usageScript)],
    ['usageBaseUrl', config.usageBaseUrl],
    ['usageAutoInterval', '30']
  ]

  if (config.model) {
    entries.splice(2, 0, ['model', config.model])
  }

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}
