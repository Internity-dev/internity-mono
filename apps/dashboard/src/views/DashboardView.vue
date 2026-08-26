<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import {
  BriefcaseIcon,
  MapPinnedIcon,
  CalendarCheckIcon,
  BookOpenIcon,
  AwardIcon,
  ClipboardListIcon,
  ClipboardCheckIcon,
  ListChecksIcon,
  UsersIcon,
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

const auth = useAuthStore()

// Overview KPI row (admin/coordinator only) — real counts pulled from each
// resource's own list endpoint (limit=1, we only need meta.pagination.total),
// not fabricated numbers.
const isStaffLead = computed(() => auth.role === 'admin' || auth.role === 'coordinator')
// The three status-breakdown charts hit admin-only aggregate endpoints (see
// brand.md-adjacent backend comments: coordinator scoping is deferred there
// since there's no clean join to a coordinator's school yet).
const isAdmin = computed(() => auth.role === 'admin')

function useCountQuery(key: string, path: string) {
  return useQuery({
    queryKey: [key, 'count'],
    queryFn: () => http.get<ApiSuccess<unknown[]>>(path, { params: { limit: 1 } }).then((r) => r.data.meta?.pagination?.total ?? 0),
    enabled: isStaffLead,
  })
}

const schoolsCount = useCountQuery('schools', '/schools')
const companiesCount = useCountQuery('companies', '/companies')
const vacanciesCount = useCountQuery('vacancies', '/vacancies')
const coursesCount = useCountQuery('courses', '/courses')

const overviewStats = computed(() => [
  { label: 'Sekolah', value: schoolsCount.data.value, icon: School2Icon, accent: 'primary' as const },
  { label: 'Perusahaan mitra', value: companiesCount.data.value, icon: Building2Icon, accent: 'accent' as const },
  { label: 'Lowongan', value: vacanciesCount.data.value, icon: BriefcaseIcon, accent: 'warning' as const },
  { label: 'Jurusan', value: coursesCount.data.value, icon: LibraryBigIcon, accent: 'primary' as const },
])

// --- Analytics: real GROUP BY counts from the backend, not fabricated ---

function useStatusCountsQuery(key: string, path: string) {
  return useQuery({
    queryKey: [key, 'status-counts'],
    queryFn: () => http.get<ApiSuccess<Record<string, number>>>(path).then((r) => r.data.data),
    enabled: isAdmin,
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

const studentLinks = [
  { to: '/vacancies', label: 'Browse vacancies', description: 'Find an internship placement', icon: BriefcaseIcon },
  { to: '/my-applications', label: 'My applications', description: 'Track your application status', icon: ClipboardListIcon },
  { to: '/my-internship', label: 'My internship', description: 'View or set your placement dates', icon: MapPinnedIcon },
  { to: '/attendance', label: 'Attendance', description: 'Check in, check out, or file an excuse', icon: CalendarCheckIcon },
  { to: '/journals', label: 'Journal', description: 'Log your daily activities', icon: BookOpenIcon },
  { to: '/certificate', label: 'Certificate', description: 'Download your completion certificate', icon: AwardIcon },
]

const staffLinks = [
  { to: '/admin/appliances', label: 'Applications', description: 'Review pending vacancy applications', icon: ClipboardListIcon },
  { to: '/admin/presence', label: 'Attendance review', description: 'Approve student check-ins', icon: ClipboardCheckIcon },
  { to: '/admin/journals', label: 'Journal review', description: 'Approve student journal entries', icon: BookOpenIcon },
  { to: '/admin/scores', label: 'Scores', description: 'Enter and review student scores', icon: ListChecksIcon },
]

const adminLinks = [
  { to: '/admin/schools', label: 'Schools', description: 'Onboard and manage schools', icon: Building2Icon },
  { to: '/admin/users', label: 'Users', description: 'Manage staff and student accounts', icon: UsersIcon },
]

const links = computed(() => {
  if (auth.role === 'student') return studentLinks
  if (auth.role === 'admin') return [...adminLinks, ...staffLinks]
  return staffLinks
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

    <div v-if="isAdmin" class="grid gap-4 lg:grid-cols-3">
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Applications by status</CardTitle>
          <CardDescription>All vacancy applications, current state</CardDescription>
        </CardHeader>
        <CardContent>
          <DonutChart :data="applianceChartData" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Vacancies by status</CardTitle>
          <CardDescription>Open vs closed listings</CardDescription>
        </CardHeader>
        <CardContent>
          <DonutChart :data="vacancyChartData" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Attendance breakdown</CardTitle>
          <CardDescription>This month's presence, by kind</CardDescription>
        </CardHeader>
        <CardContent>
          <DonutChart :data="presenceChartData" />
        </CardContent>
      </Card>
    </div>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to">
        <Card class="h-full transition-[transform,box-shadow] duration-150 ease-out hover:-translate-y-0.5 hover:shadow-md dark:hover:shadow-lg dark:hover:shadow-black/50">
          <CardHeader>
            <div class="mb-2 flex size-9 items-center justify-center rounded-lg bg-primary-50 text-primary-700 dark:bg-primary-950">
              <component :is="link.icon" class="size-5" />
            </div>
            <CardTitle class="text-base">{{ link.label }}</CardTitle>
            <CardDescription>{{ link.description }}</CardDescription>
          </CardHeader>
        </Card>
      </RouterLink>
    </div>
  </div>
</template>
