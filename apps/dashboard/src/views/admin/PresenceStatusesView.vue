<script setup lang="ts">
import { computed, ref } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, PencilIcon, Trash2Icon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import type {
  CreatePresenceStatusPayload,
  PresenceStatus,
  PresenceStatusKind,
  PresenceStatusPatch,
} from '@/types/internship'
import { fetchPresenceStatuses } from '@/types/internship'
import { activeStatus, presenceStatusKind } from '@/lib/status'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const auth = useAuthStore()
const queryClient = useQueryClient()

const KIND_OPTIONS: { value: PresenceStatusKind; label: string }[] = [
  { value: 'present', label: 'Present' },
  { value: 'permitted', label: 'Permitted' },
  { value: 'sick', label: 'Sick' },
  { value: 'absent', label: 'Absent' },
  { value: 'holiday', label: 'Holiday' },
]

// --- school scope ---
const schoolIdInput = ref<string>(auth.user?.school_id ? String(auth.user.school_id) : '')
const schoolId = computed(() => {
  const n = Number(schoolIdInput.value)
  return schoolIdInput.value !== '' && Number.isFinite(n) && n > 0 ? n : undefined
})

const listQuery = useQuery({
  queryKey: computed(() => ['presence-statuses', schoolId.value]),
  // fetchPresenceStatuses (src/types/internship.ts) normalizes the backend's
  // raw PascalCase response (PresenceStatus has no `json` tags server-side)
  // to the snake_case shape this view consumes.
  queryFn: () => fetchPresenceStatuses(schoolId.value as number),
  enabled: computed(() => schoolId.value !== undefined),
})

const items = computed(() => listQuery.data.value ?? [])
const usedKinds = computed(() => new Set(items.value.map((i) => i.kind)))

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'kind', label: 'Kind' },
  { key: 'description', label: 'Description' },
  { key: 'is_active', label: 'Status' },
  { key: 'actions', label: '' },
]

// --- create / edit dialog ---
const dialogOpen = ref(false)
const editing = ref<PresenceStatus | null>(null)

