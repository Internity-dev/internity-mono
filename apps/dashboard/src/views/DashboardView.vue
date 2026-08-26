<script setup lang="ts">
import { computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import {
  BriefcaseIcon,
  ClipboardListIcon,
  Building2Icon,
  School2Icon,
  LibraryBigIcon,
} from '@lucide/vue'
import { useAuthStore } from '@/stores/auth'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import PageHeader from '@/components/shared/PageHeader.vue'
import DonutChart from '@/components/shared/DonutChart.vue'
import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'
import type { Appliance } from '@/types/vacancy'

const auth = useAuthStore()

const isAdmin = computed(() => auth.role === 'admin')
const isCoordinator = computed(() => auth.role === 'coordinator')
// Org KPI row: admin and coordinator both manage org structure, just at a
// different scope (platform-wide vs their one school) — the backend already
// pins every one of these list endpoints to the caller's own school_id for
// anyone but admin, so no explicit school_id param is needed here.
const isStaffLead = computed(() => isAdmin.value || isCoordinator.value)
// Status-breakdown charts: admin (platform), coordinator (their school), or
// mentor (their company) — backend scopes each accordingly.
const canSeeStatusCharts = computed(() => isStaffLead.value || auth.role === 'mentor')
const isStudent = computed(() => auth.role === 'student')

type StatAccent = 'primary' | 'accent' | 'warning'

function useCountQuery(key: string, path: string, enabled: () => boolean) {
  return useQuery({
    queryKey: [key, 'count'],
    queryFn: () => http.get<ApiSuccess<unknown[]>>(path, { params: { limit: 1 } }).then((r) => r.data.meta?.pagination?.total ?? 0),
    enabled,
  })
}

// --- Org overview KPIs. Companies/Vacancies/Courses' list endpoints
// require an explicit department_id filter for anyone but admin (a
// coordinator manages a whole school's worth of departments, not one), so
// those three stay admin-only here rather than risk a 403 — see
// orgs.Service.ListCompanies/ListCourses and vacancy.Service.ListVacancies.
// Departments is the one org resource that's auto-scoped to the caller's
// own school (orgs.Service.ListDepartments), so it's what a coordinator
// gets instead.
const schoolsCount = useCountQuery('schools', '/schools', () => isAdmin.value)
const departmentsCount = useCountQuery('departments', '/departments', () => isCoordinator.value)
const companiesCount = useCountQuery('companies', '/companies', () => isAdmin.value)
const vacanciesCount = useCountQuery('vacancies', '/vacancies', () => isAdmin.value)
const coursesCount = useCountQuery('courses', '/courses', () => isAdmin.value)

const overviewStats = computed(() => {
  if (!isAdmin.value) {
    return [{ label: 'Departemen', value: departmentsCount.data.value, icon: Building2Icon, accent: 'primary' as StatAccent }]
  }
  return [
    { label: 'Sekolah', value: schoolsCount.data.value, icon: School2Icon, accent: 'primary' as StatAccent },
    { label: 'Perusahaan mitra', value: companiesCount.data.value, icon: Building2Icon, accent: 'accent' as StatAccent },
    { label: 'Lowongan', value: vacanciesCount.data.value, icon: BriefcaseIcon, accent: 'warning' as StatAccent },
    { label: 'Jurusan', value: coursesCount.data.value, icon: LibraryBigIcon, accent: 'primary' as StatAccent },
  ]
})

// --- Personal (student): same self-scoped endpoint MyApplicationsView
// already uses, fetched once and reused for both the KPI count and the
// status breakdown below — Attendance/Journal aren't shown here since
// those endpoints require a company_id (a student can have more than one
// placement across school years; MyInternshipView resolves which one),
// which this overview has no reason to ask the user to pick just to render
// a number.
const myAppliances = useQuery({
  queryKey: ['my-appliances', 'dashboard'],
  queryFn: () => http.get<ApiSuccess<Appliance[]>>('/appliances', { params: { limit: 100 } }).then((r) => r.data),
  enabled: isStudent,
})

const personalStats = computed(() => [
  {
    label: 'Lamaran diajukan',
    value: myAppliances.data.value?.meta?.pagination?.total,
    icon: ClipboardListIcon,
    accent: 'primary' as StatAccent,
  },
])

const myApplianceChartData = computed(() => {
  const rows = myAppliances.data.value?.data ?? []
  const counts: Record<Appliance['status'], number> = { pending: 0, processed: 0, accepted: 0, rejected: 0, canceled: 0 }
  for (const row of rows) counts[row.status]++
  return [
    { label: 'Pending', value: counts.pending, color: 'chart5' as const },
    { label: 'Processed', value: counts.processed, color: 'chart3' as const },
    { label: 'Accepted', value: counts.accepted, color: 'chart2' as const },
    { label: 'Rejected', value: counts.rejected, color: 'chart4' as const },
    { label: 'Canceled', value: counts.canceled, color: 'chart1' as const },
  ]
})

// --- Status-breakdown charts: real GROUP BY counts from the backend, not
// fabricated. Scoped server-side per statusCountsScopeFor (vacancy module)
// / presenceStatusCountsScopeFor (internship module).

function useStatusCountsQuery(key: string, path: string) {
  return useQuery({
    queryKey: [key, 'status-counts'],
    queryFn: () => http.get<ApiSuccess<Record<string, number>>>(path).then((r) => r.data.data),
    enabled: canSeeStatusCharts,
  })
}

const applianceCounts = useStatusCountsQuery('appliances', '/appliances/status-counts')
const vacancyCounts = useStatusCountsQuery('vacancies', '/vacancies/status-counts')
const presenceCounts = useStatusCountsQuery('presences', '/presences/status-counts')

const applianceChartData = computed(() => [
  { label: 'Pending', value: applianceCounts.data.value?.pending ?? 0, color: 'chart5' as const },
  { label: 'Processed', value: applianceCounts.data.value?.processed ?? 0, color: 'chart3' as const },
  { label: 'Accepted', value: applianceCounts.data.value?.accepted ?? 0, color: 'chart2' as const },
  { label: 'Rejected', value: applianceCounts.data.value?.rejected ?? 0, color: 'chart4' as const },
  { label: 'Canceled', value: applianceCounts.data.value?.canceled ?? 0, color: 'chart1' as const },
])

const vacancyChartData = computed(() => [
  { label: 'Open', value: vacancyCounts.data.value?.open ?? 0, color: 'chart2' as const },
  { label: 'Closed', value: vacancyCounts.data.value?.closed ?? 0, color: 'chart5' as const },
])

const presenceChartData = computed(() => [
  { label: 'Present', value: presenceCounts.data.value?.present ?? 0, color: 'chart2' as const },
  { label: 'Permitted', value: presenceCounts.data.value?.permitted ?? 0, color: 'chart1' as const },
  { label: 'Sick', value: presenceCounts.data.value?.sick ?? 0, color: 'chart3' as const },
  { label: 'Absent', value: presenceCounts.data.value?.absent ?? 0, color: 'chart4' as const },
  { label: 'Holiday', value: presenceCounts.data.value?.holiday ?? 0, color: 'chart5' as const },
])

// The scope differs per role, so the chart captions say so instead of
// implying "everyone" when it's really "your school" or "your company".
const scopeNoun = computed(() => {
  if (isAdmin.value) return 'the platform'
  if (isCoordinator.value) return 'your school'
  return 'your company'
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader :title="`Welcome back, ${auth.user?.name ?? ''}`" description="Here's what's on your plate today." />

    <div v-if="isStaffLead" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <Card v-for="stat in overviewStats" :key="stat.label" class="gap-2">
        <CardHeader class="flex-row items-center justify-between space-y-0">
          <CardDescription>{{ stat.label }}</CardDescription>
          <div
            class="flex size-8 items-center justify-center rounded-lg"
            :class="{
              'bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-300': stat.accent === 'primary',
              'bg-accent-50 text-accent-700 dark:bg-accent-950 dark:text-accent-300': stat.accent === 'accent',
              'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400': stat.accent === 'warning',
            }"
          >
            <component :is="stat.icon" class="size-4" />
          </div>
        </CardHeader>
        <div class="px-6">
          <p v-if="stat.value === undefined" class="h-8 w-16 animate-pulse rounded bg-muted" />
          <p v-else class="font-display text-2xl font-semibold tabular-nums">{{ stat.value }}</p>
        </div>
      </Card>
    </div>

    <div v-if="isStudent" class="grid gap-4 sm:grid-cols-3">
      <Card v-for="stat in personalStats" :key="stat.label" class="gap-2">
        <CardHeader class="flex-row items-center justify-between space-y-0">
          <CardDescription>{{ stat.label }}</CardDescription>
          <div
            class="flex size-8 items-center justify-center rounded-lg"
            :class="{
              'bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-300': stat.accent === 'primary',
              'bg-accent-50 text-accent-700 dark:bg-accent-950 dark:text-accent-300': stat.accent === 'accent',
              'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400': stat.accent === 'warning',
            }"
          >
            <component :is="stat.icon" class="size-4" />
          </div>
        </CardHeader>
        <div class="px-6">
          <p v-if="stat.value === undefined" class="h-8 w-16 animate-pulse rounded bg-muted" />
          <p v-else class="font-display text-2xl font-semibold tabular-nums">{{ stat.value }}</p>
        </div>
      </Card>
    </div>

    <div v-if="isStudent" class="grid gap-4 sm:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle class="text-base">My applications by status</CardTitle>
          <CardDescription>Every vacancy you've applied to, current state</CardDescription>
        </CardHeader>
        <CardContent>
          <DonutChart :data="myApplianceChartData" />
        </CardContent>
      </Card>
    </div>

    <div v-if="canSeeStatusCharts" class="grid gap-4 lg:grid-cols-3">
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Applications by status</CardTitle>
          <CardDescription>Every vacancy application in {{ scopeNoun }}, current state</CardDescription>
        </CardHeader>
        <CardContent>
          <DonutChart :data="applianceChartData" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Vacancies by status</CardTitle>
          <CardDescription>Open vs closed listings in {{ scopeNoun }}</CardDescription>
        </CardHeader>
        <CardContent>
          <DonutChart :data="vacancyChartData" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Attendance breakdown</CardTitle>
          <CardDescription>This month's presence in {{ scopeNoun }}, by kind</CardDescription>
        </CardHeader>
        <CardContent>
          <DonutChart :data="presenceChartData" />
        </CardContent>
      </Card>
    </div>
  </div>
</template>
