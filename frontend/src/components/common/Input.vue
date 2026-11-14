<template>
  <div class="w-full" data-ui="input">
    <label v-if="label" :for="fieldId" class="input-label mb-1.5 block">
      {{ label }}
      <span v-if="required" class="text-red-500" aria-hidden="true">*</span>
    </label>
    <div :class="styles.inputWrapper({ class: 'group border bg-content1 shadow-sm focus-within:border-focus focus-within:ring-2 focus-within:ring-focus/20' })">
      <!-- Prefix Icon Slot -->
      <div
        v-if="$slots.prefix"
        class="pointer-events-none flex shrink-0 items-center text-default-400"
      >
        <slot name="prefix"></slot>
      </div>

      <input
        :id="fieldId"
        ref="inputRef"
        :type="type"
        :value="modelValue"
        :disabled="disabled"
        :required="required"
        :placeholder="placeholderText"
        :autocomplete="autocomplete"
        :autofocus="autofocus"
        :readonly="readonly"
        :aria-invalid="error ? 'true' : 'false'"
        :aria-describedby="error || hint ? feedbackId : undefined"
        :class="styles.input({ class: 'min-w-0 placeholder:text-default-400 max-sm:text-base' })"
        @input="onInput"
        @change="$emit('change', ($event.target as HTMLInputElement).value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
        @keyup.enter="$emit('enter', $event)"
      />

      <!-- Suffix Slot (e.g. Password Toggle or Clear Button) -->
      <div
        v-if="$slots.suffix"
        class="flex shrink-0 items-center text-default-400"
      >
        <slot name="suffix"></slot>
      </div>
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
  modelValue: string | number | null | undefined
  type?: string
  label?: string
  placeholder?: string
  disabled?: boolean
  required?: boolean
  readonly?: boolean
  error?: string
  hint?: string
  id?: string
  autocomplete?: string
  autofocus?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  disabled: false,
  required: false,
  readonly: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'blur', event: FocusEvent): void
  (e: 'focus', event: FocusEvent): void
  (e: 'enter', event: KeyboardEvent): void
}>()

const inputRef = ref<HTMLInputElement | null>(null)
const styles = computed(() => input({
  variant: 'bordered', radius: 'md', size: 'md', labelPlacement: 'outside',
  isInvalid: Boolean(props.error), isDisabled: props.disabled
}))
const placeholderText = computed(() => props.placeholder || '')
const generatedId = `input-${Math.random().toString(36).slice(2, 10)}`
const fieldId = computed(() => props.id || generatedId)
const feedbackId = computed(() => `${fieldId.value}-feedback`)

const onInput = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  emit('update:modelValue', value)
}

// Expose focus method
defineExpose({
  focus: () => inputRef.value?.focus(),
  select: () => inputRef.value?.select()
})
</script>
