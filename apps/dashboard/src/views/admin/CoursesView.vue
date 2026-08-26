<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
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
import type { Department, Course, CourseInput, CoursePatchInput } from '@/types/orgs'
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
import { Checkbox } from '@/components/ui/checkbox'
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
const router = useRouter()
const route = useRoute()

// --- departments picker: scoped to the coordinator's own school, or every
// department for an admin. Backing both the filter select and the create
// form's department_id field. ---

const departmentsPicker = useQuery({
  queryKey: ['departments-picker', isAdmin.value ? undefined : auth.user?.school_id],
  queryFn: async () => {
    const res = await http.get<ApiSuccess<Department[]>>('/departments', {
      params: {
        limit: 100,
        sort: 'name',
        order: 'asc',
        school_id: isAdmin.value ? undefined : auth.user?.school_id,
      },
    })
    return res.data.data
  },
})

const departmentNameById = computed(() => {
  const map = new Map<number, string>()
  for (const d of departmentsPicker.data.value ?? []) map.set(d.id, d.name)
  return map
})

// --- list query ---

const { items, pagination, page, limit, search, sort, order, filters, setParams, isLoading, isError, refetch } =
  useListQuery<Course, 'department_id'>(
    'courses',
    async (params) => {
      const res = await http.get<ApiSuccess<Course[]>>('/courses', { params })
      return res.data
    },
    {
      defaultSort: 'name',
      defaultOrder: 'asc',
      filters: ['department_id'],
      // Read straight from the route query, not the destructured `filters`
      // above — referencing that here would be a circular self-reference
      // (this options object is part of `filters`'s own initializer).
      enabled: () => isAdmin.value || !!route.query.department_id,
    },
  )

// A coordinator must always scope to one of their own departments (the
// backend 403s a course list request with no department_id for anyone but
// an admin), so default to their first department once the picker loads. An
// admin's filter starts unset ('all') since department_id is a genuinely
// optional filter for them.
watch(
  () => departmentsPicker.data.value,
  (depts) => {
    const first = depts?.[0]
    if (!isAdmin.value && !filters.value.department_id && first) {
      setParams({ department_id: String(first.id) })
    }
  },
  { immediate: true },
)

const departmentFilterModel = computed<string>({
  get: () => filters.value.department_id ?? 'all',
  set: (v) => setParams({ department_id: v === 'all' ? undefined : v }),
})

interface TableColumn {
  key: string
  label: string
  sortable?: boolean
  class?: string
}

const columns: TableColumn[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'department', label: 'Department' },
  { key: 'is_active', label: 'Status' },
  { key: 'created_at', label: 'Created', sortable: true },
  { key: 'actions', label: '', class: 'w-24 text-right' },
]

function onSort(key: string) {
  if (key === 'actions' || key === 'department') return
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
    department_id: z.string().min(1, 'Select a department'),
    name: z.string().trim().min(1, 'Name is required').max(255, 'Too long'),
    description: z.string().trim().optional().or(z.literal('')),
    is_active: z.boolean(),
  }),
)

const { defineField, handleSubmit, errors, resetForm, setErrors } = useForm({
  validationSchema: schema,
  initialValues: { department_id: '', name: '', description: '', is_active: true },
})
const [departmentIdField, departmentIdAttrs] = defineField('department_id')
const [name, nameAttrs] = defineField('name')
const [description, descriptionAttrs] = defineField('description')
const [isActiveField] = defineField('is_active')

function openCreate() {
  editingId.value = null
  resetForm({ values: { department_id: filters.value.department_id ?? '', name: '', description: '', is_active: true } })
  dialogOpen.value = true
}

function openEdit(row: Course) {
  editingId.value = row.id
  resetForm({
    values: {
      department_id: String(row.department_id),
      name: row.name,
      description: row.description ?? '',
      is_active: row.is_active,
    },
  })
  dialogOpen.value = true
}

const queryClient = useQueryClient()

const createMutation = useMutation({
  mutationFn: (payload: CourseInput) => http.post<ApiSuccess<Course>>('/courses', payload),
  onSuccess: () => {
    toast.success('Course created')
    queryClient.invalidateQueries({ queryKey: ['courses'] })
    dialogOpen.value = false
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to create course'))
  },
})

const updateMutation = useMutation({
  mutationFn: ({ id, payload }: { id: number; payload: CoursePatchInput }) =>
    http.put<ApiSuccess<Course>>(`/courses/${id}`, payload),
  onSuccess: () => {
    toast.success('Course updated')
    queryClient.invalidateQueries({ queryKey: ['courses'] })
    dialogOpen.value = false
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to update course'))
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
          is_active: values.is_active,
        },
      })
    } else {
      await createMutation.mutateAsync({
        department_id: Number(values.department_id),
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
      })
    }
  } catch {
    // handled in mutation onError
  } finally {
    isSubmitting.value = false
  }
})

