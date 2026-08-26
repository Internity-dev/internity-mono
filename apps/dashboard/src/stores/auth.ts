import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { http, registerForceLogout } from '@/lib/http'
import type { ApiSuccess, User } from '@/types/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isReady = ref(false)

  const isAuthenticated = computed(() => user.value !== null)
  const role = computed(() => user.value?.role ?? null)

  async function fetchMe() {
    try {
      const res = await http.get<ApiSuccess<{ user: User }>>('/auth/me')
      user.value = res.data.data.user
    } catch {
      user.value = null
    } finally {
      isReady.value = true
    }
  }

  async function login(email: string, password: string) {
    const res = await http.post<ApiSuccess<{ user: User }>>('/auth/login', { email, password })
    user.value = res.data.data.user
  }

  interface RegisterPayload {
    name: string
    email: string
    password: string
    password_confirmation: string
    invite_code: string
  }

  async function register(payload: RegisterPayload) {
    const res = await http.post<ApiSuccess<{ user: User }>>('/auth/register', payload)
    user.value = res.data.data.user
  }

  async function logout() {
    try {
      await http.post('/auth/logout')
    } catch {
      // best-effort — clear local state regardless of network outcome
    }
    user.value = null
    window.location.href = '/login'
  }

  function clear() {
    user.value = null
  }

  registerForceLogout(() => {
    clear()
    if (window.location.pathname !== '/login') {
      window.location.href = '/login'
    }
  })

  return { user, isReady, isAuthenticated, role, fetchMe, login, register, logout, clear }
})
