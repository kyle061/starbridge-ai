<template>
  <BaseDialog :show="show" :title="title" width="narrow" @close="handleCancel">
    <div class="space-y-5">
      <div
        :class="[
          'flex items-start gap-3 rounded-2xl border px-4 py-3.5',
          danger
            ? 'border-red-200/80 bg-red-50/80 dark:border-red-900/60 dark:bg-red-950/30'
            : 'border-primary-200/80 bg-primary-50/80 dark:border-primary-900/60 dark:bg-primary-950/25'
        ]"
      >
        <span
          :class="[
            'mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl',
            danger
              ? 'bg-red-100 text-red-600 dark:bg-red-900/50 dark:text-red-300'
              : 'bg-primary-100 text-primary-600 dark:bg-primary-900/50 dark:text-primary-300'
          ]"
          aria-hidden="true"
        >
          <Icon :name="danger ? 'exclamationTriangle' : 'questionCircle'" size="md" />
        </span>
        <p class="pt-1 text-sm leading-6 text-gray-700 dark:text-gray-300">{{ message }}</p>
      </div>
      <slot></slot>
    </div>

    <template #footer>
      <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <button
          @click="handleCancel"
          type="button"
          class="btn btn-secondary btn-md w-full sm:w-auto"
        >
          {{ cancelText }}
        </button>
        <button
          @click="handleConfirm"
          type="button"
          :class="[
            'btn btn-md w-full sm:w-auto',
            danger ? 'btn-danger' : 'btn-primary'
          ]"
        >
          {{ confirmText }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from './BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

interface Props {
  show: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

interface Emits {
  (e: 'confirm'): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  danger: false
})

const confirmText = computed(() => props.confirmText || t('common.confirm'))
const cancelText = computed(() => props.cancelText || t('common.cancel'))

const emit = defineEmits<Emits>()

const handleConfirm = () => {
  emit('confirm')
}

const handleCancel = () => {
  emit('cancel')
}
</script>
