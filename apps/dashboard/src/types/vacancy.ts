export type VacancyStatus = 'open' | 'closed'

export interface Vacancy {
  id: number
  company_id: number
  name: string
  category: string | null
  description: string | null
  skills: string | null
  slots: number
  status: VacancyStatus
  created_at: string
  updated_at: string
}

export interface CreateVacancyPayload {
  company_id: number
  name: string
  category?: string
  description?: string
  skills?: string
  slots?: number
}

export interface UpdateVacancyPayload {
  name?: string
  category?: string
  description?: string
  skills?: string
  slots?: number
  status?: VacancyStatus
}

export type ApplianceStatus = 'pending' | 'processed' | 'accepted' | 'rejected' | 'canceled'

export interface Appliance {
  id: number
  user_id: string
  vacancy_id: number
  status: ApplianceStatus
  message: string | null
  created_at: string
  updated_at: string
}

// --- Internship placements (apps/api/internal/modules/internship) ---

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

// Request body for PUT /internships/:id/dates
export interface SetInternDatesPayload {
  start_date: string
  end_date: string
  extended_until?: string
  expected_version: number
}

export type AttendanceDayStatus = 'reported' | 'missing' | 'upcoming' | 'outside_range'

export interface AttendanceDay {
  date: string
  status: AttendanceDayStatus
  presence?: Record<string, unknown> | null
}

// --- Companies (apps/api/internal/modules/orgs) ---

export interface Company {
  id: number
  department_id: number
  name: string
  category: string | null
  city: string | null
  state: string | null
  country: string | null
  address: string | null
  email: string | null
  phone: string | null
  website: string | null
  logo_key: string | null
  contact_person: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}
