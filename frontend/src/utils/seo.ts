import type { RouteLocationNormalizedLoaded } from 'vue-router'
import type { CustomMenuItem } from '@/types'
import { i18n } from '@/i18n'
import { resolveRouteDocumentTitle, resolveRouteMetaKeys, type RouteTitleOptions } from '@/router/title'

const FALLBACK_DESCRIPTIONS = {
  zh: 'Starbridge AI 多模型 AI API 中转站，统一接入 OpenAI、Claude、Gemini、DeepSeek 等模型，支持 OAuth、API Key、多账号调度与用量控制。',
  en: 'Starbridge AI is a self-hosted multi-model AI API gateway for OpenAI, Claude, Gemini, DeepSeek and other compatible providers.',
} as const

function upsertMeta(attribute: 'name' | 'property', value: string, content: string): void {
  if (typeof document === 'undefined') return
  let element = document.head.querySelector<HTMLMetaElement>(`meta[${attribute}="${value}"]`)
  if (!element) {
    element = document.createElement('meta')
    element.setAttribute(attribute, value)
    document.head.appendChild(element)
  }
  element.content = content
}

function upsertCanonical(href: string): void {
  if (typeof document === 'undefined') return
  let element = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]')
  if (!element) {
    element = document.createElement('link')
    element.rel = 'canonical'
    document.head.appendChild(element)
  }
  element.href = href
}

function translate(key: string | undefined, fallback: string): string {
  if (!key) return fallback
  const value = i18n.global.t(key)
  return value && value !== key ? String(value) : fallback
}

function resolveSeoTitle(route: RouteLocationNormalizedLoaded, siteName: string, fallbackTitle: string): string {
  const key = typeof route.meta.seoTitleKey === 'string' ? route.meta.seoTitleKey : undefined
  return translate(key, fallbackTitle || siteName)
}

function resolveCanonicalPath(route: RouteLocationNormalizedLoaded): string {
  if (route.name === 'Home' || route.path === '/') return '/'
  return route.path || '/'
}

/** Synchronize crawler metadata whenever the SPA route changes. */
export function updateDocumentSeo(
  route: RouteLocationNormalizedLoaded,
  siteName = 'Starbridge AI',
  customMenuItems: CustomMenuItem[] = [],
  siteSubtitle?: string,
  options: RouteTitleOptions = {},
): void {
  if (typeof document === 'undefined') return

  const normalizedSiteName = siteName.trim() || 'Starbridge AI'
  const browserTitle = resolveRouteDocumentTitle(route, normalizedSiteName, customMenuItems, options)
  const { descriptionKey } = resolveRouteMetaKeys(route, options)
  const locale = i18n.global.locale.value === 'zh' ? 'zh' : 'en'
  const fallbackDescription = route.name === 'Home' && siteSubtitle?.trim()
    ? siteSubtitle.trim()
    : FALLBACK_DESCRIPTIONS[locale]
  const description = translate(descriptionKey, fallbackDescription)
  const seoTitle = resolveSeoTitle(route, normalizedSiteName, browserTitle)
  const indexable = route.meta.indexable === true
  const canonicalUrl = new URL(resolveCanonicalPath(route), window.location.origin).toString()
  const robots = indexable ? 'index,follow,max-image-preview:large' : 'noindex,nofollow'

  document.title = browserTitle
  document.documentElement.setAttribute('lang', locale === 'zh' ? 'zh-CN' : 'en')
  upsertMeta('name', 'description', description)
  upsertMeta('name', 'robots', robots)
  upsertMeta('property', 'og:title', seoTitle)
  upsertMeta('property', 'og:description', description)
  upsertMeta('property', 'og:url', canonicalUrl)
  upsertMeta('property', 'og:site_name', normalizedSiteName)
  upsertMeta('property', 'og:type', 'website')
  upsertMeta('name', 'twitter:title', seoTitle)
  upsertMeta('name', 'twitter:description', description)
  upsertCanonical(canonicalUrl)

  const structuredData = document.getElementById('site-structured-data')
  if (structuredData) {
    structuredData.textContent = JSON.stringify({
      '@context': 'https://schema.org',
      '@type': 'SoftwareApplication',
      name: normalizedSiteName,
      applicationCategory: 'DeveloperApplication',
      operatingSystem: 'Web',
      url: canonicalUrl,
      description,
    })
  }
}
