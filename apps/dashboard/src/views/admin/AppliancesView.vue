<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery, type FetcherParams } from '@/composables/useListQuery'
import { useLastOrgScope } from '@/composables/useLastOrgScope'
import type { ApiSuccess } from '@/types/api'
import type { Appliance } from '@/types/vacancy'
import type { Vacancy } from '@/types/vacancy'
import { applianceStatus } from '@/lib/status'

import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

interface DepartmentOption {
  id: number
  name: string
}
interface CompanyOption {
  id: number
  name: string
}

function errMessage(err: unknown): string {
  return (err as { response?: { data?: { message?: string } } })?.response?.data?.message ?? 'Something went wrong'
}

function shortId(id: string) {
  return id.length > 13 ? `${id.slice(0, 8)}…${id.slice(-4)}` : id
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

// --- org-scope pickers -------------------------------------------------

const auth = useAuthStore()
const isMentor = computed(() => auth.user?.role === 'mentor')
const route = useRoute()

// department_id/company_id/vacancy_id are all read straight from the route
// query so the cascading pickers below can exist before `list` (which owns
// the setParams calls that actually write them) is declared.
const lastScope = useLastOrgScope()
const departmentId = computed(() => lastScope.departmentDefault(route.query.department_id ? Number(route.query.department_id) : undefined))
const urlCompanyId = computed(() => lastScope.companyDefault(route.query.company_id ? Number(route.query.company_id) : undefined, departmentId.value))
const vacancyId = computed(() => (route.query.vacancy_id ? Number(route.query.vacancy_id) : undefined))

// Remembers whatever the staff picker lands on (explicit pick or a
// remembered default resolved above) so the next cascading page defaults to
// the same department/company — mentors never see this picker, so their
// company (always their own, never chosen) shouldn't overwrite it. Vacancy
// isn't part of the shared memory (it's specific to this one page).
watch([departmentId, urlCompanyId], ([d, c]) => {
  if (!isMentor.value) lastScope.remember(d, c)
})

const departmentsQuery = useQuery({
  queryKey: ['org-departments-picker'],
  queryFn: () =>
    http
      .get<ApiSuccess<DepartmentOption[]>>('/departments', { params: { limit: 100, sort: 'name', order: 'asc' } })
      .then((r) => r.data.data),
  enabled: computed(() => !isMentor.value),
})

const companiesQuery = useQuery({
  queryKey: computed(() => ['org-companies-picker', departmentId.value]),
  queryFn: () =>
    http
      .get<ApiSuccess<CompanyOption[]>>('/companies', {
        params: { department_id: departmentId.value, limit: 100, sort: 'name', order: 'asc' },
      })
      .then((r) => r.data.data),
  enabled: computed(() => !isMentor.value && !!departmentId.value),
})

const effectiveCompanyId = computed<number | undefined>(() => (isMentor.value ? auth.user?.company_id : urlCompanyId.value))

const departmentModel = computed<string | undefined>({
  get: () => (departmentId.value ? String(departmentId.value) : undefined),
  // Changing department resets company and vacancy in one atomic URL update.
  set: (v) => list.setParams({ department_id: v, company_id: undefined, vacancy_id: undefined }),
})
const companyModel = computed<string | undefined>({
  get: () => (urlCompanyId.value ? String(urlCompanyId.value) : undefined),
  set: (v) => list.setParams({ company_id: v, vacancy_id: undefined }),
})

// --- vacancy picker (which application queue to review) -----------------

const vacanciesQuery = useQuery({
  queryKey: computed(() => ['vacancies-picker', effectiveCompanyId.value]),
  queryFn: () =>
    http
      .get<ApiSuccess<Vacancy[]>>('/vacancies', {
        params: { company_id: effectiveCompanyId.value, limit: 100, sort: 'name', order: 'asc' },
      })
      .then((r) => r.data.data),
  enabled: computed(() => !!effectiveCompanyId.value),
})

const vacancyModel = computed<string | undefined>({
  get: () => (vacancyId.value ? String(vacancyId.value) : undefined),
  set: (v) => list.setParams({ vacancy_id: v }),
})

const pageDescription = computed(() => {
  const v = vacanciesQuery.data.value?.find((x) => x.id === vacancyId.value)
  return v ? `Reviewing applications for "${v.name}".` : 'Pick a vacancy to review its applications.'
})

// --- appliances list (scoped to the selected vacancy) --------------------

function fetchAppliances({ vacancy_id, ...rest }: FetcherParams<'department_id' | 'company_id' | 'vacancy_id'>) {
  return http.get<ApiSuccess<Appliance[]>>(`/vacancies/${vacancy_id}/appliances`, { params: rest }).then((r) => r.data)
}

const list = useListQuery<Appliance, 'department_id' | 'company_id' | 'vacancy_id'>('vacancy-appliances', fetchAppliances, {
  filters: [{ key: 'department_id', sendToFetcher: false }, { key: 'company_id', sendToFetcher: false }, 'vacancy_id'],
  enabled: () => !!vacancyId.value,
})

const searchModel = computed<string>({
  get: () => list.search.value,
  set: (v) => list.setParams({ search: v }),
})

const columns: Column[] = [
  { key: 'applicant', label: 'Applicant' },
  { key: 'status', label: 'Status' },
  { key: 'applied', label: 'Applied', sortable: true },
  { key: 'actions', label: '', class: 'text-right' },
]

function onSort(key: string) {
  const backendKey = key === 'applied' ? 'created_at' : key
  if (list.sort.value === backendKey) {
    list.setParams({ sort: backendKey, order: list.order.value === 'asc' ? 'desc' : 'asc' })
  } else {
    list.setParams({ sort: backendKey, order: 'asc' })
  }
}

function canProcess(row: Appliance) {
  return row.status === 'pending'
}
function canAcceptReject(row: Appliance) {
  return row.status === 'pending' || row.status === 'processed'
}

const queryClient = useQueryClient()

function invalidate() {
  queryClient.invalidateQueries({ queryKey: ['vacancy-appliances'] })
}

const processMutation = useMutation({
  mutationFn: (id: number) => http.put(`/appliances/${id}/process`),
  onSuccess: () => {
    toast.success('Marked as under review')
    invalidate()
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})

const acceptTarget = ref<Appliance | null>(null)
const acceptMutation = useMutation({
  mutationFn: (id: number) => http.put(`/appliances/${id}/accept`),
  onSuccess: () => {
    toast.success('Application accepted')
    acceptTarget.value = null
    invalidate()
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})

const rejectTarget = ref<Appliance | null>(null)
const rejectMutation = useMutation({
  mutationFn: (id: number) => http.put(`/appliances/${id}/reject`),
  onSuccess: () => {
    toast.success('Application rejected')
    rejectTarget.value = null
    invalidate()
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})
</script>

<template>
  <div class="space-y-6 p-6">
    <PageHeader title="Applications" :description="pageDescription" />

    <Card>
      <CardContent class="flex flex-wrap items-end gap-4 pt-0">
        <div v-if="!isMentor" class="w-56 space-y-1.5">
          <label class="text-sm font-medium">Department</label>
          <Select v-model="departmentModel">
            <SelectTrigger class="w-full"><SelectValue placeholder="Select department" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="d in departmentsQuery.data.value ?? []" :key="d.id" :value="String(d.id)">
                {{ d.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="!isMentor" class="w-56 space-y-1.5">
          <label class="text-sm font-medium">Company</label>
          <Select v-model="companyModel" :disabled="!departmentId || companiesQuery.isFetching.value">
            <SelectTrigger class="w-full">
              <SelectValue :placeholder="companiesQuery.isFetching.value ? 'Loading…' : 'Select company'" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="c in companiesQuery.data.value ?? []" :key="c.id" :value="String(c.id)">
                {{ c.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="w-64 space-y-1.5">
          <label class="text-sm font-medium">Vacancy</label>
          <Select v-model="vacancyModel" :disabled="!effectiveCompanyId || vacanciesQuery.isFetching.value">
            <SelectTrigger class="w-full">
              <SelectValue :placeholder="vacanciesQuery.isFetching.value ? 'Loading…' : 'Select vacancy'" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="v in vacanciesQuery.data.value ?? []" :key="v.id" :value="String(v.id)">
                {{ v.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </CardContent>
    </Card>

    <template v-if="vacancyId">
      <ListToolbar v-model="searchModel" placeholder="Search by applicant name or NIS…" />

      <DataTable
        :columns="columns"
        :rows="list.items.value"
        :is-loading="list.isLoading.value"
        :sort="list.sort.value === 'created_at' ? 'applied' : list.sort.value"
        :order="list.order.value"
        empty-title="No applications yet"
        empty-description="Applications for this vacancy will show up here."
        @sort="onSort"
      >
        <template #cell-applicant="{ row }">
          <div class="flex flex-col">
            <span class="font-medium text-foreground">{{ row.user_name || 'Unknown student' }}</span>
            <span class="font-mono text-xs text-muted-foreground" :title="row.user_id">{{ row.user_nis || shortId(row.user_id) }}</span>
          </div>
        </template>
        <template #cell-status="{ row }">
          <StatusBadge v-bind="applianceStatus(row.status)" />
        </template>
        <template #cell-applied="{ row }">{{ formatDate(row.created_at) }}</template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end gap-2">
            <Button v-if="canProcess(row)" size="sm" variant="outline" :disabled="processMutation.isPending.value" @click="processMutation.mutate(row.id)">
              Process
            </Button>
            <Button v-if="canAcceptReject(row)" size="sm" variant="outline" @click="acceptTarget = row">Accept</Button>
            <Button v-if="canAcceptReject(row)" size="sm" variant="destructive" @click="rejectTarget = row">Reject</Button>
            <span v-if="!canAcceptReject(row) && !canProcess(row)" class="text-sm text-muted-foreground">—</span>
          </div>
        </template>
      </DataTable>

      <ListPagination
        :page="list.page.value"
        :limit="list.limit.value"
        :total="list.pagination.value?.total ?? 0"
        @update:page="(p) => list.setParams({ page: p })"
      />
    </template>
    <EmptyState v-else title="Select a vacancy" description="Choose a company and vacancy above to review its applications." />

    <ConfirmDialog
      :open="!!acceptTarget"
      title="Accept this application?"
      description="This schedules the student's placement at this company. This cannot be undone from here."
      confirm-label="Accept"
      :destructive="false"
      :is-loading="acceptMutation.isPending.value"
      @update:open="(v) => { if (!v) acceptTarget = null }"
      @confirm="acceptTarget && acceptMutation.mutate(acceptTarget.id)"
    />

    <ConfirmDialog
      :open="!!rejectTarget"
      title="Reject this application?"
      description="The applicant will be notified that their application was rejected."
      confirm-label="Reject"
      :is-loading="rejectMutation.isPending.value"
      @update:open="(v) => { if (!v) rejectTarget = null }"
      @confirm="rejectTarget && rejectMutation.mutate(rejectTarget.id)"
    />
  </div>
</template>
