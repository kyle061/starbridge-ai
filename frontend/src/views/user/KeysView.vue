<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col gap-3">
          <div v-if="prepaidAccess?.enabled" class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900 dark:bg-primary-950/30" data-test="prepaid-access">
            <div class="space-y-1 text-sm">
              <p class="font-semibold text-gray-900 dark:text-white">{{ t('keys.prepaidBalance', { balance: prepaidAccess.balance.toFixed(8).replace(/0+$/, '').replace(/\.$/, '') }) }}</p>
              <p class="text-gray-600 dark:text-gray-300">{{ t(prepaidStatusMessage) }}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('keys.prepaidLifetimeHint') }}</p>
            </div>
            <a href="/redeem" class="btn btn-primary">{{ t(prepaidAccess.has_purchased ? 'keys.renewBalance' : 'keys.purchaseBalance') }}</a>
          </div>
          <p v-if="prepaidAccessError" class="text-sm text-amber-600" role="alert">{{ t('keys.prepaidAccessUnavailable') }}</p>
          <div class="grid grid-cols-2 items-center gap-3 sm:flex sm:flex-wrap">
            <SearchInput
              v-model="filterSearch"
              :placeholder="t('keys.searchPlaceholder')"
              class="col-span-2 w-full sm:w-64"
              @search="onFilterChange"
            />
            <Select
              :model-value="filterGroupId"
              class="min-w-0 w-full sm:w-40"
              :options="groupFilterOptions"
              @update:model-value="onGroupFilterChange"
            />
            <Select
              :model-value="filterStatus"
              class="min-w-0 w-full sm:w-40"
              :options="statusFilterOptions"
              @update:model-value="onStatusFilterChange"
            />
          </div>
          <EndpointPopover
            :api-base-url="effectiveApiBaseUrl"
            :custom-endpoints="publicSettings?.custom_endpoints || []"
          />
          <p class="text-xs leading-5 text-gray-500 dark:text-gray-400">
            {{ t('keys.gatewayUsageHint') }}
          </p>
          <div v-if="selectedIds.length" class="flex flex-wrap items-center gap-3 text-sm">
            <span class="text-gray-600 dark:text-gray-300">
              {{ t('keys.bulkEdit.selectedCount', { count: selectedIds.length }) }}
            </span>
            <button
              class="btn btn-primary btn-sm"
              :disabled="loading"
              data-test="bulk-edit-keys"
              @click="showBulkEditModal = true"
            >
              {{ t('keys.bulkEdit.title') }}
            </button>
            <button class="btn btn-secondary btn-sm" @click="selectedIds = []">
              {{ t('keys.bulkEdit.clearSelection') }}
            </button>
          </div>
        </div>
      </template>

      <template #actions>
        <div class="flex flex-wrap justify-end gap-3">
          <button
            @click="loadApiKeys"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <div class="relative" ref="columnDropdownRef">
            <button
              @click="showColumnDropdown = !showColumnDropdown"
              class="btn btn-secondary px-2 md:px-3"
              :title="t('keys.columnSettings')"
            >
              <svg class="h-4 w-4 md:mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" />
              </svg>
              <span class="hidden md:inline">{{ t('keys.columnSettings') }}</span>
            </button>
            <div
              v-if="showColumnDropdown"
              class="absolute left-1/2 top-full z-50 mt-1 max-h-80 w-48 -translate-x-1/2 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800 md:left-auto md:right-0 md:translate-x-0"
            >
              <button
                v-for="col in toggleableColumns"
                :key="col.key"
                @click="toggleColumn(col.key)"
                class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <span>{{ col.label }}</span>
                <Icon
                  v-if="isColumnVisible(col.key)"
                  name="check"
                  size="sm"
                  class="text-primary-500"
                  :stroke-width="2"
                />
              </button>
            </div>
          </div>
          <button @click="showCreateModal = true" :disabled="!canCreateKey" class="btn btn-primary" data-tour="keys-create-btn">
            <Icon name="plus" size="md" class="mr-2" />
            {{ t('keys.createKey') }}
          </button>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="apiKeys"
          :loading="loading"
          selectable
          row-key="id"
          :selected-keys="selectedIds"
          :selection-label="(key: ApiKey) => t('keys.bulkEdit.selectKey', { name: key.name })"
          @update:selected-keys="handleSelectionChange"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-id="{ value }">
            <span class="font-mono text-xs text-gray-500 dark:text-gray-400">#{{ value }}</span>
          </template>

          <template #cell-key="{ value, row }">
            <div class="flex items-center gap-2">
              <code class="code text-xs">
                {{ maskApiKey(value) }}
              </code>
              <button
                @click="copyToClipboard(value, row.id)"
                class="rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-dark-700"
                :class="
                  copiedKeyId === row.id
                    ? 'text-green-500'
                    : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'
                "
                :title="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
              >
                <Icon
                  v-if="copiedKeyId === row.id"
                  name="check"
                  size="sm"
                  :stroke-width="2"
                />
                <Icon v-else name="clipboard" size="sm" />
              </button>
            </div>
          </template>

          <template #cell-name="{ value, row }">
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <Icon
                v-if="row.ip_whitelist?.length > 0 || row.ip_blacklist?.length > 0"
                name="shield"
                size="sm"
                class="text-blue-500"
                :title="t('keys.ipRestrictionEnabled')"
              />
            </div>
          </template>

          <template #cell-group="{ row }">
            <div class="group/dropdown relative">
              <button
                :ref="(el) => setGroupButtonRef(row.id, el)"
                @click="openGroupSelector(row)"
                class="-my-1 flex w-full min-w-0 cursor-pointer flex-wrap items-center justify-end gap-2 rounded-lg px-2 py-1 transition-all duration-200 hover:bg-gray-100 dark:hover:bg-dark-700 md:-mx-2 md:w-auto md:flex-nowrap md:justify-start"
                :title="t('keys.clickToChangeGroup')"
              >
                <GroupBadge
                  :show-rate="false"
                  v-if="row.group"
                  :name="row.group.name"
                  :platform="row.group.platform"
                  :subscription-type="row.group.subscription_type"
                />
                <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{
                  t('keys.noGroup')
                }}</span>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('keys.selectGroup') }}</span>
                <svg
                  class="h-3.5 w-3.5 text-gray-400 opacity-60 transition-opacity group-hover/dropdown:opacity-100"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9"
                  />
                </svg>
              </button>
            </div>
          </template>

          <template #cell-current_concurrency="{ value }">
            <span
              :class="[
                'inline-flex min-w-8 items-center justify-center rounded px-2 py-1 text-sm font-semibold tabular-nums',
                (value ?? 0) > 0
                  ? 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-900/25 dark:text-emerald-300 dark:ring-emerald-800'
                  : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400'
              ]"
            >
              {{ value ?? 0 }}
            </span>
          </template>

          <template #cell-usage="{ row }">
            <span v-if="usageLoading" class="text-xs text-gray-400" role="status">{{ t('common.loading') }}</span>
            <span v-else-if="usageLoadFailed || !usageStats[row.id]" class="text-xs text-amber-600 dark:text-amber-400" role="status">
              {{ t('keys.usageUnavailable') }}
            </span>
            <div v-else class="text-sm">
              <div class="flex items-center gap-1.5">
                <span class="text-gray-500 dark:text-gray-400">{{ t('keys.today') }}:</span>
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ formatUsageCost(usageStats[row.id]?.today_actual_cost ?? 0) }}
                </span>
              </div>
              <div class="text-xs tabular-nums text-gray-500 dark:text-gray-400">
                {{ t('keys.requestCount', { count: (usageStats[row.id]?.today_requests ?? 0).toLocaleString() }) }}
                · {{ (usageStats[row.id]?.today_tokens ?? 0).toLocaleString() }} Tokens
              </div>
              <div class="mt-0.5 flex items-center gap-1.5">
                <span class="text-gray-500 dark:text-gray-400">{{ t('keys.total') }}:</span>
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ formatUsageCost(usageStats[row.id]?.total_actual_cost ?? 0) }}
                </span>
              </div>
              <div class="text-xs tabular-nums text-gray-500 dark:text-gray-400">
                {{ t('keys.requestCount', { count: (usageStats[row.id]?.total_requests ?? 0).toLocaleString() }) }}
                · {{ (usageStats[row.id]?.total_tokens ?? 0).toLocaleString() }} Tokens
              </div>
              <!-- Quota progress (if quota is set) -->
              <div v-if="row.quota > 0" class="mt-1.5">
                <div class="flex items-center gap-1.5">
                  <span class="text-gray-500 dark:text-gray-400">{{ t('keys.quota') }}:</span>
                  <span :class="[
                    'font-medium',
                    row.quota_used >= row.quota ? 'text-red-500' :
                    row.quota_used >= row.quota * 0.8 ? 'text-yellow-500' :
                    'text-gray-900 dark:text-white'
                  ]">
                    {{ formatUsageCost(row.quota_used ?? 0) }} / {{ formatUsageCost(row.quota ?? 0) }}
                  </span>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      row.quota_used >= row.quota ? 'bg-red-500' :
                      row.quota_used >= row.quota * 0.8 ? 'bg-yellow-500' :
                      'bg-primary-500'
                    ]"
                    :style="{ width: Math.min((row.quota_used / row.quota) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>
          </template>

          <template #cell-expires_at="{ value }">
            <span v-if="value" :class="[
              'text-sm',
              new Date(value) < new Date() ? 'text-red-500 dark:text-red-400' : 'text-gray-500 dark:text-dark-400'
            ]">
              {{ formatDateTime(value) }}
            </span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{ t('keys.noExpiration') }}</span>
          </template>

          <template #cell-status="{ value }">
            <span v-if="value === 'active' && prepaidAccess?.enabled && !prepaidAccess.requests_allowed" class="badge badge-warning">
              {{ t(prepaidAccess.has_purchased ? 'keys.prepaidPausedStatus' : 'keys.prepaidPurchaseStatus') }}
            </span>
            <span v-else :class="[
              'badge',
              value === 'active' ? 'badge-success' :
              value === 'quota_exhausted' ? 'badge-warning' :
              value === 'expired' ? 'badge-danger' :
              'badge-gray'
            ]">
              {{ t('keys.status.' + value) }}
            </span>
          </template>

          <template #cell-last_used_at="{ value }">
            <span v-if="value" class="text-sm text-gray-500 dark:text-dark-400">
              {{ formatDateTime(value) }}
            </span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>

          <template #cell-last_used_ip="{ value }">
            <span v-if="value" class="text-sm text-gray-500 dark:text-dark-400">
              {{ value }}
            </span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <!-- Use Key Button -->
              <button
                @click="openUseKeyModal(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400"
              >
                <Icon name="terminal" size="sm" />
                <span class="text-xs">{{ t('keys.useKey') }}</span>
              </button>
              <!-- Import to CC Switch Button -->
              <button
                v-if="!publicSettings?.hide_ccs_import_button"
                @click="importToCcswitch(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
              >
                <Icon name="upload" size="sm" />
                <span class="text-xs">{{ t('keys.importToCcSwitch') }}</span>
              </button>
              <!-- Toggle Status Button -->
              <button
                @click="toggleKeyStatus(row)"
                :class="[
                  'flex flex-col items-center gap-0.5 rounded-lg p-1.5 transition-colors',
                  row.status === 'active'
                    ? 'text-gray-500 hover:bg-yellow-50 hover:text-yellow-600 dark:hover:bg-yellow-900/20 dark:hover:text-yellow-400'
                    : 'text-gray-500 hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400'
                ]"
              >
                <Icon v-if="row.status === 'active'" name="ban" size="sm" />
                <Icon v-else name="checkCircle" size="sm" />
                <span class="text-xs">{{ row.status === 'active' ? t('keys.disable') : t('keys.enable') }}</span>
              </button>
              <!-- Edit Button -->
              <button
                @click="editKey(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <!-- Delete Button -->
              <button
                @click="confirmDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
              >
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('keys.noKeysYet')"
              :description="t('keys.createFirstKey')"
              :action-text="t('keys.createKey')"
              @action="showCreateModal = true"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
        <div>
          <label class="input-label">{{ t('keys.nameLabel') }}</label>
          <input
            v-model="formData.name"
            type="text"
            required
            class="input"
            :placeholder="t('keys.namePlaceholder')"
            data-tour="key-form-name"
          />
        </div>

        <div>
          <label class="input-label">{{ t('keys.groupLabel') }}</label>
          <Select
            v-model="formData.group_id"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            :searchable="true"
            :search-placeholder="t('keys.searchGroup')"
            data-tour="key-form-group"
          >
            <template #selected="{ option }">
              <GroupBadge
                  :show-rate="false"
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
              />
              <span v-else class="text-gray-400">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
        </div>

        <div v-if="showEditModal">
          <label class="input-label">{{ t('keys.statusLabel') }}</label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <p v-if="!showEditModal" class="input-hint">{{ t('keys.prepaidLifetimeHint') }}</p>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModals" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            form="key-form"
            type="submit"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="key-form-submit"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BulkEditKeysModal
      :show="showBulkEditModal"
      :selected-keys="selectedApiKeys"
      :groups="groups"
      @close="showBulkEditModal = false"
      @updated="handleBulkUpdated"
    />

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="effectiveApiBaseUrl"
      :platform="selectedKey?.group?.platform || null"
      :allow-messages-dispatch="selectedKey?.group?.allow_messages_dispatch || false"
      @close="closeUseKeyModal"
    />

    <!-- CCS Client Selection Dialog for Antigravity -->
    <BaseDialog
      :show="showCcsClientSelect"
      :title="t('keys.ccsClientSelect.title')"
      width="narrow"
      @close="closeCcsClientSelect"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t('keys.ccsClientSelect.description') }}
	        </p>
	        <div class="grid grid-cols-2 gap-3">
	          <button
	            @click="handleCcsClientSelect('claude')"
	            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
	          >
	            <Icon name="terminal" size="xl" class="text-gray-600 dark:text-gray-400" />
	            <span class="font-medium text-gray-900 dark:text-white">{{
	              t('keys.ccsClientSelect.claudeCode')
	            }}</span>
	            <span class="text-xs text-gray-500 dark:text-gray-400">{{
	              t('keys.ccsClientSelect.claudeCodeDesc')
	            }}</span>
	          </button>
	          <button
	            @click="handleCcsClientSelect('gemini')"
	            class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
	          >
	            <Icon name="sparkles" size="xl" class="text-gray-600 dark:text-gray-400" />
	            <span class="font-medium text-gray-900 dark:text-white">{{
	              t('keys.ccsClientSelect.geminiCli')
	            }}</span>
	            <span class="text-xs text-gray-500 dark:text-gray-400">{{
	              t('keys.ccsClientSelect.geminiCliDesc')
	            }}</span>
	          </button>
	        </div>
	      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeCcsClientSelect" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- CCS import fallback and diagnostic details -->
    <BaseDialog
      :show="showCcsImportFallback"
      :title="t('keys.ccsImportFallback.title')"
      width="narrow"
      @close="closeCcsImportFallback"
    >
      <div class="space-y-3">
        <p class="text-sm leading-5 text-gray-600 dark:text-gray-300">
          {{ t('keys.ccsImportFallback.description') }}
        </p>
        <textarea
          :value="ccsImportUrl"
          rows="5"
          readonly
          spellcheck="false"
          class="block w-full max-w-full resize-y break-all rounded-lg border border-gray-200 bg-gray-50 p-3 text-xs leading-5 text-gray-700 outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-900 dark:text-gray-200"
          :aria-label="t('keys.ccsImportFallback.linkLabel')"
          @focus="selectCcsImportLink"
          @click="selectCcsImportLink"
        />
        <p class="text-xs leading-5 text-amber-600 dark:text-amber-400">
          {{ t('keys.ccsImportFallback.warning') }}
        </p>
      </div>
      <template #footer>
        <div class="grid w-full grid-cols-1 gap-2 sm:grid-cols-2">
          <button
            type="button"
            class="btn btn-secondary w-full"
            @click="copyCcsImportLink"
          >
            {{ t('keys.ccsImportFallback.copyLink') }}
          </button>
          <button
            type="button"
            class="btn btn-primary w-full text-center"
            @click="retryCcsImport"
          >
            {{ t('keys.ccsImportFallback.retry') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-max max-w-[calc(100vw-16px)] overflow-hidden rounded-xl bg-white shadow-lg ring-1 ring-black/5 duration-200 sm:min-w-[380px] dark:bg-dark-800 dark:ring-white/10"
        style="pointer-events: auto !important;"
        :style="{
          top: dropdownPosition.top !== undefined ? dropdownPosition.top + 'px' : undefined,
          bottom: dropdownPosition.bottom !== undefined ? dropdownPosition.bottom + 'px' : undefined,
          left: dropdownPosition.left + 'px'
        }"
      >
        <!-- Search box -->
        <div class="border-b border-gray-100 p-2 dark:border-dark-700">
          <div class="relative">
            <svg class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="groupSearchQuery"
              type="text"
              class="w-full rounded-lg border border-gray-200 bg-gray-50 py-1.5 pl-8 pr-3 text-sm text-gray-900 placeholder-gray-400 outline-none focus:border-primary-300 focus:ring-1 focus:ring-primary-300 dark:border-dark-600 dark:bg-dark-700 dark:text-white dark:placeholder-gray-500 dark:focus:border-primary-600 dark:focus:ring-primary-600"
              :placeholder="t('keys.searchGroup')"
              @click.stop
            />
          </div>
        </div>
        <!-- Group list -->
        <div class="max-h-80 overflow-y-auto p-1.5">
          <button
            v-for="option in filteredGroupOptions"
            :key="option.value ?? 'null'"
            @click="changeGroup(selectedKeyForGroup!, option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
              'border-b border-gray-100 last:border-0 dark:border-dark-700',
              selectedKeyForGroup?.group_id === option.value ||
              (!selectedKeyForGroup?.group_id && option.value === null)
                ? 'bg-primary-50 dark:bg-primary-900/20'
                : 'hover:bg-gray-100 dark:hover:bg-dark-700'
            ]"
            :title="option.description || undefined"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :description="option.description"
              :selected="
                selectedKeyForGroup?.group_id === option.value ||
                (!selectedKeyForGroup?.group_id && option.value === null)
              "
            />
          </button>
          <!-- Empty state when search has no results -->
          <div v-if="filteredGroupOptions.length === 0" class="py-4 text-center text-sm text-gray-400 dark:text-gray-500">
            {{ t('keys.noGroupFound') }}
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
	import { ref, reactive, computed, watch, onMounted, onUnmounted, type ComponentPublicInstance } from 'vue'
	import { useI18n } from 'vue-i18n'
	import { useAppStore } from '@/stores/app'
	import { useOnboardingStore } from '@/stores/onboarding'
	import { useClipboard } from '@/composables/useClipboard'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'

