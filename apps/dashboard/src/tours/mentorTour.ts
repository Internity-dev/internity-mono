import type { DriveStep } from 'driver.js'

export const mentorTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'As a company mentor, you supervise interns placed at your company: reviewing applications, approving attendance and journals, and scoring their performance.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/appliances"]',
    popover: {
      title: 'Review applicants',
      description: 'As a mentor, I want to review students who applied to my company\'s vacancies and accept the ones I want to take on.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/presence"]',
    popover: {
      title: 'Approve attendance',
      description: 'As a mentor, I want to approve my interns\' daily check-ins once they\'ve completed a full day.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/journals"]',
    popover: {
      title: 'Review journals',
      description: 'As a mentor, I want to read and approve my interns\' daily work journals.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/scores"]',
    popover: {
      title: 'Score your interns',
      description: 'As a mentor, I want to enter technical and non-technical scores for each intern before their internship ends.',
    },
  },
]
