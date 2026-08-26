import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import type { Role } from '@/types/api'
import { useAuthStore } from '@/stores/auth'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    guestOnly?: boolean
    roles?: Role[]
  }
}

const STAFF: Role[] = ['admin', 'coordinator', 'mentor']

const guestRoutes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/GuestLayout.vue'),
    meta: { guestOnly: true },
    children: [
      { path: '', redirect: '/login' },
      { path: 'login', name: 'login', component: () => import('@/views/auth/LoginView.vue') },
      { path: 'register', name: 'register', component: () => import('@/views/auth/RegisterView.vue') },
      { path: 'forgot-password', name: 'forgot-password', component: () => import('@/views/auth/ForgotPasswordView.vue') },
      { path: 'reset-password', name: 'reset-password', component: () => import('@/views/auth/ResetPasswordView.vue') },
    ],
  },
]

const appRoutes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/DefaultLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: 'dashboard', name: 'dashboard', component: () => import('@/views/DashboardView.vue') },

      // --- Student ---
      { path: 'vacancies', name: 'vacancies', component: () => import('@/views/vacancy/VacancyListView.vue'), meta: { roles: ['student'] } },
      { path: 'vacancies/:id', name: 'vacancy-detail', component: () => import('@/views/vacancy/VacancyDetailView.vue'), meta: { roles: ['student'] } },
      { path: 'my-applications', name: 'my-applications', component: () => import('@/views/vacancy/MyApplicationsView.vue'), meta: { roles: ['student'] } },
      { path: 'my-internship', name: 'my-internship', component: () => import('@/views/internship/MyInternshipView.vue'), meta: { roles: ['student'] } },
      { path: 'attendance', name: 'attendance', component: () => import('@/views/attendance/AttendanceView.vue'), meta: { roles: ['student'] } },
      { path: 'journals', name: 'journals', component: () => import('@/views/journal/JournalView.vue'), meta: { roles: ['student'] } },
      { path: 'certificate', name: 'certificate', component: () => import('@/views/certificate/CertificateView.vue'), meta: { roles: ['student'] } },

      // --- Shared ---
      { path: 'news', name: 'news', component: () => import('@/views/news/NewsListView.vue') },
      { path: 'news/:slug', name: 'news-detail', component: () => import('@/views/news/NewsDetailView.vue') },
      { path: 'faq', name: 'faq', component: () => import('@/views/faq/FaqView.vue') },
      { path: 'notifications', name: 'notifications', component: () => import('@/views/notifications/NotificationsView.vue') },
      { path: 'profile', name: 'profile', component: () => import('@/views/profile/ProfileView.vue') },
      { path: 'profile/change-password', name: 'change-password', component: () => import('@/views/profile/ChangePasswordView.vue') },

      // --- Admin / staff ---
      { path: 'admin/schools', name: 'admin-schools', component: () => import('@/views/admin/SchoolsView.vue'), meta: { roles: ['admin'] } },
      { path: 'admin/departments', name: 'admin-departments', component: () => import('@/views/admin/DepartmentsView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/courses', name: 'admin-courses', component: () => import('@/views/admin/CoursesView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/companies', name: 'admin-companies', component: () => import('@/views/admin/CompaniesView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/users', name: 'admin-users', component: () => import('@/views/admin/UsersView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/vacancies', name: 'admin-vacancies', component: () => import('@/views/admin/AdminVacanciesView.vue'), meta: { roles: STAFF } },
      { path: 'admin/appliances', name: 'admin-appliances', component: () => import('@/views/admin/AppliancesView.vue'), meta: { roles: STAFF } },
      { path: 'admin/presence', name: 'admin-presence', component: () => import('@/views/admin/PresenceReviewView.vue'), meta: { roles: STAFF } },
      { path: 'admin/journals', name: 'admin-journals', component: () => import('@/views/admin/JournalReviewView.vue'), meta: { roles: STAFF } },
      { path: 'admin/scores', name: 'admin-scores', component: () => import('@/views/admin/ScoresView.vue'), meta: { roles: STAFF } },
      { path: 'admin/presence-statuses', name: 'admin-presence-statuses', component: () => import('@/views/admin/PresenceStatusesView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/score-predicates', name: 'admin-score-predicates', component: () => import('@/views/admin/ScorePredicatesView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/monitors', name: 'admin-monitors', component: () => import('@/views/admin/MonitorsView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/questions', name: 'admin-questions', component: () => import('@/views/admin/QuestionsView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/news', name: 'admin-news', component: () => import('@/views/admin/AdminNewsView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/faqs', name: 'admin-faqs', component: () => import('@/views/admin/AdminFaqsView.vue'), meta: { roles: ['admin', 'coordinator'] } },
      { path: 'admin/reports', name: 'admin-reports', component: () => import('@/views/admin/ReportsView.vue'), meta: { roles: ['admin', 'coordinator'] } },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    ...guestRoutes,
    ...appRoutes,
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFoundView.vue') },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.isReady) {
    await auth.fetchMe()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }

  if (to.meta.roles && auth.role && !to.meta.roles.includes(auth.role)) {
    return { name: 'dashboard' }
  }

  return true
})

export default router
