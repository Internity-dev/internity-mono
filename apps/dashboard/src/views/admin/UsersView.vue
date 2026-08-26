<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useQuery, useMutation } from '@tanstack/vue-query'
import { isAxiosError } from 'axios'
import { toast } from 'vue-sonner'
import { CopyIcon, TicketIcon, AlertCircleIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useListQuery } from '@/composables/useListQuery'
import { useAuthStore } from '@/stores/auth'
import type { ApiSuccess, ApiErrorBody, User, Role } from '@/types/api'
import type { School, Department, Course, Company, InviteCode, InviteCodeInput } from '@/types/orgs'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable from '@/components/shared/DataTable.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'

const auth = useAuthStore()
const isAdmin = computed(() => auth.user?.role === 'admin')

// --- department -> course cascade (mirrors CoursesView.vue's scoping);
// doubles as the department-name lookup for the user table's scope column ---

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

// --- school/company name lookups for the user table — admin-only queries,
// since GET /schools is admin-only and a coordinator's user list never
// includes mentors (see ListUsers RBAC comment in identity/service.go), so
// they never need a company name resolved either. ---

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

const companiesPicker = useQuery({
  queryKey: ['companies-picker-all'],
  queryFn: async () => {
    const res = await http.get<ApiSuccess<Company[]>>('/companies', { params: { limit: 100, sort: 'name', order: 'asc' } })
    return res.data.data
  },
  enabled: () => isAdmin.value,
})

const companyNameById = computed(() => {
  const map = new Map<number, string>()
  for (const c of companiesPicker.data.value ?? []) map.set(c.id, c.name)
  return map
})

// --- user directory ---

const roleFilter = ref<Role | 'all'>('all')

const {
  items: users,
  pagination,
  page,
  limit,
  search,
  sort,
  order,
  setParams,
  isLoading: isUsersLoading,
  isError: isUsersError,
  refetch: refetchUsers,
} = useListQuery<User>(
  'users',
  async (params) => {
    const res = await http.get<ApiSuccess<User[]>>('/users', { params })
    return res.data
  },
  {
    defaultSort: 'created_at',
    defaultOrder: 'desc',
    extraParams: () => ({ role: roleFilter.value === 'all' ? undefined : roleFilter.value }),
  },
)

interface UserColumn {
  key: string
  label: string
  sortable?: boolean
  class?: string
}

const userColumns: UserColumn[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'email', label: 'Email', sortable: true },
  { key: 'role', label: 'Role' },
  { key: 'scope', label: 'School / Department / Company' },
  { key: 'created_at', label: 'Created', sortable: true },
]

function onUsersSort(key: string) {
  if (key === 'scope') return
  setParams({ sort: key, order: sort.value === key && order.value === 'asc' ? 'desc' : 'asc' })
}

// The most specific scope column a row has — company for a mentor,
// department for a student, school for a coordinator, none for an admin.
function scopeLabel(row: User): string {
  if (row.company_id != null) return companyNameById.value.get(row.company_id) ?? `Company #${row.company_id}`
  if (row.department_id != null) return departmentNameById.value.get(row.department_id) ?? `Department #${row.department_id}`
  if (row.school_id != null) return schoolNameById.value.get(row.school_id) ?? `School #${row.school_id}`
  return '—'
}

function formatUserDate(value: string) {
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

// --- invite code form ---

const schema = toTypedSchema(
  z.object({
    department_id: z.string().min(1, 'Select a department'),
    course_id: z.string().min(1, 'Select a course'),
    expires_at: z.string().optional().or(z.literal('')),
  }),
)

const { defineField, handleSubmit, errors, setErrors, setFieldValue } = useForm({
  validationSchema: schema,
  initialValues: { department_id: '', course_id: '', expires_at: '' },
})
const [departmentIdField, departmentIdAttrs] = defineField('department_id')
const [courseIdField, courseIdAttrs] = defineField('course_id')
const [expiresAt, expiresAtAttrs] = defineField('expires_at')

// Course choices depend entirely on which department is selected — reset the
// course whenever the department changes so a stale course_id from a
// different department can never be submitted.
watch(departmentIdField, () => {
  setFieldValue('course_id', '')
})

const coursesPicker = useQuery({
  queryKey: ['courses-picker', departmentIdField],
  queryFn: async () => {
    const res = await http.get<ApiSuccess<Course[]>>('/courses', {
      params: { department_id: Number(departmentIdField.value), limit: 100, sort: 'name', order: 'asc' },
    })
    return res.data.data
  },
  enabled: () => !!departmentIdField.value,
})

const generatedCodes = ref<InviteCode[]>([])

const createInviteCodeMutation = useMutation({
  mutationFn: (payload: InviteCodeInput) => http.post<ApiSuccess<InviteCode>>('/invite-codes', payload),
  onSuccess: (res) => {
    generatedCodes.value.unshift(res.data.data)
    toast.success('Invite code created')
    setFieldValue('expires_at', '')
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to create invite code'))
  },
})

const isSubmitting = ref(false)

const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  try {
    await createInviteCodeMutation.mutateAsync({
      course_id: Number(values.course_id),
      expires_at: values.expires_at ? new Date(values.expires_at).toISOString() : undefined,
    })
  } catch {
    // handled in mutation onError
  } finally {
    isSubmitting.value = false
  }
})

