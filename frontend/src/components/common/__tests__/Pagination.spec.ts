import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import Pagination from '../Pagination.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: { page: number; total: number }) =>
      key === 'pagination.pageOf' ? `${params?.page} / ${params?.total}` : key
  })
}))

describe('Pagination mobile navigation', () => {
  it('shows one empty page with both navigation buttons disabled', async () => {
    const wrapper = mount(Pagination, {
      props: { total: 0, page: 1, pageSize: 20, showPageSizeSelector: false }
    })
    const [previous, next] = wrapper.findAll('button')

    expect(wrapper.text()).toContain('1 / 1')
    expect(previous.attributes('disabled')).toBeDefined()
    expect(next.attributes('disabled')).toBeDefined()
    await next.trigger('click')
    expect(wrapper.emitted('update:page')).toBeUndefined()
  })

  it('enables next when results arrive and emits the next page', async () => {
    const wrapper = mount(Pagination, {
      props: { total: 0, page: 1, pageSize: 20, showPageSizeSelector: false }
    })
    await wrapper.setProps({ total: 21 })
    const next = wrapper.findAll('button')[1]
    expect(wrapper.text()).toContain('1 / 2')
    expect(next.attributes('disabled')).toBeUndefined()
    await next.trigger('click')
    expect(wrapper.emitted('update:page')).toEqual([[2]])
    await wrapper.setProps({ page: 2 })
    expect(next.attributes('disabled')).toBeDefined()
  })
})
