<template>
  <div class="min-w-0 space-y-4 text-sm text-blue-900 dark:text-blue-200">
    <p>{{ t(`${prefix}.description`) }}</p>
    <p class="text-xs text-blue-700 dark:text-blue-300">{{ t(`${prefix}.enableHint`) }}</p>
    <div class="min-w-0 rounded-lg bg-white/80 p-3 dark:bg-gray-800/80">
      <p class="text-xs">{{ t(`${prefix}.returnAddress`) }}</p>
      <p class="mt-1 break-all font-mono" data-testid="device-return-address">{{ returnAddress }}</p>
    </div>
    <button v-if="!session" type="button" class="btn btn-primary w-full" :disabled="starting || loading" @click="start">
      {{ t(starting ? `${prefix}.starting` : `${prefix}.start`) }}
    </button>
    <template v-else>
      <div class="min-w-0 rounded-lg border border-blue-200 bg-white p-4 text-center dark:border-blue-700 dark:bg-gray-800">
        <p>{{ t(`${prefix}.codeLabel`) }}</p>
        <p class="my-3 break-all font-mono text-2xl font-bold tracking-widest" data-testid="device-code">{{ session.user_code }}</p>
        <button type="button" class="btn btn-secondary w-full" @click="copyToClipboard(session.user_code)">{{ t(copied ? 'common.copied' : 'common.copy') }}</button>
      </div>
      <a href="https://auth.openai.com/codex/device" target="_blank" rel="noopener noreferrer" class="btn btn-primary w-full justify-center">{{ t(`${prefix}.open`) }}</a>
      <p role="status">{{ t(tokenInfo ? `${prefix}.authorized` : `${prefix}.waiting`) }}</p>
      <button v-if="tokenInfo" type="button" class="btn btn-primary w-full" :disabled="loading" @click="emit('authorized', tokenInfo)">{{ t(`${prefix}.save`) }}</button>
      <button type="button" class="btn btn-secondary w-full" :disabled="loading" @click="restart">{{ t(`${prefix}.restart`) }}</button>
    </template>
    <p v-if="error || localError" role="alert" class="break-words text-red-600 dark:text-red-400">{{ error || localError }}</p>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiClient } from '@/api/client'
import { useClipboard } from '@/composables/useClipboard'
import type { OpenAITokenInfo } from '@/composables/useOpenAIOAuth'

const props = defineProps<{ proxyId?: number | null; loading?: boolean; error?: string }>()
const emit = defineEmits<{ authorized: [tokenInfo: OpenAITokenInfo] }>()
const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()
const prefix = 'admin.accounts.oauth.openai.device'
const returnAddress = window.location.origin + window.location.pathname
interface DeviceSession { session_id: string; user_code: string; interval: number; expires_at: number }
const session = ref<DeviceSession | null>(null)
const tokenInfo = ref<OpenAITokenInfo | null>(null)
const starting = ref(false)
const localError = ref('')
let generation = 0
let timer: ReturnType<typeof setTimeout> | undefined
let controller: AbortController | undefined

function cancelSession(id: string) {
  // No credentials or device secrets are included in the URL or browser storage.
  void apiClient.post('/admin/openai/device/cancel', { session_id: id }).catch(() => {})
}
function stop() {
  generation++
  if (timer) clearTimeout(timer)
  controller?.abort()
  if (session.value) cancelSession(session.value.session_id)
  session.value = null
  tokenInfo.value = null
  starting.value = false
}
function schedule(currentGeneration: number, interval: number) {
  timer = setTimeout(() => { void poll(currentGeneration) }, Math.max(5, interval) * 1000)
}
async function poll(currentGeneration: number) {
  const active = session.value
  if (!active || currentGeneration !== generation) return
  if (Date.now() >= active.expires_at * 1000) {
    stop()
    localError.value = t(`${prefix}.expired`)
    return
  }
  try {
    const { data } = await apiClient.post<{ status: string; interval: number; token_info?: OpenAITokenInfo }>('/admin/openai/device/poll', { session_id: active.session_id }, { signal: controller?.signal })
    if (currentGeneration !== generation) return
    if (data.status === 'authorized' && data.token_info?.access_token) {
      tokenInfo.value = data.token_info
      localError.value = ''
      emit('authorized', data.token_info)
      return
    }
    schedule(currentGeneration, data.interval || active.interval)
  } catch (error: unknown) {
    if (currentGeneration !== generation) return
    const status = (error as { response?: { status?: number } }).response?.status
    if (status === 400 || status === 401 || status === 403) {
      stop()
      localError.value = t(`${prefix}.expired`)
      return
    }
    localError.value = t(`${prefix}.retrying`)
    schedule(currentGeneration, active.interval)
  }
}
async function start() {
  stop()
  const currentGeneration = generation
  starting.value = true
  localError.value = ''
  controller = new AbortController()
  try {
    const { data } = await apiClient.post<DeviceSession>('/admin/openai/device/start', { proxy_id: props.proxyId ?? null }, { signal: controller.signal })
    if (currentGeneration !== generation) { cancelSession(data.session_id); return }
    session.value = data
    schedule(currentGeneration, data.interval)
  } catch {
    if (currentGeneration === generation) localError.value = t(`${prefix}.failed`)
  } finally {
    if (currentGeneration === generation) starting.value = false
  }
}
function restart() { stop(); void start() }
watch(() => props.proxyId, () => { stop(); localError.value = '' })
onBeforeUnmount(stop)
</script>
