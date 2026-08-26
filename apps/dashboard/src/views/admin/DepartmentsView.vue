<script setup lang="ts">
import { ref, computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { isAxiosError } from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, PencilIcon, Trash2Icon, AlertCircleIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useListQuery } from '@/composables/useListQuery'
import { useAuthStore } from '@/stores/auth'
import { activeStatus } from '@/lib/status'
import type { ApiSuccess, ApiErrorBody } from '@/types/api'
import type { School, Department, DepartmentInput, DepartmentPatchInput } from '@/types/orgs'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable from '@/components/shared/DataTable.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

const auth = useAuthStore()
const isAdmin = computed(() => auth.user?.role === 'admin')

// --- schools picker (admin only — a coordinator can't call GET /schools at
// all, the backend 403s it, so this query never fires for them) ---

const schoolsPicker = useQuery({
  queryKey: ['schools-picker'],
  queryFn: async () => {
    const res = await http.get<ApiSuccess<School[]>>('/schools', { params: { limit: 100, sort: 'name', order: 'asc' } })
    return res.data.data
  },
  enabled: () => isAdmin.value,
})

const schoolNameById = computed(() => {
  const map = new Map<number, string>()
  for (const s of schoolsPicker.data.value ?? []) map.set(s.id, s.name)
  return map
})

// --- list query, scoped by school ---

const { items, pagination, page, limit, search, sort, order, filters, setParams, isLoading, isError, refetch } =
  useListQuery<Department, 'school_id'>(
    'departments',
    async (params) => {
      // A coordinator is always scoped to their own school regardless of
      // the URL's school_id — only an admin's choice is honored.
      const res = await http.get<ApiSuccess<Department[]>>('/departments', {
        params: { ...params, school_id: isAdmin.value ? params.school_id : auth.user?.school_id },
      })
      return res.data
    },
    {
      defaultSort: 'name',
      defaultOrder: 'asc',
      filters: ['school_id'],
    },
  )

// Only an admin sees the school filter at all — a coordinator is always
// scoped to their own school regardless of the URL.
const schoolFilterModel = computed<string>({
  get: () => filters.value.school_id ?? 'all',
  set: (v) => setParams({ school_id: v === 'all' ? undefined : v }),
})

interface TableColumn {
  key: string
  label: string
  sortable?: boolean
  class?: string
}

const columns = computed<TableColumn[]>(() => [
  { key: 'name', label: 'Name', sortable: true },
  ...(isAdmin.value ? [{ key: 'school', label: 'School' }] : []),
  { key: 'study_program', label: 'Study program' },
  { key: 'is_active', label: 'Status' },
  { key: 'created_at', label: 'Created', sortable: true },
  { key: 'actions', label: '', class: 'w-24 text-right' },
])

function onSort(key: string) {
  if (key === 'actions' || key === 'school' || key === 'study_program') return
  setParams({ sort: key, order: sort.value === key && order.value === 'asc' ? 'desc' : 'asc' })
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

// --- shared error helpers ---

function errorMessage(err: unknown, fallback: string): string {
  if (isAxiosError(err)) {
    const body = err.response?.data as ApiErrorBody | undefined
    if (body?.message) return body.message
  }
  return fallback
}

function fieldErrors(err: unknown): Record<string, string> {
  if (isAxiosError(err)) {
    const body = err.response?.data as ApiErrorBody | undefined
    const details = body?.error?.details
    if (details?.length) {
      const map: Record<string, string> = {}
      for (const d of details) {
        if (d.field) map[d.field] = d.issue === 'required' ? 'This field is required' : `Invalid: ${d.issue}`
      }
      return map
    }
  }
  return {}
}

// --- create / edit dialog ---

const dialogOpen = ref(false)
const editingId = ref<number | null>(null)

const schema = toTypedSchema(
  z.object({
    school_id: z.string().min(1, 'Select a school'),
    name: z.string().trim().min(2, 'At least 2 characters').max(255, 'Too long'),
    description: z.string().trim().optional().or(z.literal('')),
    study_program: z.string().trim().max(255, 'Too long').optional().or(z.literal('')),
  }),
)

const { defineField, handleSubmit, errors, resetForm, setErrors } = useForm({
  validationSchema: schema,
  initialValues: { school_id: '', name: '', description: '', study_program: '' },
})
const [schoolIdField, schoolIdAttrs] = defineField('school_id')
const [name, nameAttrs] = defineField('name')
const [description, descriptionAttrs] = defineField('description')
const [studyProgram, studyProgramAttrs] = defineField('study_program')

function openCreate() {
  editingId.value = null
  resetForm({
    values: {
      school_id: isAdmin.value ? '' : String(auth.user?.school_id ?? ''),
      name: '',
      description: '',
      study_program: '',
    },
  })
  dialogOpen.value = true
}

function openEdit(row: Department) {
  editingId.value = row.id
  resetForm({
    values: {
      school_id: String(row.school_id),
      name: row.name,
      description: row.description ?? '',
      study_program: row.study_program ?? '',
    },
  })
  dialogOpen.value = true
}

const queryClient = useQueryClient()

const createMutation = useMutation({
  mutationFn: (payload: DepartmentInput) => http.post<ApiSuccess<Department>>('/departments', payload),
  onSuccess: () => {
    toast.success('Department created')
    queryClient.invalidateQueries({ queryKey: ['departments'] })
    dialogOpen.value = false
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to create department'))
  },
})

const updateMutation = useMutation({
  mutationFn: ({ id, payload }: { id: number; payload: DepartmentPatchInput }) =>
    http.put<ApiSuccess<Department>>(`/departments/${id}`, payload),
  onSuccess: () => {
    toast.success('Department updated')
    queryClient.invalidateQueries({ queryKey: ['departments'] })
    dialogOpen.value = false
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to update department'))
  },
})

const isSubmitting = ref(false)

const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  try {
    if (editingId.value !== null) {
      await updateMutation.mutateAsync({
        id: editingId.value,
        payload: {
          name: values.name.trim(),
          description: values.description?.trim() || undefined,
          study_program: values.study_program?.trim() || undefined,
        },
      })
    } else {
      await createMutation.mutateAsync({
        school_id: Number(values.school_id),
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
        study_program: values.study_program?.trim() || undefined,
      })
    }
  } catch {
    // handled in mutation onError
  } finally {
    isSubmitting.value = false
  }
})

