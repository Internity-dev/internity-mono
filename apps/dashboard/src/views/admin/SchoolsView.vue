<script setup lang="ts">
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { isAxiosError } from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, PencilIcon, Trash2Icon, AlertCircleIcon, SchoolIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useListQuery } from '@/composables/useListQuery'
import { activeStatus } from '@/lib/status'
import type { ApiSuccess, ApiErrorBody } from '@/types/api'
import type { School, SchoolInput, SchoolPatchInput } from '@/types/orgs'
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'

// --- list query ---

const { items, pagination, page, limit, search, sort, order, setParams, isLoading, isError, refetch } =
  useListQuery<School>(
    'schools',
    async (params) => {
      const res = await http.get<ApiSuccess<School[]>>('/schools', { params })
      return res.data
    },
    { defaultSort: 'name', defaultOrder: 'asc' },
  )

interface TableColumn {
  key: string
  label: string
  sortable?: boolean
  class?: string
}

const columns: TableColumn[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'email', label: 'Email' },
  { key: 'phone', label: 'Phone' },
  { key: 'is_active', label: 'Status' },
  { key: 'created_at', label: 'Created', sortable: true },
  { key: 'actions', label: '', class: 'w-24 text-right' },
]

function onSort(key: string) {
  if (key === 'actions') return
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
    name: z.string().trim().min(2, 'At least 2 characters').max(255, 'Too long'),
    email: z.string().trim().max(255).email('Enter a valid email').optional().or(z.literal('')),
    phone: z.string().trim().max(50, 'Too long').optional().or(z.literal('')),
    address: z.string().trim().optional().or(z.literal('')),
    website: z.string().trim().url('Enter a valid URL').optional().or(z.literal('')),
  }),
)

const { defineField, handleSubmit, errors, resetForm, setErrors } = useForm({
  validationSchema: schema,
  initialValues: { name: '', email: '', phone: '', address: '', website: '' },
})
const [name, nameAttrs] = defineField('name')
const [email, emailAttrs] = defineField('email')
const [phone, phoneAttrs] = defineField('phone')
const [address, addressAttrs] = defineField('address')
const [website, websiteAttrs] = defineField('website')

function openCreate() {
  editingId.value = null
  resetForm({ values: { name: '', email: '', phone: '', address: '', website: '' } })
  dialogOpen.value = true
}

function openEdit(row: School) {
  editingId.value = row.id
  resetForm({
    values: {
      name: row.name,
      email: row.email ?? '',
      phone: row.phone ?? '',
      address: row.address ?? '',
      website: row.website ?? '',
    },
  })
  dialogOpen.value = true
}

const queryClient = useQueryClient()

const createMutation = useMutation({
  mutationFn: (payload: SchoolInput) => http.post<ApiSuccess<School>>('/schools', payload),
  onSuccess: () => {
    toast.success('School created')
    queryClient.invalidateQueries({ queryKey: ['schools'] })
    dialogOpen.value = false
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to create school'))
  },
})

const updateMutation = useMutation({
  mutationFn: ({ id, payload }: { id: number; payload: SchoolPatchInput }) =>
    http.put<ApiSuccess<School>>(`/schools/${id}`, payload),
  onSuccess: () => {
    toast.success('School updated')
    queryClient.invalidateQueries({ queryKey: ['schools'] })
    dialogOpen.value = false
  },
  onError: (err: unknown) => {
    const fields = fieldErrors(err)
    if (Object.keys(fields).length) setErrors(fields)
    toast.error(errorMessage(err, 'Failed to update school'))
  },
})

const isSubmitting = ref(false)

const onSubmit = handleSubmit(async (values) => {
  isSubmitting.value = true
  try {
    const payload = {
      name: values.name.trim(),
      email: values.email?.trim() || undefined,
      phone: values.phone?.trim() || undefined,
      address: values.address?.trim() || undefined,
      website: values.website?.trim() || undefined,
    }
    if (editingId.value !== null) {
      await updateMutation.mutateAsync({ id: editingId.value, payload })
    } else {
      await createMutation.mutateAsync(payload)
    }
  } catch {
    // handled in mutation onError
  } finally {
    isSubmitting.value = false
  }
})

