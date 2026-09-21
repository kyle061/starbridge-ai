<template>
  <div class="min-w-0 space-y-1.5 text-xs">
    <span class="text-gray-500 dark:text-dark-400">{{ t('payment.planCard.models') }}</span>
    <div v-if="models.length" class="flex min-w-0 flex-wrap gap-1">
      <span v-for="model in models" :key="model" class="max-w-full break-words rounded bg-gray-200/80 px-1.5 py-0.5 font-medium text-gray-600 [overflow-wrap:anywhere] dark:bg-dark-600 dark:text-gray-300">{{ model }}</span>
    </div>
    <p v-else class="text-gray-500 dark:text-dark-400">{{ t('payment.planCard.modelsByRoute') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
const props = defineProps<{ allowlist?: SubscriptionPlan['model_allowlist']; platform?: string; scopes?: string[] }>()
const { t } = useI18n()
const labels: Record<string, string> = { claude: 'Claude', gemini_text: 'Gemini', gemini_image: 'Imagen' }
const models = computed(() => props.allowlist?.enabled
  ? (props.allowlist.models || [])
  : props.platform === 'antigravity' ? (props.scopes || []).map(scope => labels[scope] || scope) : [])
</script>
