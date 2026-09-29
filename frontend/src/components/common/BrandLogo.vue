<template>
  <span class="brand-logo" :class="{ 'brand-logo-compact': compact }">
    <img v-if="customLogo" :src="customLogo" :alt="name" class="h-full w-full object-contain" />
    <img v-else-if="compact" src="/logo-mark.png" :alt="name" class="h-full w-full object-contain" />
    <template v-else>
      <img src="/logo.png" :alt="name" class="h-full w-full object-contain dark:hidden" />
      <img src="/logo-dark.png" :alt="name" class="hidden h-full w-full object-contain dark:block" />
    </template>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(defineProps<{
  name?: string
  logo?: string
  compact?: boolean
}>(), {
  name: 'Starbridge AI',
  logo: '',
  compact: false,
})

const customLogo = computed(() => sanitizeUrl(props.logo, {
  allowRelative: true,
  allowDataUrl: true,
}))
</script>

<style scoped>
.brand-logo {
  display: inline-flex;
  width: 12rem;
  max-width: 100%;
  min-width: 0;
  height: 2.25rem;
  vertical-align: middle;
}

.brand-logo-compact {
  width: 2.25rem;
  flex-shrink: 0;
}
</style>
