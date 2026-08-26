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

  it('translates known backend auth messages to Indonesian', () => {
    expect(errorMessage(fakeAxiosError({ message: 'Invalid email or password' }), 'fallback')).toBe(
      'Email atau kata sandi salah',
    )
    expect(
      errorMessage(fakeAxiosError({ message: 'An account with this email already exists' }), 'fallback'),
    ).toBe('Akun dengan email ini sudah terdaftar')
    expect(errorMessage(fakeAxiosError({ message: 'Invalid invite code' }), 'fallback')).toBe(
      'Kode undangan tidak valid',
    )
    expect(
      errorMessage(fakeAxiosError({ message: 'This reset link is invalid or has expired' }), 'fallback'),
    ).toBe('Tautan reset ini tidak valid atau sudah kedaluwarsa')
  })

  it('leaves an unrecognized backend message untranslated', () => {
    expect(errorMessage(fakeAxiosError({ message: 'Some future backend message' }), 'fallback')).toBe(
      'Some future backend message',
    )
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

  it('translates known backend issue strings to Indonesian', () => {
    const err = fakeAxiosError({
      error: {
        details: [
          { field: 'email', issue: 'already registered' },
          { field: 'invite_code', issue: 'not found' },
        ],
      },
    })
    expect(fieldErrors(err)).toEqual({ email: 'sudah terdaftar', invite_code: 'tidak ditemukan' })
  })

  it('leaves an unrecognized issue string untranslated', () => {
    const err = fakeAxiosError({ error: { details: [{ field: 'name', issue: 'Some future issue' }] } })
    expect(fieldErrors(err)).toEqual({ name: 'Some future issue' })
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
