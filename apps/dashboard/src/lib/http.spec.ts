import { describe, it, expect, beforeEach, vi } from 'vitest'
import axios, { AxiosError, AxiosHeaders } from 'axios'
import { toast } from 'vue-sonner'
import { http, registerForceLogout } from './http'

vi.mock('vue-sonner', () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

// Reach into axios' real (if internal) InterceptorManager to grab the exact
// functions http.ts registered, so we exercise the real interceptor logic
// without needing a live backend or a network-level mock library.
function requestInterceptor() {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (http.interceptors.request as any).handlers[0].fulfilled as (config: unknown) => unknown
}

function responseErrorInterceptor() {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (http.interceptors.response as any).handlers[0].rejected as (error: unknown) => Promise<unknown>
}

function fakeConfig(overrides: Record<string, unknown> = {}) {
  return {
    method: 'get',
    url: '/things',
    headers: new AxiosHeaders(),
    ...overrides,
  }
}

function fakeAxiosError(status: number, config: ReturnType<typeof fakeConfig>, data?: unknown) {
  const response = {
    status,
    data,
    statusText: '',
    headers: new AxiosHeaders(),
    config,
  }
  const error = new AxiosError('Request failed', String(status), config as never, {}, response as never)
  return error
}

function clearCsrfCookie() {
  document.cookie = 'internity_csrf=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/'
}

describe('http instance configuration', () => {
  it('targets the versioned API base URL and sends credentials', () => {
    // baseURL is relative ("/api/v1") unless VITE_API_BASE_URL is set in the
    // local environment (e.g. apps/dashboard/.env.local for dev against a
    // non-proxied API) — either way it must end with the versioned path.
    expect(http.defaults.baseURL).toMatch(/\/api\/v1$/)
    expect(http.defaults.withCredentials).toBe(true)
  })
})

describe('request interceptor — CSRF header attachment', () => {
  beforeEach(() => {
    clearCsrfCookie()
  })

  it('does not attach a CSRF header to safe (GET) requests even if the cookie is present', async () => {
    document.cookie = 'internity_csrf=abc123'
    const config = fakeConfig({ method: 'get' })
    const result = (await requestInterceptor()(config)) as typeof config
    expect(result.headers.get('X-CSRF-Token')).toBeFalsy()
  })

  it('attaches the decoded CSRF cookie as X-CSRF-Token on mutating requests', async () => {
    document.cookie = `internity_csrf=${encodeURIComponent('abc 123')}`
    const config = fakeConfig({ method: 'post' })
    const result = (await requestInterceptor()(config)) as typeof config
    expect(result.headers.get('X-CSRF-Token')).toBe('abc 123')
  })

  it.each(['put', 'patch', 'delete'])('also attaches the header for %s requests', async (method) => {
    document.cookie = 'internity_csrf=xyz'
    const config = fakeConfig({ method })
    const result = (await requestInterceptor()(config)) as typeof config
    expect(result.headers.get('X-CSRF-Token')).toBe('xyz')
  })

  it('leaves the header unset for mutating requests when there is no cookie', async () => {
    const config = fakeConfig({ method: 'post' })
    const result = (await requestInterceptor()(config)) as typeof config
    expect(result.headers.get('X-CSRF-Token')).toBeFalsy()
  })

  it('defaults a config with no method to GET (no header attached)', async () => {
    document.cookie = 'internity_csrf=abc123'
    const config = fakeConfig({ method: undefined })
    const result = (await requestInterceptor()(config)) as typeof config
    expect(result.headers.get('X-CSRF-Token')).toBeFalsy()
  })
})

describe('response interceptor — error toasts', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    registerForceLogout(vi.fn())
  })

  it('shows a network-error toast when there is no response at all (e.g. request timed out)', async () => {
    const err = new AxiosError('timeout of 5000ms exceeded', 'ECONNABORTED', fakeConfig() as never)
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).toHaveBeenCalledWith('Koneksi bermasalah. Periksa jaringan Anda dan coba lagi.')
  })

  it('shows the same network-error toast for a completely non-axios error', async () => {
    const err = new Error('boom')
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).toHaveBeenCalledWith('Koneksi bermasalah. Periksa jaringan Anda dan coba lagi.')
  })

  it('shows the server-provided message on 403', async () => {
    const config = fakeConfig({ url: '/admin/x' })
    const err = fakeAxiosError(403, config, { message: 'Forbidden zone' })
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).toHaveBeenCalledWith('Forbidden zone')
  })

  it('falls back to a default message on 403 when the body has none', async () => {
    const config = fakeConfig({ url: '/admin/x' })
    const err = fakeAxiosError(403, config, {})
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).toHaveBeenCalledWith('Anda tidak memiliki izin untuk melakukan ini.')
  })

  it('does not toast on 422 — field errors are left for the calling form', async () => {
    const config = fakeConfig({ url: '/things', method: 'post' })
    const err = fakeAxiosError(422, config, { message: 'Validation failed' })
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).not.toHaveBeenCalled()
  })

  it('shows the rate-limit message on 429', async () => {
    const config = fakeConfig()
    const err = fakeAxiosError(429, config, {})
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).toHaveBeenCalledWith('Terlalu banyak percobaan. Coba lagi nanti.')
  })

  it('does not toast a 429 from an auth endpoint without a session — the calling form shows it inline instead', async () => {
    const config = fakeConfig({ url: '/auth/login', method: 'post' })
    const err = fakeAxiosError(429, config, { message: 'Too many requests. Please try again later.' })
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).not.toHaveBeenCalled()
  })

  it('shows a generic message on 500', async () => {
    const config = fakeConfig()
    const err = fakeAxiosError(500, config, { message: 'ignored, 500 always uses the generic copy' })
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).toHaveBeenCalledWith('Terjadi kesalahan pada server. Silakan coba lagi.')
  })

  it('toasts the body message for other status codes when present', async () => {
    const config = fakeConfig()
    const err = fakeAxiosError(418, config, { message: "I'm a teapot" })
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).toHaveBeenCalledWith("I'm a teapot")
  })

  it('stays silent for other status codes with no message', async () => {
    const config = fakeConfig()
    const err = fakeAxiosError(418, config, {})
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)
    expect(toast.error).not.toHaveBeenCalled()
  })
})

