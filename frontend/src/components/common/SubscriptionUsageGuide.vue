<template>
  <section class="rounded-2xl border border-primary-100 bg-primary-50/50 p-4 dark:border-primary-900/50 dark:bg-primary-950/20 sm:p-5" :aria-label="t('userSubscriptions.guide.title')">
    <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('userSubscriptions.guide.title') }}</h2>
      <button type="button" class="min-h-11 shrink-0 text-sm font-medium text-primary-600 hover:underline dark:text-primary-400" :aria-expanded="expanded" aria-controls="subscription-usage-instructions" @click="expanded = !expanded">
        {{ t(expanded ? 'userSubscriptions.guide.hide' : 'userSubscriptions.guide.show') }}
      </button>
    </div>
    <div v-show="expanded" id="subscription-usage-instructions">
      <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('userSubscriptions.guide.description') }}</p>

      <ol class="mt-4 grid gap-4 lg:grid-cols-3">
        <li v-for="step in steps" :key="step" class="flex min-w-0 items-start gap-3">
          <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-100 text-xs font-semibold text-primary-700 dark:bg-primary-900/50 dark:text-primary-300" aria-hidden="true">{{ step }}</span>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t(`userSubscriptions.guide.step${step}.title`) }}</h3>
            <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t(`userSubscriptions.guide.step${step}.description`) }}</p>
          </div>
        </li>
      </ol>

      <div class="mt-4 space-y-1 border-t border-primary-100 pt-3 text-xs leading-6 text-gray-600 dark:border-primary-900/50 dark:text-gray-300">
        <p>{{ t('userSubscriptions.guide.billingHint') }}</p>
        <p>{{ t('userSubscriptions.guide.limitHint') }}</p>
      </div>
      <div class="mt-3 flex flex-wrap gap-2">
        <RouterLink to="/keys" class="btn btn-primary btn-sm min-h-11">{{ t('userSubscriptions.guide.openKeys') }}</RouterLink>
        <RouterLink to="/usage" class="btn btn-secondary btn-sm min-h-11">{{ t('userSubscriptions.guide.openUsage') }}</RouterLink>
        <RouterLink v-if="showSubscriptionsLink" to="/subscriptions" class="btn btn-secondary btn-sm min-h-11">{{ t('userSubscriptions.title') }}</RouterLink>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref } from 'vue'
import { RouterLink } from 'vue-router'

defineProps<{ showSubscriptionsLink?: boolean }>()
const { t } = useI18n()
const expanded = ref(true)
const steps = [1, 2, 3] as const
</script>
