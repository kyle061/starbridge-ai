<template>
  <BaseDialog :show="show" :title="`${t('payment.admin.configureModels')} · ${plan?.name || ''}`" width="wide" @close="emit('close')">
    <div class="min-w-0 space-y-4">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('payment.admin.modelsHint') }}</p>
      <p v-if="sharedPlans.length > 1" class="rounded-lg bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-300">{{ t('payment.admin.sharedModelsWarning', { plans: sharedPlans.map(p => p.name).join('、') }) }}</p>
      <p v-if="loading" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
      <div v-else-if="loadFailed" class="space-y-2">
        <p class="text-sm text-red-600">{{ t('payment.admin.modelsLoadError') }}</p>
        <button type="button" class="btn btn-secondary" @click="loadModels">{{ t('payment.admin.retryModels') }}</button>
      </div>
      <fieldset v-else :disabled="saving" class="min-w-0 space-y-3">
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input v-model="state.enabled" type="checkbox" class="checkbox" />{{ t('payment.admin.modelsEnabled') }}
        </label>
        <template v-if="state.enabled">
          <input v-model="search" type="search" class="input" :placeholder="t('payment.admin.modelSearch')" :aria-label="t('payment.admin.modelSearch')" />
          <p class="text-xs text-gray-500">{{ t('admin.groups.modelAllowlist.selectedSummary', { selected: selectedCount, total: state.items.length }) }}</p>
          <div class="max-h-64 space-y-1 overflow-y-auto rounded-lg border border-gray-200 p-2 dark:border-dark-600">
            <label v-for="item in filteredItems" :key="item.id" class="flex min-w-0 items-start gap-2 rounded p-2 text-sm hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-700">
              <input v-model="item.selected" type="checkbox" class="checkbox mt-0.5 shrink-0" />
              <span class="min-w-0 break-words [overflow-wrap:anywhere]">{{ item.id }}</span>
            </label>
            <p v-if="!filteredItems.length" class="p-2 text-xs text-gray-500">{{ t('payment.admin.noModelMatches') }}</p>
          </div>
          <div class="flex flex-col gap-2 sm:flex-row">
            <input v-model="customModel" class="input min-w-0 flex-1" :placeholder="t('payment.admin.modelCustom')" :aria-label="t('payment.admin.modelCustom')" @keydown.enter.prevent="addModel" />
            <button type="button" class="btn btn-secondary shrink-0" @click="addModel">{{ t('admin.groups.modelAllowlist.addCustom') }}</button>
          </div>
          <p v-if="customError" role="alert" class="text-xs text-red-600">{{ t(customError) }}</p>
        </template>
      </fieldset>
    </div>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="loading || loadFailed || saving || !plan" @click="saveModels">{{ t(saving ? 'common.saving' : 'common.save') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import * as groupsAPI from '@/api/admin/groups'
import type { SubscriptionPlan } from '@/types/payment'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { extractApiErrorMessage } from '@/utils/apiError'
import { addCustomModelAllowlistItem, buildModelAllowlistConfig, hydrateModelAllowlistState } from '../groupModelAllowlist'
import { filterSupportedModelIds } from '@/composables/useModelWhitelist'

const props = defineProps<{ show: boolean; plan: SubscriptionPlan | null; plans: SubscriptionPlan[] }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const state = ref(hydrateModelAllowlistState(undefined, []))
const loading = ref(false)
const loadFailed = ref(false)
const saving = ref(false)
const search = ref('')
const customModel = ref('')
const customError = ref('')
let loadVersion = 0
const sharedPlans = computed(() => props.plans.filter(p => p.group_id === props.plan?.group_id))
const selectedCount = computed(() => state.value.items.filter(item => item.selected).length)
const filteredItems = computed(() => state.value.items.filter(item => item.id.toLowerCase().includes(search.value.trim().toLowerCase())))

async function loadModels() {
  const version = ++loadVersion
  const groupID = props.plan?.group_id
  if (!props.show || !groupID) return
  loading.value = true
  loadFailed.value = false
  search.value = ''
  customModel.value = ''
  customError.value = ''
  try {
    const [group, candidates] = await Promise.all([groupsAPI.getById(groupID), groupsAPI.getModelAllowlistCandidates(groupID)])
    if (version !== loadVersion) return
    state.value = hydrateModelAllowlistState(
      group.model_allowlist,
      filterSupportedModelIds(group.platform, candidates)
    )
  } catch {
    if (version === loadVersion) loadFailed.value = true
  } finally {
    if (version === loadVersion) loading.value = false
  }
}
watch(() => [props.show, props.plan?.group_id], loadModels, { immediate: true })

function addModel() {
  const error = addCustomModelAllowlistItem(state.value, customModel.value)
  customError.value = error ? `admin.groups.modelAllowlist.errors.${error}` : ''
  if (!error) { customModel.value = ''; search.value = '' }
}

async function saveModels() {
  if (!props.plan || loading.value || loadFailed.value || saving.value) return
  if (state.value.enabled && selectedCount.value === 0) {
    appStore.showError(t('admin.groups.modelAllowlist.emptySelectionError'))
    return
  }
  saving.value = true
  try {
    await groupsAPI.update(props.plan.group_id, { model_allowlist: buildModelAllowlistConfig(state.value) })
    appStore.showSuccess(t('common.saved'))
    emit('saved')
    emit('close')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally { saving.value = false }
}
</script>
