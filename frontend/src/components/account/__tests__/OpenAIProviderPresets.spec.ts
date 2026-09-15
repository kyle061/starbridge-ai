import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import OpenAIProviderPresets from '../OpenAIProviderPresets.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('OpenAI provider presets', () => {
  it('selects a chat-only provider without submitting the form', async () => {
    const wrapper = mount(OpenAIProviderPresets, { props: { currentUrl: '', responsesMode: 'auto' } })
    const button = wrapper.get('[data-testid="provider-preset-siliconflow"]')
    expect(button.attributes('type')).toBe('button')
    await button.trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual([{
      id: 'siliconflow', label: 'SiliconFlow', baseUrl: 'https://api.siliconflow.cn/v1', responsesMode: 'force_chat_completions'
    }])
  })

  it('highlights only when both address and protocol match and accepts custom endpoints', async () => {
    const wrapper = mount(OpenAIProviderPresets, { props: {
      currentUrl: ' https://api.openai.com/ ', responsesMode: 'force_responses'
    } })
    expect(wrapper.get('[data-testid="provider-preset-openai"]').attributes('aria-pressed')).toBe('true')
    await wrapper.setProps({ responsesMode: 'auto' })
    expect(wrapper.findAll('[aria-pressed="true"]')).toHaveLength(0)
    await wrapper.setProps({ currentUrl: 'https://models.example.com/v1' })
    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('shows local deployment guidance for Docker Ollama and loopback hosts', async () => {
    const wrapper = mount(OpenAIProviderPresets, { props: { currentUrl: '', responsesMode: 'auto' } })
    for (const currentUrl of ['http://host.docker.internal:11434/v1', 'http://127.0.0.1:11434/v1']) {
      await wrapper.setProps({ currentUrl })
      expect(wrapper.get('[role="status"]').text()).toContain('localHint')
    }
    await wrapper.setProps({ currentUrl: 'https://api.openai.com' })
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })
})
