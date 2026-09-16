<template>
  <section
    v-if="showSection"
    class="border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900 sm:p-5"
    data-testid="platform-quota-summary"
  >
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-2">
        <Icon name="shield" size="md" class="shrink-0 text-primary-500" />
        <h2 class="truncate text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('dashboard.platformQuota.title') }}
        </h2>
      </div>
      <div v-if="accountBalance != null" class="text-right">
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.balance') }}</p>
        <p :class="['font-mono text-sm font-semibold', accountBalance > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-500']">
          {{ formatMoney(accountBalance) }}
        </p>
      </div>
    </div>

    <div v-if="configuredQuotas.length" class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="item in configuredQuotas"
        :key="item.platform"
        class="min-w-0 rounded-lg border border-gray-200 p-3 dark:border-dark-700"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ platformLabel(item.platform) }}</span>
          <span v-if="isExhausted(item)" class="shrink-0 text-xs font-medium text-rose-500">{{ t('dashboard.platformQuota.disabled') }}</span>
        </div>
        <div class="mt-2 space-y-2">
          <div v-for="window in quotaWindows" :key="window.key" v-show="getLimit(item, window.key) !== null" class="space-y-1">
            <div class="flex items-center justify-between gap-2 text-xs">
              <span class="text-gray-500 dark:text-gray-400">{{ t(window.label) }}</span>
              <span class="font-mono text-gray-700 dark:text-gray-200">
                {{ formatMoney(getUsage(item, window.key)) }} / {{ formatMoney(getLimit(item, window.key)) }}
              </span>
            </div>
            <div class="h-1.5 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <div
                class="h-full rounded-full transition-all"
                :class="quotaBarClass(getPercent(item, window.key))"
                :style="{ width: `${getPercent(item, window.key)}%` }"
              />
            </div>
            <p v-if="getReset(item, window.key)" class="text-[10px] text-gray-400">
              {{ t('dashboard.platformQuota.resetsAt', { time: formatReset(getReset(item, window.key)) }) }}
            </p>
          </div>
        </div>
      </div>
    </div>
    <p v-else class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.platformQuota.noLimit') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { PlatformQuotaItem } from '@/api/admin/users'

type QuotaWindow = 'daily' | 'weekly' | 'monthly'
type QuotaField = `${QuotaWindow}_limit_usd` | `${QuotaWindow}_usage_usd` | `${QuotaWindow}_window_resets_at`

const props = withDefaults(defineProps<{
  quotas?: PlatformQuotaItem[] | null
  accountBalance?: number | null
}>(), {
  quotas: () => [],
  accountBalance: null,
})

const { t } = useI18n()

const quotaWindows: Array<{ key: QuotaWindow; label: string }> = [
  { key: 'daily', label: 'dashboard.platformQuota.daily' },
  { key: 'weekly', label: 'dashboard.platformQuota.weekly' },
  { key: 'monthly', label: 'dashboard.platformQuota.monthly' },
]

const platformLabels: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  kimi: 'Kimi',
  deepseek: 'DeepSeek',
  minimax: 'MiniMax',
  zhipu: 'Zhipu GLM',
  opencode_go: 'OpenCode Go',
}

const configuredQuotas = computed(() => (props.quotas ?? []).filter((item) => hasAnyLimit(item)))
const showSection = computed(() => props.accountBalance != null || configuredQuotas.value.length > 0)

function platformLabel(platform: string): string {
  return platformLabels[platform] ?? platform
}

function getField(item: PlatformQuotaItem, window: QuotaWindow, suffix: 'limit_usd' | 'usage_usd' | 'window_resets_at'): PlatformQuotaItem[QuotaField] {
  return item[`${window}_${suffix}` as QuotaField]
}

function getLimit(item: PlatformQuotaItem, window: QuotaWindow): number | null {
  return getField(item, window, 'limit_usd') as number | null
}

function getUsage(item: PlatformQuotaItem, window: QuotaWindow): number {
  return (getField(item, window, 'usage_usd') as number | null) ?? 0
}

function getReset(item: PlatformQuotaItem, window: QuotaWindow): string | null {
  return getField(item, window, 'window_resets_at') as string | null
}

function hasAnyLimit(item: PlatformQuotaItem): boolean {
  return item.daily_limit_usd != null || item.weekly_limit_usd != null || item.monthly_limit_usd != null
}

function isExhausted(item: PlatformQuotaItem): boolean {
  return quotaWindows.some((window) => {
    const limit = getLimit(item, window.key)
    return limit === 0 || (limit != null && getUsage(item, window.key) >= limit)
  })
}

function getPercent(item: PlatformQuotaItem, window: QuotaWindow): number {
  const limit = getLimit(item, window)
  if (limit == null || limit <= 0) return limit === 0 ? 100 : 0
  return Math.min(100, Math.max(0, Math.round((getUsage(item, window) / limit) * 100)))
}

function quotaBarClass(percent: number): string {
  if (percent >= 95) return 'bg-rose-500'
  if (percent >= 75) return 'bg-amber-500'
  return 'bg-emerald-500'
}

function formatMoney(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '-'
  const absolute = Math.abs(value)
  const digits = absolute > 0 && absolute < 0.01 ? 8 : 4
  return `$${value.toFixed(digits)}`
}

function formatReset(value: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}
</script>
