import { isAxiosError } from 'axios'
import type { ApiErrorBody } from '@/types/api'

// The auth flow (login/register/forgot-password/reset-password) is entirely
// Indonesian, but apps/api/internal/modules/identity/service.go returns a
// small, fixed set of English strings for its known failure cases. Showing
// those verbatim breaks the Indonesian-only experience right when a
// first-time user most needs clarity. This is not a general i18n system —
// just coverage for the exact strings that endpoint returns today. Anything
// not listed here falls through untranslated.
const MESSAGE_TRANSLATIONS: Record<string, string> = {
  'Invalid email or password': 'Email atau kata sandi salah',
  'An account with this email already exists': 'Akun dengan email ini sudah terdaftar',
  'Invalid invite code': 'Kode undangan tidak valid',
  'This invite code has expired': 'Kode undangan ini sudah kedaluwarsa',
  'Password confirmation does not match': 'Kata sandi tidak cocok',
  'This reset link is invalid or has expired': 'Tautan reset ini tidak valid atau sudah kedaluwarsa',
}

const ISSUE_TRANSLATIONS: Record<string, string> = {
  'already registered': 'sudah terdaftar',
  'not found': 'tidak ditemukan',
  expired: 'sudah kedaluwarsa',
  'must match password': 'Kata sandi tidak cocok',
  'invalid or expired': 'tidak valid atau sudah kedaluwarsa',
}

export function errorMessage(err: unknown, fallback: string): string {
  if (isAxiosError(err)) {
    const body = err.response?.data as ApiErrorBody | undefined
    if (body?.message) return MESSAGE_TRANSLATIONS[body.message] ?? body.message
  }
  return fallback
}

export function fieldErrors(err: unknown): Record<string, string> {
  if (isAxiosError(err)) {
    const details = (err.response?.data as ApiErrorBody | undefined)?.error?.details
    if (details?.length) {
      const map: Record<string, string> = {}
      for (const d of details) if (d.field) map[d.field] = ISSUE_TRANSLATIONS[d.issue] ?? d.issue
      return map
    }
  }
  return {}
}

export function retryAfterSeconds(err: unknown): number | null {
  if (isAxiosError(err)) {
    const header = err.response?.headers?.['retry-after']
    const seconds = Number(header)
    if (Number.isFinite(seconds) && seconds > 0) return seconds
  }
  return null
}
