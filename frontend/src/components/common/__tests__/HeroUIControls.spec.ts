import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Button from '../Button.vue'
import Input from '../Input.vue'
import TextArea from '../TextArea.vue'
import Toggle from '../Toggle.vue'

describe('HeroUI Vue controls', () => {
  it('blocks submission while a button is loading', async () => {
    const wrapper = mount(Button, { props: { type: 'submit', loading: true }, slots: { default: 'Save' } })
    expect(wrapper.attributes('type')).toBe('submit')
    expect(wrapper.attributes('aria-busy')).toBe('true')
    expect(wrapper.attributes('disabled')).toBeDefined()
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toBeUndefined()
    await wrapper.setProps({ loading: false })
    expect(wrapper.attributes('disabled')).toBeUndefined()
  })

  it.each([Input, TextArea])('keeps field values and validation feedback connected', async (component) => {
    const wrapper = mount(component, {
      props: { id: 'field', modelValue: '', label: 'Name', error: 'Required' },
      global: { stubs: { Icon: true } }
    })
    const field = wrapper.get('input, textarea')
    await field.setValue('Starbridge')
    expect(wrapper.emitted('update:modelValue')).toEqual([['Starbridge']])
    expect(field.attributes('aria-invalid')).toBe('true')
    expect(wrapper.get(`#${field.attributes('aria-describedby')}`).text()).toContain('Required')
    expect(wrapper.get('label').attributes('for')).toBe('field')
  })

  it('preserves switch state and ignores a disabled switch', async () => {
    const wrapper = mount(Toggle, { props: { modelValue: false }, attrs: { 'aria-label': 'Auto refresh' } })
    await wrapper.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[true]])
    await wrapper.setProps({ modelValue: true, disabled: true })
    expect(wrapper.attributes('aria-checked')).toBe('true')
    await wrapper.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toHaveLength(1)
  })
})
