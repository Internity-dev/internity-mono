import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'

// ConfirmDialog wraps a Reka UI Dialog, which teleports its content to
// document.body and mounts it asynchronously (via a Presence wrapper) once
// `open` is true — so every assertion needs a flush after mount before the
// teleported DOM is queryable. attachTo puts the wrapper's own root in the
// document too, and we track + unmount every wrapper so state (and the
// teleported nodes) don't leak between tests.
let wrapper: VueWrapper | undefined

function findButtonByText(text: string): HTMLButtonElement | undefined {
  return Array.from(document.querySelectorAll('button')).find(b => b.textContent?.trim() === text)
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

describe('confirmDialog', () => {
  it('renders no dialog content in the document when closed', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: false, title: 'Delete item' },
      attachTo: document.body,
    })
    await flushPromises()

    expect(document.body.textContent).not.toContain('Delete item')
  })

  it('renders the title and description when open', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item', description: 'This cannot be undone.' },
      attachTo: document.body,
    })
    await flushPromises()

    expect(document.body.textContent).toContain('Delete item')
    expect(document.body.textContent).toContain('This cannot be undone.')
  })

  it('does not render a description when none is given', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item' },
      attachTo: document.body,
    })
    await flushPromises()

    expect(document.querySelector('[data-slot="dialog-description"]')).toBeNull()
  })

  it('uses "Confirm" as the default confirm button label', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item' },
      attachTo: document.body,
    })
    await flushPromises()

    expect(findButtonByText('Confirm')).toBeDefined()
  })

  it('uses a custom confirm label when given', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item', confirmLabel: 'Delete forever' },
      attachTo: document.body,
    })
    await flushPromises()

    expect(findButtonByText('Delete forever')).toBeDefined()
  })

  it('emits "confirm" when the confirm button is clicked', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item', confirmLabel: 'Delete forever' },
      attachTo: document.body,
    })
    await flushPromises()

    findButtonByText('Delete forever')?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()

    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('emits update:open with false when the cancel button is clicked', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item' },
      attachTo: document.body,
    })
    await flushPromises()

    findButtonByText('Cancel')?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()

    expect(wrapper.emitted('update:open')).toEqual([[false]])
  })

  it('does not emit "confirm" when the cancel button is clicked', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item' },
      attachTo: document.body,
    })
    await flushPromises()

    findButtonByText('Cancel')?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()

    expect(wrapper.emitted('confirm')).toBeUndefined()
  })

  it('shows a loading label and disables both buttons while isLoading is true', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item', isLoading: true },
      attachTo: document.body,
    })
    await flushPromises()

    const cancelButton = findButtonByText('Cancel')
    const confirmButton = findButtonByText('Please wait…')

    expect(cancelButton?.disabled).toBe(true)
    expect(confirmButton?.disabled).toBe(true)
  })

  it('defaults to the destructive button variant', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item', confirmLabel: 'Delete forever' },
      attachTo: document.body,
    })
    await flushPromises()

    expect(findButtonByText('Delete forever')?.getAttribute('data-variant')).toBe('destructive')
  })

  it('uses the default button variant when destructive is false', async () => {
    wrapper = mount(ConfirmDialog, {
      props: { open: true, title: 'Delete item', confirmLabel: 'Save', destructive: false },
      attachTo: document.body,
    })
    await flushPromises()

    expect(findButtonByText('Save')?.getAttribute('data-variant')).toBe('default')
  })
})
