import type { DriveStep } from 'driver.js'

// Each step is written as a user story, per the product decision to frame
// onboarding around "as a <role>, I want <goal> so <reason>" rather than a
// dry feature list — see plan's "Onboarding tours (driver.js)" section.
export const studentTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'As a student, you\'ll use this dashboard to find an internship, track your attendance, log your daily work, and download your certificate. Quick tour?',
    },
  },
  {
    element: '[data-tour-nav-target="/vacancies"]',
    popover: {
      title: 'Find a placement',
      description: 'As a student, I want to browse open internship vacancies at companies in my department so I can apply to one that fits me.',
    },
  },
  {
    element: '[data-tour-nav-target="/my-applications"]',
    popover: {
      title: 'Track your applications',
      description: 'As a student, I want to see the status of every application I\'ve submitted: pending, under review, accepted, or rejected.',
    },
  },
  {
    element: '[data-tour-nav-target="/my-internship"]',
    popover: {
      title: 'Set your placement dates',
      description: 'As a student, once accepted, I want to set my internship start and end dates so my attendance and journal periods are defined.',
    },
  },
  {
    element: '[data-tour-nav-target="/attendance"]',
    popover: {
      title: 'Check in every day',
      description: 'As a student, I want to check in/out with a photo and my location each workday, or file an excuse if I\'m sick or on leave.',
    },
  },
  {
    element: '[data-tour-nav-target="/journals"]',
    popover: {
      title: 'Log your work',
      description: 'As a student, I want to write a short journal entry for each day I attend, so my mentor can see and approve what I worked on.',
    },
  },
  {
    element: '[data-tour-nav-target="/certificate"]',
    popover: {
      title: 'Get your certificate',
      description: 'As a student, once my internship is complete and scored, I want to download my official completion certificate here.',
    },
  },
  {
    element: '[data-tour="notifications"]',
    popover: {
      title: 'Stay in the loop',
      description: 'Application updates, approvals, and school announcements all show up here.',
    },
  },
]
