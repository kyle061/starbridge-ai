import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

import type { ApiKey } from '@/types'
import KeysView from '../KeysView.vue'

const {
  getAccess,
  listKeys,
  createKey,
  updateKey,
  getPublicSettings,
  getDashboardApiKeysUsage,
  getAvailableGroups,
  getUserGroupRates,
  showError,
  showSuccess,
  copyToClipboard,
  isCurrentStep,
  nextStep,
} = vi.hoisted(() => ({
  getAccess: vi.fn(),
  listKeys: vi.fn(),
  createKey: vi.fn(),
  updateKey: vi.fn(),
  getPublicSettings: vi.fn(),
  getDashboardApiKeysUsage: vi.fn(),
  getAvailableGroups: vi.fn(),
  getUserGroupRates: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  copyToClipboard: vi.fn(),
  isCurrentStep: vi.fn(),
  nextStep: vi.fn(),
}))

const messages: Record<string, string> = {
  'common.actions': 'Actions',
  'common.name': 'Name',
  'common.refresh': 'Refresh',
  'common.status': 'Status',
  'keys.apiKey': 'API Key',
  'keys.allGroups': 'All Groups',
  'keys.allStatus': 'All Status',
  'keys.columnSettings': 'Column Settings',
  'keys.createKey': 'Create API Key',
  'keys.created': 'Created',
  'keys.expiresAt': 'Expires',
  'keys.group': 'Group',
  'keys.id': 'ID',
  'keys.importToCcSwitch': 'Import to CC Switch',
  'keys.currentConcurrency': 'Current Concurrency',
  'keys.lastUsedAt': 'Last Used',
  'keys.lastUsedIP': 'Last Used IP',
  'keys.rateLimitColumn': 'Rate Limit',
  'keys.searchPlaceholder': 'Search name or key...',
  'keys.status.active': 'Active',
  'keys.status.expired': 'Expired',
  'keys.status.inactive': 'Inactive',
  'keys.status.quota_exhausted': 'Quota exhausted',
  'keys.usage': 'Usage',
  'keys.today': 'Today',
  'keys.total': 'Last 30d',
  'keys.requestCount': '{count} requests',
  'keys.usageUnavailable': 'Usage unavailable. Refresh to retry.',
}

vi.mock('@/api', () => ({
  keysAPI: {
    getAccess,
    list: listKeys,
    create: createKey,
    update: updateKey,
    delete: vi.fn(),
    toggleStatus: vi.fn(),
  },
  authAPI: {
    getPublicSettings,
  },
  usageAPI: {
    getDashboardApiKeysUsage,
  },
  userGroupsAPI: {
    getAvailable: getAvailableGroups,
    getUserGroupRates,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep,
    nextStep,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params: Record<string, string | number> = {}) =>
        (messages[key] ?? key).replace(/\{(\w+)\}/g, (_, name) => String(params[name] ?? '')),
    }),
  }
})

const createApiKey = (): ApiKey => ({
  id: 1,
  user_id: 1,
  key: 'sk-test-key',
  name: 'test-key',
  group_id: null,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  last_used_ip: null,
  quota: 0,
  quota_used: 0,
  expires_at: null,
  created_at: '2026-06-27T00:00:00Z',
  updated_at: '2026-06-27T00:00:00Z',
  current_concurrency: 3,
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
})

const AppLayoutStub = {
  template: '<div><slot /></div>',
}

const TablePageLayoutStub = {
  template: `
    <div>
      <slot name="filters" />
      <slot name="actions" />
      <slot name="table" />
      <slot name="pagination" />
    </div>
  `,
}

