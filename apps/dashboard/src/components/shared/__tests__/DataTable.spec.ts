import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'

interface Row {
  id: number
  name: string
  status: string
}

const columns: Column[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'status', label: 'Status' },
]

const rows: Row[] = [
  { id: 1, name: 'Alice', status: 'active' },
  { id: 2, name: 'Bob', status: 'inactive' },
]

describe('dataTable', () => {
  it('renders a header cell for every column', () => {
    const wrapper = mount(DataTable, { props: { columns, rows: [] as Row[] } })
    const headers = wrapper.findAll('th')
    expect(headers).toHaveLength(2)
    expect(headers[0]?.text()).toContain('Name')
    expect(headers[1]?.text()).toContain('Status')
  })

  it('shows 5 skeleton rows while loading, and no data rows', () => {
    const wrapper = mount(DataTable, { props: { columns, rows, isLoading: true } })
    const bodyRows = wrapper.findAll('tbody > tr')
    expect(bodyRows).toHaveLength(5)
    expect(wrapper.text()).not.toContain('Alice')
    expect(wrapper.findAll('[data-slot="skeleton"]')).toHaveLength(5 * columns.length)
  })

  it('shows the empty state when there are no rows and it is not loading', () => {
    const wrapper = mount(DataTable, { props: { columns, rows: [] as Row[] } })
    expect(wrapper.text()).toContain('No results')
  })

  it('shows a custom empty title and description when provided', () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        rows: [] as Row[],
        emptyTitle: 'No students yet',
        emptyDescription: 'Invite one to get started.',
      },
    })
    expect(wrapper.text()).toContain('No students yet')
    expect(wrapper.text()).toContain('Invite one to get started.')
  })

  it('renders each row using the default stringified cell value when no slot is given', () => {
    const wrapper = mount(DataTable, { props: { columns, rows } })
    const bodyRows = wrapper.findAll('tbody > tr')
    expect(bodyRows).toHaveLength(2)
    expect(bodyRows[0]?.text()).toContain('Alice')
    expect(bodyRows[0]?.text()).toContain('active')
    expect(bodyRows[1]?.text()).toContain('Bob')
    expect(bodyRows[1]?.text()).toContain('inactive')
  })

  it('overrides the default cell rendering with a named #cell-{key} slot', () => {
    const wrapper = mount(DataTable, {
      props: { columns, rows },
      slots: {
        'cell-status': `<template #cell-status="{ row }"><span class="pill">{{ row.status.toUpperCase() }}</span></template>`,
      },
    })
    const bodyRows = wrapper.findAll('tbody > tr')
    // The slotted column renders the custom markup instead of the raw value.
    expect(bodyRows[0]?.find('span.pill').text()).toBe('ACTIVE')
    expect(bodyRows[1]?.find('span.pill').text()).toBe('INACTIVE')
    // The un-slotted column still falls back to the default stringified value.
    expect(bodyRows[0]?.text()).toContain('Alice')
  })

  it('shows a "no results for" message and Clear search action when a search is active and rows are empty', () => {
    const wrapper = mount(DataTable, {
      props: {
        columns,
        rows: [] as Row[],
        search: 'zzzzz',
        emptyTitle: 'No students yet',
        emptyDescription: 'Invite one to get started.',
      },
    })
    expect(wrapper.text()).toContain("No results for 'zzzzz'")
    expect(wrapper.text()).not.toContain('No students yet')
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toBe('Clear search')
  })

  it('emits "clear-search" when the Clear search action is clicked', async () => {
    const wrapper = mount(DataTable, {
      props: { columns, rows: [] as Row[], search: 'zzzzz' },
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('clear-search')).toHaveLength(1)
  })

  it('falls back to the view-supplied empty copy when there is no active search', () => {
    const wrapper = mount(DataTable, {
      props: { columns, rows: [] as Row[], emptyTitle: 'No students yet' },
    })
    expect(wrapper.text()).toContain('No students yet')
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('emits "sort" with the column key when a sortable header is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, rows } })
    const headers = wrapper.findAll('th')
    await headers[0]?.trigger('click')
    expect(wrapper.emitted('sort')).toEqual([['name']])
  })

  it('does not emit "sort" when a non-sortable header is clicked', async () => {
    const wrapper = mount(DataTable, { props: { columns, rows } })
    const headers = wrapper.findAll('th')
    await headers[1]?.trigger('click')
    expect(wrapper.emitted('sort')).toBeUndefined()
  })

  it('shows a sort indicator icon matching the active sort column and order', () => {
    const wrapper = mount(DataTable, { props: { columns, rows, sort: 'name', order: 'asc' } })
    const headers = wrapper.findAll('th')
    expect(headers[0]?.findAll('svg')).toHaveLength(1)
    expect(headers[1]?.findAll('svg')).toHaveLength(0)
  })
})
