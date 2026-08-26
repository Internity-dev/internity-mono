export type Tone = 'success' | 'warning' | 'danger' | 'info' | 'neutral'

interface StatusMeta {
  label: string
  tone: Tone
}

function mapper(dict: Record<string, StatusMeta>) {
  return (status: string): StatusMeta => dict[status] ?? { label: status, tone: 'neutral' }
}

export const applianceStatus = mapper({
  pending: { label: 'Pending', tone: 'warning' },
  processed: { label: 'Under review', tone: 'info' },
  accepted: { label: 'Accepted', tone: 'success' },
  rejected: { label: 'Rejected', tone: 'danger' },
  canceled: { label: 'Canceled', tone: 'neutral' },
})

export const vacancyStatus = mapper({
  open: { label: 'Open', tone: 'success' },
  closed: { label: 'Closed', tone: 'neutral' },
})

export const newsStatus = mapper({
  draft: { label: 'Draft', tone: 'warning' },
  published: { label: 'Published', tone: 'success' },
})

export const internDateStatus = mapper({
  scheduled: { label: 'Scheduled', tone: 'info' },
  completed: { label: 'Completed', tone: 'success' },
})

export function approvalStatus(isApproved: boolean): StatusMeta {
  return isApproved ? { label: 'Approved', tone: 'success' } : { label: 'Pending approval', tone: 'warning' }
}

export function activeStatus(isActive: boolean): StatusMeta {
  return isActive ? { label: 'Active', tone: 'success' } : { label: 'Inactive', tone: 'neutral' }
}

export const presenceStatusKind = mapper({
  present: { label: 'Present', tone: 'success' },
  permitted: { label: 'Permitted', tone: 'info' },
  sick: { label: 'Sick', tone: 'warning' },
  absent: { label: 'Absent', tone: 'danger' },
  holiday: { label: 'Holiday', tone: 'neutral' },
})

export const attendanceDayStatus = mapper({
  reported: { label: 'Reported', tone: 'success' },
  missing: { label: 'Missing', tone: 'danger' },
  upcoming: { label: 'Upcoming', tone: 'neutral' },
  outside_range: { label: 'Outside range', tone: 'neutral' },
})