const { t } = useI18n()
import { keysAPI, authAPI, usageAPI, userGroupsAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BulkEditKeysModal from '@/components/keys/BulkEditKeysModal.vue'
	import DataTable from '@/components/common/DataTable.vue'
	import Pagination from '@/components/common/Pagination.vue'
	import BaseDialog from '@/components/common/BaseDialog.vue'
	import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
	import EmptyState from '@/components/common/EmptyState.vue'
	import Select from '@/components/common/Select.vue'
	import SearchInput from '@/components/common/SearchInput.vue'
	import Icon from '@/components/icons/Icon.vue'
	import UseKeyModal from '@/components/keys/UseKeyModal.vue'
	import EndpointPopover from '@/components/keys/EndpointPopover.vue'
	import GroupBadge from '@/components/common/GroupBadge.vue'
	import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
	import type { ApiKey, Group, PublicSettings, SubscriptionType, GroupPlatform, UpdateApiKeyRequest } from '@/types'
import type { Column } from '@/components/common/types'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import type { PrepaidAccess } from '@/api/keys'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'
import {
  buildCcSwitchImportDeeplink,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'

interface GroupOption {
  value: number
  label: string
  description: string | null
  subscriptionType: SubscriptionType
  platform: GroupPlatform
}

const appStore = useAppStore()
const onboardingStore = useOnboardingStore()
const { copyToClipboard: clipboardCopy } = useClipboard()

const allColumns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name'), sortable: true },
  { key: 'id', label: t('keys.id'), sortable: true },
  { key: 'key', label: t('keys.apiKey'), sortable: false },
  { key: 'group', label: t('keys.group'), sortable: false },
  { key: 'current_concurrency', label: t('keys.currentConcurrency'), sortable: true },
  { key: 'usage', label: t('keys.usage'), sortable: false },
  { key: 'expires_at', label: t('keys.expiresAt'), sortable: true },
  { key: 'status', label: t('common.status'), sortable: true },
  { key: 'last_used_at', label: t('keys.lastUsedAt'), sortable: true },
  { key: 'last_used_ip', label: t('keys.lastUsedIP'), sortable: false },
  { key: 'created_at', label: t('keys.created'), sortable: true },
  { key: 'actions', label: t('common.actions'), sortable: false }
])

