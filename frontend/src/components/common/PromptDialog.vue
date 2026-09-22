<template>
  <BaseDialog :show="show" :title="title" width="narrow" @close="handleCancel">
    <form class="space-y-5" @submit.prevent="handleConfirm">
      <div
        class="flex items-start gap-3 rounded-2xl border border-primary-200/80 bg-primary-50/80 px-4 py-3.5 dark:border-primary-900/60 dark:bg-primary-950/25"
      >
        <span
          class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-primary-100 text-primary-600 dark:bg-primary-900/50 dark:text-primary-300"
          aria-hidden="true"
        >
          <Icon name="infoCircle" size="md" />
        </span>
        <p class="pt-1 text-sm leading-6 text-gray-700 dark:text-gray-300">{{ message }}</p>
      </div>
      <div>
        <label :for="inputId" class="sr-only">{{ message }}</label>
        <input
          :id="inputId"
          ref="inputRef"
          v-model="inputValue"
          :type="inputType"
          :placeholder="placeholder"
          class="input"
          autocomplete="off"
        />
      </div>
    </form>

    <template #footer>
      <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <button
          type="button"
          class="btn btn-secondary btn-md w-full sm:w-auto"
          @click="handleCancel"
        >
          {{ cancelText }}
        </button>
        <button
          type="button"
          :class="[
            'btn btn-md w-full sm:w-auto',
            danger ? 'btn-danger' : 'btn-primary'
          ]"
          @click="handleConfirm"
        >
          {{ confirmText }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from './BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  show: boolean
  title: string
  message: string
  modelValue: string
  confirmText?: string
  cancelText?: string
  placeholder?: string
  inputType?: 'text' | 'password'
  danger?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  inputType: 'text',
  danger: false
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'confirm', value: string): void
  (event: 'cancel'): void
}>()

const { t } = useI18n()
const inputRef = ref<HTMLInputElement | null>(null)
const inputId = `dialog-input-${Math.random().toString(36).slice(2, 10)}`

const inputValue = computed({
  get: () => props.modelValue,
  set: (value: string) => emit('update:modelValue', value)
})
const confirmText = computed(() => props.confirmText || t('common.confirm'))
const cancelText = computed(() => props.cancelText || t('common.cancel'))

watch(
  () => props.show,
  async (show) => {
    if (show) {
      await nextTick()
      inputRef.value?.focus()
      inputRef.value?.select()
    }
  }
)

function handleConfirm(): void {
  emit('confirm', inputValue.value)
}

function handleCancel(): void {
  emit('cancel')
}
</script>
