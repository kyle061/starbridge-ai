import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import SecretRevealButton from '../SecretRevealButton.vue'

const { revealSecret } = vi.hoisted(() => ({ revealSecret: vi.fn() }))
vi.mock('@/api/admin/settings', () => ({ revealSecret }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

let wrapper: VueWrapper
function setup() {
  wrapper = mount(SecretRevealButton, {
    props: { target: { key: 'resend_api_key' } },
    global: {
      stubs: { BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /><button data-close @click="$emit(\'close\')">close</button></div>' } },
    },
  })
  return wrapper
}
async function submit(password = 'admin-password') {
  await wrapper.get('input[type=password]').setValue(password)
  await wrapper.get('form').trigger('submit')
  await flushPromises()
}

beforeEach(() => vi.resetAllMocks())
afterEach(() => { wrapper?.unmount(); vi.useRealTimers() })

describe('saved credential password verification', () => {
  it('fetches only after password submission, clears password, and requires verification again after closing', async () => {
    setup()
    expect(revealSecret).not.toHaveBeenCalled()
    await wrapper.get('button').trigger('click')
    expect(revealSecret).not.toHaveBeenCalled()
    revealSecret.mockResolvedValue('re_saved_test_secret')
    await submit()
    expect(revealSecret).toHaveBeenCalledWith({ key: 'resend_api_key' }, 'admin-password')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('re_saved_test_secret')
    expect(wrapper.find('input').exists()).toBe(false)
    await wrapper.get('[data-close]').trigger('click')
    expect(wrapper.find('textarea').exists()).toBe(false)
    await wrapper.get('button').trigger('click')
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('')
    expect(wrapper.find('textarea').exists()).toBe(false)
  })

  it('does not display a secret after wrong password and clears the rejected password', async () => {
    setup()
    await wrapper.get('button').trigger('click')
    revealSecret.mockRejectedValue({ reason: 'PASSWORD_INCORRECT', message: 'untrusted upstream value' })
    await submit('wrong')
    expect(wrapper.get('[role=alert]').text()).toBe('admin.settings.secretReveal.incorrect')
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('')
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('untrusted upstream value')
  })

  it('hides the value after 30 seconds', async () => {
    vi.useFakeTimers()
    setup()
    await wrapper.get('button').trigger('click')
    revealSecret.mockResolvedValue('re_temporary')
    await submit()
    await vi.advanceTimersByTimeAsync(30_000)
    expect(wrapper.find('textarea').exists()).toBe(false)
    await wrapper.get('button').trigger('click')
    expect(wrapper.find('textarea').exists()).toBe(false)
  })

  it('ignores a successful response arriving after the dialog was closed and reopened', async () => {
    let resolve!: (value: string) => void
    revealSecret.mockImplementation(() => new Promise<string>(r => { resolve = r }))
    setup()
    await wrapper.get('button').trigger('click')
    await wrapper.get('input').setValue('admin-password')
    await wrapper.get('form').trigger('submit')
    await wrapper.get('[data-close]').trigger('click')
    await wrapper.get('button').trigger('click')
    resolve('must-not-appear')
    await flushPromises()
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect((wrapper.get('input').element as HTMLInputElement).value).toBe('')
  })

  it('clears visible credentials when the target changes', async () => {
    setup()
    await wrapper.get('button').trigger('click')
    revealSecret.mockResolvedValue('re_old')
    await submit()
    await wrapper.setProps({ target: { key: 'brevo_api_key' } })
    expect(wrapper.find('textarea').exists()).toBe(false)
  })

  it('clears visible credentials when the browser tab is hidden', async () => {
    setup()
    await wrapper.get('button').trigger('click')
    revealSecret.mockResolvedValue('re_old')
    await submit()
    const hidden = vi.spyOn(document, 'hidden', 'get').mockReturnValue(true)
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()
    expect(wrapper.find('textarea').exists()).toBe(false)
    hidden.mockRestore()
  })
})
