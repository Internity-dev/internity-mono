import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuery, keepPreviousData } from '@tanstack/vue-query'
import type { ApiSuccess } from '@/types/api'

export interface ListParams {
  page: number
  limit: number
  search: string
  sort: string
  order: 'asc' | 'desc'
}

interface UseListQueryOptions {
  defaultSort?: string
  defaultOrder?: 'asc' | 'desc'
  defaultLimit?: number
  /** Extra filter params (e.g. company_id) — included in the query key so
   * changing a filter re-fetches, but NOT synced to the URL by this composable. */
  extraParams?: () => Record<string, string | number | undefined>
  enabled?: () => boolean
}

/**
 * The one composable every listing page uses: syncs page/limit/search/sort/order
 * to the URL query string (so filters survive refresh/back-button) and drives a
 * TanStack Query fetch against the exact same param shape every backend list
 * endpoint accepts (?page=&limit=&search=&sort=&order=), per spec requirement #12.
 */
export function useListQuery<T>(
  resourceKey: string,
  fetcher: (params: ListParams & Record<string, string | number | undefined>) => Promise<ApiSuccess<T[]>>,
  options: UseListQueryOptions = {},
) {
  const route = useRoute()
  const router = useRouter()

  const page = computed(() => Number(route.query.page ?? 1) || 1)
  const limit = computed(() => Number(route.query.limit ?? options.defaultLimit ?? 20) || 20)
  const search = computed(() => (route.query.search as string) ?? '')
  const sort = computed(() => (route.query.sort as string) ?? options.defaultSort ?? 'created_at')
  const order = computed(() => ((route.query.order as string) as 'asc' | 'desc') ?? options.defaultOrder ?? 'desc')

  function setParams(patch: Partial<{ page: number; limit: number; search: string; sort: string; order: string }>) {
    const nextQuery: Record<string, string> = {}
    for (const [k, v] of Object.entries(route.query)) {
      if (typeof v === 'string') nextQuery[k] = v
    }
    for (const [k, v] of Object.entries(patch)) {
      if (v === undefined || v === '') delete nextQuery[k]
      else nextQuery[k] = String(v)
    }
    // Any filter/search/sort change resets pagination back to page 1.
    if (!('page' in patch)) nextQuery.page = '1'
    router.replace({ query: nextQuery })
  }

  const extra = computed(() => options.extraParams?.() ?? {})

  const queryKey = computed(() => [
    resourceKey,
    page.value,
    limit.value,
    search.value,
    sort.value,
    order.value,
    extra.value,
  ])

  const query = useQuery({
    queryKey,
    queryFn: () =>
      fetcher({
        page: page.value,
        limit: limit.value,
        search: search.value,
        sort: sort.value,
        order: order.value,
        ...extra.value,
      }),
    placeholderData: keepPreviousData,
    enabled: options.enabled,
  })

  const items = computed(() => query.data.value?.data ?? [])
  const pagination = computed(() => query.data.value?.meta?.pagination)
  const hasActiveFilters = computed(() => search.value !== '' || Object.values(extra.value).some((v) => v !== undefined))

  return { items, pagination, page, limit, search, sort, order, setParams, hasActiveFilters, ...query }
}
