import type { DriveStep } from 'driver.js'
import { menuStep } from './menuInfo'

// Short critical-path tour (per the onboard skill: 3-7 steps max) covering
// the core loop: set up org structure, review applications, approve
// attendance, score, announce. Every other menu gets a one-time spotlight
// the first time the coordinator actually visits it — see useMenuHints.ts.
export const coordinatorCoreHintPaths = ['/admin/departments', '/admin/appliances', '/admin/presence', '/admin/scores', '/admin/news']

export const coordinatorTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'As a coordinator, you manage your school\'s internship program end to end: departments, companies, applications, attendance, scoring, and announcements.',
    },
  },
  ...coordinatorCoreHintPaths.map(menuStep),
]
