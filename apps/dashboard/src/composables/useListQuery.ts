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

/** A URL-synced extra filter key. A bare string is forwarded to the fetcher
 * under the same name; `{ key, sendToFetcher: false }` still round-trips
 * through the URL and is still readable via `filters.value[key]`, but is
 * omitted from the object passed to `fetcher` — for UI-only cascading keys
 * (e.g. a department_id that only narrows a company <Select>, never itself
 * sent to a backend endpoint that only accepts company_id). */
export type FilterDecl<F extends string> = F | { key: F; sendToFetcher: false }

interface NormalizedFilterDecl<F extends string> {
  key: F
  sendToFetcher: boolean
}

function normalizeFilterDecls<F extends string>(decls: FilterDecl<F>[]): NormalizedFilterDecl<F>[] {
  return decls.map((d) => (typeof d === 'string' ? { key: d, sendToFetcher: true } : { key: d.key, sendToFetcher: d.sendToFetcher }))
}

interface UseListQueryOptions<F extends string = never> {
  defaultSort?: string
  defaultOrder?: 'asc' | 'desc'
  defaultLimit?: number
  /** Every extra filter key this list accepts, synced to the URL query
   * string exactly like search/sort/order/page — reload, back-button, and
   * sharing a link all preserve them. */
  filters?: FilterDecl<F>[]
  enabled?: () => boolean
}

export type FetcherParams<F extends string> = ListParams & Partial<Record<F, string | number | undefined>>

/**
 * The one composable every listing page uses: syncs page/limit/search/sort/order
 * plus any declared extra filters to the URL query string (so filters survive
 * refresh/back-button/sharing a link) and drives a TanStack Query fetch against
 * the exact same param shape every backend list endpoint accepts
 * (?page=&limit=&search=&sort=&order=&...filters), per spec requirement #12.
 */
export function useListQuery<T, F extends string = never>(
  resourceKey: string,
  fetcher: (params: FetcherParams<F>) => Promise<ApiSuccess<T[]>>,
  options: UseListQueryOptions<F> = {},
) {
  const route = useRoute()
  const router = useRouter()

  const page = computed(() => Number(route.query.page ?? 1) || 1)
  const limit = computed(() => Number(route.query.limit ?? options.defaultLimit ?? 20) || 20)
  const search = computed(() => (route.query.search as string) ?? '')
  const sort = computed(() => (route.query.sort as string) ?? options.defaultSort ?? 'created_at')
  const order = computed(() => ((route.query.order as string) as 'asc' | 'desc') ?? options.defaultOrder ?? 'desc')

  const filterDecls = normalizeFilterDecls(options.filters ?? [])

  /** Reactive, URL-sourced values for every declared filter key — raw
   * strings, exactly like `search`/`sort` already are. Callers `Number()`-cast
   * where they need a number, same convention already used for page/limit. */
  const filters = computed(
    () =>
      Object.fromEntries(filterDecls.map((d) => [d.key, (route.query[d.key] as string) || undefined])) as Partial<
        Record<F, string>
      >,
  )

  function setParams(patch: Partial<{ page: number; limit: number; search: string; sort: string; order: string } & Record<F, string | number | undefined>>) {
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

  const fetcherFilters = computed(
    () =>
      Object.fromEntries(filterDecls.filter((d) => d.sendToFetcher).map((d) => [d.key, filters.value[d.key]])) as Partial<
        Record<F, string | undefined>
      >,
  )

  const queryKey = computed<unknown[]>(() => [resourceKey, page.value, limit.value, search.value, sort.value, order.value, filters.value])

  const query = useQuery({
    queryKey,
    queryFn: () =>
      fetcher({
        page: page.value,
        limit: limit.value,
        search: search.value,
        sort: sort.value,
        order: order.value,
        ...fetcherFilters.value,
      } as FetcherParams<F>),
    placeholderData: keepPreviousData,
    enabled: options.enabled,
  })

  const items = computed(() => query.data.value?.data ?? [])
  const pagination = computed(() => query.data.value?.meta?.pagination)
  const hasActiveFilters = computed(() => search.value !== '' || Object.values(filters.value).some((v) => v !== undefined))

  return { items, pagination, page, limit, search, sort, order, filters, setParams, hasActiveFilters, ...query }
}
