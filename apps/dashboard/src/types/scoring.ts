export type ScoreType = 'teknis' | 'non-teknis'

export interface Score {
  id: number
  user_id: string
  company_id: number
  name: string
  score: number
  type: ScoreType
  created_at: string
  updated_at: string
}

export interface CreateScorePayload {
  user_id: string
  company_id: number
  name: string
  score: number
  type: ScoreType
}

export interface UpdateScorePayload {
  name?: string
  score?: number
  type?: ScoreType
}

// --- Score predicates (per-school letter-grade bands) ---

export interface ScorePredicate {
  id: number
  school_id: number
  name: string
  description: string | null
  color: string | null
  min: number
  max: number
  created_at: string
  updated_at: string
  [key: string]: unknown
}

export interface CreateScorePredicatePayload {
  school_id: number
  name: string
  description?: string
  color?: string
  min: number
  max: number
}

export interface ScorePredicatePatch {
  name?: string
  description?: string
  color?: string
  min?: number
  max?: number
}

function pascalToSnake(key: string): string {
  return key.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toLowerCase()
}

/** Converts a PascalCase-keyed response (or array of them) to snake_case, recursively. No-op on already-snake_case keys — kept as a defensive no-op now that the backend serializes snake_case directly. */
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
