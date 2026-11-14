<template>
  <button
    :type="type"
    :class="classes"
    :disabled="disabled || loading"
    :data-disabled="disabled || loading || undefined"
    :aria-busy="loading || undefined"
  >
    <span v-if="loading" class="h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true" />
    <slot />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { button } from '@heroui/theme'

const props = withDefaults(defineProps<{
  type?: 'button' | 'submit' | 'reset'
  color?: 'default' | 'primary' | 'secondary' | 'success' | 'warning' | 'danger'
  variant?: 'solid' | 'bordered' | 'light' | 'flat' | 'faded' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  loading?: boolean
  isIconOnly?: boolean
  fullWidth?: boolean
}>(), {
  type: 'button', color: 'default', variant: 'solid', size: 'md',
  disabled: false, loading: false, isIconOnly: false, fullWidth: false
})

const classes = computed(() => button({
  color: props.color,
  variant: props.variant,
  size: props.size,
  radius: 'md',
  isDisabled: props.disabled || props.loading,
  isIconOnly: props.isIconOnly,
  fullWidth: props.fullWidth,
  class: 'gap-2 font-medium hover:opacity-90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus active:opacity-80 motion-reduce:transition-none max-sm:min-h-11 max-sm:min-w-11'
}))
</script>