// --- delete ---

const deleteTarget = ref<School | null>(null)

const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/schools/${id}`),
  onSuccess: () => {
    toast.success('School deleted')
    queryClient.invalidateQueries({ queryKey: ['schools'] })
    deleteTarget.value = null
  },
  onError: (err: unknown) => {
    // 409 CONFLICT — the school still has departments/companies under it.
    // Surface the server's own message rather than a generic one, since it
    // explains exactly what the user needs to do next.
    toast.error(errorMessage(err, 'Failed to delete school'))
  },
})

function confirmDelete() {
  if (deleteTarget.value) deleteMutation.mutate(deleteTarget.value.id)
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Schools" description="Manage the schools onboarded onto Internity.">
      <template #actions>
        <Button size="sm" @click="openCreate">
          <PlusIcon class="size-4" />
          Add school
        </Button>
      </template>
    </PageHeader>

    <ListToolbar :model-value="search" placeholder="Search schools…" @update:model-value="(v) => setParams({ search: v })" />

    <EmptyState
      v-if="isError"
      :icon="AlertCircleIcon"
      title="Couldn't load schools"
      description="Something went wrong. Try again."
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
        empty-title="No schools yet"
        empty-description="Get started by adding your first school."
        @sort="onSort"
      >
        <template #cell-name="{ row }">
          <div class="flex items-center gap-2">
            <SchoolIcon class="size-4 shrink-0 text-muted-foreground" />
            <span class="font-medium text-foreground">{{ row.name }}</span>
          </div>
        </template>
        <template #cell-email="{ row }">
          <span class="text-muted-foreground">{{ row.email || '—' }}</span>
        </template>
        <template #cell-phone="{ row }">
          <span class="text-muted-foreground">{{ row.phone || '—' }}</span>
        </template>
        <template #cell-is_active="{ row }">
          <StatusBadge v-bind="activeStatus(row.is_active)" />
        </template>
        <template #cell-created_at="{ row }">
          <span class="text-muted-foreground">{{ formatDate(row.created_at) }}</span>
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end gap-1">
            <Button variant="ghost" size="icon-sm" aria-label="Edit school" @click="openEdit(row)">
              <PencilIcon class="size-4" />
            </Button>
            <Button variant="ghost" size="icon-sm" aria-label="Delete school" @click="deleteTarget = row">
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
          <DialogTitle>{{ editingId !== null ? 'Edit school' : 'Add school' }}</DialogTitle>
          <DialogDescription>
            {{ editingId !== null ? 'Update the details for this school.' : 'Onboard a new school onto Internity.' }}
          </DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <label for="school-name" class="text-sm font-medium">Name</label>
            <Input id="school-name" v-model="name" v-bind="nameAttrs" />
            <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="school-email" class="text-sm font-medium">Email</label>
            <Input id="school-email" v-model="email" v-bind="emailAttrs" type="email" />
            <p v-if="errors.email" class="text-sm text-destructive">{{ errors.email }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="school-phone" class="text-sm font-medium">Phone</label>
            <Input id="school-phone" v-model="phone" v-bind="phoneAttrs" />
            <p v-if="errors.phone" class="text-sm text-destructive">{{ errors.phone }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="school-website" class="text-sm font-medium">Website</label>
            <Input id="school-website" v-model="website" v-bind="websiteAttrs" placeholder="https://" />
            <p v-if="errors.website" class="text-sm text-destructive">{{ errors.website }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="school-address" class="text-sm font-medium">Address</label>
            <Textarea id="school-address" v-model="address" v-bind="addressAttrs" rows="2" />
            <p v-if="errors.address" class="text-sm text-destructive">{{ errors.address }}</p>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isSubmitting" @click="dialogOpen = false">Cancel</Button>
            <Button type="submit" :disabled="isSubmitting">
              {{ isSubmitting ? 'Saving…' : editingId !== null ? 'Save changes' : 'Create school' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="deleteTarget !== null"
      title="Delete school?"
      :description="`This will permanently remove '${deleteTarget?.name}'.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => { if (!v) deleteTarget = null }"
      @confirm="confirmDelete"
    />
  </div>
</template>
