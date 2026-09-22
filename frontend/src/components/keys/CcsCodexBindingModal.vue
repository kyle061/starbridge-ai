<template>
  <BaseDialog :show="show" :title="t('keys.ccsCodex.title')" width="wide" @close="emit('close')">
    <div class="space-y-4 text-sm">
      <p class="leading-6 text-gray-600 dark:text-gray-300">{{ t('keys.ccsCodex.description') }}</p>
      <div class="grid gap-2 sm:grid-cols-2" role="radiogroup" :aria-label="t('keys.ccsCodex.modeLabel')">
        <button
          v-for="option in modes"
          :key="option"
          type="button"
          role="radio"
          :aria-checked="mode === option"
          :class="['min-h-11 rounded-lg border px-3 py-2 text-sm font-medium', mode === option ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-950/30 dark:text-primary-300' : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-gray-300']"
          @click="mode = option"
        >{{ t(`keys.ccsCodex.modes.${option}`) }}</button>
      </div>
      <template v-if="mode === 'new'">
        <p class="leading-6 text-gray-600 dark:text-gray-300">{{ t('keys.ccsCodex.newConfigHint') }}</p>
        <button type="button" class="btn btn-primary min-h-11" @click="emit('import-new')">{{ t('keys.ccsCodex.importNew') }}</button>
      </template>
      <template v-else>
      <ol class="list-inside list-decimal space-y-2 leading-6 text-gray-700 dark:text-gray-200">
        <li>{{ t('keys.ccsCodex.step1') }}</li>
        <li>{{ t('keys.ccsCodex.step2') }}</li>
        <li>{{ t('keys.ccsCodex.step3') }}</li>
      </ol>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary min-h-11" @click="copyToClipboard(endpoint)">{{ t('keys.ccsCodex.copyEndpoint') }}</button>
        <button type="button" class="btn btn-secondary min-h-11" @click="copyToClipboard(apiKey)">{{ t('keys.ccsCodex.copyKey') }}</button>
      </div>
      <p class="rounded-lg bg-blue-50 p-3 leading-6 text-blue-800 dark:bg-blue-950/30 dark:text-blue-200">{{ t('keys.ccsCodex.preserveHint') }}</p>
      <label for="ccs-existing-config" class="block font-medium">{{ t('keys.ccsCodex.configLabel') }}</label>
      <textarea id="ccs-existing-config" v-model="source" class="input w-full resize-y whitespace-pre-wrap break-all font-mono text-xs" rows="8" spellcheck="false" autocomplete="off" :placeholder="t('keys.ccsCodex.placeholder')" />
      <p class="text-xs leading-5 text-gray-500">{{ t('keys.ccsCodex.localOnly') }}</p>
      <p v-if="error" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ error }}</p>
      <template v-if="updated">
        <label for="ccs-updated-config" class="block font-medium">{{ t('keys.ccsCodex.resultLabel') }}</label>
        <textarea id="ccs-updated-config" :value="updated" readonly class="input w-full resize-y whitespace-pre-wrap break-all font-mono text-xs" rows="8" />
      </template>
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-primary min-h-11" :disabled="!source.trim() || generating" @click="generate">{{ t('keys.ccsCodex.generate') }}</button>
        <button v-if="updated" type="button" class="btn btn-secondary min-h-11" @click="copyToClipboard(updated)">{{ t('keys.ccsCodex.copyResult') }}</button>
      </div>
      </template>
    </div>
    <template #footer><button class="btn btn-secondary" @click="emit('close')">{{ t('common.close') }}</button></template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useClipboard } from '@/composables/useClipboard'

const props = defineProps<{ show: boolean; apiKey: string; endpoint: string }>()
const emit = defineEmits<{ (event: 'close'): void; (event: 'import-new'): void }>()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const source = ref('')
const updated = ref('')
const error = ref('')
const generating = ref(false)
const modes = ['preserve', 'new'] as const
const mode = ref<typeof modes[number]>('preserve')
let revision = 0
watch([source, mode, () => props.apiKey, () => props.endpoint, () => props.show], () => {
  revision++
  updated.value = ''
  error.value = ''
  if (!props.show || mode.value === 'new') source.value = ''
  if (!props.show) mode.value = 'preserve'
})

async function generate() {
  const currentRevision = revision
  generating.value = true
  try {
    const { updateCodexConnection, CodexConnectionError } = await import('@/utils/codexConnectionConfig')
    if (currentRevision !== revision) return
    try {
      updated.value = updateCodexConnection(source.value, props.endpoint, props.apiKey)
      error.value = ''
    } catch (cause) {
      updated.value = ''
      error.value = t(`keys.ccsCodex.errors.${cause instanceof CodexConnectionError ? cause.code : 'invalid'}`)
    }
  } catch {
    if (currentRevision === revision) error.value = t('keys.ccsCodex.errors.invalid')
  } finally { generating.value = false }
}
</script>
