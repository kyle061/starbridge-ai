import { describe, expect, it, vi } from 'vitest'
import { useChannelMonitorFormat } from '../useChannelMonitorFormat'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('channel monitor status labels', () => {
  it('distinguishes slow probes from quota warnings', () => {
    const { statusLabel, statusHint } = useChannelMonitorFormat()

    expect(statusLabel('degraded', 'probe')).toBe('monitorCommon.status.slowResponse')
    expect(statusLabel('degraded', 'quota_probe')).toBe('monitorCommon.status.slowResponse')
    expect(statusHint('degraded', 'probe')).toBe('monitorCommon.statusHint.slowResponse')
    expect(statusLabel('degraded', 'quota')).toBe('monitorCommon.status.degraded')
    expect(statusHint('degraded', 'quota')).toBeUndefined()
    expect(statusLabel('operational', 'probe')).toBe('monitorCommon.status.operational')
  })
})
