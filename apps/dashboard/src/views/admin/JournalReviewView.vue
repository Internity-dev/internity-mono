<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery, type FetcherParams } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { Journal, BulkApproveResult } from '@/types/internship'
import { approvalStatus } from '@/lib/status'

import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

interface DepartmentOption {
  ID: number
  Name: string
}
interface CompanyOption {
  ID: number
  Name: string
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

// --- org-scope picker ----------------------------------------------------

const auth = useAuthStore()
const isMentor = computed(() => auth.user?.role === 'mentor')
const route = useRoute()

const departmentId = computed(() => (route.query.department_id ? Number(route.query.department_id) : undefined))
const urlCompanyId = computed(() => (route.query.company_id ? Number(route.query.company_id) : undefined))

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
  set: (v) => list.setParams({ department_id: v, company_id: undefined }),
})
const companyModel = computed<string | undefined>({
  get: () => (urlCompanyId.value ? String(urlCompanyId.value) : undefined),
  set: (v) => list.setParams({ company_id: v }),
})

const companyNameQuery = useQuery({
  queryKey: computed(() => ['company-name', effectiveCompanyId.value]),
  queryFn: () => http.get<ApiSuccess<CompanyOption>>(`/companies/${effectiveCompanyId.value}`).then((r) => r.data.data),
  enabled: computed(() => !!effectiveCompanyId.value),
})
const pageDescription = computed(() =>
  companyNameQuery.data.value
    ? `Reviewing journal entries for ${companyNameQuery.data.value.Name}.`
    : 'Select a company to review pending journal entries.',
)

// --- journals list --------------------------------------------------------

function fetchJournals(params: FetcherParams<'department_id' | 'company_id'>) {
  return http.get<ApiSuccess<Journal[]>>('/journals/pending-approval', { params: { ...params, company_id: effectiveCompanyId.value } }).then((r) => r.data)
}

const list = useListQuery<Journal, 'department_id' | 'company_id'>('journals-review', fetchJournals, {
  defaultSort: 'date',
  filters: [{ key: 'department_id', sendToFetcher: false }, { key: 'company_id', sendToFetcher: false }],
  enabled: () => !!effectiveCompanyId.value,
})

const searchModel = computed<string>({
  get: () => list.search.value,
  set: (v) => list.setParams({ search: v }),
})

// --- selection -------------------------------------------------------------

const selected = ref<Set<number>>(new Set())

function isBulkEligible(row: Journal) {
  return !!row.work_type && !!row.description && !row.is_approved
}

function toggleSelected(id: number, checked: boolean) {
  if (checked) selected.value.add(id)
  else selected.value.delete(id)
}

const eligibleRows = computed(() => list.items.value.filter(isBulkEligible))
const allEligibleSelected = computed(() => eligibleRows.value.length > 0 && eligibleRows.value.every((j) => selected.value.has(j.id)))

function toggleSelectAll(checked: boolean) {
  if (checked) eligibleRows.value.forEach((j) => selected.value.add(j.id))
  else eligibleRows.value.forEach((j) => selected.value.delete(j.id))
}

watch([effectiveCompanyId, () => list.page.value], () => selected.value.clear())

const columns: Column[] = [
  { key: 'select', label: '' },
  { key: 'user', label: 'Student' },
  { key: 'date', label: 'Date', sortable: true },
  { key: 'work_type', label: 'Work type' },
  { key: 'description', label: 'Description' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '', class: 'text-right' },
]

function onSort(key: string) {
  if (list.sort.value === key) {
    list.setParams({ sort: key, order: list.order.value === 'asc' ? 'desc' : 'asc' })
  } else {
    list.setParams({ sort: key, order: 'asc' })
  }
}

// --- mutations ---------------------------------------------------------

const queryClient = useQueryClient()
function invalidate() {
  queryClient.invalidateQueries({ queryKey: ['journals-review'] })
}

