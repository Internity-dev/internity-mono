import { isAxiosError } from 'axios'
import type { ApiErrorBody } from '@/types/api'

export function errorMessage(err: unknown, fallback: string): string {
  if (isAxiosError(err)) {
    const body = err.response?.data as ApiErrorBody | undefined
    if (body?.message) return body.message
  }
  return fallback
}

export function fieldErrors(err: unknown): Record<string, string> {
  if (isAxiosError(err)) {
    const details = (err.response?.data as ApiErrorBody | undefined)?.error?.details
    if (details?.length) {
      const map: Record<string, string> = {}
      for (const d of details) if (d.field) map[d.field] = d.issue
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
