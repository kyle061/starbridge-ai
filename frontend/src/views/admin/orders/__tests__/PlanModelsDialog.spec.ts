import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PlanModelsDialog from '../PlanModelsDialog.vue'
import type { SubscriptionPlan } from '@/types/payment'

const { getById, getModelAllowlistCandidates, update, showError } = vi.hoisted(() => ({
  getById: vi.fn(), getModelAllowlistCandidates: vi.fn(), update: vi.fn(), showError: vi.fn(),
}))
vi.mock('@/api/admin/groups', () => ({ getById, getModelAllowlistCandidates, update }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError, showSuccess: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: { plans: string }) => params?.plans ? `${key} ${params.plans}` : key }) }))
const plan = { id: 1, group_id: 10, name: 'Starter' } as SubscriptionPlan
const createWrapper = (plans = [plan]) => mount(PlanModelsDialog, {
  props: { show: true, plan, plans },
  global: { stubs: { BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' } } },
})
const save = (wrapper: ReturnType<typeof createWrapper>) => wrapper.findAll('button').find(b => b.text() === 'common.save')!

beforeEach(() => {
  vi.resetAllMocks()
  getById.mockResolvedValue({
    platform: 'composite',
    model_allowlist: { enabled: true, models: ['deepseek-v4-pro'] }
  })
  getModelAllowlistCandidates.mockResolvedValue(['deepseek-v4-pro', 'gpt-5.6-sol'])
  update.mockResolvedValue({})
})

describe('PlanModelsDialog', () => {
  it('saves selected models to the plan group without modifying billing settings', async () => {
    const wrapper = createWrapper()
    await flushPromises()
    await wrapper.findAll('input[type="checkbox"]')[2].setValue(true)
    await save(wrapper).trigger('click')
    await flushPromises()
    expect(update).toHaveBeenCalledWith(10, { model_allowlist: { enabled: true, models: ['deepseek-v4-pro', 'gpt-5.6-sol'] } })
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })
  it('blocks empty enabled policies and does not save', async () => {
    const wrapper = createWrapper()
    await flushPromises()
    await wrapper.findAll('input[type="checkbox"]')[1].setValue(false)
    await save(wrapper).trigger('click')
    expect(update).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.groups.modelAllowlist.emptySelectionError')
  })
  it('shows other plans that share the same policy', async () => {
    const wrapper = createWrapper([plan, { ...plan, id: 2, name: 'Pro' }])
    await flushPromises()
    expect(wrapper.text()).toContain('Starter、Pro')
  })
  it('blocks saving after a load failure', async () => {
    getById.mockRejectedValue(new Error('offline'))
    const wrapper = createWrapper()
    await flushPromises()
    expect(save(wrapper).attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('payment.admin.modelsLoadError')
    expect(update).not.toHaveBeenCalled()
  })
  it('keeps the dialog open and reports a failed save', async () => {
    update.mockRejectedValue(new Error('offline'))
    const wrapper = createWrapper()
    await flushPromises()
    await save(wrapper).trigger('click')
    await flushPromises()
    expect(wrapper.emitted('saved')).toBeUndefined()
    expect(wrapper.emitted('close')).toBeUndefined()
    expect(showError).toHaveBeenCalled()
  })
  it('ignores a stale response when switching plans', async () => {
    let resolveOld!: (value: unknown) => void
    getById.mockReturnValueOnce(new Promise(resolve => { resolveOld = resolve }))
    const wrapper = createWrapper()
    await wrapper.setProps({ plan: { ...plan, group_id: 11 } })
    await flushPromises()
    resolveOld({ model_allowlist: { enabled: true, models: ['old-model'] } })
    await flushPromises()
    expect(wrapper.text()).not.toContain('old-model')
    await save(wrapper).trigger('click')
    expect(update).toHaveBeenCalledWith(11, { model_allowlist: { enabled: true, models: ['deepseek-v4-pro'] } })
  })
})
