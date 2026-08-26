import type { DriveStep } from 'driver.js'
import { menuStep } from './menuInfo'

// Short critical-path tour (per the onboard skill: 3-7 steps max). Every
// other menu (Vacancies, News, FAQ, Notifications, Profile) gets a one-time
// spotlight the first time the mentor actually visits it — see useMenuHints.ts.
export const mentorCoreHintPaths = ['/admin/appliances', '/admin/presence', '/admin/journals', '/admin/scores']

export const mentorTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'As a company mentor, you supervise the interns placed with you: reviewing applicants, approving their attendance and journals, and scoring their performance.',
    },
  },
  ...mentorCoreHintPaths.map(menuStep),
]
