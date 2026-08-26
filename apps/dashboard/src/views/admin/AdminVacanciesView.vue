<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { PlusIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery, type FetcherParams } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { Vacancy, VacancyStatus } from '@/types/vacancy'
import { vacancyStatus } from '@/lib/status'

import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

// --- org-scope pickers -------------------------------------------------
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

const auth = useAuthStore()
const isMentor = computed(() => auth.user?.role === 'mentor')
const route = useRoute()

// department_id is a UI-only cascade key (narrows the company picker, never
// itself sent to /vacancies); company_id is resolved to effectiveCompanyId
// below (a mentor's own company overrides whatever's in the URL) before
// being sent, so it's also read directly rather than auto-forwarded.
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
  // Changing department resets company in the same atomic URL update — no
  // separate watcher needed.
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
  companyNameQuery.data.value ? `Managing vacancies for ${companyNameQuery.data.value.name}.` : 'Select a company to manage its vacancies.',
)

// --- vacancy list --------------------------------------------------------

function fetchVacancies(params: FetcherParams<'department_id' | 'company_id'>) {
  return http.get<ApiSuccess<Vacancy[]>>('/vacancies', { params: { ...params, company_id: effectiveCompanyId.value } }).then((r) => r.data)
}

const list = useListQuery<Vacancy, 'department_id' | 'company_id'>('vacancies', fetchVacancies, {
  filters: [{ key: 'department_id', sendToFetcher: false }, { key: 'company_id', sendToFetcher: false }],
  enabled: () => !!effectiveCompanyId.value,
})

const columns: Column[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'category', label: 'Category' },
  { key: 'slots', label: 'Slots' },
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

const searchModel = computed({
  get: () => list.search.value,
  set: (v: string) => list.setParams({ search: v }),
})

// --- create / edit dialog --------------------------------------------------

