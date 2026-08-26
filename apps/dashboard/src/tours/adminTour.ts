import type { DriveStep } from 'driver.js'
import { menuStep } from './menuInfo'

// Short critical-path tour (per the onboard skill: 3-7 steps max). Every
// other menu gets a one-time spotlight the first time the admin actually
// visits it — see useMenuHints.ts.
export const adminCoreHintPaths = ['/admin/schools', '/admin/users', '/admin/appliances', '/admin/reports']

export const adminTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description: 'As the platform admin, everything here spans every school, not just one.',
    },
  },
  ...adminCoreHintPaths.map(menuStep),
]