describe('response interceptor — 401 single-flight refresh + retry', () => {
  const originalAdapter = http.defaults.adapter

  beforeEach(() => {
    vi.clearAllMocks()
    registerForceLogout(vi.fn())
  })

  it('immediately forces logout when the request was already retried once', async () => {
    const forceLogout = vi.fn()
    registerForceLogout(forceLogout)
    const adapter = vi.fn()
    http.defaults.adapter = adapter

    const config = fakeConfig({ url: '/things', _retried: true })
    const err = fakeAxiosError(401, config, {})
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)

    expect(forceLogout).toHaveBeenCalledTimes(1)
    expect(adapter).not.toHaveBeenCalled()

    http.defaults.adapter = originalAdapter
  })

  it('immediately forces logout when the 401 comes from the refresh endpoint itself (guards against a refresh loop)', async () => {
    const forceLogout = vi.fn()
    registerForceLogout(forceLogout)
    const adapter = vi.fn()
    http.defaults.adapter = adapter

    const config = fakeConfig({ url: '/auth/refresh', method: 'post' })
    const err = fakeAxiosError(401, config, {})
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)

    expect(forceLogout).toHaveBeenCalledTimes(1)
    expect(adapter).not.toHaveBeenCalled()

    http.defaults.adapter = originalAdapter
  })

  it('refreshes once and retries the original request on a first 401', async () => {
    const forceLogout = vi.fn()
    registerForceLogout(forceLogout)

    const calls: string[] = []
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter = vi.fn(async (config: any) => {
      calls.push(config.url)
      return {
        data: { success: true, data: { ok: true }, message: 'ok' },
        status: 200,
        statusText: 'OK',
        headers: new AxiosHeaders(),
        config,
      }
    })
    http.defaults.adapter = adapter

    const config = fakeConfig({ url: '/things' })
    const err = fakeAxiosError(401, config, {})
    const result = (await responseErrorInterceptor()(err)) as { data: { success: boolean } }

    expect(calls).toEqual(['/auth/refresh', '/things'])
    expect(result.data.success).toBe(true)
    expect(forceLogout).not.toHaveBeenCalled()
    expect((config as unknown as { _retried?: boolean })._retried).toBe(true)

    http.defaults.adapter = originalAdapter
  })

  it('only issues a single /auth/refresh call for two concurrent 401s (single-flight)', async () => {
    const forceLogout = vi.fn()
    registerForceLogout(forceLogout)

    const calls: string[] = []
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter = vi.fn(async (config: any) => {
      calls.push(config.url)
      return {
        data: { success: true, data: {}, message: 'ok' },
        status: 200,
        statusText: 'OK',
        headers: new AxiosHeaders(),
        config,
      }
    })
    http.defaults.adapter = adapter

    const configA = fakeConfig({ url: '/a' })
    const configB = fakeConfig({ url: '/b' })
    const errA = fakeAxiosError(401, configA, {})
    const errB = fakeAxiosError(401, configB, {})

    const [resA, resB] = await Promise.all([
      responseErrorInterceptor()(errA),
      responseErrorInterceptor()(errB),
    ])

    expect(calls.filter((u) => u === '/auth/refresh')).toHaveLength(1)
    expect(calls.filter((u) => u === '/a')).toHaveLength(1)
    expect(calls.filter((u) => u === '/b')).toHaveLength(1)
    expect((resA as { data: unknown }).data).toBeTruthy()
    expect((resB as { data: unknown }).data).toBeTruthy()
    expect(forceLogout).not.toHaveBeenCalled()

    http.defaults.adapter = originalAdapter
  })

  it('forces logout when the refresh call itself fails', async () => {
    const forceLogout = vi.fn()
    registerForceLogout(forceLogout)

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const adapter = vi.fn(async (config: any) => {
      if (config.url === '/auth/refresh') {
        const refreshConfig = fakeConfig({ url: '/auth/refresh', method: 'post' })
        throw fakeAxiosError(401, refreshConfig, {})
      }
      return {
        data: { success: true, data: {}, message: 'ok' },
        status: 200,
        statusText: 'OK',
        headers: new AxiosHeaders(),
        config,
      }
    })
    http.defaults.adapter = adapter

    const config = fakeConfig({ url: '/things' })
    const err = fakeAxiosError(401, config, {})

    await expect(responseErrorInterceptor()(err)).rejects.toBeTruthy()
    expect(forceLogout).toHaveBeenCalled()

    http.defaults.adapter = originalAdapter
  })

  it('toasts a 401 from /auth/login instead of retrying it as a session refresh', async () => {
    const forceLogout = vi.fn()
    registerForceLogout(forceLogout)
    const adapter = vi.fn()
    http.defaults.adapter = adapter

    const config = fakeConfig({ url: '/auth/login', method: 'post' })
    const err = fakeAxiosError(401, config, { message: 'Invalid email or password' })
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)

    expect(toast.error).toHaveBeenCalledWith('Invalid email or password')
    expect(forceLogout).not.toHaveBeenCalled()
    expect(adapter).not.toHaveBeenCalled()

    http.defaults.adapter = originalAdapter
  })

  it('toasts a 401 from /auth/register the same way', async () => {
    const forceLogout = vi.fn()
    registerForceLogout(forceLogout)
    const adapter = vi.fn()
    http.defaults.adapter = adapter

    const config = fakeConfig({ url: '/auth/register', method: 'post' })
    const err = fakeAxiosError(401, config, {})
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)

    expect(toast.error).toHaveBeenCalledWith('Email atau kata sandi salah.')
    expect(forceLogout).not.toHaveBeenCalled()
    expect(adapter).not.toHaveBeenCalled()

    http.defaults.adapter = originalAdapter
  })
})

describe('registerForceLogout', () => {
  it('lets the caller replace the redirect strategy used on an unrecoverable 401', async () => {
    const spy = vi.fn()
    registerForceLogout(spy)

    const adapter = vi.fn()
    const originalAdapter = http.defaults.adapter
    http.defaults.adapter = adapter

    const config = fakeConfig({ url: '/things', _retried: true })
    const err = fakeAxiosError(401, config, {})
    await expect(responseErrorInterceptor()(err)).rejects.toBe(err)

    expect(spy).toHaveBeenCalledTimes(1)
    http.defaults.adapter = originalAdapter
  })
})

// axios re-export sanity check: confirms fakeAxiosError() below actually produces
// objects axios.isAxiosError() recognizes, so the "network error" branch tests above
// are exercising the real `!axios.isAxiosError(error)` check rather than trivially
// falling through it.
describe('sanity: fake errors are real AxiosErrors', () => {
  it('axios.isAxiosError recognizes our fakeAxiosError helper', () => {
    const err = fakeAxiosError(500, fakeConfig(), {})
    expect(axios.isAxiosError(err)).toBe(true)
  })
})
