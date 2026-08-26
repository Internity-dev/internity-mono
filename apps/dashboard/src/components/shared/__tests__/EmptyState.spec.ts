import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { h } from 'vue'
import EmptyState from '@/components/shared/EmptyState.vue'

describe('emptyState', () => {
  it('renders the title', () => {
    const wrapper = mount(EmptyState, { props: { title: 'Nothing here' } })
    expect(wrapper.text()).toContain('Nothing here')
  })

  it('does not render a description when none is given', () => {
    const wrapper = mount(EmptyState, { props: { title: 'Nothing here' } })
    expect(wrapper.findAll('p')).toHaveLength(1)
  })

  it('renders the description when given', () => {
    const wrapper = mount(EmptyState, {
      props: { title: 'Nothing here', description: 'Try adjusting your filters.' },
    })
    expect(wrapper.text()).toContain('Try adjusting your filters.')
  })

  it('does not render an action button when no actionLabel is given', () => {
    const wrapper = mount(EmptyState, { props: { title: 'Nothing here' } })
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('renders the action button and emits "action" when clicked', async () => {
    const wrapper = mount(EmptyState, {
      props: { title: 'Nothing here', actionLabel: 'Create one' },
    })
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toBe('Create one')

    await button.trigger('click')

    expect(wrapper.emitted('action')).toHaveLength(1)
  })

  it('renders the default inbox icon when no icon prop is given', () => {
    const wrapper = mount(EmptyState, { props: { title: 'Nothing here' } })
    expect(wrapper.find('svg').exists()).toBe(true)
  })

  it('renders a custom icon component when one is provided', () => {
    const CustomIcon = { name: 'CustomIcon', render: () => h('svg', { class: 'my-custom-icon' }) }
    const wrapper = mount(EmptyState, {
      props: { title: 'Nothing here', icon: CustomIcon },
    })
    expect(wrapper.find('svg.my-custom-icon').exists()).toBe(true)
  })
})