const ALWAYS_VISIBLE_COLUMNS = new Set(['name', 'actions'])
const DEFAULT_HIDDEN_COLUMNS = ['id', 'last_used_at', 'last_used_ip']
const HIDDEN_COLUMNS_KEY = 'api-key-hidden-columns'
const COLUMN_SETTINGS_VERSION_KEY = 'api-key-column-settings-version'
const COLUMN_SETTINGS_VERSION = 3
const VERSION_NEW_HIDDEN_COLUMNS: Record<number, string[]> = {
  2: ['last_used_ip'],
  3: ['id']
}

const toggleableColumns = computed(() =>
  allColumns.value.filter((col) => !ALWAYS_VISIBLE_COLUMNS.has(col.key))
)

const hiddenColumns = reactive<Set<string>>(new Set())

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
    localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
  } catch (error) {
    console.error('Failed to save API key table columns:', error)
  }
}

const loadSavedColumns = () => {
  hiddenColumns.clear()
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      const validColumnKeys = new Set(allColumns.value.map((col) => col.key))
      parsed
        .filter((key) =>
          typeof key === 'string' &&
          validColumnKeys.has(key) &&
          !ALWAYS_VISIBLE_COLUMNS.has(key)
        )
        .forEach((key) => hiddenColumns.add(key))
      const storedVersion = Number(localStorage.getItem(COLUMN_SETTINGS_VERSION_KEY) ?? '1')
      if (storedVersion < COLUMN_SETTINGS_VERSION) {
        for (let v = storedVersion + 1; v <= COLUMN_SETTINGS_VERSION; v++) {
          for (const key of VERSION_NEW_HIDDEN_COLUMNS[v] ?? []) {
            if (validColumnKeys.has(key) && !ALWAYS_VISIBLE_COLUMNS.has(key)) {
              hiddenColumns.add(key)
            }
          }
        }
        saveColumnsToStorage()
      } else {
        localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
      }
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
      localStorage.setItem(COLUMN_SETTINGS_VERSION_KEY, String(COLUMN_SETTINGS_VERSION))
    }
  } catch (error) {
    console.error('Failed to load API key table columns:', error)
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => hiddenColumns.add(key))
  }
}

