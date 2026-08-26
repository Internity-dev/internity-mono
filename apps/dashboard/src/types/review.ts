// IMPORTANT — response field casing:
// apps/api/internal/modules/review/domain.go's Monitor/Question/Review
// structs carry ONLY `gorm:"column:..."` tags, no `json:"..."` tags. Go's
// encoding/json therefore serializes them using the literal exported field
// name (PascalCase), NOT snake_case — same backend gap documented in
// src/types/vacancy.ts and src/types/internship.ts. These interfaces are
// typed snake_case (the shape this app's views actually consume); every
// fetch goes through normalizeKeys() below to convert at the fetch boundary.
// Request bodies (POST/PUT) DO use snake_case already, per the handler's
// binding structs (createMonitorRequest / createQuestionRequest /
// updateQuestionRequest) — see the payload types below, unaffected.

export interface Monitor {
  id: number
  coordinator_id: string
  student_id: string
  company_id: number
  date: string
  attachment_key: string | null
  notes: string | null
  suggest: string | null
  match_rating: number
  created_at: string
  updated_at: string
  [key: string]: unknown
}

export interface CreateMonitorPayload {
  student_id: string
  company_id: number
  date: string
  attachment_key?: string
  notes?: string
  suggest?: string
  match_rating: number
}

export interface Question {
  id: number
  school_id: number
  question: string
  sort_order: number
  created_at: string
  updated_at: string
  [key: string]: unknown
}

export interface CreateQuestionPayload {
  school_id: number
  question: string
  sort_order?: number
}

export interface QuestionPatch {
  question?: string
  sort_order?: number
}

export interface Review {
  id: number
  reviewer_id: string
  question_id: number | null
  reviewee_user_id: string | null
  reviewee_company_id: number | null
  title: string | null
  body: string | null
  rating: number
  created_at: string
  updated_at: string
}

export type CreateReviewPayload =
  | { reviewee_type: 'user'; reviewee_user_id: string; title?: string; body?: string; rating: number }
  | { reviewee_type: 'company'; reviewee_company_id: number; title?: string; body?: string; rating: number }

// The wrapping object comes from a literal `gin.H{"reviews": ..., "average_rating": ...}`
// map in the handler, so these two outer keys really are snake_case on the
// wire already — only the nested Review items need normalizeKeys().
export interface CompanyReviews {
  reviews: Review[]
  average_rating: number
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
