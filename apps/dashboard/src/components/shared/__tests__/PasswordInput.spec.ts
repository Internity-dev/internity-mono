import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PasswordInput from '@/components/shared/PasswordInput.vue'

describe('passwordInput', () => {
  it('starts masked as a password field', () => {
    const wrapper = mount(PasswordInput, { props: { modelValue: '' } })
    expect(wrapper.find('input').attributes('type')).toBe('password')
  })

  it('toggles to a plain text field on click, and back on a second click', async () => {
    const wrapper = mount(PasswordInput, { props: { modelValue: 'secret' } })
    const toggle = wrapper.find('button')

    await toggle.trigger('click')
    expect(wrapper.find('input').attributes('type')).toBe('text')

    await toggle.trigger('click')
    expect(wrapper.find('input').attributes('type')).toBe('password')
  })

  it('forwards attrs like id and autocomplete to the underlying input', () => {
    const wrapper = mount(PasswordInput, {
      props: { modelValue: '' },
      attrs: { id: 'password', autocomplete: 'current-password' },
    })
    const input = wrapper.find('input')
    expect(input.attributes('id')).toBe('password')
    expect(input.attributes('autocomplete')).toBe('current-password')
  })

  it('emits update:modelValue as the user types', async () => {
    const wrapper = mount(PasswordInput, { props: { modelValue: '' } })
    await wrapper.find('input').setValue('hunter2')
    expect(wrapper.emitted('update:modelValue')).toEqual([['hunter2']])
  })
})
