import { describe, expect, it, vi } from 'vitest'

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
}))

import { buildModelMappingObject, filterSupportedModelIds, getModelsByPlatform, getPresetMappingsByPlatform, splitModelMappingObject } from '../useModelWhitelist'

describe('useModelWhitelist', () => {
  it('openai 模型列表只包含当前支持的 GPT 和 GPT Image 系列', () => {
    const models = getModelsByPlatform('openai')

    expect(models).toEqual([
      'gpt-5.5',
      'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna',
      'gpt-6-astra', 'gpt-6-sol', 'gpt-6-luna',
      'gpt-image-2', 'gpt-image-2.5-flare', 'gpt-image-2.5-sunburst'
    ])
  })

  it('openai 预设只列具体 GPT 模型，不重复列出别名', () => {
    const supported = new Set(getModelsByPlatform('openai'))
    const presets = getPresetMappingsByPlatform('openai')

    expect(supported.has('gpt-5.6')).toBe(false)
    expect(supported.has('gpt-6')).toBe(false)
    expect(presets).toEqual(expect.arrayContaining([
      expect.objectContaining({ label: 'GPT-5.6 Sol', from: 'gpt-5.6-sol', to: 'gpt-5.6-sol' }),
      expect.objectContaining({ label: 'GPT-6 Astra', from: 'gpt-6-astra', to: 'gpt-6-astra' })
    ]))
    expect(presets.map(preset => preset.from)).not.toContain('gpt-5.6')
    expect(presets.map(preset => preset.from)).not.toContain('gpt-6')
    expect(presets.every(preset => supported.has(preset.from) && supported.has(preset.to))).toBe(true)
  })

  it('openai 模型列表不再暴露已下线的 ChatGPT 登录 Codex 模型', () => {
    const models = getModelsByPlatform('openai')

    expect(models).not.toContain('gpt-5')
    expect(models).not.toContain('gpt-5.1')
    expect(models).not.toContain('gpt-5.1-codex')
    expect(models).not.toContain('gpt-5.1-codex-max')
    expect(models).not.toContain('gpt-5.1-codex-mini')
    expect(models).not.toContain('gpt-5.2-codex')
    expect(models).not.toContain('gpt-5.2')
    expect(models).not.toContain('gpt-5.4')
    expect(models).not.toContain('gpt-5.3-codex-spark')
    expect(models).not.toContain('gpt-image-1')
  })

  it('openai 模型列表保留当前支持的图片模型', () => {
    const models = getModelsByPlatform('openai')

    expect(models).toEqual(expect.arrayContaining([
      'gpt-image-2', 'gpt-image-2.5-flare', 'gpt-image-2.5-sunburst'
    ]))
  })

  it('antigravity 模型列表包含图片模型兼容项', () => {
    const models = getModelsByPlatform('antigravity')

    expect(models).toContain('gemini-2.5-flash-image')
    expect(models).toContain('gemini-3.1-flash-image')
    expect(models).not.toContain('gemini-3-pro-image')
  })

  it('antigravity 映射预设保留旧版 Gemini 3 Pro Image 别名', () => {
    expect(getPresetMappingsByPlatform('antigravity')).toContainEqual(
      expect.objectContaining({
        from: 'gemini-3-pro-image',
        to: 'gemini-3.1-flash-image',
      })
    )
  })

  it('Claude 模型列表包含新发布的 Claude 模型', () => {
    expect(getModelsByPlatform('claude')).toContain('claude-fable-5-1')
    expect(getModelsByPlatform('antigravity')).toContain('claude-fable-5-1')
    expect(getModelsByPlatform('claude')).toContain('claude-fable-5')
    expect(getModelsByPlatform('antigravity')).toContain('claude-fable-5')
    expect(getModelsByPlatform('claude')).toContain('claude-opus-4-8')
    expect(getModelsByPlatform('antigravity')).toContain('claude-opus-4-8')
  })

  it('xAI 模型列表只包含官方模型 ID，不展示兼容别名', () => {
    const models = getModelsByPlatform('grok')

    expect(models).toContain('grok-4.6')
    expect(models).toContain('grok-4.5')
    expect(models).toContain('grok-imagine-image-2.0')
    expect(models).toContain('grok-imagine-video-1.5')
    expect(models).not.toContain('grok-4.6-latest')
    expect(models).not.toContain('grok-build-latest')
  })

  it('combined 模式支持 Grok 4.5 官方别名映射', () => {
    const mapping = buildModelMappingObject(
      'combined',
      ['grok-4.5'],
      [
        { from: 'grok-latest', to: 'grok-4.5' },
        { from: 'grok-4.5-latest', to: 'grok-4.5' },
        { from: 'grok-build-latest', to: 'grok-4.5' }
      ]
    )

    expect(mapping).toEqual({
      'grok-4.5': 'grok-4.5',
      'grok-latest': 'grok-4.5',
      'grok-4.5-latest': 'grok-4.5',
      'grok-build-latest': 'grok-4.5'
    })
  })

  it('grok 模型列表包含 Composer 官方模型且不展示兼容别名', () => {
    const models = getModelsByPlatform('grok')

    expect(models).toContain('grok-composer-2.5-fast')
    expect(models).not.toContain('grok-composer')
    expect(models).not.toContain('composer-2.5')
  })

  it('gemini 模型列表包含原生生图模型', () => {
    const models = getModelsByPlatform('gemini')

    expect(models).toContain('gemini-2.5-flash-image')
    expect(models).toContain('gemini-3.1-flash-image')
    expect(models.indexOf('gemini-3.1-flash-image')).toBeLessThan(models.indexOf('gemini-3.5-flash'))
    expect(models.indexOf('gemini-2.5-flash-image')).toBeLessThan(models.indexOf('gemini-2.5-flash'))
  })

  it('antigravity 模型列表会把新的 Gemini 图片模型排在前面', () => {
    const models = getModelsByPlatform('antigravity')

    expect(models.indexOf('gemini-3.1-flash-image')).toBeLessThan(models.indexOf('gemini-2.5-flash'))
    expect(models.indexOf('gemini-2.5-flash-image')).toBeLessThan(models.indexOf('gemini-2.5-flash-lite'))
  })

  it('antigravity 模型列表只包含精确的 Gemini 3.1 Pro 模型 ID', () => {
    const models = getModelsByPlatform('antigravity')

    expect(models).toContain('gemini-3.1-pro-high')
    expect(models).toContain('gemini-3.1-pro-low')
    expect(models).not.toContain('gemini-3.1-pro')
  })

  it('同步上游模型时过滤旧版、预览版和兼容别名', () => {
    expect(filterSupportedModelIds('gemini', [
      'gemini-2.0-flash', 'gemini-2.5-flash', 'gemini-3-pro-preview', 'gemini-unknown'
    ])).toEqual(['gemini-2.5-flash'])
    expect(filterSupportedModelIds('openai', [
      'gpt-image-1.5', 'gpt-image-2', 'gpt-5.4'
    ])).toEqual(['gpt-image-2'])
  })

  it('保留 Bedrock 指向官方 Claude 模型的 provider-qualified 映射', () => {
    const presets = getPresetMappingsByPlatform('bedrock')

    expect(presets).toEqual(expect.arrayContaining([
      expect.objectContaining({
        from: 'claude-opus-4-6',
        to: 'us.anthropic.claude-opus-4-6-v1'
      }),
      expect.objectContaining({
        from: 'claude-sonnet-4-5-20250929',
        to: 'us.anthropic.claude-sonnet-4-5-20250929-v1:0'
      })
    ]))
  })

  it('whitelist 模式会保留合法通配符身份映射', () => {
    const mapping = buildModelMappingObject('whitelist', ['claude-*', 'gpt-*', 'gemini-3.1-flash-image', 'gpt-*-broken'], [])
    expect(mapping).toEqual({
      'claude-*': 'claude-*',
      'gpt-*': 'gpt-*',
      'gemini-3.1-flash-image': 'gemini-3.1-flash-image'
    })
  })

  it('whitelist 模式会保留 GPT-5.6 的精确映射', () => {
    const mapping = buildModelMappingObject('whitelist', ['gpt-5.6'], [])

    expect(mapping).toEqual({
      'gpt-5.6': 'gpt-5.6'
    })
  })

  it('whitelist keeps GPT-5.6 Sol exact mappings', () => {
    const mapping = buildModelMappingObject('whitelist', ['gpt-5.6-sol'], [])

    expect(mapping).toEqual({
      'gpt-5.6-sol': 'gpt-5.6-sol'
    })
  })

  it('combined 模式会同时保留白名单身份映射和模型映射', () => {
    const mapping = buildModelMappingObject(
      'combined',
      ['gpt-5.6', 'claude-*'],
      [
        { from: 'gpt-latest', to: 'gpt-5.6' },
        { from: 'gpt-5.6', to: 'gpt-5.6-sol' }
      ]
    )

    expect(mapping).toEqual({
      'gpt-5.6': 'gpt-5.6-sol',
      'claude-*': 'claude-*',
      'gpt-latest': 'gpt-5.6'
    })
  })

  it('splitModelMappingObject 会把身份映射还原成白名单，其余保留为映射', () => {
    const parsed = splitModelMappingObject({
      'gpt-5.6': 'gpt-5.6',
      'gpt-latest': 'gpt-5.6',
      ' ': 'gpt-empty',
      broken: 123
    })

    expect(parsed).toEqual({
      allowedModels: ['gpt-5.6'],
      modelMappings: [{ from: 'gpt-latest', to: 'gpt-5.6' }]
    })
  })
})