const DataTableStub = {
  name: 'DataTable',
  props: { columns: Array, data: Array, selectedKeys: Array, selectable: Boolean },
  emits: ['sort', 'update:selectedKeys'],
  template: `
    <div>
      <div data-test="columns">{{ columns.map((col) => col.key).join(',') }}</div>
      <div data-test="columns-meta">{{ JSON.stringify(columns.map((col) => ({ key: col.key, sortable: !!col.sortable }))) }}</div>
      <button data-test="sort-current-concurrency" @click="$emit('sort', 'current_concurrency', 'asc')">
        Sort Current Concurrency
      </button>
      <div v-for="row in data" :key="row.id">
        <div
          v-if="columns.some((col) => col.key === 'id')"
          data-test="key-id"
        >
          <slot name="cell-id" :value="row.id" :row="row" />
        </div>
        <slot name="cell-name" :value="row.name" :row="row" />
        <div data-test="key-usage"><slot name="cell-usage" :row="row" /></div>
        <slot name="cell-actions" :row="row" />
        <div data-test="current-concurrency">
          <slot name="cell-current_concurrency" :value="row.current_concurrency" :row="row" />
        </div>
        <div
          v-if="columns.some((col) => col.key === 'last_used_ip')"
          data-test="last-used-ip"
        >
          <slot name="cell-last_used_ip" :value="row.last_used_ip" :row="row" />
        </div>
      </div>
      <slot name="empty" />
    </div>
  `,
}

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"></select>',
}

const SearchInputStub = {
  name: 'SearchInput',
  props: ['modelValue'],
  emits: ['update:modelValue', 'search'],
  template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
}

