<template>
  <BaseDialog :show="show" :title="title" width="narrow" @close="handleCancel">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <p class="text-sm text-gray-600 dark:text-gray-400">{{ message }}</p>
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
      <div class="flex justify-end space-x-3">
        <button
          type="button"
          class="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-dark-600 dark:focus:ring-offset-dark-800"
          @click="handleCancel"
        >
          {{ cancelText }}
        </button>
        <button
          type="button"
          :class="[
            'rounded-md px-4 py-2 text-sm font-medium text-white focus:outline-none focus:ring-2 focus:ring-offset-2 dark:focus:ring-offset-dark-800',
            danger
              ? 'bg-red-600 hover:bg-red-700 focus:ring-red-500'
              : 'bg-primary-600 hover:bg-primary-700 focus:ring-primary-500'
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
