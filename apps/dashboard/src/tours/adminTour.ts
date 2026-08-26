import type { DriveStep } from 'driver.js'

// Every step targets a real sidebar item (via data-tour-nav-target) or the
// header bell, so the tour stays in sync with nav.ts by construction — add a
// menu there and its tour step is the only other thing to add here.
export const adminTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'As the platform admin, everything here spans every school, not just one. Here\'s what every menu does.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/schools"]',
    popover: {
      title: 'Schools',
      description: 'Onboard new schools onto the platform — each one gets its own coordinators, departments, and student roster, isolated from every other school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/departments"]',
    popover: {
      title: 'Departments',
      description: 'Manage academic departments (jurusan) across every school, or scope to one — the same tool a school\'s own coordinator uses.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/courses"]',
    popover: {
      title: 'Courses',
      description: 'Manage class groups (rombel) across every school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/companies"]',
    popover: {
      title: 'Companies',
      description: 'Manage every partner company on the platform, across every school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/users"]',
    popover: {
      title: 'Users',
      description: 'Every account on the platform — students, mentors, coordinators, other admins — across every school. Edit roles, reset access, issue invite codes.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/vacancies"]',
    popover: {
      title: 'Vacancies',
      description: 'See and edit every internship opening posted across every school and company.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/appliances"]',
    popover: {
      title: 'Applications',
      description: 'Review student applications platform-wide, filterable by school, department, company, or vacancy.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/presence"]',
    popover: {
      title: 'Attendance review',
      description: 'Approve or audit student check-ins and filed excuses across every school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/journals"]',
    popover: {
      title: 'Journal review',
      description: 'Read student work journals across every school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/scores"]',
    popover: {
      title: 'Scores',
      description: 'Review scores entered by mentors and coordinators across every school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/monitors"]',
    popover: {
      title: 'Monitoring visits',
      description: 'See the monitoring visit logs coordinators have filed for their companies, across every school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/questions"]',
    popover: {
      title: 'Reviews & questions',
      description: 'Manage the review questionnaire template used platform-wide.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/reports"]',
    popover: {
      title: 'Reports',
      description: 'Export student rosters and attendance reports for any school on the platform.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/news"]',
    popover: {
      title: 'Manage news',
      description: 'Publish platform-wide announcements, or scope one to a single school.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/faqs"]',
    popover: {
      title: 'Manage FAQ',
      description: 'Edit the FAQ entries shown to every school\'s students.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/presence-statuses"]',
    popover: {
      title: 'Presence statuses',
      description: 'Configure the attendance status types available platform-wide — Present, Sick, Permit, Absent — each with its own label and color.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/score-predicates"]',
    popover: {
      title: 'Score predicates',
      description: 'Configure the letter-grade bands used to summarize scores, platform-wide or per school.',
    },
  },
  {
    element: '[data-tour-nav-target="/news"]',
    popover: {
      title: 'News',
      description: 'Preview announcements the way students see them.',
    },
  },
  {
    element: '[data-tour-nav-target="/faq"]',
    popover: {
      title: 'FAQ',
      description: 'Preview the FAQ the way students see it.',
    },
  },
  {
    element: '[data-tour="notifications"]',
    popover: {
      title: 'Notifications',
      description: 'Platform-wide alerts and anything waiting on your review land here first.',
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
