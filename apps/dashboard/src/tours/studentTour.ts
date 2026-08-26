import type { DriveStep } from 'driver.js'
import { menuStep } from './menuInfo'

// Short critical-path tour (per the onboard skill: 3-7 steps max). Every
// other menu (News, FAQ, Notifications, Profile) gets a one-time spotlight
// the first time the student actually visits it — see useMenuHints.ts.
export const studentCoreHintPaths = ['/vacancies', '/my-applications', '/my-internship', '/attendance', '/journals', '/certificate']

export const studentTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'This dashboard covers your whole internship: finding a placement, checking in each day, logging your work, and getting your certificate at the end. Quick tour?',
    },
  },
  ...studentCoreHintPaths.map(menuStep),
]
