<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, Trash2Icon, StarIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { CreateMonitorPayload, Monitor } from '@/types/review'
import { normalizeKeys } from '@/types/review'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

// orgs.Department / orgs.Company (apps/api/.../orgs/domain.go) carry no
// `json` tags either, so these pickers normalize the raw PascalCase (ID,
// Name) response the same way as the review-module types above.
interface Department {
  id: number
  name: string
}
interface Company {
  id: number
  name: string
}

const auth = useAuthStore()
const queryClient = useQueryClient()

// --- school -> department -> company cascade ---
const schoolIdInput = ref<string>(auth.user?.school_id ? String(auth.user.school_id) : '')
const schoolId = computed(() => {
  const n = Number(schoolIdInput.value)
  return schoolIdInput.value !== '' && Number.isFinite(n) && n > 0 ? n : undefined
})

const departmentId = ref<number | undefined>(undefined)
const departmentsQuery = useQuery({
  queryKey: computed(() => ['departments-picker', schoolId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<unknown[]>>('/departments', {
      params: { school_id: schoolId.value, limit: 100 },
    })
    return normalizeKeys<Department[]>(res.data.data)
  },
  enabled: computed(() => schoolId.value !== undefined),
})
const departments = computed(() => departmentsQuery.data.value ?? [])

const companyId = ref<number | undefined>(undefined)
const companiesQuery = useQuery({
  queryKey: computed(() => ['companies-picker', departmentId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<unknown[]>>('/companies', {
      params: { department_id: departmentId.value, limit: 100 },
    })
    return normalizeKeys<Company[]>(res.data.data)
  },
  enabled: computed(() => departmentId.value !== undefined),
})
const companies = computed(() => companiesQuery.data.value ?? [])

watch(departmentId, () => {
  companyId.value = undefined
})

const studentIdFilter = ref('')

// --- list ---
const listQuery = useListQuery<Monitor>(
  'monitors',
  async (params) => {
    // Monitor (apps/api/.../review/domain.go) carries no `json` tags, so the
    // raw response is PascalCase — normalize just the `data` payload, the
    // envelope itself (success/data/message/meta) is already snake_case.
    const res = await http.get<ApiSuccess<unknown[]>>('/monitors', { params })
    return { ...res.data, data: normalizeKeys<Monitor[]>(res.data.data) }
  },
  {
    defaultSort: 'date',
    extraParams: () => ({
      company_id: companyId.value,
      student_id: studentIdFilter.value || undefined,
    }),
    enabled: () => companyId.value !== undefined || studentIdFilter.value !== '',
  },
)

const columns: Column[] = [
  { key: 'student_id', label: 'Student' },
  { key: 'date', label: 'Date', sortable: true },
  { key: 'match_rating', label: 'Match rating' },
  { key: 'notes', label: 'Notes' },
  { key: 'actions', label: '' },
]

function truncate(value: string | null | undefined, len = 40) {
  if (!value) return '—'
  return value.length > len ? `${value.slice(0, len)}…` : value
}

// --- log visit dialog ---
const dialogOpen = ref(false)

const formSchema = toTypedSchema(
  z.object({
    student_id: z.string().uuid('Enter a valid student UUID'),
    date: z.string().min(1, 'Date is required'),
    match_rating: z.coerce.number({ invalid_type_error: 'Pick a rating' }).int().min(1).max(4),
    notes: z.string().optional(),
    suggest: z.string().optional(),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: formSchema,
  initialValues: { student_id: '', date: '', match_rating: undefined, notes: '', suggest: '' },
})
const [studentId, studentIdAttrs] = defineField('student_id')
const [date, dateAttrs] = defineField('date')
const [matchRating, matchRatingAttrs] = defineField('match_rating')
const [notes, notesAttrs] = defineField('notes')
const [suggest, suggestAttrs] = defineField('suggest')

function openCreate() {
  resetForm({ values: { student_id: '', date: '', match_rating: undefined, notes: '', suggest: '' } })
  dialogOpen.value = true
}

function handle422(err: unknown) {
  if (axios.isAxiosError(err) && err.response?.status === 422) {
    toast.error(err.response.data?.message ?? 'Check the form for errors')
  }
}

const createMutation = useMutation({
  mutationFn: (payload: CreateMonitorPayload) => http.post('/monitors', payload),
  onSuccess: () => {
    toast.success('Monitoring visit logged')
    queryClient.invalidateQueries({ queryKey: ['monitors'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const onSubmit = handleSubmit((values) => {
  if (!companyId.value) {
    toast.error('Pick a company first')
    return
  }
  createMutation.mutate({
    student_id: values.student_id,
    company_id: companyId.value,
    date: values.date,
    match_rating: values.match_rating,
    notes: values.notes || undefined,
    suggest: values.suggest || undefined,
  })
})

// --- delete ---
const deleteTarget = ref<Monitor | null>(null)
const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/monitors/${id}`),
  onSuccess: () => {
    toast.success('Monitoring visit deleted')
    queryClient.invalidateQueries({ queryKey: ['monitors'] })
    deleteTarget.value = null
  },
  onSettled: () => {
    deleteTarget.value = null
  },
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Monitoring Visits" description="Log site visits for students placed at companies.">
      <template #actions>
        <Button :disabled="!companyId" @click="openCreate">
          <PlusIcon class="size-4" />
          Log visit
        </Button>
      </template>
    </PageHeader>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3">
        <div class="space-y-1.5">
          <Label for="school-id">School ID</Label>
          <Input id="school-id" v-model="schoolIdInput" type="number" placeholder="e.g. 1" class="w-32" />
        </div>
        <div class="space-y-1.5">
          <Label for="department">Department</Label>
          <Select v-model="departmentId">
            <SelectTrigger id="department" class="w-56">
              <SelectValue placeholder="Select department" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="space-y-1.5">
          <Label for="company">Company</Label>
          <Select v-model="companyId" :disabled="!departmentId">
            <SelectTrigger id="company" class="w-56">
              <SelectValue placeholder="Select company" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="c in companies" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="space-y-1.5">
          <Label for="student-filter">Student ID (optional)</Label>
          <Input id="student-filter" v-model="studentIdFilter" placeholder="Filter by UUID" class="w-56" />
        </div>
      </CardContent>
    </Card>

    <DataTable
      :columns="columns"
      :rows="listQuery.items.value"
      :is-loading="listQuery.isLoading.value"
      :sort="listQuery.sort.value"
      :order="listQuery.order.value"
      empty-title="No monitoring visits logged"
      empty-description="Pick a company (and optionally a student) above, then log a visit."
      @sort="(key) => listQuery.setParams({ sort: key, order: listQuery.order.value === 'asc' ? 'desc' : 'asc' })"
    >
      <template #cell-student_id="{ row }">
        <span class="font-mono text-xs" :title="row.student_id">{{ row.student_id.slice(0, 8) }}…</span>
      </template>
      <template #cell-match_rating="{ row }">
        <span class="inline-flex items-center gap-1 font-medium">
          <StarIcon class="size-3.5 text-warning" />
          {{ row.match_rating }}/4
        </span>
      </template>
      <template #cell-notes="{ row }">
        <span class="text-muted-foreground">{{ truncate(row.notes) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex justify-end">
          <Button variant="ghost" size="icon-sm" aria-label="Delete monitoring visit" @click="deleteTarget = row">
            <Trash2Icon class="size-4 text-destructive" />
          </Button>
        </div>
      </template>
    </DataTable>

    <ListPagination
      v-if="listQuery.pagination.value"
      :page="listQuery.page.value"
      :limit="listQuery.limit.value"
      :total="listQuery.pagination.value.total"
      @update:page="(p) => listQuery.setParams({ page: p })"
    />

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Log monitoring visit</DialogTitle>
          <DialogDescription>Paste the student's UUID — no picker yet.</DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <Label for="mv-student">Student ID (UUID)</Label>
            <Input id="mv-student" v-model="studentId" v-bind="studentIdAttrs" placeholder="00000000-0000-0000-0000-000000000000" />
            <p v-if="errors.student_id" class="text-sm text-destructive">{{ errors.student_id }}</p>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label for="mv-date">Date</Label>
              <Input id="mv-date" v-model="date" v-bind="dateAttrs" type="date" />
              <p v-if="errors.date" class="text-sm text-destructive">{{ errors.date }}</p>
            </div>
            <div class="space-y-1.5">
              <Label for="mv-rating">Match rating</Label>
              <Select v-model="matchRating" v-bind="matchRatingAttrs">
                <SelectTrigger id="mv-rating" class="w-full">
                  <SelectValue placeholder="1–4" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="n in [1, 2, 3, 4]" :key="n" :value="n">{{ n }}</SelectItem>
                </SelectContent>
              </Select>
              <p v-if="errors.match_rating" class="text-sm text-destructive">{{ errors.match_rating }}</p>
            </div>
          </div>
          <div class="space-y-1.5">
            <Label for="mv-notes">Notes</Label>
            <Textarea id="mv-notes" v-model="notes" v-bind="notesAttrs" rows="2" />
          </div>
          <div class="space-y-1.5">
            <Label for="mv-suggest">Suggestions</Label>
            <Textarea id="mv-suggest" v-model="suggest" v-bind="suggestAttrs" rows="2" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" :disabled="createMutation.isPending.value" @click="dialogOpen = false">
              Cancel
            </Button>
            <Button type="submit" :disabled="createMutation.isPending.value">
              {{ createMutation.isPending.value ? 'Saving…' : 'Log visit' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="!!deleteTarget"
      title="Delete monitoring visit?"
      description="This permanently removes this visit log. Only the coordinator who logged it, or an admin, can delete it."
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => !v && (deleteTarget = null)"
      @confirm="deleteTarget && deleteMutation.mutate(deleteTarget.id)"
    />
  </div>
</template>