// --- delete ---

const deleteTarget = ref<Department | null>(null)

const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/departments/${id}`),
  onSuccess: () => {
    toast.success('Department deleted')
    queryClient.invalidateQueries({ queryKey: ['departments'] })
    deleteTarget.value = null
  },
  onError: (err: unknown) => {
    // 409 CONFLICT — courses/companies still reference this department.
    toast.error(errorMessage(err, 'Failed to delete department'))
  },
})

function confirmDelete() {
  if (deleteTarget.value) deleteMutation.mutate(deleteTarget.value.id)
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Departments" description="Manage departments within a school.">
      <template #actions>
        <Button size="sm" @click="openCreate">
          <PlusIcon class="size-4" />
          Add department
        </Button>
      </template>
    </PageHeader>

    <ListToolbar :model-value="search" placeholder="Search departments…" @update:model-value="(v) => setParams({ search: v })">
      <Select v-if="isAdmin" v-model="schoolFilterModel">
        <SelectTrigger class="w-48">
          <SelectValue placeholder="All schools" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All schools</SelectItem>
          <SelectItem v-for="s in schoolsPicker.data.value" :key="s.id" :value="String(s.id)">{{ s.name }}</SelectItem>
        </SelectContent>
      </Select>
    </ListToolbar>

    <EmptyState
      v-if="isError"
      :icon="AlertCircleIcon"
      title="Couldn't load departments"
      description="Please try again."
      action-label="Retry"
      @action="refetch()"
    />

    <template v-else>
      <DataTable
        :columns="columns"
        :rows="items"
        :is-loading="isLoading"
        :sort="sort"
        :order="order"
        empty-title="No departments yet"
        empty-description="Get started by adding your first department."
        @sort="onSort"
      >
        <template #cell-name="{ row }">
          <span class="font-medium text-foreground">{{ row.name }}</span>
        </template>
        <template #cell-school="{ row }">
          <span class="text-muted-foreground">{{ schoolNameById.get(row.school_id) ?? `School #${row.school_id}` }}</span>
        </template>
        <template #cell-study_program="{ row }">
          <span class="text-muted-foreground">{{ row.study_program || '—' }}</span>
        </template>
        <template #cell-is_active="{ row }">
          <StatusBadge v-bind="activeStatus(row.is_active)" />
        </template>
        <template #cell-created_at="{ row }">
          <span class="text-muted-foreground">{{ formatDate(row.created_at) }}</span>
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end gap-1">
            <Button variant="ghost" size="icon-sm" aria-label="Edit department" @click="openEdit(row)">
              <PencilIcon class="size-4" />
            </Button>
            <Button variant="ghost" size="icon-sm" aria-label="Delete department" @click="deleteTarget = row">
              <Trash2Icon class="size-4 text-destructive" />
            </Button>
          </div>
        </template>
      </DataTable>

      <ListPagination :page="page" :limit="limit" :total="pagination?.total ?? 0" @update:page="(p) => setParams({ page: p })" />
    </template>

    <Dialog v-model:open="dialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ editingId !== null ? 'Edit department' : 'Add department' }}</DialogTitle>
          <DialogDescription>
            {{ editingId !== null ? 'Update the details for this department.' : 'Create a new department.' }}
          </DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div v-if="editingId === null && isAdmin" class="space-y-1.5">
            <label for="dept-school" class="text-sm font-medium">School</label>
            <Select v-model="schoolIdField" v-bind="schoolIdAttrs">
              <SelectTrigger id="dept-school" class="w-full">
                <SelectValue placeholder="Select a school" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="s in schoolsPicker.data.value" :key="s.id" :value="String(s.id)">{{ s.name }}</SelectItem>
              </SelectContent>
            </Select>
            <p v-if="errors.school_id" class="text-sm text-destructive">{{ errors.school_id }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="dept-name" class="text-sm font-medium">Name</label>
            <Input id="dept-name" v-model="name" v-bind="nameAttrs" />
            <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="dept-study-program" class="text-sm font-medium">Study program</label>
            <Input id="dept-study-program" v-model="studyProgram" v-bind="studyProgramAttrs" />
            <p v-if="errors.study_program" class="text-sm text-destructive">{{ errors.study_program }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="dept-description" class="text-sm font-medium">Description</label>
            <Textarea id="dept-description" v-model="description" v-bind="descriptionAttrs" rows="3" />
            <p v-if="errors.description" class="text-sm text-destructive">{{ errors.description }}</p>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isSubmitting" @click="dialogOpen = false">Cancel</Button>
            <Button type="submit" :disabled="isSubmitting">
              {{ isSubmitting ? 'Saving…' : editingId !== null ? 'Save changes' : 'Create department' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="deleteTarget !== null"
      title="Delete department?"
      :description="`This will permanently remove '${deleteTarget?.name}'.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => { if (!v) deleteTarget = null }"
      @confirm="confirmDelete"
    />
  </div>
</template>
