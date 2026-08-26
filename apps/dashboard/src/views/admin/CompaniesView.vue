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
import type { Department, Company, CompanyInput, CompanyPatchInput } from '@/types/orgs'
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
  useListQuery<Company, 'department_id'>(
    'companies',
    async (params) => {
      const res = await http.get<ApiSuccess<Company[]>>('/companies', { params })
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
// backend 403s a company list request with no department_id for anyone but
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
  { key: 'city', label: 'City' },
  { key: 'is_active', label: 'Status' },
  { key: 'created_at', label: 'Created', sortable: true },
  { key: 'actions', label: '', class: 'w-24 text-right' },
]

function onSort(key: string) {
  if (key === 'actions' || key === 'department' || key === 'city') return
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
    name: z.string().trim().min(2, 'At least 2 characters').max(255, 'Too long'),
    category: z.string().trim().max(255, 'Too long').optional().or(z.literal('')),
    city: z.string().trim().max(255, 'Too long').optional().or(z.literal('')),
    state: z.string().trim().max(255, 'Too long').optional().or(z.literal('')),
    country: z.string().trim().max(255, 'Too long').optional().or(z.literal('')),
    address: z.string().trim().optional().or(z.literal('')),
    email: z.string().trim().max(255).email('Enter a valid email').optional().or(z.literal('')),
    phone: z.string().trim().max(50, 'Too long').optional().or(z.literal('')),
    website: z.string().trim().url('Enter a valid URL').optional().or(z.literal('')),
    contact_person: z.string().trim().max(255, 'Too long').optional().or(z.literal('')),
    is_active: z.boolean(),
  }),
)

const defaultValues = {
  department_id: '',
  name: '',
  category: '',
  city: '',
  state: '',
  country: '',
  address: '',
  email: '',
  phone: '',
  website: '',
  contact_person: '',
  is_active: true,
}

const { defineField, handleSubmit, errors, resetForm, setErrors } = useForm({
  validationSchema: schema,
  initialValues: defaultValues,
})
const [departmentIdField, departmentIdAttrs] = defineField('department_id')
const [name, nameAttrs] = defineField('name')
const [category, categoryAttrs] = defineField('category')
const [city, cityAttrs] = defineField('city')
const [state, stateAttrs] = defineField('state')
const [country, countryAttrs] = defineField('country')
const [address, addressAttrs] = defineField('address')
const [email, emailAttrs] = defineField('email')
const [phone, phoneAttrs] = defineField('phone')
const [website, websiteAttrs] = defineField('website')
const [contactPerson, contactPersonAttrs] = defineField('contact_person')
const [isActiveField] = defineField('is_active')

function openCreate() {
  editingId.value = null
  resetForm({ values: { ...defaultValues, department_id: filters.value.department_id ?? '' } })
  dialogOpen.value = true
}

function openEdit(row: Company) {
  editingId.value = row.id
  resetForm({
    values: {
      department_id: String(row.department_id),
      name: row.name,
      category: row.category ?? '',
      city: row.city ?? '',
      state: row.state ?? '',
      country: row.country ?? '',
      address: row.address ?? '',
      email: row.email ?? '',
      phone: row.phone ?? '',
      website: row.website ?? '',
      contact_person: row.contact_person ?? '',
      is_active: row.is_active,
    },
  })
  dialogOpen.value = true
}

const queryClient = useQueryClient()

const createMutation = useMutation({
  mutationFn: (payload: CompanyInput) => http.post<ApiSuccess<Company>>('/companies', payload),
  onSuccess: () => {
    toast.success('Company created')
    queryClient.invalidateQueries({ queryKey: ['companies'] })
    dialogOpen.value = false
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to create company'))
  },
})

