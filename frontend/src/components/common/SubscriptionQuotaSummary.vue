<template>
  <div class="space-y-2" data-testid="subscription-quota-summary">
    <template v-if="quota > 0">
      <div class="grid grid-cols-3 gap-2 text-xs">
        <div><p class="text-gray-500">{{ t('userSubscriptions.totalQuota') }}</p><p class="break-all font-medium">¥{{ formatSubscriptionQuota(quota) }}</p></div>
        <div><p class="text-gray-500">{{ t('userSubscriptions.usedQuota') }}</p><p class="break-all font-medium">¥{{ formatSubscriptionQuota(used) }}</p></div>
        <div><p class="text-gray-500">{{ t('userSubscriptions.remainingQuota') }}</p><p class="break-all font-medium text-primary-600 dark:text-primary-400">¥{{ formatSubscriptionQuota(Math.max(0, quota - used)) }}</p></div>
      </div>
      <div class="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600" role="progressbar" :aria-label="t('userSubscriptions.subscriptionQuota')" :aria-valuenow="percentage" aria-valuemin="0" aria-valuemax="100">
        <div class="h-full rounded-full transition-all" :class="percentage >= 100 ? 'bg-red-500' : 'bg-primary-500'" :style="{ width: `${percentage}%` }" />
      </div>
      <p v-if="used >= quota" class="text-xs text-red-600 dark:text-red-400">{{ t('userSubscriptions.quotaExhausted') }}</p>
    </template>
    <p v-else class="text-xs text-gray-500">{{ t('userSubscriptions.quotaNotConfigured') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatSubscriptionQuota } from '@/utils/subscriptionQuota'
const props = defineProps<{ quota: number; used: number }>()
const { t } = useI18n()
const percentage = computed(() => props.quota > 0 ? Math.max(0, Math.min(100, props.used / props.quota * 100)) : 0)
</script>
