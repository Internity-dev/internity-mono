import type { Component } from 'vue'
import {
  LayoutDashboardIcon,
  BriefcaseIcon,
  ClipboardListIcon,
  MapPinnedIcon,
  CalendarCheckIcon,
  BookOpenIcon,
  AwardIcon,
  NewspaperIcon,
  HelpCircleIcon,
  SchoolIcon,
  Building2Icon,
  UsersIcon,
  GraduationCapIcon,
  ClipboardCheckIcon,
  StarIcon,
  FileBarChartIcon,
  BellIcon,
  UserIcon,
  ListChecksIcon,
} from '@lucide/vue'
import type { Role } from '@/types/api'

export interface NavItem {
  label: string
  to: string
  icon: Component
  roles: Role[]
}

export interface NavSection {
  label?: string
  items: NavItem[]
}

const studentNav: NavItem[] = [
  { label: 'Dashboard', to: '/dashboard', icon: LayoutDashboardIcon, roles: ['student'] },
  { label: 'Vacancies', to: '/vacancies', icon: BriefcaseIcon, roles: ['student'] },
  { label: 'My Applications', to: '/my-applications', icon: ClipboardListIcon, roles: ['student'] },
  { label: 'My Internship', to: '/my-internship', icon: MapPinnedIcon, roles: ['student'] },
  { label: 'Attendance', to: '/attendance', icon: CalendarCheckIcon, roles: ['student'] },
  { label: 'Journal', to: '/journals', icon: BookOpenIcon, roles: ['student'] },
  { label: 'Certificate', to: '/certificate', icon: AwardIcon, roles: ['student'] },
]

const sharedNav: NavItem[] = [
  { label: 'News', to: '/news', icon: NewspaperIcon, roles: ['admin', 'coordinator', 'mentor', 'student'] },
  { label: 'FAQ', to: '/faq', icon: HelpCircleIcon, roles: ['admin', 'coordinator', 'mentor', 'student'] },
]

const orgNav: NavItem[] = [
  { label: 'Schools', to: '/admin/schools', icon: SchoolIcon, roles: ['admin'] },
  { label: 'Departments', to: '/admin/departments', icon: Building2Icon, roles: ['admin', 'coordinator'] },
  { label: 'Courses', to: '/admin/courses', icon: GraduationCapIcon, roles: ['admin', 'coordinator'] },
  { label: 'Companies', to: '/admin/companies', icon: Building2Icon, roles: ['admin', 'coordinator'] },
  { label: 'Users', to: '/admin/users', icon: UsersIcon, roles: ['admin', 'coordinator'] },
]

const operationsNav: NavItem[] = [
  { label: 'Vacancies', to: '/admin/vacancies', icon: BriefcaseIcon, roles: ['admin', 'coordinator', 'mentor'] },
  { label: 'Applications', to: '/admin/appliances', icon: ClipboardListIcon, roles: ['admin', 'coordinator', 'mentor'] },
  { label: 'Attendance review', to: '/admin/presence', icon: ClipboardCheckIcon, roles: ['admin', 'coordinator', 'mentor'] },
  { label: 'Journal review', to: '/admin/journals', icon: BookOpenIcon, roles: ['admin', 'coordinator', 'mentor'] },
  { label: 'Scores', to: '/admin/scores', icon: ListChecksIcon, roles: ['admin', 'coordinator', 'mentor'] },
  { label: 'Monitoring visits', to: '/admin/monitors', icon: MapPinnedIcon, roles: ['admin', 'coordinator'] },
  { label: 'Reviews & questions', to: '/admin/questions', icon: StarIcon, roles: ['admin', 'coordinator'] },
  { label: 'Reports', to: '/admin/reports', icon: FileBarChartIcon, roles: ['admin', 'coordinator'] },
]

const contentNav: NavItem[] = [
  { label: 'Manage news', to: '/admin/news', icon: NewspaperIcon, roles: ['admin', 'coordinator'] },
  { label: 'Manage FAQ', to: '/admin/faqs', icon: HelpCircleIcon, roles: ['admin', 'coordinator'] },
  { label: 'Presence statuses', to: '/admin/presence-statuses', icon: CalendarCheckIcon, roles: ['admin', 'coordinator'] },
  { label: 'Score predicates', to: '/admin/score-predicates', icon: AwardIcon, roles: ['admin', 'coordinator'] },
]

const accountNav: NavItem[] = [
  { label: 'Notifications', to: '/notifications', icon: BellIcon, roles: ['admin', 'coordinator', 'mentor', 'student'] },
  { label: 'Profile', to: '/profile', icon: UserIcon, roles: ['admin', 'coordinator', 'mentor', 'student'] },
]

export function navSectionsForRole(role: Role): NavSection[] {
  const filter = (items: NavItem[]) => items.filter((i) => i.roles.includes(role))
  const sections: NavSection[] = [
    { items: filter([{ label: 'Dashboard', to: '/dashboard', icon: LayoutDashboardIcon, roles: ['admin', 'coordinator', 'mentor', 'student'] }, ...studentNav.slice(1)]) },
    { label: 'Organization', items: filter(orgNav) },
    { label: 'Operations', items: filter(operationsNav) },
    { label: 'Content', items: filter(contentNav) },
    { label: 'General', items: filter([...sharedNav, ...accountNav]) },
  ]
  return sections.filter((s) => s.items.length > 0)
}
