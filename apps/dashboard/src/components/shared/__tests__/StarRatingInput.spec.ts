import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StarRatingInput from '@/components/shared/StarRatingInput.vue'

describe('starRatingInput', () => {
  it('renders five star buttons', () => {
    const wrapper = mount(StarRatingInput, { props: { modelValue: 0 } })
    expect(wrapper.findAll('button')).toHaveLength(5)
  })

  it('marks stars up to and including the current value as filled', () => {
    const wrapper = mount(StarRatingInput, { props: { modelValue: 3 } })
    const buttons = wrapper.findAll('button')
    buttons.slice(0, 3).forEach((b) => expect(b.find('svg').classes()).toContain('fill-warning'))
    buttons.slice(3).forEach((b) => expect(b.find('svg').classes()).not.toContain('fill-warning'))
  })

  it('emits update:modelValue with the clicked star\'s value', async () => {
    const wrapper = mount(StarRatingInput, { props: { modelValue: 0 } })
    await wrapper.findAll('button')[3]!.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[4]])
  })

  it('marks only the star matching modelValue as checked', () => {
    const wrapper = mount(StarRatingInput, { props: { modelValue: 2 } })
    const buttons = wrapper.findAll('button')
    expect(buttons[1]!.attributes('aria-checked')).toBe('true')
    expect(buttons[0]!.attributes('aria-checked')).toBe('false')
    expect(buttons[2]!.attributes('aria-checked')).toBe('false')
  })
})
