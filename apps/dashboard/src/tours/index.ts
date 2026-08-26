import type { DriveStep } from 'driver.js'
import type { Role } from '@/types/api'
import { studentTourSteps, studentCoreHintPaths } from './studentTour'
import { coordinatorTourSteps, coordinatorCoreHintPaths } from './coordinatorTour'
import { mentorTourSteps, mentorCoreHintPaths } from './mentorTour'
import { adminTourSteps, adminCoreHintPaths } from './adminTour'

export function tourStepsForRole(role: Role): DriveStep[] {
  switch (role) {
    case 'student':
      return studentTourSteps
    case 'coordinator':
      return coordinatorTourSteps
    case 'mentor':
      return mentorTourSteps
    case 'admin':
      return adminTourSteps
  }
}

// The paths a role's core tour already spotlights — passed to
// markHintsSeenFor() so useMenuHints.ts doesn't immediately re-explain a
// menu the tour just covered.
export function coreHintPathsForRole(role: Role): string[] {
  switch (role) {
    case 'student':
      return studentCoreHintPaths
    case 'coordinator':
      return coordinatorCoreHintPaths
    case 'mentor':
      return mentorCoreHintPaths
    case 'admin':
      return adminCoreHintPaths
  }
}
