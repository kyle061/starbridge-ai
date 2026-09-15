import { describe, expect, it } from 'vitest'
import { formatPublicVersion } from '../publicVersion'

describe('formatPublicVersion', () => {
  it.each([
    ['0.1.179', '0.01'],
    ['v1.1.42', '1.01'],
    ['starbridge-0.12.3', '0.12'],
    ['0.147.0', '0.14']
  ])('formats %s as the compact public version %s', (version, expected) => {
    expect(formatPublicVersion(version)).toBe(expected)
  })

  it('uses the initial public version for commit-based builds', () => {
    expect(formatPublicVersion('starbridge-4d7036a2e4b654c5f')).toBe('0.01')
  })
})