const toggleColumn = (key: string) => {
  if (ALWAYS_VISIBLE_COLUMNS.has(key)) return
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
}

const isColumnVisible = (key: string) => !hiddenColumns.has(key)

const columns = computed<Column[]>(() =>
  allColumns.value.filter((col) => ALWAYS_VISIBLE_COLUMNS.has(col.key) || !hiddenColumns.has(col.key))
)

const apiKeys = ref<ApiKey[]>([])
const selectedIds = ref<number[]>([])
const showBulkEditModal = ref(false)
const selectedApiKeys = computed(() => apiKeys.value.filter((key) => selectedIds.value.includes(key.id)))

const handleSelectionChange = (ids: Array<string | number>) => {
  const visibleIds = new Set(apiKeys.value.map((key) => key.id))
  selectedIds.value = [...new Set(ids.map(Number))].filter((id) => visibleIds.has(id))
}

const handleBulkUpdated = (succeededIds: number[]) => {
  const succeeded = new Set(succeededIds)
  selectedIds.value = selectedIds.value.filter((id) => !succeeded.has(id))
  loadApiKeys()
}

const groups = ref<Group[]>([])
const loading = ref(false)
const submitting = ref(false)
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({})
const usageLoading = ref(false)
const usageLoadFailed = ref(false)

