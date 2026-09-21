import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import AccountGroupsCell from '../AccountGroupsCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string, params?: { count?: number }) => `${key}:${params?.count ?? ''}` }) }
})

const groups = Array.from({ length: 5 }, (_, index) => ({
  id: index + 1,
  name: `A very long group name ${index + 1}`,
  platform: 'openai',
  subscription_type: 'standard',
  rate_multiplier: 1
})) as any

const mountCell = (value = groups) => mount(AccountGroupsCell, {
  props: { groups: value },
  global: {
    stubs: {
      GroupBadge: {
        props: ['name'],
        template: '<span data-group-badge :title="$attrs.title">{{ name }}</span>'
      }
    }
  }
})

describe('AccountGroupsCell', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('keeps the preview readable and exposes the full group name as a title', () => {
    const wrapper = mountCell()
    const preview = wrapper.get('[data-testid="account-groups-preview"]')

    expect(preview.classes()).not.toContain('overflow-hidden')
    expect(preview.classes()).not.toContain('max-h-14')
    expect(preview.findAll('[data-group-badge]')).toHaveLength(3)
    expect(preview.get('[data-group-badge]').attributes('title')).toBe(groups[0].name)
    expect(wrapper.get('button').text()).toContain('+2')
  })

  it('shows every group in the popover when the preview is expanded', async () => {
    const wrapper = mountCell()

    await wrapper.get('button').trigger('click')

    const popover = document.body.querySelector('[data-testid="account-groups-popover"]')
    expect(popover).not.toBeNull()
    expect(popover?.querySelectorAll('[data-group-badge]')).toHaveLength(groups.length)
    expect(wrapper.get('button').attributes('aria-expanded')).toBe('true')
  })

  it('shows a placeholder when an account has no groups', () => {
    expect(mountCell([]).text()).toBe('-')
  })
})