const updateMutation = useMutation({
  mutationFn: ({ id, payload }: { id: number; payload: CompanyPatchInput }) =>
    http.put<ApiSuccess<Company>>(`/companies/${id}`, payload),
  onSuccess: () => {
    toast.success('Company updated')
    queryClient.invalidateQueries({ queryKey: ['companies'] })
    dialogOpen.value = false
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to update company'))
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
          category: values.category?.trim() || undefined,
          city: values.city?.trim() || undefined,
          state: values.state?.trim() || undefined,
          country: values.country?.trim() || undefined,
          address: values.address?.trim() || undefined,
          email: values.email?.trim() || undefined,
          phone: values.phone?.trim() || undefined,
          website: values.website?.trim() || undefined,
          contact_person: values.contact_person?.trim() || undefined,
          is_active: values.is_active,
        },
      })
    } else {
      await createMutation.mutateAsync({
        department_id: Number(values.department_id),
        name: values.name.trim(),
        category: values.category?.trim() || undefined,
        city: values.city?.trim() || undefined,
        state: values.state?.trim() || undefined,
        country: values.country?.trim() || undefined,
        address: values.address?.trim() || undefined,
        email: values.email?.trim() || undefined,
        phone: values.phone?.trim() || undefined,
        website: values.website?.trim() || undefined,
        contact_person: values.contact_person?.trim() || undefined,
      })
    }
  } catch {
    // handled in mutation onError
  } finally {
    isSubmitting.value = false
  }
})

// --- delete ---

const deleteTarget = ref<Company | null>(null)

