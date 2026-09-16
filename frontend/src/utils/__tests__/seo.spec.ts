import type { RouteLocationNormalizedLoaded } from 'vue-router'
import { beforeEach, describe, expect, it } from 'vitest'
import { loadLocaleMessages, i18n } from '@/i18n'
import { updateDocumentSeo } from '../seo'

function route(overrides: Partial<RouteLocationNormalizedLoaded>): RouteLocationNormalizedLoaded {
  return {
    name: 'Home',
    path: '/home',
    fullPath: '/home',
    hash: '',
    query: {},
    params: {},
    matched: [],
    redirectedFrom: undefined,
    meta: {},
    ...overrides,
  } as RouteLocationNormalizedLoaded
}

describe('updateDocumentSeo', () => {
  beforeEach(async () => {
    await loadLocaleMessages('zh')
    i18n.global.locale.value = 'zh'
    document.head.innerHTML = '<link rel="canonical" href="/" />'
    document.title = ''
    window.history.replaceState({}, '', '/home')
  })

  it('marks public routes indexable and points home at the root canonical URL', () => {
    updateDocumentSeo(route({
      meta: {
        indexable: true,
        title: '多模型 AI API 中转站',
      },
    }), 'Starbridge AI')

    expect(document.head.querySelector('meta[name="robots"]')?.getAttribute('content'))
      .toBe('index,follow,max-image-preview:large')
    expect(document.head.querySelector('meta[name="description"]')?.getAttribute('content'))
      .toContain('OpenAI')
    expect(document.head.querySelector('link[rel="canonical"]')?.getAttribute('href'))
      .toBe(new URL('/', window.location.origin).toString())
    expect(document.head.querySelector('meta[property="og:title"]')?.getAttribute('content'))
      .toContain('多模型 AI API 中转站')
    expect(document.title).toContain('多模型 AI API 中转站')
  })

  it('uses the configured public subtitle for crawler descriptions', () => {
    updateDocumentSeo(route({
      meta: { indexable: true },
    }), 'Starbridge AI', [], '统一接入 OpenAI 与 Claude 的 API 网关')

    expect(document.head.querySelector('meta[name="description"]')?.getAttribute('content'))
      .toBe('统一接入 OpenAI 与 Claude 的 API 网关')
    expect(document.head.querySelector('meta[property="og:description"]')?.getAttribute('content'))
      .toBe('统一接入 OpenAI 与 Claude 的 API 网关')
  })

  it('marks account routes noindex and keeps their route canonical URL', () => {
    updateDocumentSeo(route({
      name: 'Keys',
      path: '/keys',
      meta: { indexable: false, descriptionKey: 'keys.description' },
    }), 'Starbridge AI')

    expect(document.head.querySelector('meta[name="robots"]')?.getAttribute('content'))
      .toBe('noindex,nofollow')
    expect(document.head.querySelector('link[rel="canonical"]')?.getAttribute('href'))
      .toBe(new URL('/keys', window.location.origin).toString())
  })
})
