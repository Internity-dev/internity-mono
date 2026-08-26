import { describe, it, expect, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import { VueQueryPlugin, QueryClient } from '@tanstack/vue-query'
import { useListQuery, type ListParams } from './useListQuery'
import type { ApiSuccess } from '@/types/api'

type Fetcher<T = unknown> = (params: ListParams & Record<string, string | number | undefined>) => Promise<ApiSuccess<T[]>>

function okResponse<T>(data: T[] = [], pagination?: { page: number; limit: number; total: number }): ApiSuccess<T[]> {
  return { success: true, message: 'ok', data, meta: pagination ? { pagination } : undefined }
}

async function setup(
  initialPath: string,
  fetcher: Fetcher,
  options: Parameters<typeof useListQuery>[2] = {},
) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/list', component: { template: '<div/>' } }],
  })
  await router.push(initialPath)
  await router.isReady()

  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let result: any
  const Host = defineComponent({
    setup() {
      result = useListQuery('things', fetcher, options)
      return () => h('div')
    },
  })

  const wrapper = mount(Host, {
    global: { plugins: [router, [VueQueryPlugin, { queryClient }]] },
  })

  await flushPromises()

  return { wrapper, router, result: result as ReturnType<typeof useListQuery>, queryClient }
}

describe('useListQuery — reading state from the URL', () => {
  it('defaults page/limit/search/sort/order when the URL has no query params', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result } = await setup('/list', fetcher)

    expect(result.page.value).toBe(1)
    expect(result.limit.value).toBe(20)
    expect(result.search.value).toBe('')
    expect(result.sort.value).toBe('created_at')
    expect(result.order.value).toBe('desc')
  })

  it('honors defaultSort/defaultOrder/defaultLimit options when the URL has no query params', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result } = await setup('/list', fetcher, {
      defaultSort: 'name',
      defaultOrder: 'asc',
      defaultLimit: 50,
    })

    expect(result.limit.value).toBe(50)
    expect(result.sort.value).toBe('name')
    expect(result.order.value).toBe('asc')
  })

  it('reads existing query params from the URL instead of the defaults', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result } = await setup('/list?page=3&limit=50&search=foo&sort=name&order=asc', fetcher)

    expect(result.page.value).toBe(3)
    expect(result.limit.value).toBe(50)
    expect(result.search.value).toBe('foo')
    expect(result.sort.value).toBe('name')
    expect(result.order.value).toBe('asc')
  })

  it('falls back to 1/20 for garbage numeric query params', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result } = await setup('/list?page=not-a-number&limit=nope', fetcher)

    expect(result.page.value).toBe(1)
    expect(result.limit.value).toBe(20)
  })
})

describe('useListQuery — fetching', () => {
  it('calls the fetcher with the page/limit/search/sort/order shape every list endpoint expects', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse([{ id: 1 }]))
    await setup('/list?page=2&limit=10&search=bob&sort=name&order=asc', fetcher)

    await vi.waitFor(() => expect(fetcher).toHaveBeenCalled())
    expect(fetcher).toHaveBeenCalledWith({
      page: 2,
      limit: 10,
      search: 'bob',
      sort: 'name',
      order: 'asc',
    })
  })

  it('merges extraParams into both the fetcher call and hasActiveFilters', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result } = await setup('/list', fetcher, {
      extraParams: () => ({ company_id: 42 }),
    })

    await vi.waitFor(() => expect(fetcher).toHaveBeenCalled())
    expect(fetcher).toHaveBeenCalledWith(expect.objectContaining({ company_id: 42 }))
    expect(result.hasActiveFilters.value).toBe(true)
  })

  it('does not fetch at all when enabled() is false', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    await setup('/list', fetcher, { enabled: () => false })

    await flushPromises()
    expect(fetcher).not.toHaveBeenCalled()
  })

  it('exposes items and pagination derived from the resolved response', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse([{ id: 1 }, { id: 2 }], { page: 1, limit: 20, total: 2 }))
    const { result } = await setup('/list', fetcher)

    await vi.waitFor(() => expect(result.isSuccess.value).toBe(true))
    expect(result.items.value).toEqual([{ id: 1 }, { id: 2 }])
    expect(result.pagination.value).toEqual({ page: 1, limit: 20, total: 2 })
  })

  it('defaults items to [] and pagination to undefined before data arrives', async () => {
    const fetcher = vi.fn().mockReturnValue(new Promise(() => {})) // never resolves
    const { result } = await setup('/list', fetcher)

    expect(result.items.value).toEqual([])
    expect(result.pagination.value).toBeUndefined()
  })
})

describe('useListQuery — hasActiveFilters', () => {
  it('is false with no search and no extraParams', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result } = await setup('/list', fetcher)
    expect(result.hasActiveFilters.value).toBe(false)
  })

  it('is true as soon as a search term is present in the URL', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result } = await setup('/list?search=x', fetcher)
    expect(result.hasActiveFilters.value).toBe(true)
  })
})

describe('useListQuery — setParams / URL syncing', () => {
  it('writes a patched param into the URL and resets page back to 1', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result, router } = await setup('/list?page=5', fetcher)

    result.setParams({ search: 'abc' })
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({ search: 'abc', page: '1' })
  })

  it('does not force page back to 1 when the patch explicitly sets page', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result, router } = await setup('/list?search=abc', fetcher)

    result.setParams({ page: 3 })
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({ search: 'abc', page: '3' })
  })

  it('removes a key from the URL when patched with an empty string', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result, router } = await setup('/list?search=abc&sort=name', fetcher)

    result.setParams({ search: '' })
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({ sort: 'name', page: '1' })
  })

  it('removes a key from the URL when patched with undefined', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result, router } = await setup('/list?search=abc&sort=name', fetcher)

    result.setParams({ search: undefined })
    await flushPromises()

    expect(router.currentRoute.value.query).toEqual({ sort: 'name', page: '1' })
  })

  it('preserves unrelated existing query params untouched', async () => {
    const fetcher = vi.fn().mockResolvedValue(okResponse())
    const { result, router } = await setup('/list?sort=name&order=asc&limit=50', fetcher)

    result.setParams({ search: 'x' })
    await flushPromises()

    expect(router.currentRoute.value.query).toMatchObject({ sort: 'name', order: 'asc', limit: '50', search: 'x' })
  })
})
