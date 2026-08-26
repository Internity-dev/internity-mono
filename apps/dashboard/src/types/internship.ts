export interface Presence {
  id: number
  user_id: string
  // Only populated by GET /presences/pending-approval (its join against
  // users) — absent (omitempty backend-side) from every other endpoint that
  // returns a Presence.
  user_name?: string
  user_nis?: string | null
  company_id: number
  presence_status_id: number
  date: string
  check_in_at: string | null
  check_out_at: string | null
  check_in_lat: number | null
  check_in_lng: number | null
  attachment_key: string | null
  is_approved: boolean
  description: string | null
  created_at: string
  updated_at: string
}

export interface Journal {
  id: number
  user_id: string
  // Only populated by GET /journals/pending-approval (its join against
  // users) — absent (omitempty backend-side) from every other endpoint that
  // returns a Journal.
  user_name?: string
  user_nis?: string | null
  company_id: number
  date: string
  work_type: string | null
  description: string | null
  is_approved: boolean
  created_at: string
  updated_at: string
}

// Bulk-approve responses are built from a literal gin.H{"approved_count": ...}
// map (not a tagged struct), so this one IS genuinely snake_case.
export interface BulkApproveResult {
  approved_count: number
}

// --- InternDate (a student's placement) ---
export type InternDateStatus = 'scheduled' | 'completed'

export interface InternDate {
  id: number
  user_id: string
  company_id: number
  appliance_id: number
  start_date: string | null
  end_date: string | null
  extended_until: string | null
  status: InternDateStatus
  version: number
  created_at: string
  updated_at: string
}

// --- Presence statuses (school-configured, e.g. "Present", "Sick") ---
export type PresenceStatusKind = 'present' | 'permitted' | 'sick' | 'absent' | 'holiday'

export interface PresenceStatus {
  id: number
  school_id: number
  name: string
  kind: PresenceStatusKind
  description: string | null
  color: string | null
  icon: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreatePresenceStatusPayload {
  school_id: number
  name: string
  kind: PresenceStatusKind
  description?: string
  color?: string
  icon?: string
}

export interface PresenceStatusPatch {
  name?: string
  description?: string
  color?: string
  icon?: string
  is_active?: boolean
}

// --- Backend key-casing normalizer ---
//
// See the PresenceStatus note above. Converts a raw response object/array
// from PascalCase (the actual wire format for any untagged Go struct in
// this API) to snake_case. It's a no-op on keys that are already
// snake_case, so it stays correct if/when json tags get added backend-side.
function pascalToSnake(key: string): string {
  return key.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toLowerCase()
}

export function normalizeKeys<T>(raw: unknown): T {
  if (Array.isArray(raw)) {
    return raw.map((item) => normalizeKeys(item)) as unknown as T
  }
  if (raw !== null && typeof raw === 'object') {
    const out: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(raw as Record<string, unknown>)) {
      out[pascalToSnake(key)] = value
    }
    return out as T
  }
  return raw as T
}

// --- "My placements" — shared by every student page that needs the company switcher ---

import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'

export interface Placement {
  id: number
  user_id: string
  company_id: number
  appliance_id: number
  start_date: string | null
  end_date: string | null
  extended_until: string | null
  status: InternDateStatus
  version: number
  created_at: string
  updated_at: string
  company_name: string
}

/**
 * Fetches the student's placements (GET /internships/mine) and best-effort
 * resolves each distinct company_id to a display name (GET /companies/:id)
 * so the company switcher can show "Acme Corp" instead of "Company #5". A
 * company lookup failure never blocks the page — it just falls back to the
 * numeric label. Both endpoints hit the same untagged-struct issue as
 * PresenceStatus above, so the raw response is run through normalizeKeys().
 */
export async function fetchMyPlacements(): Promise<Placement[]> {
  const res = await http.get<ApiSuccess<unknown[]>>('/internships/mine')
  const placements = normalizeKeys<Omit<Placement, 'company_name'>[]>(res.data.data)

  const uniqueCompanyIds = [...new Set(placements.map((p) => p.company_id))]
  const companies = await Promise.all(
    uniqueCompanyIds.map(async (id) => {
      try {
        const companyRes = await http.get<ApiSuccess<unknown>>(`/companies/${id}`)
        return normalizeKeys<{ id: number; name: string }>(companyRes.data.data)
      } catch {
        return null
      }
    }),
  )
  const nameById = new Map(companies.filter((c): c is { id: number; name: string } => c !== null).map((c) => [c.id, c.name]))

  return placements.map((p) => ({ ...p, company_name: nameById.get(p.company_id) ?? `Company #${p.company_id}` }))
}

/** Local YYYY-MM-DD for "today", matching the date-only fields this API uses. */
export function todayISODate(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

/**
 * GET /presence-statuses?school_id=X, normalized to the snake_case
 * PresenceStatus shape (see the note on that interface above).
 */
// /presence-statuses now paginates (defaults to 20); a school has at most
// one status per PresenceStatusKind (5 total), so this asks for a generous
// ceiling instead of paginating a lookup that's always small.
export async function fetchPresenceStatuses(schoolId: number): Promise<PresenceStatus[]> {
  const res = await http.get<ApiSuccess<unknown[]>>('/presence-statuses', { params: { school_id: schoolId, limit: 100 } })
  return normalizeKeys<PresenceStatus[]>(res.data.data)
}
