import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { OpsAlertRule } from '@/api/admin/ops'
import zhOps from '@/i18n/locales/zh/admin/ops'
import OpsAlertRulesCard from '../OpsAlertRulesCard.vue'

const mocks = vi.hoisted(() => ({
  list: vi.fn(), update: vi.fn(), create: vi.fn(),
  error: vi.fn(), success: vi.fn(), desktop: true,
}))
vi.mock('@/api', () => ({ adminAPI: { groups: { getAll: vi.fn().mockResolvedValue([{ id: 7, name: 'OpenAI' }]) } } }))
vi.mock('@/api/admin/ops', () => ({ opsAPI: {
  listAlertRules: mocks.list, updateAlertRule: mocks.update, createAlertRule: mocks.create,
} }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.error, showSuccess: mocks.success }) }))
vi.mock('vue-i18n', async (original) => ({
  ...await original<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string, params: Record<string, unknown> = {}) => {
    const messages = { admin: zhOps, common: { edit: '编辑', delete: '删除', save: '保存', cancel: '取消', refresh: '刷新', enabled: '启用' } }
    const value = key.split('.').reduce<unknown>((current, part) => {
      return current && typeof current === 'object' ? (current as Record<string, unknown>)[part] : undefined
    }, messages)
    return typeof value === 'string' ? value.replace(/\{(\w+)\}/g, (_, name: string) => String(params[name] ?? `{${name}}`)) : key
  } }),
}))
vi.mock('@vueuse/core', async (original) => ({
  ...await original<typeof import('@vueuse/core')>(),
  useMediaQuery: () => mocks.desktop,
}))

let wrapper: VueWrapper | undefined
const cases = [
  { metric: 'p95_latency_ms', label: 'P95', threshold: 2000 },
  { metric: 'p99_latency_ms', label: 'P99', threshold: 3000 },
] as const

async function render() {
  wrapper = mount(OpsAlertRulesCard, {
    global: {
      stubs: {
        BaseDialog: { props: ['show'], template: '<div v-if="show" role="dialog"><slot /><slot name="footer" /></div>' },
        ConfirmDialog: true,
        teleport: true,
      },
    },
  })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.desktop = true
  mocks.list.mockResolvedValue([])
  mocks.update.mockResolvedValue({})
  mocks.create.mockResolvedValue({})
})
afterEach(() => wrapper?.unmount())

describe('P95/P99 latency alert editor', () => {
  for (const desktop of [true, false]) {
    for (const { metric, label, threshold } of cases) {
      it(`preserves the saved ${label} rule on ${desktop ? 'desktop' : 'mobile'}`, async () => {
        mocks.desktop = desktop
        const rule: OpsAlertRule = {
          id: 4, name: `${label}延迟过高`, description: 'Existing rule',
          enabled: true, metric_type: metric, operator: '>', threshold,
          window_minutes: 5, sustained_minutes: 10, cooldown_minutes: 30,
          severity: 'P2', notify_email: true, filters: { group_id: 7, platform: 'openai' },
        }
        mocks.list.mockResolvedValue([rule])
        const w = await render()
        await w.findAll('button').find(b => b.text() === '编辑')!.trigger('click')
        const dialog = w.get('[role="dialog"]')
        const metricButton = dialog.get('button[aria-label="指标"]')
        expect(metricButton.text()).toContain(label)
        expect(dialog.text()).toContain('包含流式生成时间')
        expect(dialog.text()).toContain('不同于首 Token 延迟')
        expect(dialog.get('input[type="number"]').element).toHaveProperty('value', String(threshold))
        await metricButton.trigger('click')
        expect(w.findAll('[role="option"]').some(o => o.text().includes('P95'))).toBe(true)
        expect(w.findAll('[role="option"]').some(o => o.text().includes('P99'))).toBe(true)
        await metricButton.trigger('click')
        await dialog.findAll('button').find(b => b.text() === '保存')!.trigger('click')
        await flushPromises()
        expect(mocks.update).toHaveBeenCalledWith(4, rule)
        expect(mocks.create).not.toHaveBeenCalled()
        expect(mocks.error).not.toHaveBeenCalled()
      })
    }
  }

  for (const { metric, label, threshold } of cases) {
    it(`allows creating a ${label} latency rule in milliseconds`, async () => {
      const w = await render()
      await w.findAll('button').find(b => b.text() === zhOps.ops.alertRules.create)!.trigger('click')
      const dialog = w.get('[role="dialog"]')
      await dialog.get('input[type="text"]').setValue(`${label} latency`)
      await dialog.get('button[aria-label="指标"]').trigger('click')
      await w.findAll('[role="option"]').find(o => o.text().includes(label))!.trigger('click')
      await dialog.get('input[type="number"]').setValue(threshold)
      await dialog.findAll('button').find(b => b.text() === '保存')!.trigger('click')
      await flushPromises()
      expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ metric_type: metric, threshold, notify_email: true }))
      expect(mocks.update).not.toHaveBeenCalled()
      expect(mocks.error).not.toHaveBeenCalled()
    })
  }
})
