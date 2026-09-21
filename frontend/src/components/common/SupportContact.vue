<template>
  <span v-if="value" class="inline-flex min-w-0 max-w-full flex-wrap items-center gap-x-2 gap-y-1 text-sm">
    <a v-if="href" :href="href" target="_blank" rel="noopener noreferrer" class="inline-flex min-h-11 min-w-0 items-center gap-2 text-primary-600 hover:underline dark:text-primary-400">
      <span class="shrink-0">{{ t('common.contactSupport') }}</span>
      <span v-if="showValue" class="[overflow-wrap:anywhere]">{{ value }}</span>
    </a>
    <template v-else>
      <span>{{ t(isWeChat ? 'common.contactWeChat' : 'common.contactSupport') }}</span>
      <span class="min-w-0 font-medium [overflow-wrap:anywhere]">{{ value }}</span>
      <button type="button" class="inline-flex min-h-11 min-w-11 items-center justify-center rounded-lg px-2 text-primary-600 hover:bg-primary-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-primary-400 dark:hover:bg-primary-900/30" :aria-label="t(isWeChat ? 'common.copyWeChat' : 'common.copy')" @click="copyToClipboard(copyValue, t('common.copiedToClipboard'))">
        <span aria-live="polite">{{ t(copied ? 'common.copied' : 'common.copy') }}</span>
      </button>
    </template>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(defineProps<{ contact: string; showValue?: boolean }>(), { showValue: false })
const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()
const value = computed(() => props.contact.trim())
const href = computed(() => {
  if (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.value)) return `mailto:${value.value}`
  return sanitizeUrl(value.value)
})
const weChatPrefix = /^(?:微信(?:号)?|wechat|weixin)\s*[:：]\s*/i
const copyValue = computed(() => value.value.replace(weChatPrefix, '').trim())
const isWeChat = computed(() => weChatPrefix.test(value.value) || /^[a-zA-Z][\w-]{5,19}$/.test(value.value))
</script>