const vacancyFormSchema = toTypedSchema(
  z.object({
    name: z.string().min(2, 'Enter a name').max(255),
    category: z.string().max(255).optional(),
    description: z.string().optional(),
    skills: z.string().optional(),
    slots: z.coerce.number().int().min(1, 'At least 1 slot'),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({ validationSchema: vacancyFormSchema })
const [name, nameAttrs] = defineField('name')
const [category, categoryAttrs] = defineField('category')
const [description, descriptionAttrs] = defineField('description')
const [skills, skillsAttrs] = defineField('skills')
const [slots, slotsAttrs] = defineField('slots')

const formOpen = ref(false)
const editingVacancy = ref<Vacancy | null>(null)

function openCreate() {
  editingVacancy.value = null
  resetForm({ values: { name: '', category: '', description: '', skills: '', slots: 1 } })
  formOpen.value = true
}

function openEdit(v: Vacancy) {
  editingVacancy.value = v
  resetForm({
    values: {
      name: v.name,
      category: v.category ?? '',
      description: v.description ?? '',
      skills: v.skills ?? '',
      slots: v.slots,
    },
  })
  formOpen.value = true
}

function orUndef(v?: string) {
  const trimmed = v?.trim()
  return trimmed ? trimmed : undefined
}

const queryClient = useQueryClient()

const saveMutation = useMutation({
  mutationFn: async (values: { name: string; category?: string; description?: string; skills?: string; slots: number }) => {
    const payload = {
      name: values.name,
      category: orUndef(values.category),
      description: orUndef(values.description),
      skills: orUndef(values.skills),
      slots: values.slots,
    }
    if (editingVacancy.value) {
      return http.put(`/vacancies/${editingVacancy.value.id}`, payload)
    }
    return http.post('/vacancies', { ...payload, company_id: effectiveCompanyId.value })
  },
  onSuccess: () => {
    toast.success(editingVacancy.value ? 'Vacancy updated' : 'Vacancy created')
    formOpen.value = false
    queryClient.invalidateQueries({ queryKey: ['vacancies'] })
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})

const onSubmit = handleSubmit((values) => saveMutation.mutate(values))

// --- open/close toggle ---

const toggleStatusMutation = useMutation({
  mutationFn: (v: Vacancy) =>
    http.put(`/vacancies/${v.id}`, { status: (v.status === 'open' ? 'closed' : 'open') as VacancyStatus }),
  onSuccess: () => {
    toast.success('Vacancy status updated')
    queryClient.invalidateQueries({ queryKey: ['vacancies'] })
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})

// --- delete ---

const deleteTarget = ref<Vacancy | null>(null)
const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/vacancies/${id}`),
  onSuccess: () => {
    toast.success('Vacancy deleted')
    deleteTarget.value = null
    queryClient.invalidateQueries({ queryKey: ['vacancies'] })
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})
</script>

<template>
  <div class="space-y-6 p-6">
    <PageHeader title="Vacancies" :description="pageDescription">
      <template #actions>
        <Button :disabled="!effectiveCompanyId" @click="openCreate">
          <PlusIcon class="size-4" />
          Add vacancy
        </Button>
      </template>
    </PageHeader>

    <Card v-if="!isMentor">
      <CardContent class="flex flex-wrap items-end gap-4 pt-0">
        <div class="w-56 space-y-1.5">
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
        <div class="w-56 space-y-1.5">
          <label class="text-sm font-medium">Company</label>
          <Select v-model="companyModel" :disabled="!departmentId">
            <SelectTrigger class="w-full"><SelectValue placeholder="Select company" /></SelectTrigger>
            <SelectContent>
              <SelectItem v-for="c in companiesQuery.data.value ?? []" :key="c.id" :value="String(c.id)">
                {{ c.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </CardContent>
    </Card>

    <template v-if="effectiveCompanyId">
      <ListToolbar v-model="searchModel" placeholder="Search vacancies…" />

      <DataTable
        :columns="columns"
        :rows="list.items.value"
        :is-loading="list.isLoading.value"
        :sort="list.sort.value"
        :order="list.order.value"
        empty-title="No vacancies yet"
        empty-description="Create your first vacancy to start receiving applications."
        @sort="onSort"
      >
        <template #cell-name="{ row }">
          <span class="font-medium text-foreground">{{ row.name }}</span>
        </template>
        <template #cell-category="{ row }">{{ row.category ?? '—' }}</template>
        <template #cell-slots="{ row }">{{ row.slots }}</template>
        <template #cell-status="{ row }">
          <StatusBadge v-bind="vacancyStatus(row.status)" />
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end gap-2">
            <Button size="sm" variant="outline" @click="toggleStatusMutation.mutate(row)">
              {{ row.status === 'open' ? 'Close' : 'Reopen' }}
            </Button>
            <Button size="sm" variant="outline" @click="openEdit(row)">Edit</Button>
            <Button size="sm" variant="destructive" @click="deleteTarget = row">Delete</Button>
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
    <EmptyState
      v-else
      title="Select a company"
      description="Choose a department and company above to manage its vacancies."
    />

    <Dialog :open="formOpen" @update:open="formOpen = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editingVacancy ? 'Edit vacancy' : 'Add vacancy' }}</DialogTitle>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <label for="v-name" class="text-sm font-medium">Name</label>
            <Input id="v-name" v-model="name" v-bind="nameAttrs" />
            <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="v-category" class="text-sm font-medium">Category</label>
            <Input id="v-category" v-model="category" v-bind="categoryAttrs" />
            <p v-if="errors.category" class="text-sm text-destructive">{{ errors.category }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="v-slots" class="text-sm font-medium">Slots</label>
            <Input id="v-slots" v-model="slots" v-bind="slotsAttrs" type="number" min="1" />
            <p v-if="errors.slots" class="text-sm text-destructive">{{ errors.slots }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="v-skills" class="text-sm font-medium">Skills</label>
            <Input id="v-skills" v-model="skills" v-bind="skillsAttrs" placeholder="e.g. Figma, HTML, CSS" />
            <p v-if="errors.skills" class="text-sm text-destructive">{{ errors.skills }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="v-description" class="text-sm font-medium">Description</label>
            <Textarea id="v-description" v-model="description" v-bind="descriptionAttrs" rows="4" />
            <p v-if="errors.description" class="text-sm text-destructive">{{ errors.description }}</p>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="formOpen = false">Cancel</Button>
            <Button type="submit" :disabled="saveMutation.isPending.value">
              {{ saveMutation.isPending.value ? 'Saving…' : 'Save' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="!!deleteTarget"
      title="Delete vacancy?"
      :description="`This will permanently remove '${deleteTarget?.name}'.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => { if (!v) deleteTarget = null }"
      @confirm="deleteTarget && deleteMutation.mutate(deleteTarget.id)"
    />
  </div>
</template>