const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/companies/${id}`),
  onSuccess: () => {
    toast.success('Company deleted')
    queryClient.invalidateQueries({ queryKey: ['companies'] })
    deleteTarget.value = null
  },
  onError: (err: unknown) => {
    // 409 CONFLICT — vacancies/appliances still reference this company.
    toast.error(errorMessage(err, 'Failed to delete company'))
  },
})

function confirmDelete() {
  if (deleteTarget.value) deleteMutation.mutate(deleteTarget.value.id)
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Companies" description="Manage partner companies within a department.">
      <template #actions>
        <Button size="sm" :disabled="departmentsPicker.data.value?.length === 0" @click="openCreate">
          <PlusIcon class="size-4" />
          Add company
        </Button>
      </template>
    </PageHeader>

    <EmptyState
      v-if="!isAdmin && departmentsPicker.isSuccess.value && departmentsPicker.data.value?.length === 0"
      title="No departments yet"
      description="Create a department before adding companies."
      action-label="Go to departments"
      @action="router.push('/admin/departments')"
    />

    <template v-else>
      <ListToolbar :model-value="search" placeholder="Search companies…" @update:model-value="(v) => setParams({ search: v })">
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
        title="Couldn't load companies"
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
          empty-title="No companies yet"
          empty-description="Get started by adding your first partner company."
          @sort="onSort"
        >
          <template #cell-name="{ row }">
            <span class="font-medium text-foreground">{{ row.name }}</span>
          </template>
          <template #cell-department="{ row }">
            <span class="text-muted-foreground">{{ departmentNameById.get(row.department_id) ?? `Department #${row.department_id}` }}</span>
          </template>
          <template #cell-city="{ row }">
            <span class="text-muted-foreground">{{ row.city || '—' }}</span>
          </template>
          <template #cell-is_active="{ row }">
            <StatusBadge v-bind="activeStatus(row.is_active)" />
          </template>
          <template #cell-created_at="{ row }">
            <span class="text-muted-foreground">{{ formatDate(row.created_at) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex justify-end gap-1">
              <Button variant="ghost" size="icon-sm" aria-label="Edit company" @click="openEdit(row)">
                <PencilIcon class="size-4" />
              </Button>
              <Button variant="ghost" size="icon-sm" aria-label="Delete company" @click="deleteTarget = row">
                <Trash2Icon class="size-4 text-destructive" />
              </Button>
            </div>
          </template>
        </DataTable>

        <ListPagination :page="page" :limit="limit" :total="pagination?.total ?? 0" @update:page="(p) => setParams({ page: p })" />
      </template>
    </template>

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="max-h-[85vh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editingId !== null ? 'Edit company' : 'Add company' }}</DialogTitle>
          <DialogDescription>
            {{ editingId !== null ? 'Update the details for this company.' : 'Add a new partner company.' }}
          </DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div v-if="editingId === null" class="space-y-1.5">
            <label for="company-department" class="text-sm font-medium">Department</label>
            <Select v-model="departmentIdField" v-bind="departmentIdAttrs">
              <SelectTrigger id="company-department" class="w-full">
                <SelectValue placeholder="Select a department" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="d in departmentsPicker.data.value" :key="d.id" :value="String(d.id)">{{ d.name }}</SelectItem>
              </SelectContent>
            </Select>
            <p v-if="errors.department_id" class="text-sm text-destructive">{{ errors.department_id }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="company-name" class="text-sm font-medium">Name</label>
            <Input id="company-name" v-model="name" v-bind="nameAttrs" />
            <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <label for="company-category" class="text-sm font-medium">Category</label>
              <Input id="company-category" v-model="category" v-bind="categoryAttrs" />
              <p v-if="errors.category" class="text-sm text-destructive">{{ errors.category }}</p>
            </div>
            <div class="space-y-1.5">
              <label for="company-contact-person" class="text-sm font-medium">Contact person</label>
              <Input id="company-contact-person" v-model="contactPerson" v-bind="contactPersonAttrs" />
              <p v-if="errors.contact_person" class="text-sm text-destructive">{{ errors.contact_person }}</p>
            </div>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <label for="company-email" class="text-sm font-medium">Email</label>
              <Input id="company-email" v-model="email" v-bind="emailAttrs" type="email" />
              <p v-if="errors.email" class="text-sm text-destructive">{{ errors.email }}</p>
            </div>
            <div class="space-y-1.5">
              <label for="company-phone" class="text-sm font-medium">Phone</label>
              <Input id="company-phone" v-model="phone" v-bind="phoneAttrs" />
              <p v-if="errors.phone" class="text-sm text-destructive">{{ errors.phone }}</p>
            </div>
          </div>
          <div class="space-y-1.5">
            <label for="company-website" class="text-sm font-medium">Website</label>
            <Input id="company-website" v-model="website" v-bind="websiteAttrs" placeholder="https://" />
            <p v-if="errors.website" class="text-sm text-destructive">{{ errors.website }}</p>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div class="space-y-1.5">
              <label for="company-city" class="text-sm font-medium">City</label>
              <Input id="company-city" v-model="city" v-bind="cityAttrs" />
              <p v-if="errors.city" class="text-sm text-destructive">{{ errors.city }}</p>
            </div>
            <div class="space-y-1.5">
              <label for="company-state" class="text-sm font-medium">State</label>
              <Input id="company-state" v-model="state" v-bind="stateAttrs" />
              <p v-if="errors.state" class="text-sm text-destructive">{{ errors.state }}</p>
            </div>
            <div class="space-y-1.5">
              <label for="company-country" class="text-sm font-medium">Country</label>
              <Input id="company-country" v-model="country" v-bind="countryAttrs" />
              <p v-if="errors.country" class="text-sm text-destructive">{{ errors.country }}</p>
            </div>
          </div>
          <div class="space-y-1.5">
            <label for="company-address" class="text-sm font-medium">Address</label>
            <Textarea id="company-address" v-model="address" v-bind="addressAttrs" rows="2" />
            <p v-if="errors.address" class="text-sm text-destructive">{{ errors.address }}</p>
          </div>
          <label v-if="editingId !== null" class="flex items-center gap-2 text-sm font-medium">
            <Checkbox v-model="isActiveField" />
            Active
          </label>

          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isSubmitting" @click="dialogOpen = false">Cancel</Button>
            <Button type="submit" :disabled="isSubmitting">
              {{ isSubmitting ? 'Saving…' : editingId !== null ? 'Save changes' : 'Create company' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="deleteTarget !== null"
      title="Delete company?"
      :description="`This will permanently remove '${deleteTarget?.name}'.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => { if (!v) deleteTarget = null }"
      @confirm="confirmDelete"
    />
  </div>
</template>
