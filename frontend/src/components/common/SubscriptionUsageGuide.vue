<template>
  <div class="flex justify-end">
    <button type="button" class="btn btn-secondary btn-sm min-h-11" aria-haspopup="dialog" @click="showGuide = true">
      <Icon name="questionCircle" size="sm" />
      {{ t('userSubscriptions.guide.title') }}
    </button>
    <BaseDialog :show="showGuide" :title="t('userSubscriptions.guide.title')" width="wide" close-on-click-outside @close="showGuide = false">
      <p class="text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('userSubscriptions.guide.description') }}</p>

      <ol class="mt-4 grid gap-4 lg:grid-cols-3">
        <li v-for="step in steps" :key="step" class="flex min-w-0 items-start gap-3">
          <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-100 text-xs font-semibold text-primary-700 dark:bg-primary-900/50 dark:text-primary-300" aria-hidden="true">{{ step }}</span>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t(`userSubscriptions.guide.step${step}.title`) }}</h3>
            <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t(`userSubscriptions.guide.step${step}.description`) }}</p>
          </div>
        </li>
      </ol>

      <div class="mt-4 space-y-1 border-t border-gray-200 pt-3 text-xs leading-6 text-gray-600 dark:border-dark-700 dark:text-gray-300">
        <p>{{ t('userSubscriptions.guide.billingHint') }}</p>
        <p>{{ t('userSubscriptions.guide.limitHint') }}</p>
      </div>
      <template #footer>
        <div class="flex flex-wrap justify-end gap-2">
          <RouterLink to="/keys" class="btn btn-primary btn-sm min-h-11" @click="showGuide = false">{{ t('userSubscriptions.guide.openKeys') }}</RouterLink>
          <RouterLink to="/usage" class="btn btn-secondary btn-sm min-h-11" @click="showGuide = false">{{ t('userSubscriptions.guide.openUsage') }}</RouterLink>
          <RouterLink v-if="showSubscriptionsLink" to="/subscriptions" class="btn btn-secondary btn-sm min-h-11" @click="showGuide = false">{{ t('userSubscriptions.title') }}</RouterLink>
        </div>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{ showSubscriptionsLink?: boolean }>()
const { t } = useI18n()
const showGuide = ref(false)
const steps = [1, 2, 3] as const
</script>
