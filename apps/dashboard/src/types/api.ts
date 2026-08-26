export interface Pagination {
  page: number
  limit: number
  total: number
}

export interface ApiMeta {
  request_id?: string
  pagination?: Pagination
}

export interface ApiSuccess<T> {
  success: true
  data: T
  message: string
  meta?: ApiMeta
}

export interface ApiErrorDetail {
  field?: string
  issue: string
}

export interface ApiErrorBody {
  success: false
  data: null
  message: string
  error: {
    code: string
    details?: ApiErrorDetail[]
  }
  meta?: ApiMeta
}

export type Role = 'admin' | 'coordinator' | 'mentor' | 'student'

export interface User {
  id: string
  role: Role
  school_id?: number
  department_id?: number
  company_id?: number
  course_id?: number
  name: string
  email: string
  nis?: string
  gender?: 'male' | 'female'
  bio?: string
  address?: string
  phone?: string
  date_of_birth?: string
  avatar_key?: string
  resume_key?: string
  skills?: string
  is_active: boolean
  created_at: string
}
