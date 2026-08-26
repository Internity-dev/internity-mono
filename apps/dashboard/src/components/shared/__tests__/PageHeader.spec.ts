import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import PageHeader from '@/components/shared/PageHeader.vue'

describe('pageHeader', () => {
  it('renders the title in a heading', () => {
    const wrapper = mount(PageHeader, { props: { title: 'Students' } })
    expect(wrapper.find('h1').text()).toBe('Students')
  })

  it('does not render a description paragraph when none is given', () => {
    const wrapper = mount(PageHeader, { props: { title: 'Students' } })
    expect(wrapper.find('p').exists()).toBe(false)
  })

  it('renders the description when given', () => {
    const wrapper = mount(PageHeader, {
      props: { title: 'Students', description: 'Manage enrolled students.' },
    })
    expect(wrapper.find('p').text()).toBe('Manage enrolled students.')
  })

  it('renders the "actions" slot content', () => {
    const wrapper = mount(PageHeader, {
      props: { title: 'Students' },
      slots: { actions: '<button class="add-btn">Add student</button>' },
    })
    const actionButton = wrapper.find('button.add-btn')
    expect(actionButton.exists()).toBe(true)
    expect(actionButton.text()).toBe('Add student')
  })

  it('renders no actions content when the slot is not used', () => {
    const wrapper = mount(PageHeader, { props: { title: 'Students' } })
    expect(wrapper.find('button').exists()).toBe(false)
  })
})
