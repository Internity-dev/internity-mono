import type { DriveStep } from 'driver.js'

export const adminTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'As the platform admin, you onboard schools and oversee every school\'s program from one place.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/schools"]',
    popover: {
      title: 'Onboard schools',
      description: 'As an admin, I want to add new schools onto the platform so their coordinators can start managing their programs.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/users"]',
    popover: {
      title: 'Manage access',
      description: 'As an admin, I want to see and manage the accounts using the platform, and issue invite codes for new students.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/reports"]',
    popover: {
      title: 'Export reports',
      description: 'As an admin, I want to export student rosters and attendance reports for record-keeping.',
    },
  },
]
