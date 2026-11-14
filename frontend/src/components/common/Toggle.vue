<template>
  <button
    type="button"
    @click="toggle"
    :class="styles.base({ class: 'min-h-11 min-w-11 shrink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus' })"
    :data-selected="modelValue"
    :disabled="disabled"
    role="switch"
    :aria-checked="modelValue"
  >
    <span :class="styles.wrapper({ class: 'm-0' })" aria-hidden="true">
      <span :class="styles.thumb()" />
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { toggle as toggleStyles } from '@heroui/theme'

const props = defineProps<{
  modelValue: boolean
  disabled?: boolean
}>()

const styles = computed(() => toggleStyles({ size: 'sm', color: 'secondary', isDisabled: props.disabled }))

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>