const formSchema = toTypedSchema(
  z.object({
    name: z.string().min(1, 'Name is required').max(100, 'Max 100 characters'),
    kind: z.enum(['present', 'permitted', 'sick', 'absent', 'holiday'], {
      errorMap: () => ({ message: 'Pick a kind' }),
    }),
    description: z.string().max(500, 'Max 500 characters').optional(),
    color: z.string().max(20, 'Max 20 characters').optional(),
    icon: z.string().max(50, 'Max 50 characters').optional(),
    is_active: z.boolean().optional(),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: formSchema,
  initialValues: { name: '', kind: undefined, description: '', color: '', icon: '', is_active: true },
})
const [name, nameAttrs] = defineField('name')
const [kind, kindAttrs] = defineField('kind')
const [description, descriptionAttrs] = defineField('description')
const [color, colorAttrs] = defineField('color')
const [icon, iconAttrs] = defineField('icon')
const [isActive] = defineField('is_active')

function openCreate() {
  editing.value = null
  resetForm({ values: { name: '', kind: undefined, description: '', color: '', icon: '', is_active: true } })
  dialogOpen.value = true
}

function openEdit(row: PresenceStatus) {
  editing.value = row
  resetForm({
    values: {
      name: row.name,
      kind: row.kind,
      description: row.description ?? '',
      color: row.color ?? '',
      icon: row.icon ?? '',
      is_active: row.is_active,
    },
  })
  dialogOpen.value = true
}

function handle422(err: unknown) {
  if (axios.isAxiosError(err) && err.response?.status === 422) {
    toast.error(err.response.data?.message ?? 'Check the form for errors')
  }
}

const createMutation = useMutation({
  mutationFn: (payload: CreatePresenceStatusPayload) => http.post('/presence-statuses', payload),
  onSuccess: () => {
    toast.success('Presence status created')
    queryClient.invalidateQueries({ queryKey: ['presence-statuses'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const updateMutation = useMutation({
  mutationFn: ({ id, patch }: { id: number; patch: PresenceStatusPatch }) => http.put(`/presence-statuses/${id}`, patch),
  onSuccess: () => {
    toast.success('Presence status updated')
    queryClient.invalidateQueries({ queryKey: ['presence-statuses'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const onSubmit = handleSubmit((values) => {
  if (editing.value) {
    updateMutation.mutate({
      id: editing.value.id,
      patch: {
        name: values.name,
        description: values.description || undefined,
        color: values.color || undefined,
        icon: values.icon || undefined,
        is_active: values.is_active,
      },
    })
    return
  }
  if (!schoolId.value) {
    toast.error('Enter a school ID first')
    return
  }
  createMutation.mutate({
    school_id: schoolId.value,
    name: values.name,
    kind: values.kind as PresenceStatusKind,
    description: values.description || undefined,
    color: values.color || undefined,
    icon: values.icon || undefined,
  })
})

const isSaving = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

// --- delete ---
const deleteTarget = ref<PresenceStatus | null>(null)
const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/presence-statuses/${id}`),
  onSuccess: () => {
    toast.success('Presence status deleted')
    queryClient.invalidateQueries({ queryKey: ['presence-statuses'] })
    deleteTarget.value = null
  },
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader
      title="Presence Statuses"
      description="Configure the attendance statuses students and mentors can use for this school."
    >
      <template #actions>
        <Button :disabled="!schoolId" @click="openCreate">
          <PlusIcon class="size-4" />
          New status
        </Button>
      </template>
    </PageHeader>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3">
        <div class="space-y-1.5">
          <Label for="school-id">School ID</Label>
          <Input id="school-id" v-model="schoolIdInput" type="number" placeholder="e.g. 1" class="w-40" />
        </div>
        <p class="pb-1.5 text-sm text-muted-foreground">
          A school needs a <strong>present</strong> status configured before students can check in.
        </p>
      </CardContent>
    </Card>

    <DataTable
      :columns="columns"
      :rows="items"
      :is-loading="listQuery.isLoading.value"
      empty-title="No presence statuses yet"
      empty-description="Create statuses like Hadir, Izin, Sakit, and Alpa for this school."
    >
      <template #cell-kind="{ row }">
        <StatusBadge :tone="presenceStatusKind(row.kind).tone" :label="presenceStatusKind(row.kind).label" />
      </template>
      <template #cell-description="{ row }">
        <span class="text-muted-foreground">{{ row.description || '—' }}</span>
      </template>
      <template #cell-is_active="{ row }">
        <StatusBadge :tone="activeStatus(row.is_active).tone" :label="activeStatus(row.is_active).label" />
      </template>
      <template #cell-actions="{ row }">
        <div class="flex justify-end gap-1">
          <Button variant="ghost" size="icon-sm" aria-label="Edit presence status" @click="openEdit(row)">
            <PencilIcon class="size-4" />
          </Button>
          <Button variant="ghost" size="icon-sm" aria-label="Delete presence status" @click="deleteTarget = row">
            <Trash2Icon class="size-4 text-destructive" />
          </Button>
        </div>
      </template>
    </DataTable>

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editing ? 'Edit presence status' : 'New presence status' }}</DialogTitle>
          <DialogDescription>
            {{ editing ? 'Update this status.' : `For school #${schoolId}. Exactly one status per kind is allowed.` }}
          </DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <Label for="ps-name">Name</Label>
            <Input id="ps-name" v-model="name" v-bind="nameAttrs" placeholder="e.g. Hadir" />
            <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="ps-kind">Kind</Label>
            <Select v-model="kind" v-bind="kindAttrs" :disabled="!!editing">
              <SelectTrigger id="ps-kind" class="w-full">
                <SelectValue placeholder="Select a kind" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="opt in KIND_OPTIONS"
                  :key="opt.value"
                  :value="opt.value"
                  :disabled="!editing && usedKinds.has(opt.value)"
                >
                  {{ opt.label }}
                  <span v-if="!editing && usedKinds.has(opt.value)" class="text-xs text-muted-foreground">(already configured)</span>
                </SelectItem>
              </SelectContent>
            </Select>
            <p v-if="errors.kind" class="text-sm text-destructive">{{ errors.kind }}</p>
            <p v-if="editing" class="text-xs text-muted-foreground">Kind can't be changed after creation.</p>
          </div>
          <div class="space-y-1.5">
            <Label for="ps-description">Description</Label>
            <Textarea id="ps-description" v-model="description" v-bind="descriptionAttrs" rows="2" />
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label for="ps-color">Color</Label>
              <Input id="ps-color" v-model="color" v-bind="colorAttrs" placeholder="#22c55e" />
            </div>
            <div class="space-y-1.5">
              <Label for="ps-icon">Icon</Label>
              <Input id="ps-icon" v-model="icon" v-bind="iconAttrs" placeholder="check-circle" />
            </div>
          </div>
          <div v-if="editing" class="flex items-center gap-2">
            <Checkbox id="ps-active" v-model="isActive" />
            <Label for="ps-active">Active</Label>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" :disabled="isSaving" @click="dialogOpen = false">Cancel</Button>
            <Button type="submit" :disabled="isSaving">{{ isSaving ? 'Saving…' : 'Save' }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <ConfirmDialog
      :open="!!deleteTarget"
      title="Delete presence status?"
      :description="`This removes '${deleteTarget?.name}'. Presences already recorded with this status are unaffected.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => !v && (deleteTarget = null)"
      @confirm="deleteTarget && deleteMutation.mutate(deleteTarget.id)"
    />
  </div>
</template>
