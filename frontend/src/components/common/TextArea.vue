<template>
  <div class="w-full">
    <label v-if="label" :for="fieldId" class="input-label mb-1.5 block">
      {{ label }}
      <span v-if="required" class="text-red-500" aria-hidden="true">*</span>
    </label>
    <div :class="styles.inputWrapper({ class: 'group border bg-content1 shadow-sm focus-within:border-focus focus-within:ring-2 focus-within:ring-focus/20' })">
      <textarea
        :id="fieldId"
        ref="textAreaRef"
        :value="modelValue"
        :disabled="disabled"
        :required="required"
        :placeholder="placeholderText"
        :readonly="readonly"
        :aria-invalid="error ? 'true' : 'false'"
        :aria-describedby="error || hint ? feedbackId : undefined"
        :rows="rows"
        :class="styles.input({ class: 'min-h-20 resize-y placeholder:text-default-400 max-sm:text-base' })"
        @input="onInput"
        @change="$emit('change', ($event.target as HTMLTextAreaElement).value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
      ></textarea>
    </div>
    <!-- Hint / Error Text -->
    <p v-if="error" :id="feedbackId" class="input-error-text mt-1.5" role="alert">
      <Icon name="exclamationCircle" size="sm" aria-hidden="true" />
      {{ error }}
    </p>
    <p v-else-if="hint" :id="feedbackId" class="input-hint mt-1.5">
      {{ hint }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { input } from '@heroui/theme'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  modelValue: string | null | undefined
  label?: string
  placeholder?: string
  disabled?: boolean
  required?: boolean
  readonly?: boolean
  error?: string
  hint?: string
  id?: string
  rows?: number | string
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  required: false,
  readonly: false,
  rows: 3
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'blur', event: FocusEvent): void
  (e: 'focus', event: FocusEvent): void
}>()

const textAreaRef = ref<HTMLTextAreaElement | null>(null)
const styles = computed(() => input({
  variant: 'bordered', radius: 'md', size: 'md', labelPlacement: 'outside',
  isMultiline: true, isInvalid: Boolean(props.error), isDisabled: props.disabled
}))
const placeholderText = computed(() => props.placeholder || '')
const generatedId = `textarea-${Math.random().toString(36).slice(2, 10)}`
const fieldId = computed(() => props.id || generatedId)
const feedbackId = computed(() => `${fieldId.value}-feedback`)

const onInput = (event: Event) => {
  const value = (event.target as HTMLTextAreaElement).value
  emit('update:modelValue', value)
}

// Expose focus method
defineExpose({
  focus: () => textAreaRef.value?.focus(),
  select: () => textAreaRef.value?.select()
})
</script>
