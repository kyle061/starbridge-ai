<template>
  <div :class="styles.base()">
    <span :class="styles.dot()" aria-hidden="true" />
    <span :class="styles.content()">
      {{ label }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { chip } from '@heroui/theme'

const props = defineProps<{
  status: string
  label: string
}>()

const color = computed(() => {
  switch (props.status) {
    case 'active':
    case 'success':
      return 'success'
    case 'disabled':
    case 'inactive':
    case 'warning':
      return 'warning'
    case 'error':
    case 'danger':
      return 'danger'
    default:
      return 'default'
  }
})

const styles = computed(() => chip({ variant: 'dot', size: 'sm', radius: 'sm', color: color.value }))
</script>
