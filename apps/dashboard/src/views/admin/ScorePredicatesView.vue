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
import type { ApiSuccess } from '@/types/api'
import type { CreateScorePredicatePayload, ScorePredicate, ScorePredicatePatch } from '@/types/scoring'
import { normalizeKeys } from '@/types/scoring'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'

const auth = useAuthStore()
const queryClient = useQueryClient()

// --- school scope ---
const schoolIdInput = ref<string>(auth.user?.school_id ? String(auth.user.school_id) : '')
const schoolId = computed(() => {
  const n = Number(schoolIdInput.value)
  return schoolIdInput.value !== '' && Number.isFinite(n) && n > 0 ? n : undefined
})

const listQuery = useQuery({
  queryKey: computed(() => ['score-predicates', schoolId.value]),
  queryFn: async () => {
    // ScorePredicate (apps/api/.../scoring/domain.go) has no `json` tags, so
    // the raw response is PascalCase — normalizeKeys() converts it to the
    // snake_case shape this view consumes.
    const res = await http.get<ApiSuccess<unknown[]>>('/score-predicates', {
      params: { school_id: schoolId.value },
    })
    return normalizeKeys<ScorePredicate[]>(res.data.data)
  },
  enabled: computed(() => schoolId.value !== undefined),
})

const items = computed(() => [...(listQuery.data.value ?? [])].sort((a, b) => a.min - b.min))

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'range', label: 'Range' },
  { key: 'description', label: 'Description' },
  { key: 'actions', label: '' },
]

// --- create / edit dialog ---
const dialogOpen = ref(false)
const editing = ref<ScorePredicate | null>(null)

const formSchema = toTypedSchema(
  z
    .object({
      name: z.string().min(1, 'Name is required').max(50, 'Max 50 characters'),
      description: z.string().max(500, 'Max 500 characters').optional(),
      color: z.string().max(20, 'Max 20 characters').optional(),
      min: z.coerce.number({ invalid_type_error: 'Min is required' }).min(0).max(100),
      max: z.coerce.number({ invalid_type_error: 'Max is required' }).min(0).max(100),
    })
    .refine((v) => v.min <= v.max, { message: 'Min must be less than or equal to max', path: ['min'] }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: formSchema,
  initialValues: { name: '', description: '', color: '', min: 0, max: 100 },
})
const [name, nameAttrs] = defineField('name')
const [description, descriptionAttrs] = defineField('description')
const [color, colorAttrs] = defineField('color')
const [min, minAttrs] = defineField('min')
const [max, maxAttrs] = defineField('max')

function openCreate() {
  editing.value = null
  resetForm({ values: { name: '', description: '', color: '', min: 0, max: 100 } })
  dialogOpen.value = true
}

function openEdit(row: ScorePredicate) {
  editing.value = row
  resetForm({
    values: {
      name: row.name,
      description: row.description ?? '',
      color: row.color ?? '',
      min: row.min,
      max: row.max,
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
  mutationFn: (payload: CreateScorePredicatePayload) => http.post('/score-predicates', payload),
  onSuccess: () => {
    toast.success('Score predicate created')
    queryClient.invalidateQueries({ queryKey: ['score-predicates'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const updateMutation = useMutation({
  mutationFn: ({ id, patch }: { id: number; patch: ScorePredicatePatch }) => http.put(`/score-predicates/${id}`, patch),
  onSuccess: () => {
    toast.success('Score predicate updated')
    queryClient.invalidateQueries({ queryKey: ['score-predicates'] })
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
        min: values.min,
        max: values.max,
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
    description: values.description || undefined,
    color: values.color || undefined,
    min: values.min,
    max: values.max,
  })
})

const isSaving = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

// --- delete ---
const deleteTarget = ref<ScorePredicate | null>(null)
const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/score-predicates/${id}`),
  onSuccess: () => {
    toast.success('Score predicate deleted')
    queryClient.invalidateQueries({ queryKey: ['score-predicates'] })
    deleteTarget.value = null
  },
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Score Predicates" description="Configure the letter-grade bands used to summarize student scores.">
      <template #actions>
        <Button :disabled="!schoolId" @click="openCreate">
          <PlusIcon class="size-4" />
          New predicate
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
          Bands are matched by score range, e.g. A = 90–100. Ranges shouldn't overlap.
        </p>
      </CardContent>
    </Card>

    <DataTable
      :columns="columns"
      :rows="items"
      :is-loading="listQuery.isLoading.value"
      empty-title="No score predicates yet"
      empty-description="Create bands like A (90–100), B (80–89), C (70–79) for this school."
    >
      <template #cell-range="{ row }">
        <div class="flex items-center gap-2">
          <span
            class="inline-block size-2.5 rounded-full"
            :style="{ backgroundColor: row.color || 'var(--color-muted-foreground)' }"
          />
          <span class="font-medium tabular-nums">{{ row.min }} – {{ row.max }}</span>
        </div>
      </template>
      <template #cell-description="{ row }">
        <span class="text-muted-foreground">{{ row.description || '—' }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex justify-end gap-1">
          <Button variant="ghost" size="icon-sm" aria-label="Edit score predicate" @click="openEdit(row)">
            <PencilIcon class="size-4" />
          </Button>
          <Button variant="ghost" size="icon-sm" aria-label="Delete score predicate" @click="deleteTarget = row">
            <Trash2Icon class="size-4 text-destructive" />
          </Button>
        </div>
      </template>
    </DataTable>

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editing ? 'Edit score predicate' : 'New score predicate' }}</DialogTitle>
          <DialogDescription>{{ editing ? 'Update this band.' : `For school #${schoolId}.` }}</DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <Label for="sp-name">Name</Label>
            <Input id="sp-name" v-model="name" v-bind="nameAttrs" placeholder="e.g. A" />
            <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
          </div>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div class="space-y-1.5">
              <Label for="sp-min">Min score</Label>
              <Input id="sp-min" v-model="min" v-bind="minAttrs" type="number" min="0" max="100" />
              <p v-if="errors.min" class="text-sm text-destructive">{{ errors.min }}</p>
            </div>
            <div class="space-y-1.5">
              <Label for="sp-max">Max score</Label>
              <Input id="sp-max" v-model="max" v-bind="maxAttrs" type="number" min="0" max="100" />
              <p v-if="errors.max" class="text-sm text-destructive">{{ errors.max }}</p>
            </div>
          </div>
          <div class="space-y-1.5">
            <Label for="sp-description">Description</Label>
            <Textarea id="sp-description" v-model="description" v-bind="descriptionAttrs" rows="2" />
          </div>
          <div class="space-y-1.5">
            <Label for="sp-color">Color</Label>
            <Input id="sp-color" v-model="color" v-bind="colorAttrs" placeholder="#22c55e" />
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
      title="Delete score predicate?"
      :description="`This removes '${deleteTarget?.name}'. Existing scores are unaffected.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => !v && (deleteTarget = null)"
      @confirm="deleteTarget && deleteMutation.mutate(deleteTarget.id)"
    />
  </div>
</template>
