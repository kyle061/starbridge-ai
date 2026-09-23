import { useAutoRefresh } from '@/composables/useAutoRefresh'

interface AdminListAutoRefreshOptions {
  storageKey: string
  onRefresh: () => Promise<void> | void
  shouldPause?: () => boolean
  /** Slower cadence for lists whose refresh also loads secondary statistics. */
  intervalSeconds?: readonly number[]
  defaultInterval?: number
}

/** Shared, low-frequency refresh behavior for admin list pages. */
export function useAdminListAutoRefresh(options: AdminListAutoRefreshOptions) {
  return useAutoRefresh({
    storageKey: `admin-list-auto-refresh:${options.storageKey}`,
    intervals: options.intervalSeconds ?? [60, 120, 300],
    defaultInterval: options.defaultInterval ?? 60,
    defaultEnabled: true,
    onRefresh: options.onRefresh,
    shouldPause: () => document.hidden || Boolean(options.shouldPause?.()),
  })
}