async function copyCode(code: string) {
  try {
    await navigator.clipboard.writeText(code)
    toast.success('Code copied to clipboard')
  } catch {
    toast.error('Could not copy. Copy it manually instead')
  }
}

function formatDate(value: string) {
  return new Date(value).toLocaleString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Users" description="Manage the people using Internity." />

    <ListToolbar :model-value="search" placeholder="Search by name or email…" @update:model-value="(v) => setParams({ search: v })">
      <Select v-model="roleFilter">
        <SelectTrigger class="w-44">
          <SelectValue placeholder="All roles" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All roles</SelectItem>
          <SelectItem value="admin">Admin</SelectItem>
          <SelectItem value="coordinator">Coordinator</SelectItem>
          <SelectItem value="mentor">Mentor</SelectItem>
          <SelectItem value="student">Student</SelectItem>
        </SelectContent>
      </Select>
    </ListToolbar>

    <EmptyState
      v-if="isUsersError"
      :icon="AlertCircleIcon"
      title="Couldn't load users"
      description="Something went wrong. Try again."
      action-label="Retry"
      @action="refetchUsers()"
    />

    <template v-else>
      <DataTable
        :columns="userColumns"
        :rows="users"
        :is-loading="isUsersLoading"
        :sort="sort"
        :order="order"
        empty-title="No users found"
        empty-description="Try adjusting your search or role filter."
        @sort="onUsersSort"
      >
        <template #cell-name="{ row }">
          <span class="font-medium text-foreground">{{ row.name }}</span>
        </template>
        <template #cell-email="{ row }">
          <span class="text-muted-foreground">{{ row.email }}</span>
        </template>
        <template #cell-role="{ row }">
          <Badge variant="secondary" class="capitalize">{{ row.role }}</Badge>
        </template>
        <template #cell-scope="{ row }">
          <span class="text-muted-foreground">{{ scopeLabel(row) }}</span>
        </template>
        <template #cell-created_at="{ row }">
          <span class="text-muted-foreground">{{ formatUserDate(row.created_at) }}</span>
        </template>
      </DataTable>

      <ListPagination :page="page" :limit="limit" :total="pagination?.total ?? 0" @update:page="(p) => setParams({ page: p })" />
    </template>

    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2">
          <TicketIcon class="size-4" />
          Generate invite code
        </CardTitle>
        <CardDescription>Create a code students can use to self-register into a course.</CardDescription>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="space-y-1.5">
              <label for="invite-department" class="text-sm font-medium">Department</label>
              <Select v-model="departmentIdField" v-bind="departmentIdAttrs">
                <SelectTrigger id="invite-department" class="w-full">
                  <SelectValue placeholder="Select a department" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="d in departmentsPicker.data.value" :key="d.id" :value="String(d.id)">{{ d.name }}</SelectItem>
                </SelectContent>
              </Select>
              <p v-if="errors.department_id" class="text-sm text-destructive">{{ errors.department_id }}</p>
              <p v-if="departmentsPicker.isSuccess.value && departmentsPicker.data.value?.length === 0" class="text-sm text-muted-foreground">
                No departments available yet.
              </p>
            </div>
            <div class="space-y-1.5">
              <label for="invite-course" class="text-sm font-medium">Course</label>
              <Select v-model="courseIdField" v-bind="courseIdAttrs" :disabled="!departmentIdField">
                <SelectTrigger id="invite-course" class="w-full">
                  <SelectValue placeholder="Select a course" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="c in coursesPicker.data.value" :key="c.id" :value="String(c.id)">{{ c.name }}</SelectItem>
                </SelectContent>
              </Select>
              <p v-if="errors.course_id" class="text-sm text-destructive">{{ errors.course_id }}</p>
              <p v-if="departmentIdField && coursesPicker.isSuccess.value && coursesPicker.data.value?.length === 0" class="text-sm text-muted-foreground">
                This department has no courses yet.
              </p>
            </div>
          </div>

          <div class="space-y-1.5">
            <label for="invite-expires" class="text-sm font-medium">Expires at (optional)</label>
            <Input id="invite-expires" v-model="expiresAt" v-bind="expiresAtAttrs" type="datetime-local" class="max-w-xs" />
            <p v-if="errors.expires_at" class="text-sm text-destructive">{{ errors.expires_at }}</p>
          </div>

          <Button type="submit" :disabled="isSubmitting">
            {{ isSubmitting ? 'Generating…' : 'Generate invite code' }}
          </Button>
        </form>

        <div v-if="generatedCodes.length > 0" class="mt-6 space-y-2 border-t pt-4">
          <p class="text-sm font-medium text-foreground">Recently generated</p>
          <div v-for="code in generatedCodes" :key="code.id" class="flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3">
            <div class="flex items-center gap-2">
              <Badge variant="outline" class="font-mono text-sm tracking-wider">{{ code.code }}</Badge>
              <span class="text-xs text-muted-foreground">
                {{ code.expires_at ? `Expires ${formatDate(code.expires_at)}` : 'No expiry' }}
              </span>
            </div>
            <Button variant="ghost" size="icon-sm" aria-label="Copy code" @click="copyCode(code.code)">
              <CopyIcon class="size-4" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
