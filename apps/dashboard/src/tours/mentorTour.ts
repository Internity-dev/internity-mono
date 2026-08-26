import type { DriveStep } from 'driver.js'

// Every step targets a real sidebar item (via data-tour-nav-target) or the
// header bell, so the tour stays in sync with nav.ts by construction — add a
// menu there and its tour step is the only other thing to add here.
export const mentorTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'As a company mentor, you supervise the interns placed with you: reviewing applicants, approving their attendance and journals, and scoring their performance. Here\'s what every menu does.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/vacancies"]',
    popover: {
      title: 'Vacancies',
      description: 'See and edit the internship openings posted for your company — required skills, open slots, and which department can apply.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/appliances"]',
    popover: {
      title: 'Applications',
      description: 'Review students who applied to your company\'s vacancies and accept the ones you\'re taking on as interns.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/presence"]',
    popover: {
      title: 'Attendance review',
      description: 'Approve your interns\' daily check-ins (photo + location) once they\'ve completed a full day, or approve their filed excuses.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/journals"]',
    popover: {
      title: 'Journal review',
      description: 'Read and approve your interns\' daily work journals.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/scores"]',
    popover: {
      title: 'Scores',
      description: 'Enter technical and non-technical scores for each intern before their placement ends — these feed their final certificate.',
    },
  },
  {
    element: '[data-tour-nav-target="/news"]',
    popover: {
      title: 'News',
      description: 'Announcements from the schools you host interns from.',
    },
  },
  {
    element: '[data-tour-nav-target="/faq"]',
    popover: {
      title: 'FAQ',
      description: 'Common questions about hosting and scoring interns, answered here.',
    },
  },
  {
    element: '[data-tour="notifications"]',
    popover: {
      title: 'Notifications',
      description: 'New applications, journal entries waiting for review, and attendance to approve all land here.',
    },
  },
  {
    element: '[data-tour-nav-target="/profile"]',
    popover: {
      title: 'Profile',
      description: 'Update your name, avatar, and password.',
    },
  },
]