const formatUsageCost = (amount: number): string => {
  if (amount > 0 && amount < 0.00000001) return '<$0.00000001'
  return `$${amount.toFixed(amount > 0 && amount < 0.0001 ? 8 : 4)}`
}

const pagination = ref({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})
const sortState = ref({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

// Filter state
const filterSearch = ref('')
const filterStatus = ref('')
const filterGroupId = ref<string | number>('')

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showUseKeyModal = ref(false)
const showCcsClientSelect = ref(false)
const showCcsImportFallback = ref(false)
const showColumnDropdown = ref(false)
const pendingCcsRow = ref<ApiKey | null>(null)
const ccsImportUrl = ref('')
const selectedKey = ref<ApiKey | null>(null)
const copiedKeyId = ref<number | null>(null)
const groupSelectorKeyId = ref<number | null>(null)
const publicSettings = ref<PublicSettings | null>(null)
const prepaidAccess = ref<PrepaidAccess | null>(null)
const prepaidAccessError = ref(false)
const canCreateKey = computed(() => !prepaidAccessError.value && !!prepaidAccess.value?.can_create_key)
const prepaidStatusMessage = computed(() => !prepaidAccess.value?.has_purchased
  ? 'keys.prepaidPurchaseRequired'
  : prepaidAccess.value.requests_allowed ? 'keys.prepaidReady' : 'keys.prepaidPaused')
let prepaidTimer: ReturnType<typeof setInterval> | null = null
let prepaidController: AbortController | null = null
let ccsImportTimer: number | null = null
let ccsImportBlurHandler: (() => void) | null = null
let ccsImportVisibilityHandler: (() => void) | null = null
let ccsImportWindow: Window | null = null

const loadPrepaidAccess = async () => {
  prepaidController?.abort()
  const controller = new AbortController()
  prepaidController = controller
  try {
    const access = await keysAPI.getAccess({ signal: controller.signal })
    if (controller.signal.aborted) return
    prepaidAccess.value = access
    prepaidAccessError.value = false
  } catch (error) {
    if (!controller.signal.aborted && !isAbortError(error)) prepaidAccessError.value = true
  }
}
const effectiveApiBaseUrl = computed(() => publicSettings.value?.api_base_url?.trim() || window.location.origin)
const dropdownRef = ref<HTMLElement | null>(null)
const columnDropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<{ top?: number; bottom?: number; left: number } | null>(null)
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map())
let abortController: AbortController | null = null

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const setGroupButtonRef = (keyId: number, el: Element | ComponentPublicInstance | null) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el)
  } else {
    groupButtonRefs.value.delete(keyId)
  }
}

