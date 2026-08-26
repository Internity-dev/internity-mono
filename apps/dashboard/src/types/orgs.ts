// Domain types for the School -> Department -> Course/Company hierarchy
// (internal/modules/orgs) plus identity's InviteCode. The backend serializes
// snake_case consistently (see apps/api's domain.go json tags) — these types
// match the real wire format.

export interface School {
  id: number
  name: string
  email: string | null
  phone: string | null
  address: string | null
  website: string | null
  logo_key: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface SchoolInput {
  name: string
  email?: string
  phone?: string
  address?: string
  website?: string
}

export type SchoolPatchInput = Partial<SchoolInput>

export interface Department {
  id: number
  school_id: number
  name: string
  description: string | null
  study_program: string | null
  logo_key: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface DepartmentInput {
  school_id: number
  name: string
  description?: string
  study_program?: string
}

export interface DepartmentPatchInput {
  name?: string
  description?: string
  study_program?: string
  is_active?: boolean
}

export interface Course {
  id: number
  department_id: number
  name: string
  description: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CourseInput {
  department_id: number
  name: string
  description?: string
}

export interface CoursePatchInput {
  name?: string
  description?: string
  is_active?: boolean
}

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

export interface CompanyInput {
  department_id: number
  name: string
  category?: string
  city?: string
  state?: string
  country?: string
  address?: string
  email?: string
  phone?: string
  website?: string
  contact_person?: string
}

export interface CompanyPatchInput {
  name?: string
  category?: string
  city?: string
  state?: string
  country?: string
  address?: string
  email?: string
  phone?: string
  website?: string
  contact_person?: string
  is_active?: boolean
}

// Identity module — not really an "org" type, but the only closely-related
// domain object UsersView.vue can act on (see that file for why there's no
// user listing here).
export interface InviteCode {
  id: number
  code: string
  course_id: number
  expires_at: string | null
  created_at: string
}

export interface InviteCodeInput {
  course_id: number
  expires_at?: string
}
