<template>
  <ConfirmDialog
    v-if="activeDialog?.type === 'confirm'"
    :show="true"
    :title="activeDialog.options.title || t('common.confirmTitle')"
    :message="activeDialog.options.message"
    :confirm-text="activeDialog.options.confirmText"
    :cancel-text="activeDialog.options.cancelText"
    :danger="activeDialog.options.danger"
    @confirm="resolveConfirm(true)"
    @cancel="resolveConfirm(false)"
  />
  <PromptDialog
    v-else-if="activeDialog?.type === 'prompt'"
    :show="true"
    :title="activeDialog.options.title || t('common.promptTitle')"
    :message="activeDialog.options.message"
    :model-value="promptValue"
    :confirm-text="activeDialog.options.confirmText"
    :cancel-text="activeDialog.options.cancelText"
    :placeholder="activeDialog.options.placeholder"
    :input-type="activeDialog.options.inputType"
    :danger="activeDialog.options.danger"
    @update:model-value="promptValue = $event"
    @confirm="resolvePrompt($event)"
    @cancel="resolvePrompt(null)"
  />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from './ConfirmDialog.vue'
import PromptDialog from './PromptDialog.vue'
import { useDialogHost } from '@/composables/useDialog'

const { t } = useI18n()
const { activeDialog, resolveConfirm, resolvePrompt } = useDialogHost()
const promptValue = ref('')

watch(
  () => activeDialog.value,
  (dialog) => {
    promptValue.value = dialog?.type === 'prompt' ? dialog.options.defaultValue || '' : ''
  },
  { immediate: true }
)
</script>
