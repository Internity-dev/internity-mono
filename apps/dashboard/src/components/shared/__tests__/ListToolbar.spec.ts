import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ListToolbar from '@/components/shared/ListToolbar.vue'

describe('listToolbar', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the modelValue in the search input', () => {
    const wrapper = mount(ListToolbar, { props: { modelValue: 'hello' } })
    expect(wrapper.find('input').element.value).toBe('hello')
  })

  it('uses the default placeholder when none is given', () => {
    const wrapper = mount(ListToolbar, { props: { modelValue: '' } })
    expect(wrapper.find('input').attributes('placeholder')).toBe('Search…')
  })

  it('uses a custom placeholder when given', () => {
    const wrapper = mount(ListToolbar, {
      props: { modelValue: '', placeholder: 'Find a student…' },
    })
    expect(wrapper.find('input').attributes('placeholder')).toBe('Find a student…')
  })

  it('emits update:modelValue after the debounce delay when typing', async () => {
    const wrapper = mount(ListToolbar, { props: { modelValue: '' } })
    const input = wrapper.find('input')

    await input.setValue('john')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    vi.advanceTimersByTime(300)
    expect(wrapper.emitted('update:modelValue')).toEqual([['john']])
  })

  it('does not emit before the debounce delay has elapsed', async () => {
    const wrapper = mount(ListToolbar, { props: { modelValue: '' } })
    await wrapper.find('input').setValue('j')

    vi.advanceTimersByTime(299)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('only emits once for rapid successive keystrokes (debounced, not throttled)', async () => {
    const wrapper = mount(ListToolbar, { props: { modelValue: '' } })
    const input = wrapper.find('input')

    await input.setValue('j')
    vi.advanceTimersByTime(100)
    await input.setValue('jo')
    vi.advanceTimersByTime(100)
    await input.setValue('joh')
    vi.advanceTimersByTime(300)

    expect(wrapper.emitted('update:modelValue')).toEqual([['joh']])
  })

  it('updates the input when modelValue prop changes externally', async () => {
    const wrapper = mount(ListToolbar, { props: { modelValue: 'first' } })
    await wrapper.setProps({ modelValue: 'second' })
    expect(wrapper.find('input').element.value).toBe('second')
  })

  it('renders slot content next to the search input', () => {
    const wrapper = mount(ListToolbar, {
      props: { modelValue: '' },
      slots: { default: '<button class="filter-btn">Filters</button>' },
    })
    expect(wrapper.find('button.filter-btn').exists()).toBe(true)
  })
})
