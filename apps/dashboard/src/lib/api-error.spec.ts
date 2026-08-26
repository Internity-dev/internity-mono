import { describe, it, expect } from 'vitest'
import { AxiosError, AxiosHeaders } from 'axios'
import { errorMessage, fieldErrors, retryAfterSeconds } from './api-error'

function fakeAxiosError(data: unknown, headers: Record<string, string> = {}) {
  const responseHeaders = new AxiosHeaders()
  for (const [key, value] of Object.entries(headers)) responseHeaders.set(key, value)
  const response = { status: 400, data, statusText: '', headers: responseHeaders, config: {} as never }
  return new AxiosError('Request failed', '400', {} as never, {}, response as never)
}

describe('errorMessage', () => {
  it('returns the server message when present', () => {
    expect(errorMessage(fakeAxiosError({ message: 'Custom failure' }), 'fallback')).toBe('Custom failure')
  })

  it('falls back when the body has no message', () => {
    expect(errorMessage(fakeAxiosError({}), 'fallback')).toBe('fallback')
  })

  it('falls back for a non-axios error', () => {
    expect(errorMessage(new Error('boom'), 'fallback')).toBe('fallback')
  })
})

describe('fieldErrors', () => {
  it('maps error details keyed by field', () => {
    const err = fakeAxiosError({ error: { details: [{ field: 'email', issue: 'Already taken' }] } })
    expect(fieldErrors(err)).toEqual({ email: 'Already taken' })
  })

  it('returns an empty object when there are no details', () => {
    expect(fieldErrors(fakeAxiosError({}))).toEqual({})
  })
})

describe('retryAfterSeconds', () => {
  it('reads a numeric Retry-After header', () => {
    expect(retryAfterSeconds(fakeAxiosError({}, { 'retry-after': '42' }))).toBe(42)
  })

  it('returns null when the header is missing', () => {
    expect(retryAfterSeconds(fakeAxiosError({}))).toBeNull()
  })

  it('returns null for a non-positive or non-numeric header', () => {
    expect(retryAfterSeconds(fakeAxiosError({}, { 'retry-after': '0' }))).toBeNull()
    expect(retryAfterSeconds(fakeAxiosError({}, { 'retry-after': 'soon' }))).toBeNull()
  })
})
