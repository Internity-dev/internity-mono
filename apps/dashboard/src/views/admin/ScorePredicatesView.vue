<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, PencilIcon, Trash2Icon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { CreateScorePredicatePayload, ScorePredicate, ScorePredicatePatch } from '@/types/scoring'
import { normalizeKeys } from '@/types/scoring'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'

const auth = useAuthStore()
const queryClient = useQueryClient()
const route = useRoute()

// --- school scope ---
// Read straight from the route query (not `listQuery.filters`) so this
// stays independent of `listQuery`'s own declaration below — its `enabled`
// option needs it, and referencing the destructured/property result of the
// same call it's part of would be a circular self-reference: `enabled`
// would only resolve correctly once `listQuery` already exists, but
// `listQuery` can't finish constructing until `enabled` is evaluated.
const schoolId = computed(() => {
  const raw = (route.query.school_id as string) ?? ''
  const n = Number(raw)
  return raw !== '' && Number.isFinite(n) && n > 0 ? n : undefined
})

const listQuery = useListQuery<ScorePredicate, 'school_id'>(
  'score-predicates',
  async (params) => {
    // ScorePredicate (apps/api/.../scoring/domain.go) has no `json` tags, so
    // the raw response is PascalCase — normalizeKeys() converts it to the
    // snake_case shape this view consumes.
    const res = await http.get<ApiSuccess<unknown[]>>('/score-predicates', { params })
    return { ...res.data, data: normalizeKeys<ScorePredicate[]>(res.data.data) }
  },
  {
    defaultSort: 'min',
    defaultOrder: 'asc',
    filters: ['school_id'],
    enabled: () => schoolId.value !== undefined,
  },
)

const schoolIdModel = computed<string>({
  get: () => (route.query.school_id as string) ?? (auth.user?.school_id ? String(auth.user.school_id) : ''),
  set: (v) => listQuery.setParams({ school_id: v || undefined }),
})

const items = computed(() => listQuery.items.value)

const columns: Column[] = [
  { key: 'name', label: 'Name', sortable: true },
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
      color: z
        .string()
        .max(20, 'Max 20 characters')
        .regex(/^#[0-9a-fA-F]{3,8}$/, 'Must be a hex color like #22c55e')
        .optional()
        .or(z.literal('')),
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
          <Input id="school-id" v-model="schoolIdModel" type="number" placeholder="e.g. 1" class="w-40" />
        </div>
        <p class="pb-1.5 text-sm text-muted-foreground">
          Bands are matched by score range, e.g. A = 90–100. Ranges shouldn't overlap.
        </p>
      </CardContent>
    </Card>

    <ListToolbar :model-value="listQuery.search.value" placeholder="Search predicates…" @update:model-value="(v) => listQuery.setParams({ search: v })" />

    <DataTable
      :columns="columns"
      :rows="items"
      :is-loading="listQuery.isLoading.value"
      :sort="listQuery.sort.value"
      :order="listQuery.order.value"
      empty-title="No score predicates yet"
      empty-description="Create bands like A (90–100), B (80–89), C (70–79) for this school."
      @sort="(key) => listQuery.setParams({ sort: key, order: listQuery.order.value === 'asc' ? 'desc' : 'asc' })"
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

    <ListPagination
      :page="listQuery.page.value"
      :limit="listQuery.limit.value"
      :total="listQuery.pagination.value?.total ?? 0"
      @update:page="(p) => listQuery.setParams({ page: p })"
    />

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
            <div class="flex items-center gap-2">
              <div
                class="size-10 shrink-0 rounded-md border"
                :style="{ backgroundColor: color || 'transparent' }"
                aria-hidden="true"
              />
              <Input id="sp-color" v-model="color" v-bind="colorAttrs" placeholder="#22c55e" class="flex-1" />
            </div>
            <p v-if="errors.color" class="text-sm text-destructive">{{ errors.color }}</p>
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
