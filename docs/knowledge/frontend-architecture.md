# Dashboard Frontend Architecture (apps/dashboard)

Vue 3 + Composition API + `<script setup lang="ts">`, TanStack Query, Pinia (auth only), vee-validate + zod, shadcn-vue, Tailwind v4. This doc is a map, not a tutorial — file:line references point at real code, read them instead of re-deriving the pattern.

## Stack summary (verified in package.json / main.ts)

- `apps/dashboard/src/main.ts`: Pinia + router + `VueQueryPlugin` installed on the root app, with global query defaults `retry: 1, refetchOnWindowFocus: false, staleTime: 30_000`. There is no manual `QueryClient` construction elsewhere — this is the only place defaults are set.
- Server state (lists, single-resource fetches, mutations) lives in TanStack Query (`@tanstack/vue-query`) via `useQuery`/`useMutation`, keyed and cached per view. It is **never** copied into a Pinia store.
- Pinia (`src/stores/auth.ts`) holds exactly one store: `useAuthStore`, for `user`/`isReady`/`isAuthenticated`/`role` — genuine client/session state, not server-list state.
- Forms: `vee-validate`'s `useForm` + `@vee-validate/zod`'s `toTypedSchema(z.object({...}))`. Real example: `src/views/admin/CompaniesView.vue:164-169` (`toTypedSchema(z.object({ department_id: z.string().min(1, ...), name: z.string().trim().min(2,...).max(255,...), ... }))`).
- UI kit: `src/components/ui/*` is shadcn-vue (reka-ui under the hood) — `alert, avatar, badge, button, calendar, card, checkbox, dialog, drawer, dropdown-menu, form, input, label, native-select, pagination, popover, select, separator, skeleton, sonner, table, tabs, textarea`. Shared higher-level pieces live in `src/components/shared/` (DataTable, ListToolbar, ListPagination, PageHeader, EmptyState, ConfirmDialog, StatusBadge, StarRatingInput).
- Styling: Tailwind v4 (`@import "tailwindcss"` in `src/assets/main.css:3`), no `tailwind.config.js` — theme is driven by CSS custom properties from `packages/design-tokens` (see dedicated section below) plus a `@theme inline` block. Dark mode is a `.dark` class toggle (`@custom-variant dark (&:where(.dark, .dark *));`, `main.css:8`).

## `useListQuery` composable — full contract

File: `src/composables/useListQuery.ts` (121 lines, read it directly for the exact code — summarized precisely below).

Every list-view page in the app calls this once. It is the single source of server-side search/sort/pagination/filter state, and it is 100% URL-synced via `router.replace` — reload, back-button, and sharing a link all reproduce the exact same list state, per the project's own spec requirement #12 (`useListQuery.ts:49`).

```ts
function useListQuery<T, F extends string = never>(
  resourceKey: string,
  fetcher: (params: FetcherParams<F>) => Promise<ApiSuccess<T[]>>,
  options?: {
    defaultSort?: string
    defaultOrder?: 'asc' | 'desc'
    defaultLimit?: number          // default 20
    filters?: FilterDecl<F>[]
    enabled?: () => boolean
  },
): {
  items: ComputedRef<T[]>              // query.data.value?.data ?? []
  pagination: ComputedRef<Pagination | undefined>
  page, limit, search, sort, order: ComputedRef<...>   // read from route.query
  filters: ComputedRef<Partial<Record<F, string>>>     // read from route.query, raw strings
  setParams(patch): void               // merges into route.query via router.replace
  hasActiveFilters: ComputedRef<boolean>
  ...query                              // full TanStack `useQuery` return (isLoading, isError, refetch, data, ...)
}
```

Mechanics (all verified against the source):

- `page`/`limit`/`search`/`sort`/`order` are `computed()`s reading straight off `useRoute().query` (lines 59-63) — there is no local `ref` duplicating them.
- `setParams(patch)` (line 77) builds the next query object by copying every existing string-valued `route.query` entry, applying the patch (deleting keys whose new value is `undefined`/`''`), and calling `router.replace({ query: nextQuery })`. **Any patch that doesn't itself set `page` resets `page` back to `'1'`** (line 87) — changing search/sort/a filter always returns you to page 1.
- `queryKey` (line 98) is `[resourceKey, page, limit, search, sort, order, filters]` — the full `filters` object (not just the sent-to-fetcher subset) is part of the key, so even a `sendToFetcher: false` filter still busts the cache and triggers a refetch when it changes.
- `placeholderData: keepPreviousData` — pagination doesn't flash an empty/loading state between pages.
- `options.enabled` is passed straight through to the underlying `useQuery({ enabled: options.enabled })` — see the gotcha section below, this is where it bites.

