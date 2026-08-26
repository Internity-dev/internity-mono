import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ListPagination from '@/components/shared/ListPagination.vue'

describe('listPagination', () => {
  it('renders nothing when total is 0', () => {
    const wrapper = mount(ListPagination, { props: { page: 1, limit: 10, total: 0 } })
    expect(wrapper.find('div').exists()).toBe(false)
    expect(wrapper.text()).toBe('')
  })

  it('renders the correct showing-range and page count text', () => {
    const wrapper = mount(ListPagination, { props: { page: 1, limit: 10, total: 25 } })
    expect(wrapper.text()).toContain('1–10')
    expect(wrapper.text()).toContain('25')
    expect(wrapper.text()).toContain('Page 1 of 3')
  })

  it('clamps the range end to the total on the last page', () => {
    const wrapper = mount(ListPagination, { props: { page: 3, limit: 10, total: 25 } })
    expect(wrapper.text()).toContain('21–25')
  })

  it('disables the Previous button on the first page', () => {
    const wrapper = mount(ListPagination, { props: { page: 1, limit: 10, total: 25 } })
    const buttons = wrapper.findAll('button')
    expect(buttons[0]?.attributes('disabled')).toBeDefined()
    expect(buttons[1]?.attributes('disabled')).toBeUndefined()
  })

  it('disables the Next button on the last page', () => {
    const wrapper = mount(ListPagination, { props: { page: 3, limit: 10, total: 25 } })
    const buttons = wrapper.findAll('button')
    expect(buttons[0]?.attributes('disabled')).toBeUndefined()
    expect(buttons[1]?.attributes('disabled')).toBeDefined()
  })

  it('emits update:page with page - 1 when Previous is clicked', async () => {
    const wrapper = mount(ListPagination, { props: { page: 2, limit: 10, total: 25 } })
    await wrapper.findAll('button')[0]?.trigger('click')
    expect(wrapper.emitted('update:page')).toEqual([[1]])
  })

  it('emits update:page with page + 1 when Next is clicked', async () => {
    const wrapper = mount(ListPagination, { props: { page: 2, limit: 10, total: 25 } })
    await wrapper.findAll('button')[1]?.trigger('click')
    expect(wrapper.emitted('update:page')).toEqual([[3]])
  })

  it('does not emit update:page when clicking a disabled Previous button', async () => {
    const wrapper = mount(ListPagination, { props: { page: 1, limit: 10, total: 25 } })
    await wrapper.findAll('button')[0]?.trigger('click')
    expect(wrapper.emitted('update:page')).toBeUndefined()
  })

  it('treats a total that does not divide evenly by rounding the last page up', () => {
    const wrapper = mount(ListPagination, { props: { page: 1, limit: 10, total: 5 } })
    expect(wrapper.text()).toContain('Page 1 of 1')
  })
})