const PaginationStub = {
  name: 'Pagination',
  props: ['page', 'total', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: `
    <div>
      <button data-test="page-size-50" @click="$emit('update:pageSize', 50)">50</button>
    </div>
  `,
}

const IconStub = {
  props: ['name'],
  template: '<span data-test="icon">{{ name }}</span>',
}

const mountView = async () => {
  const wrapper = mount(KeysView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: PaginationStub,
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        ConfirmDialog: true,
        EmptyState: true,
        Select: SelectStub,
        SearchInput: SearchInputStub,
        Icon: IconStub,
        UseKeyModal: true,
        BulkEditKeysModal: true,
        EndpointPopover: true,
        GroupBadge: true,
        GroupOptionItem: true,
        Teleport: true,
      },
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

const visibleColumnKeys = (wrapper: VueWrapper) =>
  wrapper.get('[data-test="columns"]').text().split(',').filter(Boolean)

const visibleColumnMeta = (wrapper: VueWrapper): Array<{ key: string; sortable: boolean }> =>
  JSON.parse(wrapper.get('[data-test="columns-meta"]').text())

const getButtonByText = (wrapper: VueWrapper, text: string) => {
  const button = wrapper.findAll('button').find((item) => item.text().includes(text))
  if (!button) {
    throw new Error(`Button not found: ${text}`)
  }
  return button
}

describe('user KeysView column settings', () => {
  beforeEach(() => {
    localStorage.clear()

    getAccess.mockReset()
    getAccess.mockResolvedValue({ enabled: false, can_create_key: true, requests_allowed: true, has_purchased: false, balance: 10 })
    listKeys.mockReset()
    createKey.mockReset().mockResolvedValue(createApiKey())
    updateKey.mockReset()
    getPublicSettings.mockReset()
    getDashboardApiKeysUsage.mockReset()
    getAvailableGroups.mockReset()
    getUserGroupRates.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    copyToClipboard.mockReset()
    isCurrentStep.mockReset()
    nextStep.mockReset()

    listKeys.mockResolvedValue({
      items: [createApiKey()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getPublicSettings.mockResolvedValue({})
    getDashboardApiKeysUsage.mockResolvedValue({ stats: {} })
    getAvailableGroups.mockResolvedValue([])
    getUserGroupRates.mockResolvedValue({})
    isCurrentStep.mockReturnValue(false)
  })

  it('requires credit before creating keys and links to code redemption', async () => {
    getAccess.mockResolvedValue({ enabled: true, has_purchased: false, balance: 0, can_create_key: false, requests_allowed: false })
    const wrapper = await mountView()
    expect(wrapper.get('[data-tour="keys-create-btn"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-test="prepaid-access"]').text()).toContain('keys.prepaidPurchaseRequired')
    expect(wrapper.get('[data-test="prepaid-access"] a').attributes('href')).toBe('/redeem')
    wrapper.unmount()
  })

  it('keeps the same key and enables it after a renewal restores the balance', async () => {
    getAccess.mockResolvedValue({ enabled: true, has_purchased: true, balance: 0, can_create_key: false, requests_allowed: false })
    const wrapper = await mountView()
    expect(wrapper.get('[data-tour="keys-create-btn"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-test="prepaid-access"]').text()).toContain('keys.prepaidPaused')
    getAccess.mockResolvedValue({ enabled: true, has_purchased: true, balance: 20, can_create_key: true, requests_allowed: true })
    window.dispatchEvent(new Event('focus'))
    await flushPromises()
    expect(wrapper.get('[data-tour="keys-create-btn"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.get('[data-test="prepaid-access"]').text()).toContain('keys.prepaidReady')
    expect(listKeys).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('disables key creation when balance verification fails', async () => {
    getAccess.mockRejectedValue(new Error('unavailable'))
    const wrapper = await mountView()
    expect(wrapper.get('[data-tour="keys-create-btn"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('keys.prepaidAccessUnavailable')
    wrapper.unmount()
  })

  it('shows the current site endpoint when no custom API base URL is configured', async () => {
    const wrapper = await mountView()
    expect(wrapper.findComponent({ name: 'EndpointPopover' }).props('apiBaseUrl')).toBe(window.location.origin)
    expect(wrapper.findComponent({ name: 'UseKeyModal' }).props('baseUrl')).toBe(window.location.origin)
    wrapper.unmount()
  })

  it('uses the configured endpoint consistently', async () => {
    getPublicSettings.mockResolvedValue({ api_base_url: ' https://gateway.example ' })
    const wrapper = await mountView()
    expect(wrapper.findComponent({ name: 'EndpointPopover' }).props('apiBaseUrl')).toBe('https://gateway.example')
    expect(wrapper.findComponent({ name: 'UseKeyModal' }).props('baseUrl')).toBe('https://gateway.example')
    wrapper.unmount()
  })

  it('opens CCS in a new tab and shows an error when launch is not observed', async () => {
    vi.useFakeTimers()
    const close = vi.fn()
    const openedWindow = { closed: false, close } as unknown as Window
    const open = vi.spyOn(window, 'open').mockReturnValue(openedWindow)
    try {
      const wrapper = await mountView()
      await getButtonByText(wrapper, 'Import to CC Switch').trigger('click')

      expect(open).toHaveBeenCalledWith(expect.stringContaining('ccswitch://v1/import?'), '_blank')
      vi.advanceTimersByTime(1200)
      await nextTick()

      const importLink = wrapper.find('textarea[aria-label="keys.ccsImportFallback.linkLabel"]')
      expect(importLink.exists()).toBe(true)
      expect((importLink.element as HTMLTextAreaElement).value).toContain('ccswitch://v1/import?')
      expect(close).toHaveBeenCalledOnce()
      wrapper.unmount()
    } finally {
      open.mockRestore()
      vi.useRealTimers()
    }
  })

  it('shows measured tokens and requests even when dollar amounts are tiny', async () => {
    getDashboardApiKeysUsage.mockResolvedValue({ stats: { 1: {
      api_key_id: 1, today_actual_cost: 0.0000000001, total_actual_cost: 0.00001234,
      today_requests: 1, total_requests: 3, today_tokens: 42, total_tokens: 123,
    } } })
    const wrapper = await mountView()
    const usage = wrapper.get('[data-test="key-usage"]').text()
    expect(usage).toContain('<$0.00000001')
    expect(usage).toContain('$0.00001234')
    expect(usage).toContain('1 requests')
    expect(usage).toContain('3 requests')
    expect(usage).toContain('42 Tokens')
    expect(usage).toContain('123 Tokens')
    wrapper.unmount()
  })

  it('distinguishes a statistics failure from zero usage and allows refreshing', async () => {
    const errorLog = vi.spyOn(console, 'error').mockImplementation(() => {})
    getDashboardApiKeysUsage.mockRejectedValueOnce(new Error('statistics unavailable'))
    const wrapper = await mountView()
    expect(wrapper.get('[data-test="key-usage"]').text()).toContain('Usage unavailable')
    expect(wrapper.get('[data-test="key-usage"]').text()).not.toContain('$0.0000')

    getDashboardApiKeysUsage.mockResolvedValue({ stats: { 1: {
      api_key_id: 1, today_actual_cost: 0, total_actual_cost: 0,
      today_requests: 1, total_requests: 1, today_tokens: 42, total_tokens: 42,
    } } })
    await wrapper.get('button[title="Refresh"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-test="key-usage"]').text()).not.toContain('Usage unavailable')
    expect(wrapper.get('[data-test="key-usage"]').text()).toContain('42 Tokens')
    expect(wrapper.get('[data-test="key-usage"]').text()).toContain('$0.0000')
    wrapper.unmount()
    errorLog.mockRestore()
  })

  it('creates a funded key using only its name and group', async () => {
    getAccess.mockResolvedValue({ enabled: true, has_purchased: true, balance: 20, can_create_key: true, requests_allowed: true })
    getAvailableGroups.mockResolvedValue([{ id: 1, name: 'Default', rate_multiplier: 2, peak_rate_enabled: true }])
    const wrapper = await mountView()
    await wrapper.get('[data-tour="keys-create-btn"]').trigger('click')
    await wrapper.get('[data-tour="key-form-name"]').setValue('My key')
    const groupSelect = wrapper.findAllComponents({ name: 'Select' })
      .find(select => select.attributes('data-tour') === 'key-form-group')!
    expect(groupSelect.props('options')[0]).not.toHaveProperty('rate')
    expect(groupSelect.props('options')[0]).not.toHaveProperty('peakRateMultiplier')
    groupSelect.vm.$emit('update:modelValue', 1)
    await nextTick()
    expect(wrapper.get('#key-form').find('input[type="number"]').exists()).toBe(false)
    await wrapper.get('#key-form').trigger('submit')
    await flushPromises()
    expect(createKey).toHaveBeenCalledWith('My key', 1)
    wrapper.unmount()
  })

  it('preserves administrator limits when renaming a key', async () => {
    const key = { ...createApiKey(), group_id: 1, quota: 10, quota_used: 4,
      rate_limit_5h: 2, rate_limit_1d: 8, rate_limit_7d: 30, expires_at: '2030-01-01T00:00:00Z' }
    listKeys.mockResolvedValue({ items: [key], total: 1, page: 1, page_size: 20, pages: 1 })
    updateKey.mockResolvedValue(key)
    const wrapper = await mountView()
    await getButtonByText(wrapper, 'common.edit').trigger('click')
    const form = wrapper.get('#key-form')
    expect(form.find('input[type="number"]').exists()).toBe(false)
    expect(form.find('input[type="datetime-local"]').exists()).toBe(false)
    expect(form.text()).not.toContain('keys.rateLimitSection')
    expect(form.text()).not.toContain('keys.customKeyLabel')
    await wrapper.get('[data-tour="key-form-name"]').setValue('Renamed')
    await form.trigger('submit')
    await flushPromises()
    expect(updateKey).toHaveBeenCalledWith(key.id, { name: 'Renamed', group_id: 1, status: 'active' })
    wrapper.unmount()
  })

  it('uses the default API key columns with low-frequency columns hidden', async () => {
    const wrapper = await mountView()

    expect(visibleColumnKeys(wrapper)).toEqual([
      'name',
      'key',
      'group',
      'current_concurrency',
      'usage',
      'expires_at',
      'status',
      'created_at',
      'actions',
    ])
    expect(visibleColumnKeys(wrapper)).not.toContain('rate_limit')
    expect(visibleColumnKeys(wrapper)).not.toContain('last_used_at')
    expect(visibleColumnKeys(wrapper)).not.toContain('last_used_ip')
    expect(visibleColumnKeys(wrapper)).not.toContain('id')
  })

  it('opens bulk editing with only selected visible keys', async () => {
    const wrapper = await mountView()
    const table = wrapper.findComponent({ name: 'DataTable' })
    expect(table.props('selectable')).toBe(true)
    table.vm.$emit('update:selectedKeys', [1, 99])
    await nextTick()
    await wrapper.get('[data-test="bulk-edit-keys"]').trigger('click')
    const modal = wrapper.findComponent({ name: 'BulkEditKeysModal' })
    expect(modal.props('show')).toBe(true)
    expect(modal.props('selectedKeys').map((key: ApiKey) => key.id)).toEqual([1])
    wrapper.unmount()
  })

  it.each(['filter', 'page size', 'sort'])('clears selection on %s changes', async (change) => {
    const wrapper = await mountView()
    const table = wrapper.findComponent({ name: 'DataTable' })
    table.vm.$emit('update:selectedKeys', [1])
    await nextTick()
    if (change === 'filter') {
      wrapper.findComponent({ name: 'SearchInput' }).vm.$emit('search')
    } else if (change === 'page size') {
      await wrapper.get('[data-test="page-size-50"]').trigger('click')
    } else {
      table.vm.$emit('sort', 'created_at', 'asc')
    }
    await flushPromises()
    expect(table.props('selectedKeys')).toEqual([])
    expect(wrapper.find('[data-test="bulk-edit-keys"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('removes successful keys from the selection and refreshes the table', async () => {
    listKeys.mockResolvedValue({
      items: [createApiKey(), { ...createApiKey(), id: 2, name: 'Second' }],
      total: 2, pages: 1
    })
    const wrapper = await mountView()
    const table = wrapper.findComponent({ name: 'DataTable' })
    table.vm.$emit('update:selectedKeys', [1, 2])
    await nextTick()
    await wrapper.get('[data-test="bulk-edit-keys"]').trigger('click')
    wrapper.findComponent({ name: 'BulkEditKeysModal' }).vm.$emit('updated', [1])
    await flushPromises()
    expect(listKeys).toHaveBeenCalledTimes(2)
    expect(table.props('selectedKeys')).toEqual([2])
    wrapper.unmount()
  })

  it('drops keys that are no longer visible after a refresh', async () => {
    const wrapper = await mountView()
    const table = wrapper.findComponent({ name: 'DataTable' })
    table.vm.$emit('update:selectedKeys', [1])
    await nextTick()
    listKeys.mockResolvedValue({ items: [], total: 0, pages: 0 })
    await wrapper.get('button[title="Refresh"]').trigger('click')
    await flushPromises()
    expect(table.props('selectedKeys')).toEqual([])
    wrapper.unmount()
  })

  it('shows a hidden column when toggled and persists the preference', async () => {
    const wrapper = await mountView()

    await wrapper.get('button[title="Column Settings"]').trigger('click')
    await getButtonByText(wrapper, 'Last Used').trigger('click')
    await nextTick()

    expect(visibleColumnKeys(wrapper)).toContain('last_used_at')
    expect(localStorage.getItem('api-key-hidden-columns')).toBe(
      JSON.stringify(['id', 'last_used_ip'])
    )
    expect(localStorage.getItem('api-key-column-settings-version')).toBe('3')
  })

  it('shows the API key ID column when toggled', async () => {
    const wrapper = await mountView()

    await wrapper.get('button[title="Column Settings"]').trigger('click')
    await getButtonByText(wrapper, 'ID').trigger('click')
    await nextTick()

    expect(visibleColumnKeys(wrapper)).toContain('id')
    expect(wrapper.get('[data-test="key-id"]').text()).toBe('#1')
    expect(visibleColumnMeta(wrapper).find((column) => column.key === 'id')?.sortable).toBe(true)
  })

  it('shows the last used IP column when toggled', async () => {
    listKeys.mockResolvedValueOnce({
      items: [{ ...createApiKey(), last_used_ip: '203.0.113.10' }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    const wrapper = await mountView()

    await wrapper.get('button[title="Column Settings"]').trigger('click')
    await getButtonByText(wrapper, 'Last Used IP').trigger('click')
    await nextTick()

    expect(visibleColumnKeys(wrapper)).toContain('last_used_ip')
    expect(wrapper.get('[data-test="last-used-ip"]').text()).toBe('203.0.113.10')
  })

  it('restores column preferences from localStorage on mount', async () => {
    localStorage.setItem('api-key-hidden-columns', JSON.stringify(['group', 'created_at']))
    localStorage.setItem('api-key-column-settings-version', '1')

    const wrapper = await mountView()

    expect(visibleColumnKeys(wrapper)).toEqual([
      'name',
      'key',
      'current_concurrency',
      'usage',
      'expires_at',
      'status',
      'last_used_at',
      'actions',
    ])
    expect(localStorage.getItem('api-key-hidden-columns')).toBe(
      JSON.stringify(['group', 'created_at', 'last_used_ip', 'id'])
    )
    expect(localStorage.getItem('api-key-column-settings-version')).toBe('3')
  })

  it('does not include always-visible columns in the toggleable menu', async () => {
    const wrapper = await mountView()

    await wrapper.get('button[title="Column Settings"]').trigger('click')
    await nextTick()

    const columnMenuText = wrapper.text()
    expect(columnMenuText).toContain('API Key')
    expect(columnMenuText).toContain('ID')
    expect(columnMenuText).toContain('Current Concurrency')
    expect(columnMenuText).not.toContain('Rate Limit')
    expect(columnMenuText).toContain('Last Used IP')
    expect(columnMenuText).not.toContain('Name')
    expect(columnMenuText).not.toContain('Actions')
  })

  it('renders the current concurrency value', async () => {
    const wrapper = await mountView()

    expect(wrapper.get('[data-test="current-concurrency"]').text()).toBe('3')
  })

  it('marks current concurrency as sortable', async () => {
    const wrapper = await mountView()

    const currentConcurrencyColumn = visibleColumnMeta(wrapper).find(
      (column) => column.key === 'current_concurrency'
    )
    expect(currentConcurrencyColumn?.sortable).toBe(true)
  })

  it('keeps filters and selected page size when sorting by current concurrency', async () => {
    getAvailableGroups.mockResolvedValue([{ id: 42, name: 'OpenAI' }])
    const wrapper = await mountView()

    await wrapper.get('[data-test="page-size-50"]').trigger('click')
    await flushPromises()

    await wrapper.findComponent({ name: 'SearchInput' }).vm.$emit('update:modelValue', 'target')
    await wrapper.findComponent({ name: 'SearchInput' }).vm.$emit('search')
    await flushPromises()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    await selects[0].vm.$emit('update:modelValue', 42)
    await flushPromises()
    await selects[1].vm.$emit('update:modelValue', 'active')
    await flushPromises()

    listKeys.mockClear()

    await wrapper.get('[data-test="sort-current-concurrency"]').trigger('click')
    await flushPromises()

    expect(listKeys).toHaveBeenLastCalledWith(
      1,
      50,
      {
        search: 'target',
        status: 'active',
        group_id: 42,
        sort_by: 'current_concurrency',
        sort_order: 'asc',
      },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
  })
})
