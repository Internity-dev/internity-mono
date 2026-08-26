import type { DriveStep } from 'driver.js'

// Every step targets a real sidebar item (via data-tour-nav-target) or the
// header bell, so the tour stays in sync with nav.ts by construction — add a
// menu there and its tour step is the only other thing to add here.
export const studentTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'This dashboard covers your whole internship: finding a placement, checking in each day, logging your work, and getting your certificate at the end. Quick tour of every menu?',
    },
  },
  {
    element: '[data-tour-nav-target="/vacancies"]',
    popover: {
      title: 'Vacancies',
      description:
        'Browse every open internship listing from companies in your department. Search by company or skill, and check required skills, open slots, and the deadline before you apply.',
    },
  },
  {
    element: '[data-tour-nav-target="/my-applications"]',
    popover: {
      title: 'My Applications',
      description:
        'Every vacancy you\'ve applied to, with its status: pending, accepted, or rejected. Your application note is here too, so you can see exactly what you sent.',
    },
  },
  {
    element: '[data-tour-nav-target="/my-internship"]',
    popover: {
      title: 'My Internship',
      description:
        'Once a company accepts you, this becomes your placement record: which company, who your mentor is, and the start/end dates you set — those dates define your attendance and journal periods.',
    },
  },
  {
    element: '[data-tour-nav-target="/attendance"]',
    popover: {
      title: 'Attendance',
      description:
        'Check in and out each workday with a photo and your location, or file an excuse (sick, permit, leave) on a day you can\'t make it. Your mentor approves every entry.',
    },
  },
  {
    element: '[data-tour-nav-target="/journals"]',
    popover: {
      title: 'Journal',
      description:
        'Write a short entry for each day you attend, describing what you worked on. Your mentor reads and approves these, and they become part of your final record.',
    },
  },
  {
    element: '[data-tour-nav-target="/certificate"]',
    popover: {
      title: 'Certificate',
      description:
        'Once your internship ends and your mentor has entered your scores, your official completion certificate becomes downloadable here.',
    },
  },
  {
    element: '[data-tour-nav-target="/news"]',
    popover: {
      title: 'News',
      description: 'Announcements from your school and coordinator — new vacancies, deadline reminders, and program updates.',
    },
  },
  {
    element: '[data-tour-nav-target="/faq"]',
    popover: {
      title: 'FAQ',
      description: 'Common questions about applying, attendance rules, and how certification works, answered before you have to ask.',
    },
  },
  {
    element: '[data-tour="notifications"]',
    popover: {
      title: 'Notifications',
      description: 'Application status changes, approvals, and announcements all land here first.',
    },
  },
  {
    element: '[data-tour-nav-target="/profile"]',
    popover: {
      title: 'Profile',
      description: 'Update your name, avatar, and password, and check your account details.',
    },
  },
]