const approveTarget = ref<Journal | null>(null)
const approveMutation = useMutation({
  mutationFn: (id: number) => http.put(`/journals/${id}/approve`),
  onSuccess: () => {
    toast.success('Journal entry approved')
    approveTarget.value = null
    invalidate()
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})

const bulkConfirmOpen = ref(false)
const bulkApproveMutation = useMutation({
  mutationFn: () =>
    http
      .put<ApiSuccess<BulkApproveResult>>('/journals/bulk-approve', {
        company_id: effectiveCompanyId.value,
        ids: Array.from(selected.value),
      })
      .then((r) => r.data.data),
  onSuccess: (data) => {
    const requested = selected.value.size
    toast.success(`Approved ${data.approved_count} of ${requested} selected`)
    selected.value.clear()
    bulkConfirmOpen.value = false
    invalidate()
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})
</script>

<template>
  <div class="space-y-6 p-6">
    <PageHeader title="Journal Review" :description="pageDescription" />

    <Card v-if="!isMentor">
      <CardContent class="flex flex-wrap items-end gap-4 pt-0">
        <div class="w-56 space-y-1.5">
          <label class="text-sm font-medium">Department</label>
          <Select v-model="departmentModel">
            <SelectTrigger class="w-full"><SelectValue placeholder="Select department" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="d in departmentsQuery.data.value ?? []" :key="d.ID" :value="String(d.ID)">
                {{ d.Name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="w-56 space-y-1.5">
          <label class="text-sm font-medium">Company</label>
          <Select v-model="companyModel" :disabled="!departmentId">
            <SelectTrigger class="w-full"><SelectValue placeholder="Select company" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="c in companiesQuery.data.value ?? []" :key="c.ID" :value="String(c.ID)">
                {{ c.Name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </CardContent>
    </Card>

    <template v-if="effectiveCompanyId">
      <ListToolbar v-model="searchModel" placeholder="Search by student name, NIS, work type, or description…" />

      <div class="flex flex-wrap items-center justify-between gap-3">
        <p class="text-sm text-muted-foreground">
          {{ selected.size }} selected
          <span v-if="eligibleRows.length < list.items.value.length" class="text-xs">
            (entries missing a work type or description are skipped by bulk approval)
          </span>
        </p>
        <Button size="sm" :disabled="selected.size === 0" @click="bulkConfirmOpen = true">Approve selected</Button>
      </div>

      <DataTable
        :columns="columns"
        :rows="list.items.value"
        :is-loading="list.isLoading.value"
        :sort="list.sort.value"
        :order="list.order.value"
        empty-title="Nothing pending"
        empty-description="All journal entries for this company have been reviewed."
        @sort="onSort"
      >
        <template #cell-select="{ row }">
          <Checkbox
            :model-value="selected.has(row.id)"
            :disabled="!isBulkEligible(row)"
            @update:model-value="(v) => toggleSelected(row.id, !!v)"
          />
        </template>
        <template #cell-user="{ row }">
          <span class="font-mono text-xs" :title="row.user_id">{{ shortId(row.user_id) }}</span>
        </template>
        <template #cell-date="{ row }">{{ formatDate(row.date) }}</template>
        <template #cell-work_type="{ row }">{{ row.work_type ?? '—' }}</template>
        <template #cell-description="{ row }">
          <span class="line-clamp-1 max-w-xs text-muted-foreground">{{ row.description ?? '—' }}</span>
        </template>
        <template #cell-status="{ row }">
          <StatusBadge v-bind="approvalStatus(row.is_approved)" />
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end">
            <Button v-if="!row.is_approved" size="sm" variant="outline" @click="approveTarget = row">Approve</Button>
          </div>
        </template>
      </DataTable>

      <div v-if="list.items.value.length" class="flex items-center gap-2 text-sm text-muted-foreground">
        <Checkbox :model-value="allEligibleSelected" @update:model-value="(v) => toggleSelectAll(!!v)" />
        Select all eligible on this page
      </div>

      <ListPagination
        :page="list.page.value"
        :limit="list.limit.value"
        :total="list.pagination.value?.total ?? 0"
        @update:page="(p) => list.setParams({ page: p })"
      />
    </template>
    <EmptyState v-else title="Select a company" description="Choose a department and company above to review journal entries." />

    <ConfirmDialog
      :open="!!approveTarget"
      title="Approve this journal entry?"
      :destructive="false"
      confirm-label="Approve"
      :is-loading="approveMutation.isPending.value"
      @update:open="(v) => { if (!v) approveTarget = null }"
      @confirm="approveTarget && approveMutation.mutate(approveTarget.id)"
    />

    <ConfirmDialog
      :open="bulkConfirmOpen"
      title="Approve selected journal entries?"
      :description="`This will approve up to ${selected.size} entr${selected.size === 1 ? 'y' : 'ies'}. Entries missing a work type or description will be skipped.`"
      :destructive="false"
      confirm-label="Approve selected"
      :is-loading="bulkApproveMutation.isPending.value"
      @update:open="(v) => (bulkConfirmOpen = v)"
      @confirm="bulkApproveMutation.mutate()"
    />
  </div>
</template>