### `FilterDecl<F>` / `sendToFetcher: false` — cascading filters

```ts
export type FilterDecl<F extends string> = F | { key: F; sendToFetcher: false }
```

A bare string filter key (e.g. `'company_id'`) round-trips through the URL **and** is forwarded to `fetcher` under that name. `{ key: 'department_id', sendToFetcher: false }` still round-trips through the URL and is still readable via `filters.value.department_id` (for driving a `<Select>`), but is **omitted** from the object passed into `fetcher` (see `fetcherFilters`, line 91-96, which filters `filterDecls` by `.sendToFetcher` before building the object merged into the fetcher call).

This exists for cascading pickers where a parent selector narrows a child selector's options but the backend endpoint being listed only accepts the child's id. Concrete example, `src/views/admin/AdminVacanciesView.vue:110-117`:

```ts
function fetchVacancies(params: FetcherParams<'department_id' | 'company_id'>) {
  return http.get<ApiSuccess<Vacancy[]>>('/vacancies', { params: { ...params, company_id: effectiveCompanyId.value } }).then((r) => r.data)
}
const list = useListQuery<Vacancy, 'department_id' | 'company_id'>('vacancies', fetchVacancies, {
  filters: [{ key: 'department_id', sendToFetcher: false }, { key: 'company_id', sendToFetcher: false }],
  enabled: () => !!effectiveCompanyId.value,
})
```

`department_id` only exists to narrow the company `<Select>`'s options client-side; `/vacancies` itself only takes `company_id`. `company_id` is *also* `sendToFetcher: false` here because the fetcher composes it manually from `effectiveCompanyId` (which folds in `useLastOrgScope` defaults, see below) rather than the raw URL value — declaring it `sendToFetcher: false` avoids the fetcher call double-sending it. Same pattern repeats in `AppliancesView.vue:131-134`, `PresenceReviewView.vue:116-120`, `JournalReviewView.vue:113-117`, `MonitorsView.vue:108-117`, `ScoresView.vue:121-125`.

## THE gotcha: `enabled` must never read the composable's own return value

