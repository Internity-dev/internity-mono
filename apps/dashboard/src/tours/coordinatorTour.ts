import type { DriveStep } from 'driver.js'

export const coordinatorTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'As a coordinator, you manage your school\'s internship program end to end: departments, companies, applications, attendance, scoring, and announcements.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/departments"]',
    popover: {
      title: 'Set up your org structure',
      description: 'As a coordinator, I want to manage departments, classes, and partner companies within my school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/appliances"]',
    popover: {
      title: 'Review applications',
      description: 'As a coordinator, I want to review student applications to vacancies and accept or reject them.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/presence"]',
    popover: {
      title: 'Approve attendance',
      description: 'As a coordinator, I want to review and bulk-approve student check-ins so attendance records are finalized.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/scores"]',
    popover: {
      title: 'Score & certify',
      description: 'As a coordinator, I want to enter student scores and configure the letter-grade bands used on their certificates.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/news"]',
    popover: {
      title: 'Publish announcements',
      description: 'As a coordinator, I want to publish news to my school so students and staff are notified automatically.',
    },
  },
]