// --- delete ---

const deleteTarget = ref<Course | null>(null)

const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/courses/${id}`),
  onSuccess: () => {
    toast.success('Course deleted')
    queryClient.invalidateQueries({ queryKey: ['courses'] })
    deleteTarget.value = null
  },
  onError: (err: unknown) => {
    // 409 CONFLICT — students/invite codes still reference this course.
    toast.error(errorMessage(err, 'Failed to delete course'))
  },
})

function confirmDelete() {
  if (deleteTarget.value) deleteMutation.mutate(deleteTarget.value.id)
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Courses" description="Manage courses (kelas) within a department.">
      <template #actions>
        <Button size="sm" :disabled="departmentsPicker.data.value?.length === 0" @click="openCreate">
          <PlusIcon class="size-4" />
          Add course
        </Button>
      </template>
    </PageHeader>

    <EmptyState
      v-if="!isAdmin && departmentsPicker.isSuccess.value && departmentsPicker.data.value?.length === 0"
      title="No departments yet"
      description="Create a department before adding courses."
      action-label="Go to departments"
      @action="router.push('/admin/departments')"
    />

    <template v-else>
      <ListToolbar :model-value="search" placeholder="Search courses…" @update:model-value="(v) => setParams({ search: v })">
        <Select v-model="departmentFilterModel">
          <SelectTrigger class="w-56">
            <SelectValue placeholder="Select a department" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-if="isAdmin" value="all">All departments</SelectItem>
            <SelectItem v-for="d in departmentsPicker.data.value" :key="d.id" :value="String(d.id)">{{ d.name }}</SelectItem>
          </SelectContent>
        </Select>
      </ListToolbar>

      <EmptyState
        v-if="isError"
        :icon="AlertCircleIcon"
        title="Couldn't load courses"
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
          empty-title="No courses yet"
          empty-description="Get started by adding your first course."
          @sort="onSort"
        >
          <template #cell-name="{ row }">
            <span class="font-medium text-foreground">{{ row.name }}</span>
          </template>
          <template #cell-department="{ row }">
            <span class="text-muted-foreground">{{ departmentNameById.get(row.department_id) ?? `Department #${row.department_id}` }}</span>
          </template>
          <template #cell-is_active="{ row }">
            <StatusBadge v-bind="activeStatus(row.is_active)" />
          </template>
          <template #cell-created_at="{ row }">
            <span class="text-muted-foreground">{{ formatDate(row.created_at) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex justify-end gap-1">
              <Button variant="ghost" size="icon-sm" aria-label="Edit course" @click="openEdit(row)">
                <PencilIcon class="size-4" />
              </Button>
              <Button variant="ghost" size="icon-sm" aria-label="Delete course" @click="deleteTarget = row">
                <Trash2Icon class="size-4 text-destructive" />
              </Button>
            </div>
          </template>
        </DataTable>

        <ListPagination :page="page" :limit="limit" :total="pagination?.total ?? 0" @update:page="(p) => setParams({ page: p })" />
      </template>
    </template>

    <Dialog v-model:open="dialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ editingId !== null ? 'Edit course' : 'Add course' }}</DialogTitle>
          <DialogDescription>
            {{ editingId !== null ? 'Update the details for this course.' : 'Create a new course (kelas).' }}
          </DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div v-if="editingId === null" class="space-y-1.5">
            <label for="course-department" class="text-sm font-medium">Department</label>
            <Select v-model="departmentIdField" v-bind="departmentIdAttrs">
              <SelectTrigger id="course-department" class="w-full">
                <SelectValue placeholder="Select a department" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="d in departmentsPicker.data.value" :key="d.id" :value="String(d.id)">{{ d.name }}</SelectItem>
              </SelectContent>
            </Select>
            <p v-if="errors.department_id" class="text-sm text-destructive">{{ errors.department_id }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="course-name" class="text-sm font-medium">Name</label>
            <Input id="course-name" v-model="name" v-bind="nameAttrs" />
            <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="course-description" class="text-sm font-medium">Description</label>
            <Textarea id="course-description" v-model="description" v-bind="descriptionAttrs" rows="3" />
            <p v-if="errors.description" class="text-sm text-destructive">{{ errors.description }}</p>
          </div>
          <label v-if="editingId !== null" class="flex items-center gap-2 text-sm font-medium">
            <Checkbox v-model="isActiveField" />
            Active
          </label>

          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isSubmitting" @click="dialogOpen = false">Cancel</Button>
            <Button type="submit" :disabled="isSubmitting">
              {{ isSubmitting ? 'Saving…' : editingId !== null ? 'Save changes' : 'Create course' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="deleteTarget !== null"
      title="Delete course?"
      :description="`This will permanently remove '${deleteTarget?.name}'.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => { if (!v) deleteTarget = null }"
      @confirm="confirmDelete"
    />
  </div>
</template>
