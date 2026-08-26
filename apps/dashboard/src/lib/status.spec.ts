import { describe, it, expect } from 'vitest'
import {
  applianceStatus,
  vacancyStatus,
  newsStatus,
  internDateStatus,
  approvalStatus,
  activeStatus,
  presenceStatusKind,
  attendanceDayStatus,
} from './status'

describe('applianceStatus', () => {
  it('maps every known status to its label and tone', () => {
    expect(applianceStatus('pending')).toEqual({ label: 'Pending', tone: 'warning' })
    expect(applianceStatus('processed')).toEqual({ label: 'Under review', tone: 'info' })
    expect(applianceStatus('accepted')).toEqual({ label: 'Accepted', tone: 'success' })
    expect(applianceStatus('rejected')).toEqual({ label: 'Rejected', tone: 'danger' })
    expect(applianceStatus('canceled')).toEqual({ label: 'Canceled', tone: 'neutral' })
  })

  it('falls back to the raw status as the label with a neutral tone when unknown', () => {
    expect(applianceStatus('some_unknown_status')).toEqual({ label: 'some_unknown_status', tone: 'neutral' })
  })
})

describe('vacancyStatus', () => {
  it('maps known statuses', () => {
    expect(vacancyStatus('open')).toEqual({ label: 'Open', tone: 'success' })
    expect(vacancyStatus('closed')).toEqual({ label: 'Closed', tone: 'neutral' })
  })

  it('falls back for unknown statuses', () => {
    expect(vacancyStatus('archived')).toEqual({ label: 'archived', tone: 'neutral' })
  })
})

describe('newsStatus', () => {
  it('maps known statuses', () => {
    expect(newsStatus('draft')).toEqual({ label: 'Draft', tone: 'warning' })
    expect(newsStatus('published')).toEqual({ label: 'Published', tone: 'success' })
  })

  it('falls back for unknown statuses', () => {
    expect(newsStatus('')).toEqual({ label: '', tone: 'neutral' })
  })
})

describe('internDateStatus', () => {
  it('maps known statuses', () => {
    expect(internDateStatus('scheduled')).toEqual({ label: 'Scheduled', tone: 'info' })
    expect(internDateStatus('completed')).toEqual({ label: 'Completed', tone: 'success' })
  })

  it('falls back for unknown statuses', () => {
    expect(internDateStatus('missed')).toEqual({ label: 'missed', tone: 'neutral' })
  })
})

describe('presenceStatusKind', () => {
  it('maps every known presence kind', () => {
    expect(presenceStatusKind('present')).toEqual({ label: 'Present', tone: 'success' })
    expect(presenceStatusKind('permitted')).toEqual({ label: 'Permitted', tone: 'info' })
    expect(presenceStatusKind('sick')).toEqual({ label: 'Sick', tone: 'warning' })
    expect(presenceStatusKind('absent')).toEqual({ label: 'Absent', tone: 'danger' })
    expect(presenceStatusKind('holiday')).toEqual({ label: 'Holiday', tone: 'neutral' })
  })
})

describe('attendanceDayStatus', () => {
  it('maps every known attendance-day status', () => {
    expect(attendanceDayStatus('reported')).toEqual({ label: 'Reported', tone: 'success' })
    expect(attendanceDayStatus('missing')).toEqual({ label: 'Missing', tone: 'danger' })
    expect(attendanceDayStatus('upcoming')).toEqual({ label: 'Upcoming', tone: 'neutral' })
    expect(attendanceDayStatus('outside_range')).toEqual({ label: 'Outside range', tone: 'neutral' })
  })
})

describe('approvalStatus', () => {
  it('returns Approved/success when true', () => {
    expect(approvalStatus(true)).toEqual({ label: 'Approved', tone: 'success' })
  })

  it('returns Pending approval/warning when false', () => {
    expect(approvalStatus(false)).toEqual({ label: 'Pending approval', tone: 'warning' })
  })
})

describe('activeStatus', () => {
  it('returns Active/success when true', () => {
    expect(activeStatus(true)).toEqual({ label: 'Active', tone: 'success' })
  })

  it('returns Inactive/neutral when false', () => {
    expect(activeStatus(false)).toEqual({ label: 'Inactive', tone: 'neutral' })
  })
})
