import type { DriveStep } from 'driver.js'

export interface MenuInfo {
  selector: string
  title: string
  description: string
}

// One entry per sidebar route that has something worth explaining (see
// nav.ts for the canonical per-role menu list). Shared by the short core
// tours (studentTour.ts etc — a handful of steps for the critical-path
// workflow) and useMenuHints.ts (a one-time spotlight the first time a user
// actually visits that page), so the copy for a given menu is written once
// and used both ways instead of duplicated.
export const menuInfo: Record<string, MenuInfo> = {
  '/vacancies': {
    selector: '[data-tour-nav-target="/vacancies"]',
    title: 'Vacancies',
    description:
      'Browse every open internship listing from companies in your department. Search by company or skill, and check required skills, open slots, and the deadline before you apply.',
  },
  '/my-applications': {
    selector: '[data-tour-nav-target="/my-applications"]',
    title: 'My Applications',
    description:
      'Every vacancy you\'ve applied to, with its status: pending, accepted, or rejected. Your application note is here too, so you can see exactly what you sent.',
  },
  '/my-internship': {
    selector: '[data-tour-nav-target="/my-internship"]',
    title: 'My Internship',
    description:
      'Once a company accepts you, this becomes your placement record: which company, who your mentor is, and the start/end dates you set — those dates define your attendance and journal periods.',
  },
  '/attendance': {
    selector: '[data-tour-nav-target="/attendance"]',
    title: 'Attendance',
    description:
      'Check in and out each workday with a photo and your location, or file an excuse (sick, permit, leave) on a day you can\'t make it. Your mentor approves every entry.',
  },
  '/journals': {
    selector: '[data-tour-nav-target="/journals"]',
    title: 'Journal',
    description:
      'Write a short entry for each day you attend, describing what you worked on. Your mentor reads and approves these, and they become part of your final record.',
  },
  '/certificate': {
    selector: '[data-tour-nav-target="/certificate"]',
    title: 'Certificate',
    description:
      'Once your internship ends and your mentor has entered your scores, your official completion certificate becomes downloadable here.',
  },
  '/news': {
    selector: '[data-tour-nav-target="/news"]',
    title: 'News',
    description: 'Announcements and updates — new vacancies, deadline reminders, and program news.',
  },
  '/faq': {
    selector: '[data-tour-nav-target="/faq"]',
    title: 'FAQ',
    description: 'Answers to common questions about applying, attendance, and how certification works.',
  },
  '/notifications': {
    selector: '[data-tour-nav-target="/notifications"]',
    title: 'Notifications',
    description: 'Every update lands here — status changes, approvals, and new announcements.',
  },
  '/profile': {
    selector: '[data-tour-nav-target="/profile"]',
    title: 'Profile',
    description: 'Update your name, avatar, and password, and check your account details.',
  },
  '/admin/schools': {
    selector: '[data-tour-nav-target="/admin/schools"]',
    title: 'Schools',
    description:
      'Onboard new schools onto the platform — each one gets its own coordinators, departments, and student roster, isolated from every other school.',
  },
  '/admin/departments': {
    selector: '[data-tour-nav-target="/admin/departments"]',
    title: 'Departments',
    description: 'Manage academic departments (jurusan) — each one groups the courses and companies its students can be matched with.',
  },
  '/admin/courses': {
    selector: '[data-tour-nav-target="/admin/courses"]',
    title: 'Courses',
    description: 'Manage class groups (rombel) within each department — the roster grouping used when issuing invite codes and filtering students.',
  },
  '/admin/companies': {
    selector: '[data-tour-nav-target="/admin/companies"]',
    title: 'Companies',
    description: 'Manage the partner companies students can intern at — contact info, linked department, and the vacancies each one has posted.',
  },
  '/admin/users': {
    selector: '[data-tour-nav-target="/admin/users"]',
    title: 'Users',
    description: 'Every account you manage — students, mentors, staff. Edit roles, reset access, and issue invite codes for self-registration.',
  },
  '/admin/vacancies': {
    selector: '[data-tour-nav-target="/admin/vacancies"]',
    title: 'Vacancies',
    description: 'Post and edit internship openings — title, required skills, open slots, and which department can apply.',
  },
  '/admin/appliances': {
    selector: '[data-tour-nav-target="/admin/appliances"]',
    title: 'Applications',
    description: 'Review student applications to a vacancy — accept or reject, filter by department, company, or vacancy, and read each applicant\'s note.',
  },
  '/admin/presence': {
    selector: '[data-tour-nav-target="/admin/presence"]',
    title: 'Attendance review',
    description: 'Approve or reject student check-ins (photo + location) and filed excuses — this is what finalizes an attendance record.',
  },
  '/admin/journals': {
    selector: '[data-tour-nav-target="/admin/journals"]',
    title: 'Journal review',
    description: 'Read and approve students\' daily work journals.',
  },
  '/admin/scores': {
    selector: '[data-tour-nav-target="/admin/scores"]',
    title: 'Scores',
    description: 'Enter or review student scores per placement. Grades map to the letter bands set up under Score predicates.',
  },
  '/admin/monitors': {
    selector: '[data-tour-nav-target="/admin/monitors"]',
    title: 'Monitoring visits',
    description: 'Log site visits to partner companies — notes on how a student is doing, plus suggestions for their mentor.',
  },
  '/admin/questions': {
    selector: '[data-tour-nav-target="/admin/questions"]',
    title: 'Reviews & questions',
    description: 'Manage the questionnaire used to review a company or mentor once a placement ends.',
  },
  '/admin/reports': {
    selector: '[data-tour-nav-target="/admin/reports"]',
    title: 'Reports',
    description: 'Export student rosters, attendance summaries, and program reports for record-keeping or accreditation.',
  },
  '/admin/news': {
    selector: '[data-tour-nav-target="/admin/news"]',
    title: 'Manage news',
    description: 'Publish announcements — students and staff are notified automatically.',
  },
  '/admin/faqs': {
    selector: '[data-tour-nav-target="/admin/faqs"]',
    title: 'Manage FAQ',
    description: 'Write and edit the FAQ entries your students see on the public FAQ page.',
  },
  '/admin/presence-statuses': {
    selector: '[data-tour-nav-target="/admin/presence-statuses"]',
    title: 'Presence statuses',
    description:
      'Define the attendance status types available when reviewing check-ins — Present, Sick, Permit, Absent — each with its own label and color.',
  },
  '/admin/score-predicates': {
    selector: '[data-tour-nav-target="/admin/score-predicates"]',
    title: 'Score predicates',
    description: 'Configure the letter-grade bands (like A = 90–100) used to summarize scores on a certificate.',
  },
}

// Turns a menuInfo entry into a driver.js step — used by the role core
// tours to reuse this copy instead of re-writing it per step. The `!` is
// safe here: every call site passes one of the hardcoded literal keys
// declared right above menuInfo in each tours/*Tour.ts file, not a
// runtime-derived path (see useMenuHints.ts for the arbitrary-path lookup,
// which does check for a miss).
export function menuStep(path: string): DriveStep {
  const info = menuInfo[path]!
  return { element: info.selector, popover: { title: info.title, description: info.description } }
}
