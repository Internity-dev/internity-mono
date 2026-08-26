import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import NotificationBell from '@/components/shared/NotificationBell.vue'

// NotificationBell talks to the network purely through useQuery — mock it
// directly so each test can control exactly what `data` looks like, without
// needing a real QueryClient or HTTP layer. `setup()` re-runs (and so calls
// useQuery again) on every mount(), so a fresh mockReturnValue per test is
// enough — no need to re-import the component.
vi.mock('@tanstack/vue-query', () => ({
  useQuery: vi.fn(),
}))

// The dropdown menu (Reka UI) teleports/portals its content and only mounts
// it once open — none of that popover/positioning machinery is what this
// component's own logic is about (unread count, item list, empty state), so
// it's stubbed with plain always-rendered elements to keep the test focused
// and deterministic.
vi.mock('@/components/ui/dropdown-menu', () => ({
  DropdownMenu: { template: '<div><slot /></div>' },
  DropdownMenuTrigger: { template: '<div><slot /></div>' },
  DropdownMenuContent: { template: '<div><slot /></div>' },
  DropdownMenuItem: { template: '<div class="dm-item"><slot /></div>' },
  DropdownMenuLabel: { template: '<div><slot /></div>' },
  DropdownMenuSeparator: { template: '<hr>' },
}))

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

function mountBell() {
  return mount(NotificationBell, {
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('notificationBell', () => {
  it('does not show an unread badge when unread_count is 0', () => {
    vi.mocked(useQuery).mockReturnValue({
      data: ref({ notifications: [], unread_count: 0 }),
    } as never)

    const wrapper = mountBell()

    expect(wrapper.find('.bg-danger').exists()).toBe(false)
  })

  it('does not show an unread badge while data is still loading (undefined)', () => {
    vi.mocked(useQuery).mockReturnValue({ data: ref(undefined) } as never)

    const wrapper = mountBell()

    expect(wrapper.find('.bg-danger').exists()).toBe(false)
  })

  it('shows the unread count in the badge', () => {
    vi.mocked(useQuery).mockReturnValue({
      data: ref({ notifications: [], unread_count: 3 }),
    } as never)

    const wrapper = mountBell()

    expect(wrapper.find('.bg-danger').text()).toBe('3')
  })

  it('caps the displayed badge count at "9+" for double-digit unread counts', () => {
    vi.mocked(useQuery).mockReturnValue({
      data: ref({ notifications: [], unread_count: 42 }),
    } as never)

    const wrapper = mountBell()

    expect(wrapper.find('.bg-danger').text()).toBe('9+')
  })

  it('renders each notification item title and body', () => {
    vi.mocked(useQuery).mockReturnValue({
      data: ref({
        notifications: [
          { id: 1, type: 'info', title: 'Welcome', body: 'Thanks for joining.', read_at: null, created_at: '2026-01-01' },
          { id: 2, type: 'info', title: 'Reminder', body: 'Submit your journal.', read_at: '2026-01-02', created_at: '2026-01-02' },
        ],
        unread_count: 1,
      }),
    } as never)

    const wrapper = mountBell()

    expect(wrapper.text()).toContain('Welcome')
    expect(wrapper.text()).toContain('Thanks for joining.')
    expect(wrapper.text()).toContain('Reminder')
    expect(wrapper.text()).toContain('Submit your journal.')
  })

  it('shows the empty state when there are no notifications', () => {
    vi.mocked(useQuery).mockReturnValue({
      data: ref({ notifications: [], unread_count: 0 }),
    } as never)

    const wrapper = mountBell()

    expect(wrapper.text()).toContain('No notifications yet')
  })

  it('does not show the empty state when notifications are present', () => {
    vi.mocked(useQuery).mockReturnValue({
      data: ref({
        notifications: [
          { id: 1, type: 'info', title: 'Welcome', body: 'Hi', read_at: null, created_at: '2026-01-01' },
        ],
        unread_count: 1,
      }),
    } as never)

    const wrapper = mountBell()

    expect(wrapper.text()).not.toContain('No notifications yet')
  })

  it('links "See all" to the notifications page', () => {
    vi.mocked(useQuery).mockReturnValue({
      data: ref({ notifications: [], unread_count: 0 }),
    } as never)

    const wrapper = mountBell()

    const seeAll = wrapper.findAll('a').find(a => a.text() === 'See all')
    expect(seeAll?.attributes('href')).toBe('/notifications')
  })
})
