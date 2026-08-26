import type { DriveStep } from 'driver.js'

// Every step targets a real sidebar item (via data-tour-nav-target) or the
// header bell, so the tour stays in sync with nav.ts by construction — add a
// menu there and its tour step is the only other thing to add here.
export const coordinatorTourSteps: DriveStep[] = [
  {
    popover: {
      title: 'Welcome to Internity',
      description:
        'You run your school\'s internship program end to end: org structure, applications, attendance, scoring, and the content your students see. Here\'s what every menu does.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/departments"]',
    popover: {
      title: 'Departments',
      description: 'Manage your school\'s academic departments (jurusan) — each one groups the courses and companies its students can be matched with.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/courses"]',
    popover: {
      title: 'Courses',
      description: 'Manage class groups (rombel) within each department — the roster grouping used when you issue invite codes and filter students.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/companies"]',
    popover: {
      title: 'Companies',
      description: 'Manage the partner companies your students can intern at — contact info, linked department, and the vacancies each one has posted.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/users"]',
    popover: {
      title: 'Users',
      description: 'Every account in your school — students, mentors, staff. Edit roles, reset access, and issue invite codes for self-registration.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/vacancies"]',
    popover: {
      title: 'Vacancies',
      description: 'Post and edit internship openings on behalf of your partner companies — title, required skills, open slots, which department can apply.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/appliances"]',
    popover: {
      title: 'Applications',
      description: 'Review student applications to any vacancy — accept or reject, filter by department, company, or vacancy, and read each student\'s note.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/presence"]',
    popover: {
      title: 'Attendance review',
      description: 'Approve or reject student check-ins (photo + location) and filed excuses — this is what finalizes an attendance record.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/journals"]',
    popover: {
      title: 'Journal review',
      description: 'Read and approve students\' daily work journals across your whole school, not just one company.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/scores"]',
    popover: {
      title: 'Scores',
      description: 'Enter or review student scores per placement. Grades map to the letter bands you set up under Score predicates.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/monitors"]',
    popover: {
      title: 'Monitoring visits',
      description: 'Log your site visits to partner companies — notes on how a student is doing, plus suggestions for their mentor.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/questions"]',
    popover: {
      title: 'Reviews & questions',
      description: 'Manage the questionnaire used to review a company or mentor once a placement ends.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/reports"]',
    popover: {
      title: 'Reports',
      description: 'Export student rosters, attendance summaries, and program reports for record-keeping or accreditation.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/news"]',
    popover: {
      title: 'Manage news',
      description: 'Publish announcements to your school — students and staff are notified automatically.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/faqs"]',
    popover: {
      title: 'Manage FAQ',
      description: 'Write and edit the FAQ entries your students see on the public FAQ page.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/presence-statuses"]',
    popover: {
      title: 'Presence statuses',
      description: 'Define the attendance status types available when reviewing check-ins — Present, Sick, Permit, Absent — each with its own label and color.',
    },
  },
  {
    element: '[data-tour-nav-target="/admin/score-predicates"]',
    popover: {
      title: 'Score predicates',
      description: 'Configure the letter-grade bands (like A = 90–100) used to summarize scores on a student\'s certificate.',
    },
  },
  {
    element: '[data-tour-nav-target="/news"]',
    popover: {
      title: 'News',
      description: 'Preview announcements the way your students see them.',
    },
  },
  {
    element: '[data-tour-nav-target="/faq"]',
    popover: {
      title: 'FAQ',
      description: 'Preview the FAQ the way your students see it.',
    },
  },
  {
    element: '[data-tour="notifications"]',
    popover: {
      title: 'Notifications',
      description: 'New applications, approvals waiting on you, and system alerts all land here first.',
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