const formData = ref({
  name: '',
  group_id: null as number | null,
  status: 'active' as 'active' | 'inactive'
})

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])

const shouldSubmitEditStatus = (key: ApiKey, status: 'active' | 'inactive') => {
  if (key.status === 'quota_exhausted' || key.status === 'expired') {
    return status === 'active'
  }
  return true
}

// Filter dropdown options
const groupFilterOptions = computed(() => [
  { value: '', label: t('keys.allGroups') },
  { value: 0, label: t('keys.noGroup') },
  ...groups.value.map((g) => ({ value: g.id, label: g.name }))
])

const statusFilterOptions = computed(() => [
  { value: '', label: t('keys.allStatus') },
  { value: 'active', label: t('keys.status.active') },
  { value: 'inactive', label: t('keys.status.inactive') },
  { value: 'quota_exhausted', label: t('keys.status.quota_exhausted') },
  { value: 'expired', label: t('keys.status.expired') }
])

const onFilterChange = () => {
  selectedIds.value = []
  pagination.value.page = 1
  loadApiKeys()
}

const onGroupFilterChange = (value: string | number | boolean | null) => {
  filterGroupId.value = value as string | number
  onFilterChange()
}

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string
  onFilterChange()
}

// Convert available groups to user-facing options
const groupOptions = computed(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    subscriptionType: group.subscription_type,
    platform: group.platform
  }))
)

// The default relay group already contains the OpenAI + DeepSeek account pool.
// Prefer it for new keys so users do not need to switch groups manually.
const preferredGroupId = computed<number | null>(() => {
  const compositeDefault = groupOptions.value.find((group) =>
    group.platform === 'composite' && group.label.toLowerCase() === 'composite-default'
  )
  if (compositeDefault) return compositeDefault.value

  const openAIDefault = groupOptions.value.find((group) =>
    group.platform === 'openai' && group.label.toLowerCase() === 'openai-default'
  )
  return openAIDefault?.value ?? groupOptions.value[0]?.value ?? null
})

