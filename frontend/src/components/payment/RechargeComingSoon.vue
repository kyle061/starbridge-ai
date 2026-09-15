<template>
  <div class="card space-y-3 p-6" data-testid="offline-recharge">
    <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('payment.comingSoon.availableTitle') }}</h2>
    <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('payment.comingSoon.availableDescription') }}</p>
    <a href="/redeem" class="btn btn-primary">{{ t('payment.comingSoon.redeem') }}</a>
  </div>
  <section class="card space-y-6 p-6 sm:p-8" data-testid="recharge-coming-soon">
    <div class="space-y-2 text-center">
      <span class="badge badge-warning">{{ t('payment.comingSoon.badge') }}</span>
      <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('payment.comingSoon.title') }}</h2>
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('payment.comingSoon.description') }}</p>
    </div>
    <div class="grid gap-3 sm:grid-cols-2">
      <div v-for="method in methods" :key="method.type"
        class="flex items-center gap-3 rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-800">
        <img :src="method.icon" alt="" class="h-9 w-9" />
        <div>
          <p class="font-medium text-gray-900 dark:text-white">{{ t(`payment.methods.${method.type}`) }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('payment.comingSoon.badge') }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import alipayIcon from '@/assets/icons/alipay.svg'
import wechatIcon from '@/assets/icons/wxpay.svg'

const { t } = useI18n()
const methods = [{ type: 'alipay', icon: alipayIcon }, { type: 'wxpay', icon: wechatIcon }]
</script>
