<template>
  <section class="rounded-xl border border-primary-200 bg-primary-50/50 p-3 dark:border-primary-800 dark:bg-primary-900/10">
    <h3 class="text-sm font-medium text-gray-900 dark:text-gray-100">
      {{ t('admin.accounts.providerPresets.title') }}
    </h3>
    <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-400">
      {{ t('admin.accounts.providerPresets.description') }}
    </p>
    <div class="mt-3 flex flex-wrap gap-2">
      <button
        v-for="preset in OPENAI_PROVIDER_PRESETS"
        :key="preset.id"
        type="button"
        :data-testid="`provider-preset-${preset.id}`"
        :aria-pressed="isActive(preset)"
        :class="[
          'rounded-lg border px-3 py-2 text-xs font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500',
          isActive(preset)
            ? 'border-primary-500 bg-primary-100 text-primary-800 dark:bg-primary-900 dark:text-primary-200'
            : 'border-gray-200 bg-white text-gray-700 hover:border-primary-400 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300'
        ]"
        @click="emit('select', preset)"
      >
        {{ preset.label }}
      </button>
    </div>
    <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
      {{ t('admin.accounts.providerPresets.customHint') }}
    </p>
    <p v-if="isLocalProvider" role="status" class="mt-2 text-xs leading-5 text-amber-700 dark:text-amber-400">
      {{ t('admin.accounts.providerPresets.localHint') }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { OPENAI_PROVIDER_PRESETS, type OpenAIProviderPreset } from './openaiProviderPresets'

const props = defineProps<{ currentUrl: string; responsesMode: string }>()
const emit = defineEmits<{ select: [preset: OpenAIProviderPreset] }>()
const { t } = useI18n()
const normalizeUrl = (url: string) => url.trim().replace(/\/+$/, '')
const isActive = (preset: OpenAIProviderPreset) =>
  normalizeUrl(props.currentUrl) === normalizeUrl(preset.baseUrl) && props.responsesMode === preset.responsesMode
const isLocalProvider = computed(() => {
  try {
    return ['host.docker.internal', 'localhost', '127.0.0.1', '[::1]'].includes(new URL(props.currentUrl).hostname)
  } catch {
    return false
  }
})
</script>
