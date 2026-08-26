import { describe, it, expect } from 'vitest'
import { navSectionsForRole } from './nav'
import type { Role } from '@/types/api'

function labelsOf(sections: ReturnType<typeof navSectionsForRole>) {
  return sections.map((s) => s.label)
}

function itemLabels(sections: ReturnType<typeof navSectionsForRole>, sectionLabel: string | undefined) {
  const section = sections.find((s) => s.label === sectionLabel)
  return section ? section.items.map((i) => i.label) : undefined
}

describe('navSectionsForRole', () => {
  it('gives students only the Dashboard (with their own items) and General sections', () => {
    const sections = navSectionsForRole('student')
    expect(labelsOf(sections)).toEqual([undefined, 'General'])

    const dashboardItems = itemLabels(sections, undefined)
    expect(dashboardItems).toEqual([
      'Dashboard',
      'Vacancies',
      'My Applications',
      'My Internship',
      'Attendance',
      'Journal',
      'Certificate',
    ])
    // Dashboard must appear exactly once, not duplicated by the student-specific entry.
    expect(dashboardItems?.filter((l) => l === 'Dashboard')).toHaveLength(1)

    expect(itemLabels(sections, 'General')).toEqual(['News', 'FAQ', 'Notifications', 'Profile'])
  })

  it('drops Organization and Content for mentors (no items match their role) but keeps Operations', () => {
    const sections = navSectionsForRole('mentor')
    expect(labelsOf(sections)).toEqual([undefined, 'Operations', 'General'])

    expect(itemLabels(sections, undefined)).toEqual(['Dashboard'])
    expect(itemLabels(sections, 'Operations')).toEqual([
      'Vacancies',
      'Applications',
      'Attendance review',
      'Journal review',
      'Scores',
    ])
    expect(itemLabels(sections, 'General')).toEqual(['News', 'FAQ', 'Notifications', 'Profile'])
  })

  it('gives coordinators every admin-ish section (all org/ops/content items include coordinator)', () => {
    const sections = navSectionsForRole('coordinator')
    expect(labelsOf(sections)).toEqual([undefined, 'Organization', 'Operations', 'Content', 'General'])

    expect(itemLabels(sections, 'Organization')).toEqual(['Departments', 'Courses', 'Companies', 'Users'])
    expect(itemLabels(sections, 'Operations')).toHaveLength(8)
    expect(itemLabels(sections, 'Content')).toEqual(['Manage news', 'Manage FAQ', 'Presence statuses', 'Score predicates'])
  })

  it('gives admins the full set including school-only items', () => {
    const sections = navSectionsForRole('admin')
    expect(labelsOf(sections)).toEqual([undefined, 'Organization', 'Operations', 'Content', 'General'])

    expect(itemLabels(sections, undefined)).toEqual(['Dashboard'])
    expect(itemLabels(sections, 'Organization')).toEqual(['Schools', 'Departments', 'Courses', 'Companies', 'Users'])
    expect(itemLabels(sections, 'Operations')).toHaveLength(8)
    expect(itemLabels(sections, 'Content')).toHaveLength(4)
  })

  it('never returns a section with zero items', () => {
    const roles: Role[] = ['admin', 'coordinator', 'mentor', 'student']
    for (const role of roles) {
      for (const section of navSectionsForRole(role)) {
        expect(section.items.length).toBeGreaterThan(0)
      }
    }
  })

  it('scopes every returned item to routes the role is actually allowed on', () => {
    const roles: Role[] = ['admin', 'coordinator', 'mentor', 'student']
    for (const role of roles) {
      for (const section of navSectionsForRole(role)) {
        for (const item of section.items) {
          expect(item.roles).toContain(role)
        }
      }
    }
  })

  it('is a pure function: calling it twice for the same role yields equal results', () => {
    const first = navSectionsForRole('admin')
    const second = navSectionsForRole('admin')
    expect(second).toEqual(first)
  })
})
