import router from '@/router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { updateDocumentSeo } from '@/utils/seo'

export function syncLocaleDocumentSeo(): void {
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const adminSettingsStore = useAdminSettingsStore()
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
  updateDocumentSeo(
    router.currentRoute.value,
    appStore.siteName,
    customMenuItems,
    appStore.cachedPublicSettings?.site_subtitle,
    { billingMode: resolveSiteBillingMode(appStore.cachedPublicSettings) },
  )
}
