import type { DriveStep } from 'driver.js'
import type { Role } from '@/types/api'
import { studentTourSteps } from './studentTour'
import { coordinatorTourSteps } from './coordinatorTour'
import { mentorTourSteps } from './mentorTour'
import { adminTourSteps } from './adminTour'

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