watch([showCreateModal, groups], () => {
  if (showCreateModal.value && !showEditModal.value && formData.value.group_id === null) {
    formData.value.group_id = preferredGroupId.value
  }
})

// Group dropdown search
const groupSearchQuery = ref('')
const filteredGroupOptions = computed(() => {
  const query = groupSearchQuery.value.trim().toLowerCase()
  if (!query) return groupOptions.value
  return groupOptions.value.filter((opt) => {
    return opt.label.toLowerCase().includes(query) ||
      (opt.description && opt.description.toLowerCase().includes(query))
  })
})

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t('keys.copied'))
  if (success) {
    copiedKeyId.value = keyId
    setTimeout(() => {
      copiedKeyId.value = null
    }, 800)
  }
}

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

const loadApiKeys = async () => {
  void loadPrepaidAccess()
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  usageLoading.value = true
  usageLoadFailed.value = false
  usageStats.value = {}
  try {
    // Build filters
    const filters: {
      search?: string
      status?: string
      group_id?: number | string
      sort_by?: string
      sort_order?: 'asc' | 'desc'
    } = {}
    if (filterSearch.value) filters.search = filterSearch.value
    if (filterStatus.value) filters.status = filterStatus.value
    if (filterGroupId.value !== '') filters.group_id = filterGroupId.value
    filters.sort_by = sortState.value.sort_by
    filters.sort_order = sortState.value.sort_order

    const response = await keysAPI.list(pagination.value.page, pagination.value.page_size, filters, {
      signal
    })
    if (signal.aborted) return
    apiKeys.value = response.items
    handleSelectionChange(selectedIds.value)
    pagination.value.total = response.total
    pagination.value.pages = response.pages

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id)
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, { signal })
        if (signal.aborted) return
        usageStats.value = usageResponse.stats
      } catch (e) {
        if (!signal.aborted && !isAbortError(e)) {
          usageLoadFailed.value = true
          console.error('Failed to load usage stats:', e)
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
      usageLoading.value = false
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings()
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key
  showUseKeyModal.value = true
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  selectedKey.value = null
}

const handlePageChange = (page: number) => {
  selectedIds.value = []
  pagination.value.page = page
  loadApiKeys()
}

const handlePageSizeChange = (pageSize: number) => {
  selectedIds.value = []
  pagination.value.page_size = pageSize
  pagination.value.page = 1
  loadApiKeys()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  selectedIds.value = []
  sortState.value.sort_by = key
  sortState.value.sort_order = order
  pagination.value.page = 1
  loadApiKeys()
}

const editKey = (key: ApiKey) => {
  selectedKey.value = key
  formData.value = {
    name: key.name,
    group_id: key.group_id,
    status: key.status === 'quota_exhausted' || key.status === 'expired' ? 'inactive' : key.status
  }
  showEditModal.value = true
}

const toggleKeyStatus = async (key: ApiKey) => {
  const newStatus = key.status === 'active' ? 'inactive' : 'active'
  try {
    await keysAPI.toggleStatus(key.id, newStatus)
    appStore.showSuccess(
      newStatus === 'active' ? t('keys.keyEnabledSuccess') : t('keys.keyDisabledSuccess')
    )
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToUpdateStatus'))
  }
}

const openGroupSelector = (key: ApiKey) => {
  if (groupSelectorKeyId.value === key.id) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  } else {
    const buttonEl = groupButtonRefs.value.get(key.id)
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const dropdownEstHeight = 400 // estimated max dropdown height
      const dropdownEstWidth = Math.min(380, window.innerWidth - 16)
      const spaceBelow = window.innerHeight - rect.bottom
      const spaceAbove = rect.top
      // 夹取 left，避免窄屏下浮层超出视口右缘
      const left = Math.max(8, Math.min(rect.left, window.innerWidth - dropdownEstWidth - 8))

      if (spaceBelow < dropdownEstHeight && spaceAbove > spaceBelow) {
        // Not enough space below, pop upward
        dropdownPosition.value = {
          bottom: window.innerHeight - rect.top + 4,
          left
        }
      } else {
        // Default: pop downward
        dropdownPosition.value = {
          top: rect.bottom + 4,
          left
        }
      }
    }
    groupSelectorKeyId.value = key.id
    groupSearchQuery.value = ''
  }
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
  if (key.group_id === newGroupId) return

  try {
    await keysAPI.update(key.id, { group_id: newGroupId })
    appStore.showSuccess(t('keys.groupChangedSuccess'))
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToChangeGroup'))
  }
}

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside the dropdown or the trigger button
  if (!target.closest('.group\\/dropdown') && !dropdownRef.value?.contains(target)) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  }
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(target)) {
    showColumnDropdown.value = false
  }
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  if (!showEditModal.value && !canCreateKey.value) return
  // Validate group_id is required
  if (formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }

  submitting.value = true
  try {
    if (showEditModal.value && selectedKey.value) {
      const updates: UpdateApiKeyRequest = {
        name: formData.value.name,
        group_id: formData.value.group_id,
      }
      if (shouldSubmitEditStatus(selectedKey.value, formData.value.status)) {
        updates.status = formData.value.status
      }
      await keysAPI.update(selectedKey.value.id, updates)
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
    } else {
      await keysAPI.create(formData.value.name, formData.value.group_id)
      appStore.showSuccess(t('keys.keyCreatedSuccess'))
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500)
      }
    }
    closeModals()
    loadApiKeys()
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToSave')
    appStore.showError(errorMsg)
    // Don't advance tour on error
  } finally {
    submitting.value = false
  }
}

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    loadApiKeys()
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  selectedKey.value = null
  formData.value = {
    name: '',
    group_id: null,
    status: 'active'
  }
}

