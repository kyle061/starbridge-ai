<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ t('home.dashboard') }}
          </router-link>
          <template v-else>
            <router-link
              to="/login"
              class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-800"
            >
              {{ t('home.login') }}
            </router-link>
            <router-link
              v-if="registrationEnabled"
              data-testid="compact-register-link"
              to="/register"
              class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
            >
              {{ t('home.register') }}
            </router-link>
          </template>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <p class="mt-3 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ t('home.heroDescription') }}</p>
        <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
          <router-link
            :to="primaryActionPath"
            class="inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
          >
            {{ primaryActionLabel }}
          </router-link>
          <router-link
            v-if="!isAuthenticated && registrationEnabled"
            to="/login"
            class="inline-flex min-h-10 items-center justify-center rounded-lg border border-gray-300 px-5 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-100 dark:border-dark-600 dark:text-dark-200 dark:hover:bg-dark-800"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      <div class="flex flex-wrap items-center justify-center gap-x-5 gap-y-2">
        <span>&copy; {{ currentYear }} {{ siteName }}</span>
        <SupportContact :contact="contactInfo" />
      </div>
    </footer>
  </div>

  <!-- Default Home Page -->
  <div v-else class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white">
    <header class="border-b border-gray-200 bg-white dark:border-dark-800 dark:bg-dark-900" @keydown.esc="closeMobileMenu">
      <nav class="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-3 sm:px-6" :aria-label="t('home.navigation')">
        <router-link to="/home" class="flex min-w-0 items-center gap-2.5">
          <img :src="siteLogo || '/logo.svg'" alt="" class="h-9 w-9 shrink-0 rounded-xl object-contain" />
          <span class="truncate text-base font-semibold">{{ siteName }}</span>
        </router-link>
        <div class="hidden items-center gap-4 lg:flex">
          <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="home-nav-link">{{ t('home.modelPricing') }}</router-link>
          <router-link v-if="showChannelMonitor" :to="monitorPath" class="home-nav-link">{{ t('home.channelStatus') }}</router-link>
          <a href="#quick-start" class="home-nav-link">{{ t('home.quickStart') }}</a>
          <SupportContact :contact="contactInfo" />
        </div>
        <div class="flex shrink-0 items-center gap-1 sm:gap-2">
          <div class="hidden sm:block"><LocaleSwitcher /></div>
          <button type="button" class="inline-flex home-icon-button" :title="isDark ? t('home.switchToLight') : t('home.switchToDark')" :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')" @click="toggleTheme">
            <Icon :name="isDark ? 'sun' : 'moon'" size="md" />
          </button>
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="home-nav-link rounded-lg px-2">{{ t(isAuthenticated ? 'home.dashboard' : 'home.login') }}</router-link>
          <router-link v-if="!isAuthenticated && registrationEnabled" data-testid="default-register-link" to="/register" class="btn btn-primary hidden min-h-11 sm:inline-flex">{{ t('home.register') }}</router-link>
          <button ref="menuButton" type="button" class="inline-flex home-icon-button lg:hidden" :aria-label="t(mobileMenuOpen ? 'common.close' : 'home.openMenu')" :aria-expanded="mobileMenuOpen" aria-controls="home-mobile-menu" data-testid="home-menu-toggle" @click="mobileMenuOpen = !mobileMenuOpen">
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
              <path v-if="mobileMenuOpen" d="m6 6 12 12M6 18 18 6" />
              <path v-else d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
        </div>
      </nav>
      <div v-if="mobileMenuOpen" id="home-mobile-menu" data-testid="home-mobile-menu" class="border-t border-gray-200 px-4 py-3 dark:border-dark-800 lg:hidden">
        <div class="mx-auto grid max-w-6xl gap-1" @click="mobileMenuOpen = false">
          <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="home-nav-link">{{ t('home.modelPricing') }}</router-link>
          <router-link v-if="showChannelMonitor" :to="monitorPath" class="home-nav-link">{{ t('home.channelStatus') }}</router-link>
          <a href="#quick-start" class="home-nav-link">{{ t('home.quickStart') }}</a>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-nav-link">{{ t('home.viewDocs') }}</a>
          <SupportContact :contact="contactInfo" />
        </div>
        <div class="mt-2 flex min-h-11 items-center"><LocaleSwitcher /></div>
      </div>
    </header>

    <main class="mx-auto max-w-6xl space-y-14 px-4 pb-16 pt-10 sm:space-y-20 sm:px-6 sm:pt-16">
      <section class="grid items-center gap-8 lg:grid-cols-[1.2fr_1fr] lg:gap-14">
        <div class="min-w-0">
          <p class="mb-4 text-sm font-semibold tracking-wide text-primary-700 dark:text-primary-400">{{ siteName }} · {{ t('home.heroEyebrow') }}</p>
          <h1 class="text-balance text-4xl font-bold leading-tight tracking-tight sm:text-5xl">{{ t('home.heroSubtitle') }}</h1>
          <p class="mt-5 max-w-xl text-base leading-8 text-gray-600 dark:text-dark-300">{{ t('home.heroDescription') }}</p>
          <div class="mt-7 flex flex-wrap gap-3">
            <router-link :to="primaryActionPath" class="btn btn-primary min-h-12 px-6">{{ primaryActionLabel }}<Icon name="arrowRight" size="md" class="ml-2" /></router-link>
            <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="btn btn-secondary min-h-12 px-6">{{ t('home.modelPricing') }}</router-link>
            <a v-else href="#quick-start" class="btn btn-secondary min-h-12 px-6">{{ t('home.quickStart') }}</a>
          </div>
          <p class="mt-4 text-xs leading-6 text-gray-500 dark:text-dark-400">{{ t('home.balanceNote') }}</p>
        </div>
        <div data-testid="home-connection" class="min-w-0 rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900 sm:p-6">
          <div class="flex items-center justify-between gap-3 border-b border-gray-100 pb-4 dark:border-dark-800">
            <h2 class="font-semibold">{{ t('home.connection.title') }}</h2>
            <Icon name="terminal" size="md" class="text-primary-600 dark:text-primary-400" />
          </div>
          <p class="mb-2 mt-5 text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('home.connection.endpoint') }}</p>
          <div class="flex min-w-0 items-center gap-2 rounded-xl bg-gray-50 p-3 dark:bg-dark-800">
            <code class="min-w-0 flex-1 text-sm [overflow-wrap:anywhere]">{{ publicEndpoint }}</code>
            <button type="button" class="inline-flex home-icon-button shrink-0" :aria-label="t('home.connection.copyEndpoint')" @click="copyToClipboard(publicEndpoint, t('common.copiedToClipboard'))"><Icon :name="copied ? 'check' : 'clipboard'" size="md" /></button>
          </div>
          <div class="mt-4 flex flex-wrap gap-2">
            <span v-for="client in ['Codex', 'Claude Code', 'OpenAI SDK']" :key="client" class="rounded-md border border-gray-200 px-2.5 py-1.5 text-xs text-gray-600 dark:border-dark-700 dark:text-dark-300">{{ client }}</span>
          </div>
          <p class="mt-4 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ t('home.connection.description') }}</p>
          <router-link :to="workflowKeysPath" class="mt-3 inline-flex min-h-11 items-center gap-2 text-sm font-semibold text-primary-700 dark:text-primary-400">{{ t('home.connection.action') }}<Icon name="arrowRight" size="sm" /></router-link>
        </div>
      </section>

      <section class="grid gap-4 sm:grid-cols-3" :aria-label="t('home.overview')">
        <div class="home-info-card">
          <Icon name="server" size="md" class="text-primary-600 dark:text-primary-400" />
          <h2 class="mt-3 font-semibold">{{ t('home.features.unifiedGateway') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-400">{{ t('home.features.unifiedGatewayDesc') }}</p>
        </div>
        <div class="home-info-card">
          <Icon name="chart" size="md" class="text-primary-600 dark:text-primary-400" />
          <h2 class="mt-3 font-semibold">{{ t('home.features.balanceQuota') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-400">{{ t('home.features.balanceQuotaDesc') }}</p>
        </div>
        <div class="home-info-card">
          <Icon name="shield" size="md" class="text-primary-600 dark:text-primary-400" />
          <h2 class="mt-3 font-semibold">{{ t('home.features.multiAccount') }}</h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-400">{{ t('home.features.multiAccountDesc') }}</p>
          <router-link v-if="showChannelMonitor" :to="monitorPath" class="mt-2 inline-flex min-h-11 items-center gap-1 text-sm text-primary-700 dark:text-primary-400">{{ t('home.channelStatus') }}<Icon name="arrowRight" size="sm" /></router-link>
        </div>
      </section>

      <section id="quick-start" class="scroll-mt-6" aria-labelledby="home-workflow-title">
        <h2 id="home-workflow-title" class="text-2xl font-bold">{{ t('home.workflow.title') }}</h2>
        <p class="mt-3 max-w-2xl text-sm leading-7 text-gray-600 dark:text-dark-400">{{ t('home.workflow.subtitle') }}</p>
        <div class="mt-6 grid gap-4 md:grid-cols-3">
          <router-link v-for="(step, index) in workflowSteps" :key="step" :to="step === 'account' ? workflowLoginPath : workflowKeysPath" class="home-info-card group transition-colors hover:border-primary-400 dark:hover:border-primary-600">
            <div class="mb-4 flex items-center justify-between"><span class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-50 text-sm font-bold text-primary-700 dark:bg-primary-900/30 dark:text-primary-400">{{ index + 1 }}</span><Icon name="arrowRight" size="sm" class="text-gray-400 group-hover:text-primary-600" /></div>
            <h3 class="font-semibold">{{ t(`home.workflow.steps.${step}.title`) }}</h3>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-400">{{ t(`home.workflow.steps.${step}.description`) }}</p>
          </router-link>
        </div>
      </section>

      <section aria-labelledby="home-faq-title" class="grid gap-6 lg:grid-cols-[1fr_2fr] lg:gap-12">
        <div><h2 id="home-faq-title" class="text-2xl font-bold">{{ t('home.faq.title') }}</h2><p class="mt-3 text-sm leading-7 text-gray-600 dark:text-dark-400">{{ t('home.faq.subtitle') }}</p><SupportContact :contact="contactInfo" class="mt-3" /></div>
        <div class="min-w-0 divide-y divide-gray-200 rounded-2xl border border-gray-200 bg-white px-5 dark:divide-dark-700 dark:border-dark-700 dark:bg-dark-900">
          <details v-for="item in faqItems" :key="item" class="group py-1">
            <summary class="flex min-h-14 cursor-pointer list-none items-center justify-between gap-4 py-3 text-sm font-medium [&::-webkit-details-marker]:hidden">{{ t(`home.faq.${item}.question`) }}<span aria-hidden="true" class="shrink-0 text-lg text-gray-400 group-open:rotate-45">+</span></summary>
            <p class="pb-4 text-sm leading-7 text-gray-600 dark:text-dark-400">{{ t(`home.faq.${item}.answer`) }}</p>
          </details>
        </div>
      </section>
    </main>

    <footer class="border-t border-gray-200 px-4 py-6 dark:border-dark-800 sm:px-6">
      <div class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 text-sm text-gray-500 dark:text-dark-400">
        <p class="[overflow-wrap:anywhere]">&copy; {{ currentYear }} {{ siteName }}</p>
        <div class="flex max-w-full flex-wrap items-center gap-x-5 gap-y-2"><a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-nav-link">{{ t('home.docs') }}</a><SupportContact :contact="contactInfo" /></div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import SupportContact from '@/components/common/SupportContact.vue'
import { useClipboard } from '@/composables/useClipboard'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Starbridge AI')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Multi-model AI API Gateway')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))
const registrationEnabled = computed(() => appStore.cachedPublicSettings?.registration_enabled === true)

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

const contactInfo = computed(() => (appStore.cachedPublicSettings?.contact_info || appStore.contactInfo || '').trim())

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(
  () => appStore.cachedPublicSettings?.model_plaza_require_auth === true,
)
const showModelPlazaEntry = computed(
  () => modelPlazaEnabled.value && (isAuthenticated.value || !modelPlazaRequiresAuth.value),
)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const primaryActionPath = computed(() => {
  if (isAuthenticated.value) return dashboardPath.value
  return registrationEnabled.value ? '/register' : '/login'
})
const primaryActionLabel = computed(() => {
  if (isAuthenticated.value) return t('home.goToDashboard')
  return registrationEnabled.value ? t('home.register') : t('home.getStarted')
})
const workflowLoginPath = computed(() => (isAuthenticated.value ? dashboardPath.value : '/login'))
const workflowKeysPath = computed(() => (isAuthenticated.value ? '/keys' : '/login?redirect=/keys'))
const mobileMenuOpen = ref(false)
const menuButton = ref<HTMLButtonElement | null>(null)
function closeMobileMenu() {
  mobileMenuOpen.value = false
  menuButton.value?.focus()
}
const { copied, copyToClipboard } = useClipboard()
const workflowSteps = ['account', 'key', 'connect'] as const
const faqItems = ['balance', 'pricing', 'groups', 'client'] as const
const showChannelMonitor = computed(() => isFeatureFlagEnabled(FeatureFlags.channelMonitor))
const monitorPath = computed(() => isAuthenticated.value ? '/monitor' : '/login?redirect=/monitor')
const publicEndpoint = computed(() => appStore.cachedPublicSettings?.api_base_url?.trim() || window.location.origin)

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.home-nav-link {
  @apply inline-flex min-h-11 items-center text-sm font-medium text-gray-600 transition-colors hover:text-primary-700 dark:text-dark-300 dark:hover:text-primary-400;
}
.home-icon-button {
  @apply h-11 w-11 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-dark-300 dark:hover:bg-dark-800;
}
.home-info-card {
  @apply min-w-0 rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900;
}
</style>
