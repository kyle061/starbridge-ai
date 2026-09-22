<template>
  <button type="button" class="mt-2 text-sm text-primary-600 hover:underline disabled:opacity-50" :disabled="disabled" @click="open">
    {{ t('admin.settings.secretReveal.view') }}
  </button>
  <BaseDialog :show="show" :title="t('admin.settings.secretReveal.title')" width="normal" :z-index="80" @close="close">
    <form v-if="!value" :id="formId" class="space-y-4" @submit.prevent="reveal">
      <p class="text-sm text-gray-500">{{ t('admin.settings.secretReveal.hint') }}</p>
      <label class="block">
        <span class="input-label">{{ t('admin.settings.secretReveal.password') }}</span>
        <input v-model="password" type="password" autocomplete="current-password" class="input" required :disabled="loading" />
      </label>
      <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
    </form>
    <div v-else class="space-y-3">
      <p class="text-sm text-gray-500">{{ t('admin.settings.secretReveal.expires') }}</p>
      <textarea :value="value" :aria-label="t('admin.settings.secretReveal.savedValue')" readonly rows="5" class="input w-full break-all font-mono text-sm" spellcheck="false" />
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="close">{{ t('common.close') }}</button>
      <button v-if="!value" type="submit" :form="formId" class="btn btn-primary" :disabled="loading || !password">
        {{ loading ? t('admin.settings.secretReveal.verifying') : t('admin.settings.secretReveal.confirm') }}
      </button>
    </template>
  </BaseDialog>
</template>

<script lang="ts">
let formSequence = 0
</script>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { revealSecret, type SecretRevealTarget } from '@/api/admin/settings'
import { extractApiErrorCode } from '@/utils/apiError'

const props = defineProps<{ target: SecretRevealTarget; disabled?: boolean }>()
const { t } = useI18n()
const formId = `secret-reveal-${++formSequence}`
const show = ref(false)
const password = ref('')
const value = ref('')
const error = ref('')
const loading = ref(false)
let generation = 0
let timer: ReturnType<typeof setTimeout> | undefined

function close() {
  generation++
  clearTimeout(timer)
  show.value = false
  password.value = ''
  value.value = ''
  error.value = ''
  loading.value = false
}

function open() {
  close()
  show.value = true
}

async function reveal() {
  if (loading.value || !password.value) return
  const requestGeneration = generation
  loading.value = true
  error.value = ''
  try {
    const secret = await revealSecret(props.target, password.value)
    if (!show.value || requestGeneration !== generation) return
    value.value = secret
    timer = setTimeout(close, 30_000)
  } catch (err) {
    if (requestGeneration !== generation) return
    const code = extractApiErrorCode(err)
    error.value = t(code === 'PASSWORD_INCORRECT'
      ? 'admin.settings.secretReveal.incorrect'
      : code === 'RATE_LIMITED' || code === '429'
        ? 'admin.settings.secretReveal.rateLimited'
        : 'admin.settings.secretReveal.failed')
  } finally {
    if (requestGeneration === generation) {
      password.value = ''
      loading.value = false
    }
  }
}

function onVisibilityChange() {
  if (document.hidden) close()
}
watch(() => JSON.stringify(props.target), close)
onMounted(() => document.addEventListener('visibilitychange', onVisibilityChange))
onBeforeUnmount(() => {
  close()
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>
