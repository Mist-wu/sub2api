import { describe, expect, it, vi } from 'vitest'

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
}))

import { buildModelMappingObject, getModelsByPlatform } from '../useModelWhitelist'

describe('useModelWhitelist', () => {
  it('openai 模型列表仅暴露生产白名单', () => {
    const models = getModelsByPlatform('openai')

    expect(models).toEqual([
      'gpt-5.3-codex',
      'gpt-5.3-codex-spark',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
      'gpt-image-2'
    ])
  })

  it('openai 模型列表不再暴露已下线的 ChatGPT 登录 Codex 模型', () => {
    const models = getModelsByPlatform('openai')

    expect(models).not.toContain('gpt-5')
    expect(models).not.toContain('gpt-5.1')
    expect(models).not.toContain('gpt-5.1-codex')
    expect(models).not.toContain('gpt-5.1-codex-max')
    expect(models).not.toContain('gpt-5.1-codex-mini')
    expect(models).not.toContain('gpt-5.2-codex')
  })

  it('gemini 模型列表仅暴露生产白名单', () => {
    const models = getModelsByPlatform('gemini')

    expect(models).toEqual([
      'gemini-3-flash-preview',
      'gemini-3-pro-preview',
      'gemini-3.1-pro-preview'
    ])
  })

  it('antigravity 模型列表仅暴露 Gemini preview 请求名', () => {
    const models = getModelsByPlatform('antigravity')

    expect(models).toEqual([
      'gemini-3-flash-preview',
      'gemini-3-pro-preview',
      'gemini-3.1-pro-preview'
    ])
  })

  it('whitelist 模式会忽略通配符条目', () => {
    const mapping = buildModelMappingObject('whitelist', ['claude-*', 'gemini-3.1-pro-preview'], [])
    expect(mapping).toEqual({
      'gemini-3.1-pro-preview': 'gemini-3.1-pro-preview'
    })
  })

  it('whitelist 模式不会额外允许未列入生产白名单的 GPT-5.4 快照', () => {
    const models = getModelsByPlatform('openai')

    expect(models).not.toContain('gpt-5.4-2026-03-05')
  })

  it('whitelist keeps GPT-5.4 mini exact mappings', () => {
    const mapping = buildModelMappingObject('whitelist', ['gpt-5.4-mini'], [])

    expect(mapping).toEqual({
      'gpt-5.4-mini': 'gpt-5.4-mini'
    })
  })
})
