import axios, { type AxiosRequestConfig, type InternalAxiosRequestConfig } from 'axios'
import { toast } from 'vue-sonner'
import type { ApiErrorBody } from '@/types/api'

const MUTATING_METHODS = new Set(['post', 'put', 'patch', 'delete'])

// These endpoints are expected to fail on their own (wrong password, expired
// reset token, rate limit). A 401 from them is not "your session expired,"
// so it must not trigger the refresh-and-retry dance below. A 401/429 from
// them is also not toasted globally — the calling view shows it inline
// instead, since these are form submissions where a persistent message next
// to the form beats a transient toast.
const AUTH_ENDPOINTS_WITHOUT_SESSION = ['/auth/login', '/auth/register', '/auth/forgot-password', '/auth/reset-password']

function readCookie(name: string): string | null {
  const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'))
  return match?.[1] ? decodeURIComponent(match[1]) : null
}

export const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ? `${import.meta.env.VITE_API_BASE_URL}/api/v1` : '/api/v1',
  withCredentials: true,
})

http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const method = (config.method ?? 'get').toLowerCase()
  if (MUTATING_METHODS.has(method)) {
    const csrf = readCookie('internity_csrf')
    if (csrf) config.headers.set('X-CSRF-Token', csrf)
  }
  return config
})

// forceLogout is wired up by the auth store at app boot (see stores/auth.ts)
// to avoid a circular import between this module and Pinia state.
let forceLogout: () => void = () => {
  window.location.href = '/login'
}
export function registerForceLogout(fn: () => void) {
  forceLogout = fn
}

// Single-flight refresh: every 401 that arrives while a refresh is already
// in progress awaits the SAME promise instead of firing its own
// /auth/refresh — this is what prevents duplicate concurrent refresh calls
// under a burst of parallel requests that all expired at once.
let refreshPromise: Promise<unknown> | null = null

interface RetryableConfig extends AxiosRequestConfig {
  _retried?: boolean
}

http.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (!axios.isAxiosError(error) || !error.response) {
      toast.error('Koneksi bermasalah. Periksa jaringan Anda dan coba lagi.')
      return Promise.reject(error)
    }

    const { response, config } = error
    const body = response.data as ApiErrorBody | undefined
    const retryConfig = config as RetryableConfig | undefined
    const isAuthEndpointWithoutSession = !!retryConfig?.url && AUTH_ENDPOINTS_WITHOUT_SESSION.some((path) => retryConfig.url!.includes(path))

    switch (response.status) {
      case 401: {
        // A 401 from /auth/me is the expected, normal outcome of the
        // router guard's own auth check on a visitor with no session yet —
        // stores/auth.ts's fetchMe() already catches it and sets user=null
        // silently. Without this, it fell through to the refresh-and-retry
        // dance below, which also fails (no session to refresh either), and
        // that failure calls forceLogout() — a hard `window.location.href
        // = '/login'`. That bounced every first-ever visit to any
        // guest-only route besides /login (register, forgot-password,
        // reset-password — losing its `?token=` query param in the
        // process) straight back to /login before the visitor ever saw the
        // page they were trying to reach. Confirmed live via a network
        // trace on a fresh `/register` visit, not assumed.
        if (retryConfig?.url?.includes('/auth/me')) {
          return Promise.reject(error)
        }
        if (isAuthEndpointWithoutSession) {
          toast.error(body?.message ?? 'Email atau kata sandi salah.')
          break
        }
        if (!retryConfig || retryConfig._retried || retryConfig.url?.includes('/auth/refresh')) {
          forceLogout()
          return Promise.reject(error)
        }
        retryConfig._retried = true
        refreshPromise ??= http.post('/auth/refresh').finally(() => {
          refreshPromise = null
        })
        try {
          await refreshPromise
          return http(retryConfig)
        } catch {
          forceLogout()
          return Promise.reject(error)
        }
      }
      case 403:
        toast.error(body?.message ?? 'Anda tidak memiliki izin untuk melakukan ini.')
        break
      case 422:
        // Field-level errors are surfaced by the calling form (vee-validate),
        // not a toast — pass through untouched.
        break
      case 429:
        // Auth-form rate limits are shown inline by the calling view (with
        // Retry-After), not toasted — see AUTH_ENDPOINTS_WITHOUT_SESSION above.
        if (!isAuthEndpointWithoutSession) {
          toast.error(body?.message ?? 'Terlalu banyak percobaan. Coba lagi nanti.')
        }
        break
      case 500:
        toast.error('Terjadi kesalahan pada server. Silakan coba lagi.')
        break
      default:
        if (body?.message) toast.error(body.message)
    }
    return Promise.reject(error)
  },
)