This is a real, confirmed-live runtime bug pattern, not a hypothetical. It is not caught by `tsc`/`vue-tsc`, and not caught by `useListQuery.spec.ts` either (that spec's `setup()` helper calls `useListQuery(...)` directly, not wrapped in the self-referencing shape below) — it only surfaces as a blank page in the actual running app.

**Never do this:**

```ts
const list = useListQuery('things', fetcher, {
  filters: ['department_id'],
  enabled: () => list.filters.value.department_id !== undefined,   // BOOM
})
```

`enabled` is invoked by TanStack Query as part of constructing the query inside this very `useListQuery(...)` call — i.e. before the outer `const list = ...` assignment has completed. Referencing `list` inside the closure passed to the same statement that produces `list` is a circular initialization: `list` is in the temporal dead zone at the moment `enabled()` actually runs, even though `list.filters` is a completely reasonable-looking, type-correct expression to `tsc`. The crash reads as "Cannot access 'list' before initialization" live, with nothing at typecheck or in the unit-test harness (where `enabled` callbacks are rarely exercised against the exact same self-referencing statement shape) to catch it.

**The fix, applied consistently everywhere this need has come up**: read whatever `enabled` needs directly off `useRoute().query`, in an independent `computed()` declared *before* the `useListQuery` call — never off the composable's own destructured/property result, even transitively through another `computed`.

Correct pattern (verbatim from `src/views/admin/CompaniesView.vue:70-86`, and identically in `CoursesView.vue:70-84`):

```ts
const { items, pagination, page, limit, search, sort, order, filters, setParams, isLoading, isError, refetch } =
  useListQuery<Company, 'department_id'>(
    'companies',
    async (params) => {
      const res = await http.get<ApiSuccess<Company[]>>('/companies', { params })
      return res.data
    },
    {
      defaultSort: 'name',
      defaultOrder: 'asc',
      filters: ['department_id'],
      // Read straight from the route query, not the destructured `filters`
      // above — referencing that here would be a circular self-reference
      // (this options object is part of `filters`'s own initializer).
      enabled: () => isAdmin.value || !!route.query.department_id,
    },
  )
```

The same comment (word for word or near-identical) is left as a guardrail in every view that needs `enabled` to depend on a filter value — grep `enabled: ()` and `circular self-reference` across `src/views` to find all of them: `AttendanceView.vue:51-52/61`, `JournalView.vue:39-40/61`, `MonitorsView.vue:49-52/116`, `PresenceStatusesView.vue:51-57`, `ScorePredicatesView.vue:34-40`, `QuestionsView.vue:61-67/83`, `ScoresView.vue:53-59/124` (routed through `useLastOrgScope`, see below), `CoursesView.vue:81-84`, `CompaniesView.vue:81-84`. The clearest statement of *why* is in `PresenceStatusesView.vue:54-56` / `ScorePredicatesView.vue:37-39`: *"`enabled` would only resolve correctly once `listQuery` already exists, but `listQuery` can't finish constructing until `enabled` is evaluated."*

When writing a new list view: declare every value `enabled` needs as a `computed(() => route.query.xxx ...)` above the `useListQuery` call, never derive it from the call's own return.

## Auth flow (cookie session + CSRF)

Cookie names, verified in `apps/api/internal/modules/identity/handler.go:15-17`: `CookieSession = "internity_session"`, `CookieRefresh = "internity_refresh"`, `CookieCSRF = "internity_csrf"`. Session and refresh are `HttpOnly` (set/read only by the backend); the CSRF cookie is deliberately **not** `HttpOnly` so the frontend can read it and echo it back per the double-submit pattern.

All of this is centralized in `src/lib/http.ts` (single axios instance, `withCredentials: true`, baseURL `${VITE_API_BASE_URL}/api/v1` or `/api/v1`):

- **Request interceptor** (`http.ts:25-32`): for any mutating method (`post|put|patch|delete`), reads the `internity_csrf` cookie via `readCookie()` and sets it as the `X-CSRF-Token` header. GET requests never get the header.
- **Response interceptor, 401 handling** (`http.ts:53-105`):
  - A 401 from `/auth/me` is passed straight through unmodified (`http.ts:80-82`) — it's the expected outcome of the router guard's own initial auth probe (`stores/auth.ts`'s `fetchMe()` already catches it and sets `user = null`). The code comment (`http.ts:73-79`) documents that *without* this early-out, the failed refresh-and-retry below would call `forceLogout()` — a hard `window.location.href = '/login'` — bouncing the very first visit to any guest-only route *other than* `/login` (register, forgot-password, reset-password) straight back to `/login`, dropping `/reset-password`'s `?token=` query param in the process. Confirmed via a live network trace, not assumed, per the comment.
  - `AUTH_ENDPOINTS_WITHOUT_SESSION = ['/auth/login', '/auth/register', '/auth/forgot-password', '/auth/reset-password']` (`http.ts:13`): a 401/429 from these is expected user-facing failure (wrong password, expired token, rate limit) — not a "your session expired" condition, so it skips the refresh dance and is **not** toasted (the calling form shows it inline instead, to avoid duplicating the inline alert + toast).
  - Otherwise: single-flight refresh. `refreshPromise ??= http.post('/auth/refresh').finally(() => refreshPromise = null)` (`http.ts:95-97`) — every 401 arriving while a refresh is already in flight awaits the *same* promise instead of firing its own `/auth/refresh`, preventing duplicate refresh calls under a burst of simultaneously-expiring parallel requests. On success, the original request is retried once (`retryConfig._retried` guards against retry loops); on failure, `forceLogout()`.
  - `forceLogout` (`http.ts:36-41`) defaults to `window.location.href = '/login'` but is overridden by `stores/auth.ts:56-61` at app boot (`registerForceLogout(() => { clear(); ... })`) — done this way specifically to avoid a circular import between `lib/http.ts` and the Pinia store.
- 403 → toast; 422 → passed through untouched (vee-validate/forms handle field errors); 429 → toast unless it's an auth-endpoint-without-session; 500 → generic toast; network error with no `error.response` → generic "Koneksi bermasalah" toast.

## Routing (`src/router/index.ts`)

- Two route groups: `guestRoutes` (under `GuestLayout.vue`, `meta: { guestOnly: true }` — login/register/forgot/reset) and `appRoutes` (under `DefaultLayout.vue`, `meta: { requiresAuth: true }`, each leaf route additionally carrying `meta: { roles: Role[] }` where restricted).
- `router.beforeEach` (`index.ts:88-112`): if `!auth.isReady`, awaits `auth.fetchMe()` first (so a hard refresh always resolves auth state before any guard decision). Then: `requiresAuth && !isAuthenticated` → redirect to `login` with `?redirect=<fullPath>`; `guestOnly && isAuthenticated` → redirect to `dashboard` (with a special toast for `reset-password` specifically, explaining you must log out first — the other guest routes redirect silently); `meta.roles` set and `auth.role` not in it → redirect to `dashboard`.
- Role sets used in `meta.roles`: student-only routes (`vacancies`, `my-applications`, `my-internship`, `attendance`, `journals`, `certificate`), `STAFF = ['admin','coordinator','mentor']` (vacancies/appliances/presence/journals-review/scores admin views), and various `['admin']`-only or `['admin','coordinator']`-only admin views. `news`, `faq`, `notifications`, `profile` have no `roles` restriction (shared across all authenticated roles).

## Onboarding: core tours vs. menu hints

Two deliberately separate systems, both driven by `driver.js`:

**Core tours** — `src/composables/useTour.ts` + `src/tours/{studentTour,coordinatorTour,mentorTour,adminTour}.ts` + `src/tours/index.ts`. One short tour per role (capped at a handful of steps — the header comment in `adminTour.ts:4` cites the "onboard skill" convention of 3-7 steps max), auto-started once from `DefaultLayout.vue:83-93` on first authenticated mount, replayable anytime via a "Replay tour" menu action (`DefaultLayout.vue:78-81`).

**Menu hints** — `src/composables/useMenuHints.ts` + `src/tours/menuInfo.ts`. Everything the core tour doesn't cover gets a single one-time spotlight the first time the user actually navigates to that route (`showMenuHintIfFirstVisit`, called from a `watch(() => route.path, ...)` in `DefaultLayout.vue:98-105`). `menuInfo.ts` is a `Record<path, {selector, title, description}>` — one entry per sidebar item worth explaining, and its `menuStep()` helper (`menuInfo.ts` bottom) is reused by the core tours themselves so the copy for a shared menu item is written once, not duplicated between the two systems.

Why split this way (per `useMenuHints.ts:4-9`): front-loading every menu into one long tour would blow past the 3-7 step convention, so only the critical-path items are in a role's core tour (see e.g. `adminCoreHintPaths` in `adminTour.ts:7`); everything else is progressive disclosure, shown lazily the first time it's actually relevant. `DefaultLayout.vue:87-90` also calls `markHintsSeenFor(coreHintPathsForRole(role))` right before offering the core tour, specifically so the tour's own steps aren't immediately re-explained by a menu hint the instant the user clicks one of the paths the tour just covered.

**localStorage key scoping — a real fixed bug**: `useTour.ts`'s dismissal key is `tour-dismissed:${tourKey}:${userId}` (`useTour.ts:18`) — scoped by **tour key AND user id**, not just by role/tour. The doc comment (`useTour.ts:13-15`) states the reason explicitly: keying only by role would let the first account that ever logs into a shared machine (e.g. a school lab computer) permanently dismiss the tour for every other account that logs in afterward, since they'd share the same role and thus the same localStorage key. `useTour.spec.ts:64-74` has a dedicated regression test for exactly this ("scopes the dismissal flag per userId, not just per role (shared browser profile)"). Note by contrast that `useMenuHints.ts`'s key is `menu-hint-seen:${path}` only (`useMenuHints.ts:10`) — **not** scoped per user — so a shared-machine's menu hints are effectively shared/dismissed across accounts on that machine; if that class of bug needs fixing again, it needs fixing here too, following the same `:${userId}` suffix pattern already proven in `useTour.ts`.

Both `useTour.ts` and `useMenuHints.ts` wrap every `localStorage` access in try/catch and degrade to "hasn't been seen" on failure (private browsing / storage blocked) rather than throwing — see `useTour.spec.ts:120-129` and `:131-141` for the regression tests covering that.

## Design tokens: tokens.json → theme.css → app

- Source of truth: `packages/design-tokens/tokens.json` (palette ramps for `primary`/`accent`/`neutral`, flat `success`/`warning`/`danger`/`info`, `semantic.light`/`semantic.dark` mappings, `radius.base`, `typography.sans`/`display`/`mono`).
- Build: `packages/design-tokens/scripts/build.mjs`, run via `pnpm --filter @internity/design-tokens build`, aliased at the repo root as `pnpm tokens:build` (`package.json:13`). It reads `tokens.json` and writes `packages/design-tokens/dist/theme.css` — a generated file (`/* GENERATED FILE — do not edit by hand. */`) containing a `:root { --color-primary-500: ...; --radius: ...; ... }` block, a `.dark { ... }` block for the dark semantic overrides, and a Tailwind v4 `@theme inline { ... }` block that turns both into actual Tailwind utility classes (`bg-primary-500` etc.).
- Consumption: `apps/dashboard/src/assets/main.css:5` does `@import "@internity/design-tokens/theme.css"`, which resolves via the package's `exports` map (`design-tokens/package.json:9-12`, `"./theme.css": "./dist/theme.css"`) straight to the generated file.
- **Gotcha, confirmed**: `packages/design-tokens/dist/` is gitignored (`.gitignore:14`, `dist/`). A fresh clone has no `dist/theme.css` at all until `tokens:build` runs — `main.css`'s import will fail to resolve otherwise. `apps/dashboard/Dockerfile.prod:12-15` handles this explicitly and calls it out in a comment: `# packages/design-tokens/dist is gitignored (generated locally, never committed) — a fresh clone has no dist/theme.css until this runs, and dashboard's main.css imports @internity/design-tokens/theme.css.` followed by `RUN pnpm run tokens:build`, run **before** `RUN pnpm run build` in the dashboard's own `WORKDIR`. This is necessary specifically because the Dockerfile's final build step is scoped to `apps/dashboard`'s own `build` script (`type-check` + `vite build`, no dependency-graph awareness) rather than the root `pnpm -r build` (which — being pnpm workspaces — would build `design-tokens` first automatically via topological ordering since `dashboard` depends on `@internity/design-tokens: workspace:*`). Any fresh-clone/local/CI setup that runs the dashboard build in isolation (not via root `pnpm -r build`) needs an explicit `pnpm tokens:build` first.

## Shared list-view trio + cascading org-scope pattern

**The trio every admin CRUD list view composes**, always in this combination:
- `src/components/shared/DataTable.vue` — generic `<script setup generic="T extends object">`, takes `columns`/`rows`/`isLoading`/`sort`/`order`/`search`, renders a skeleton-row loading state, and a context-aware empty state: if `search` is truthy and `rows` is empty it shows `"No results for '<search>'"` with a "Clear search" action (`isFilteredEmpty`, `DataTable.vue:35`); otherwise the view's own `emptyTitle`/`emptyDescription`. Per-column custom rendering via `#cell-{key}` scoped slots (`cellValue()` fallback just stringifies the raw field).
- `src/components/shared/ListToolbar.vue` — debounced (300ms, `useDebounceFn`) search input wrapping a `modelValue`, plus a default `<slot>` for per-view filter controls (selects, etc.) laid out with `flex flex-wrap`.
- `src/components/shared/ListPagination.vue` — `page`/`limit`/`total` in, `update:page` out. Clamps a possibly-out-of-range `page` (stale bookmark, hand-edited URL, or a Back-button after deletions shrank the result count) to `[1, lastPage]` before computing the displayed range, so it renders "last valid page" instead of a nonsensical range like `19961–20 of 20` (`ListPagination.vue:15-19`).

**Cascading department → company `<Select>`**, repeated across ~6 admin views (`AdminVacanciesView.vue`, `AppliancesView.vue`, `JournalReviewView.vue`, `MonitorsView.vue`, `PresenceReviewView.vue`, `ScoresView.vue` — all import `useLastOrgScope`): a department picker narrows a company picker's options (`companiesQuery`'s `queryKey` includes `departmentId`, and it's `enabled: () => !isMentor && !!departmentId`, see `ScoresView.vue:79-88`), and `effectiveCompanyId` feeds `useListQuery`'s `enabled` (via a route-query-derived computed, never the list's own return — see the gotcha above).

`src/composables/useLastOrgScope.ts` (68 lines) backs this cascade with a remembered last-used default, so navigating between these ~6 pages doesn't reset both dropdowns every time:
- Backed by `sessionStorage` (key `internity:last-org-scope`) — deliberately per-tab/session, not `localStorage`, because a department picked in a *previous* session would be a stale surprise days later (comment, `useLastOrgScope.ts:5-6`).
- `departmentDefault(routeDepartmentId)`: returns the route value if present, else the remembered one.
- `companyDefault(routeCompanyId, effectiveDepartmentId)`: returns the route value if present; else the remembered company **only if** its remembered department still matches `effectiveDepartmentId` — otherwise `undefined`, specifically to avoid resurrecting a company that no longer belongs to the now-current department (`useLastOrgScope.ts:44-49`).
- `remember(departmentId, companyId)`: callers `watch` their resolved department/company and call this to persist it — e.g. `ScoresView.vue:66-68` explicitly skips remembering for mentors (`if (!isMentor.value) lastScope.remember(d, c)`), since a mentor's company is always their own fixed `auth.user.company_id`, never a picked value, and shouldn't overwrite what staff last picked.
- Explicit URL query params always win over the remembered default — the helpers only supply a fallback when the caller's own route-query read comes back `undefined`, preserving deep-link/Back-button behavior exactly.
