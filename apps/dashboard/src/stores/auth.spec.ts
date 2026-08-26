import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { User } from '@/types/api'

const httpGetMock = vi.fn()
const httpPostMock = vi.fn()
const registerForceLogoutMock = vi.fn()

vi.mock('@/lib/http', () => ({
  http: {
    get: (...args: unknown[]) => httpGetMock(...args),
    post: (...args: unknown[]) => httpPostMock(...args),
  },
  registerForceLogout: (fn: () => void) => registerForceLogoutMock(fn),
}))

const { useAuthStore } = await import('./auth')

function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: '1',
    role: 'student',
    name: 'Jane Doe',
    email: 'jane@example.com',
    is_active: true,
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('useAuthStore', () => {
  const originalLocation = window.location

  beforeEach(() => {
    setActivePinia(createPinia())
    httpGetMock.mockReset()
    httpPostMock.mockReset()
    registerForceLogoutMock.mockClear()

    Object.defineProperty(window, 'location', {
      configurable: true,
      writable: true,
      value: { ...originalLocation, href: '', pathname: '/' },
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      configurable: true,
      writable: true,
      value: originalLocation,
    })
  })

  it('starts unauthenticated with no role and isReady false', () => {
    const store = useAuthStore()

    expect(store.user).toBeNull()
    expect(store.isReady).toBe(false)
    expect(store.isAuthenticated).toBe(false)
    expect(store.role).toBeNull()
  })

  describe('fetchMe', () => {
    it('loads the current user and flips isReady/isAuthenticated on success', async () => {
      const user = makeUser({ role: 'admin' })
      httpGetMock.mockResolvedValue({ data: { data: { user } } })

      const store = useAuthStore()
      await store.fetchMe()

      expect(httpGetMock).toHaveBeenCalledWith('/auth/me')
      expect(store.user).toEqual(user)
      expect(store.isAuthenticated).toBe(true)
      expect(store.role).toBe('admin')
      expect(store.isReady).toBe(true)
    })

    it('clears the user but still flips isReady on failure (e.g. 401 on boot)', async () => {
      httpGetMock.mockRejectedValue(new Error('401'))

      const store = useAuthStore()
      store.user = makeUser()
      await store.fetchMe()

      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
      expect(store.isReady).toBe(true)
    })
  })

  describe('login', () => {
    it('posts credentials and stores the returned user', async () => {
      const user = makeUser({ role: 'mentor' })
      httpPostMock.mockResolvedValue({ data: { data: { user } } })

      const store = useAuthStore()
      await store.login('jane@example.com', 'secret')

      expect(httpPostMock).toHaveBeenCalledWith('/auth/login', { email: 'jane@example.com', password: 'secret' })
      expect(store.user).toEqual(user)
      expect(store.isAuthenticated).toBe(true)
      expect(store.role).toBe('mentor')
    })

    it('propagates the error and leaves the user unset on invalid credentials', async () => {
      httpPostMock.mockRejectedValue(new Error('invalid credentials'))

      const store = useAuthStore()
      await expect(store.login('jane@example.com', 'wrong')).rejects.toThrow('invalid credentials')

      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('register', () => {
    it('posts the payload and stores the returned user', async () => {
      const user = makeUser({ role: 'student' })
      httpPostMock.mockResolvedValue({ data: { data: { user } } })

      const payload = {
        name: 'Jane Doe',
        email: 'jane@example.com',
        password: 'secret',
        password_confirmation: 'secret',
        invite_code: 'ABC123',
      }

      const store = useAuthStore()
      await store.register(payload)

      expect(httpPostMock).toHaveBeenCalledWith('/auth/register', payload)
      expect(store.user).toEqual(user)
      expect(store.isAuthenticated).toBe(true)
    })
  })

  describe('logout', () => {
    it('clears the user and redirects to /login on success', async () => {
      httpPostMock.mockResolvedValue({ data: { success: true } })

      const store = useAuthStore()
      store.user = makeUser()
      await store.logout()

      expect(httpPostMock).toHaveBeenCalledWith('/auth/logout')
      expect(store.user).toBeNull()
      expect(window.location.href).toBe('/login')
    })

    it('still clears the user and redirects even if the logout request fails (best-effort)', async () => {
      httpPostMock.mockRejectedValue(new Error('network error'))

      const store = useAuthStore()
      store.user = makeUser()
      await store.logout()

      expect(store.user).toBeNull()
      expect(window.location.href).toBe('/login')
    })
  })

  describe('clear', () => {
    it('resets the user to null without touching isReady', () => {
      const store = useAuthStore()
      store.user = makeUser()
      store.isReady = true

      store.clear()

      expect(store.user).toBeNull()
      expect(store.isReady).toBe(true)
    })
  })

  describe('forceLogout wiring (registered for http.ts to call on an unrecoverable 401)', () => {
    it('clears store state and redirects to /login when not already there', () => {
      const store = useAuthStore()
      store.user = makeUser()
      window.location.pathname = '/dashboard'

      const forceLogout = registerForceLogoutMock.mock.calls.at(-1)?.[0] as () => void
      forceLogout()

      expect(store.user).toBeNull()
      expect(window.location.href).toBe('/login')
    })

    it('clears store state but does not redirect again when already on /login', () => {
      const store = useAuthStore()
      store.user = makeUser()
      window.location.pathname = '/login'
      window.location.href = ''

      const forceLogout = registerForceLogoutMock.mock.calls.at(-1)?.[0] as () => void
      forceLogout()

      expect(store.user).toBeNull()
      expect(window.location.href).toBe('')
    })
  })
})
