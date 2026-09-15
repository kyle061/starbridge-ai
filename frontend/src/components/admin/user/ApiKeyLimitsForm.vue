<template>
  <form class="mt-4 space-y-3 border-t border-gray-200 pt-4 dark:border-dark-600" @submit.prevent="save">
    <fieldset :disabled="saving" class="grid gap-3 sm:grid-cols-2">
      <label v-for="field in fields" :key="field.key" class="block">
        <span class="input-label">{{ t(field.label) }}</span>
        <input v-model.number="limits[field.key]" type="number" min="0" step="any" required
          class="input" :data-test="field.key" />
      </label>
    </fieldset>
    <p class="input-hint">{{ t('keys.quotaAmountHint') }}</p>
    <div class="flex justify-end gap-2">
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ t('common.cancel') }}</button>
      <button type="submit" class="btn btn-primary" :disabled="saving || !valid">{{ t(saving ? 'keys.saving' : 'common.save') }}</button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { ApiKey } from '@/types'
import type { ApiKeyLimits } from '@/api/admin/apiKeys'

const props = defineProps<{ apiKey: ApiKey }>()
const emit = defineEmits<{ updated: [key: ApiKey]; close: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const fields: Array<{ key: keyof ApiKeyLimits; label: string }> = [
  { key: 'quota', label: 'keys.quotaLimit' },
  { key: 'rate_limit_5h', label: 'keys.rateLimit5h' },
  { key: 'rate_limit_1d', label: 'keys.rateLimit1d' },
  { key: 'rate_limit_7d', label: 'keys.rateLimit7d' }
]
const initial: ApiKeyLimits = {
  quota: props.apiKey.quota, rate_limit_5h: props.apiKey.rate_limit_5h,
  rate_limit_1d: props.apiKey.rate_limit_1d, rate_limit_7d: props.apiKey.rate_limit_7d
}
const limits = reactive({ ...initial })
const valid = computed(() => Object.values(limits).every(value => typeof value === 'number' && Number.isFinite(value) && value >= 0))
const save = async () => {
  if (saving.value || !valid.value) return
  const updates: Partial<ApiKeyLimits> = {}
  for (const { key } of fields) {
    if (limits[key] !== initial[key]) updates[key] = limits[key]
  }
  if (!Object.keys(updates).length) { emit('close'); return }
  saving.value = true
  try {
    const key = await adminAPI.apiKeys.updateApiKeyLimits(props.apiKey.id, updates)
    emit('updated', key)
    appStore.showSuccess(t('keys.keyUpdatedSuccess'))
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || t('keys.failedToSave'))
  } finally {
    saving.value = false
  }
}
</script>
