import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BulkEditKeysModal from '../BulkEditKeysModal.vue'
import type { Group } from '@/types'

const { bulkUpdate, showSuccess, showError } = vi.hoisted(() => ({
  bulkUpdate: vi.fn(), showSuccess: vi.fn(), showError: vi.fn()
}))
vi.mock('@/api', () => ({ keysAPI: { bulkUpdate } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, params?: Record<string, unknown>) => `${key} ${JSON.stringify(params ?? {})}` })
}))

const mountModal = () => mount(BulkEditKeysModal, {
  props: {
    show: true,
    selectedKeys: [{ id: 1, name: 'First' }, { id: 2, name: 'Second' }],
    groups: [{ id: 7, name: 'Available group' }] as Group[]
  },
  global: {
    stubs: {
      BaseDialog: {
        name: 'BaseDialog',
        props: ['show', 'closeOnEscape', 'showCloseButton'],
        emits: ['close'],
        template: '<div v-if="show"><slot /><slot name="footer" /></div>'
      },
      Select: {
        props: ['modelValue', 'options', 'disabled'],
        emits: ['update:modelValue'],
        template: `<select :disabled="disabled" :value="modelValue" @change="$emit('update:modelValue', $event.target.value === '7' ? 7 : $event.target.value)">
          <option value=""></option>
          <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>`
      }
    }
  }
})

describe('BulkEditKeysModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    bulkUpdate.mockResolvedValue({ succeededIds: [1, 2], failures: [] })
  })

  it('requires an explicit field choice and sends only that field', async () => {
    const wrapper = mountModal()
    expect(wrapper.get('[data-test="submit"]').attributes('disabled')).toBeDefined()
    await wrapper.get('form').trigger('submit')
    expect(bulkUpdate).not.toHaveBeenCalled()

    await wrapper.get('[data-test="enable-status"]').setValue(true)
    await wrapper.get('[data-test="status-input"]').setValue('inactive')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(bulkUpdate).toHaveBeenCalledWith([1, 2], { status: 'inactive' })
    expect(wrapper.emitted('updated')).toEqual([[[1, 2]]])
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('does not expose administrator-managed fields', () => {
    const wrapper = mountModal()
    for (const field of ['quota', 'rate_limit_5h', 'rate_limit_1d', 'rate_limit_7d', 'expiration', 'ip_whitelist', 'ip_blacklist']) {
      expect(wrapper.find(`[data-test="enable-${field}"]`).exists()).toBe(false)
    }
  })

  it('requires an available group when changing group', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-test="enable-group"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    expect(bulkUpdate).not.toHaveBeenCalled()
    await wrapper.get('[data-test="group-input"]').setValue('7')
    await wrapper.get('form').trigger('submit')
    expect(bulkUpdate).toHaveBeenCalledWith([1, 2], { group_id: 7 })
  })

  it('reports individual failures and retries only failed keys', async () => {
    bulkUpdate.mockResolvedValueOnce({
      succeededIds: [1],
      failures: [{ id: 2, error: { status: 403, message: 'Group access denied' } }]
    }).mockResolvedValueOnce({ succeededIds: [2], failures: [] })
    const wrapper = mountModal()
    await wrapper.get('[data-test="enable-status"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.text()).toContain('#2 Second: Group access denied')
    expect(wrapper.emitted('updated')).toEqual([[[1]]])
    expect(wrapper.emitted('close')).toBeUndefined()
    expect(showSuccess).not.toHaveBeenCalled()
    await wrapper.setProps({ selectedKeys: [{ id: 2, name: 'Second' }, { id: 3, name: 'New selection' }] })
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(bulkUpdate).toHaveBeenLastCalledWith([2], { status: 'active' })
    expect(wrapper.emitted('updated')).toEqual([[[1]], [[2]]])
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('prevents duplicate submissions and closing during an update', async () => {
    let finish!: (result: { succeededIds: number[]; failures: [] }) => void
    bulkUpdate.mockReturnValue(new Promise((resolve) => { finish = resolve }))
    const wrapper = mountModal()
    await wrapper.get('[data-test="enable-status"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    await wrapper.get('form').trigger('submit')
    wrapper.findComponent({ name: 'BaseDialog' }).vm.$emit('close')
    expect(bulkUpdate).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('close')).toBeUndefined()
    expect(wrapper.get('fieldset').attributes('disabled')).toBeDefined()
    finish({ succeededIds: [1, 2], failures: [] })
    await flushPromises()
  })

  it('resets all field choices when reopened for a new selection', async () => {
    const wrapper = mountModal()
    await wrapper.get('[data-test="enable-status"]').setValue(true)
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true, selectedKeys: [{ id: 3, name: 'Third' }] })
    expect(wrapper.get('[data-test="submit"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-test="status-input"]').exists()).toBe(false)
    await wrapper.get('[data-test="enable-status"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    expect(bulkUpdate).toHaveBeenCalledWith([3], { status: 'active' })
  })
})
