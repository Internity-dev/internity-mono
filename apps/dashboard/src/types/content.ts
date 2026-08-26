// IMPORTANT — response field casing:
// apps/api/internal/modules/content/domain.go's News/FAQ structs carry ONLY
// `gorm:"column:..."` tags, no `json:"..."` tags. Go's encoding/json
// therefore serializes them using the literal exported field name
// (PascalCase), NOT snake_case — same backend gap documented in
// src/types/vacancy.ts and src/types/internship.ts. These interfaces are
// typed snake_case (the shape this app's views actually consume); every
// fetch goes through normalizeKeys() below to convert at the fetch boundary.
// Request bodies (POST/PUT) DO use snake_case already, per the handler's
// binding structs (createNewsRequest / updateNewsRequest / createFAQRequest
// / updateFAQRequest) — see the payload types below, unaffected.

export type NewsScopeType = 'school' | 'department'
export type NewsStatus = 'draft' | 'published'

export interface News {
  id: number
  author_id: string
  scope_type: NewsScopeType
  scope_id: number
  title: string
  slug: string
  content: string
  image_key: string | null
  status: NewsStatus
  published_at: string | null
  created_at: string
  updated_at: string
  [key: string]: unknown
}

export interface CreateNewsPayload {
  scope_type: NewsScopeType
  scope_id: number
  title: string
  content: string
  image_key?: string
  publish: boolean
}

export interface NewsPatch {
  title?: string
  content?: string
  image_key?: string
  publish?: boolean
}

export interface Faq {
  id: number
  question: string
  answer: string
  sort_order: number
  created_at: string
  updated_at: string
  [key: string]: unknown
}

export interface CreateFaqPayload {
  question: string
  answer: string
  sort_order?: number
}

export interface FaqPatch {
  question?: string
  answer?: string
  sort_order?: number
}

export interface NotificationItem {
  id: number
  user_id: string
  type: string
  title: string
  body: string
  read_at: string | null
  created_at: string
}

function pascalToSnake(key: string): string {
  return key.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toLowerCase()
}

/** Converts a PascalCase-keyed response (or array of them) to snake_case, recursively. No-op on already-snake_case keys. */
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