const importToCcswitch = (row: ApiKey) => {
  const platform = row.group?.platform || 'anthropic'

  // For antigravity platform, show client selection dialog
  if (platform === 'antigravity') {
    pendingCcsRow.value = row
    showCcsClientSelect.value = true
    return
  }

  // Composite groups are routed through the OpenAI-compatible Codex app.
  // Other platforms retain their existing client defaults.
  executeCcsImport(row, platform === 'gemini' ? 'gemini' : 'claude')
}

const clearCcsImportAttempt = () => {
  if (ccsImportTimer !== null) {
    window.clearTimeout(ccsImportTimer)
    ccsImportTimer = null
  }
  if (ccsImportBlurHandler) {
    window.removeEventListener('blur', ccsImportBlurHandler)
    ccsImportBlurHandler = null
  }
  if (ccsImportVisibilityHandler) {
    document.removeEventListener('visibilitychange', ccsImportVisibilityHandler)
    ccsImportVisibilityHandler = null
  }
  if (ccsImportWindow && !ccsImportWindow.closed) {
    ccsImportWindow.close()
  }
  ccsImportWindow = null
}

const closeCcsImportFallback = () => {
  clearCcsImportAttempt()
  showCcsImportFallback.value = false
  ccsImportUrl.value = ''
}

const selectCcsImportLink = (event: Event) => {
  const target = event.currentTarget
  if (target instanceof HTMLTextAreaElement) target.select()
}

const copyCcsImportLink = async () => {
  if (!ccsImportUrl.value) return
  await clipboardCopy(ccsImportUrl.value, t('keys.ccsImportFallback.linkCopied'))
}

const openCcsImportLink = (deeplink: string) => {
  // Open the custom scheme from the user click so registered desktop handlers
  // can take over. Keep the window reference to close an unhandled blank tab.
  ccsImportWindow = window.open(deeplink, '_blank')
}

const launchCcsImport = (deeplink: string) => {
  clearCcsImportAttempt()
  ccsImportUrl.value = deeplink
  showCcsImportFallback.value = false

  let launchObserved = false
  const markLaunchObserved = () => {
    launchObserved = true
    clearCcsImportAttempt()
  }

  try {
    ccsImportBlurHandler = markLaunchObserved
    ccsImportVisibilityHandler = () => {
      if (document.hidden) markLaunchObserved()
    }
    window.addEventListener('blur', ccsImportBlurHandler)
    document.addEventListener('visibilitychange', ccsImportVisibilityHandler)
    openCcsImportLink(deeplink)

    ccsImportTimer = window.setTimeout(() => {
      ccsImportTimer = null
      clearCcsImportAttempt()
      if (!launchObserved) showCcsImportFallback.value = true
    }, 1200)
  } catch (error) {
    clearCcsImportAttempt()
    showCcsImportFallback.value = true
  }
}

const retryCcsImport = () => {
  if (ccsImportUrl.value) launchCcsImport(ccsImportUrl.value)
}

const executeCcsImport = (row: ApiKey, clientType: CcSwitchClientType) => {
  const baseUrl = effectiveApiBaseUrl.value
  const platform = row.group?.platform || 'anthropic'

  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`
  const providerName = (publicSettings.value?.site_name || 'sub2api').trim() || 'sub2api'
  const deeplink = buildCcSwitchImportDeeplink({
    baseUrl,
    platform,
    clientType,
    providerName,
    apiKey: row.key,
    usageScript
  })

  launchCcsImport(deeplink)
}

const handleCcsClientSelect = (clientType: CcSwitchClientType) => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType)
  }
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

onMounted(() => {
  loadSavedColumns()
  loadApiKeys()
  loadGroups()
  loadPublicSettings()
  document.addEventListener('click', closeGroupSelector)
  window.addEventListener('focus', loadPrepaidAccess)
  prepaidTimer = setInterval(() => { if (!document.hidden) void loadPrepaidAccess() }, 30000)
})

onUnmounted(() => {
  window.removeEventListener('focus', loadPrepaidAccess)
  clearCcsImportAttempt()
  prepaidController?.abort()
  if (prepaidTimer) clearInterval(prepaidTimer)
  document.removeEventListener('click', closeGroupSelector)
})
</script>
