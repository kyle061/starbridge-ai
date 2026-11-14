<template>
  <div class="auth-layout flex min-h-screen items-center justify-center bg-content1 px-4 py-10 text-foreground">
    <!-- Content Container -->
    <div class="w-full max-w-sm">
      <!-- Logo/Brand -->
      <div class="mb-6 text-center sm:mb-8">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <h1 class="mb-4 flex justify-center">
            <BrandLogo :logo="siteLogo" :name="siteName" class="max-w-full" style="width: 17rem; height: 3rem" />
          </h1>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Card Container -->
      <div class="border-t border-divider pt-7">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import BrandLogo from '@/components/common/BrandLogo.vue'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Starbridge AI')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Multi-model AI API Gateway')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
