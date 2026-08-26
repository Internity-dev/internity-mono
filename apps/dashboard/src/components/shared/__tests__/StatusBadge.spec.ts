import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusBadge from '@/components/shared/StatusBadge.vue'

describe('statusBadge', () => {
  it('renders the label text', () => {
    const wrapper = mount(StatusBadge, { props: { label: 'Active' } })
    expect(wrapper.text()).toBe('Active')
  })

  it('defaults to the neutral tone when no tone prop is given', () => {
    const wrapper = mount(StatusBadge, { props: { label: 'Default' } })
    expect(wrapper.classes()).toContain('bg-muted')
    expect(wrapper.classes()).toContain('text-muted-foreground')
  })

  it.each([
    ['success', 'bg-success/15', 'text-success', 'border-success/30'],
    ['warning', 'bg-warning/15', 'text-warning', 'border-warning/30'],
    ['danger', 'bg-danger/15', 'text-danger', 'border-danger/30'],
    ['info', 'bg-info/15', 'text-info', 'border-info/30'],
  ] as const)('applies the %s tone classes', (tone, bgClass, textClass, borderClass) => {
    const wrapper = mount(StatusBadge, { props: { label: 'Status', tone } })
    expect(wrapper.classes()).toContain(bgClass)
    expect(wrapper.classes()).toContain(textClass)
    expect(wrapper.classes()).toContain(borderClass)
  })

  it('always renders using the outline badge variant, regardless of tone', () => {
    const wrapper = mount(StatusBadge, { props: { label: 'Status', tone: 'danger' } })
    expect(wrapper.attributes('data-variant')).toBe('outline')
  })
})
