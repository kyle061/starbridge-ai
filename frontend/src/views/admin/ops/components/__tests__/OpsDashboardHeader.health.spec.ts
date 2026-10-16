import { describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import type { OpsDashboardOverview } from '@/api/admin/ops'
import OpsDashboardHeader from '../OpsDashboardHeader.vue'

vi.mock('@/api', () => ({ adminAPI: { groups: { getAll: vi.fn().mockResolvedValue([]) } } }))
vi.mock('@/stores', () => ({ useAdminSettingsStore: () => ({ opsRealtimeMonitoringEnabled: false }) }))
vi.mock('vue-i18n', async (original) => ({
  ...await original<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))

async function render(overrides: Partial<OpsDashboardOverview> = {}, limit: number | null = 10000) {
  const wrapper = shallowMount(OpsDashboardHeader, {
    props: {
      overview: {
        request_count_total: 3, request_count_sla: 3, success_count: 3,
        health_score: 100, sla: 1, error_rate: 0, upstream_error_rate: 0,
        ttft: { p99_ms: 2325 }, ...overrides,
      } as OpsDashboardOverview,
      platform: '', groupId: null, timeRange: '1h', queryMode: 'raw',
      loading: false, lastUpdated: null,
      thresholds: { ttft_p99_ms_max: limit, sla_percent_min: 99.5, request_error_rate_percent_max: 5, upstream_error_rate_percent_max: 5 },
    },
  })
  await flushPromises()
  return wrapper
}

// The diagnosis tooltip is CSS-hidden until hover, so text assertions also
// cover the rendered diagnostic copy without depending on pointer timing.
describe('Ops dashboard health signals', () => {
  it('does not mark the reported 2.325s / 3 successes as degraded', async () => {
    const w = await render()
    expect(w.text()).not.toContain('admin.ops.diagnosis.ttftHigh')
    expect(w.text()).not.toContain('admin.ops.diagnosis.healthLow')
    expect(w.text()).toContain('admin.ops.diagnosis.healthy')
    expect(w.findAll('p').filter(p => p.text() === 'admin.ops.ttftLowSampleHint')).toHaveLength(1)
    const value = w.findAll('div.text-3xl').find(el => el.text() === '2325')!
    expect(value.classes()).not.toContain('text-red-600')
    w.unmount()
  })

  it('honors a stricter admin threshold without duplicating the same warning', async () => {
    const w = await render({ health_score: 65 }, 2000)
    expect(w.text()).toContain('admin.ops.diagnosis.ttftHigh')
    expect(w.text()).not.toContain('admin.ops.diagnosis.healthLow')
    const value = w.findAll('div.text-3xl').find(el => el.text() === '2325')!
    expect(value.classes()).toContain('text-red-600')
    w.unmount()
  })

  it('still warns about slow requests and failures with a small sample', async () => {
    const w = await render({ health_score: 30, error_rate: 1 / 3, ttft: { p99_ms: 60000 } as OpsDashboardOverview['ttft'] })
    expect(w.text()).toContain('admin.ops.diagnosis.ttftHigh')
    expect(w.text()).toContain('admin.ops.diagnosis.errorHigh')
    w.unmount()
  })

  it('does not hide infrastructure outages when traffic is idle', async () => {
    const w = await render({ request_count_total: 0, request_count_sla: 0, health_score: 88, system_metrics: { db_ok: false } as OpsDashboardOverview['system_metrics'] })
    expect(w.text()).toContain('admin.ops.diagnosis.dbDown')
    expect(w.text()).not.toContain('admin.ops.diagnosis.idle')
    w.unmount()
  })

  it('uses server schedule-aware job status', async () => {
    const w = await render({ health_score: 98, job_heartbeats: [{ job_name: 'ops_cleanup', stale: true }] as OpsDashboardOverview['job_heartbeats'] })
    expect(w.text()).toContain('admin.ops.diagnosis.jobsUnhealthy')
    await w.setProps({ overview: { ...w.props('overview'), health_score: 100, job_heartbeats: [{ job_name: 'ops_cleanup', stale: false }] } as OpsDashboardOverview })
    expect(w.text()).not.toContain('admin.ops.diagnosis.jobsUnhealthy')
    w.unmount()
  })
})
