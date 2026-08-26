<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, PencilIcon, Trash2Icon, StarIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { CreateQuestionPayload, Question, QuestionPatch, Review } from '@/types/review'
import { normalizeKeys } from '@/types/review'
import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

// orgs.Department / orgs.Company (apps/api/.../orgs/domain.go) carry no
// `json` tags, so these pickers normalize the raw PascalCase (ID, Name)
// response the same way as the review-module types below.
interface Department {
  id: number
  name: string
}
interface Company {
  id: number
  name: string
}
function pascalToSnake(key: string): string {
  return key.replace(/([a-z0-9])([A-Z])/g, '$1_$2').toLowerCase()
}
function normalizePickerKeys<T>(raw: unknown): T {
  if (Array.isArray(raw)) return raw.map((item) => normalizePickerKeys(item)) as unknown as T
  if (raw !== null && typeof raw === 'object') {
    const out: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(raw as Record<string, unknown>)) out[pascalToSnake(key)] = value
    return out as T
  }
  return raw as T
}

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

const listQuery = useListQuery<Question, 'school_id'>(
  'questions',
  async (params) => {
    const res = await http.get<ApiSuccess<unknown[]>>('/questions', { params })
    return { ...res.data, data: normalizeKeys<Question[]>(res.data.data) }
  },
  {
    defaultSort: 'sort_order',
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
  { key: 'sort_order', label: 'Order', sortable: true },
  { key: 'question', label: 'Question', sortable: true },
  { key: 'actions', label: '' },
]

// --- create / edit dialog ---
const dialogOpen = ref(false)
const editing = ref<Question | null>(null)

const formSchema = toTypedSchema(
  z.object({
    question: z.string().min(1, 'Question is required'),
    sort_order: z.coerce.number().int().min(0).optional(),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: formSchema,
  initialValues: { question: '', sort_order: 0 },
})
const [question, questionAttrs] = defineField('question')
const [sortOrder, sortOrderAttrs] = defineField('sort_order')

function openCreate() {
  editing.value = null
  resetForm({ values: { question: '', sort_order: listQuery.pagination.value?.total ?? items.value.length } })
  dialogOpen.value = true
}

function openEdit(row: Question) {
  editing.value = row
  resetForm({ values: { question: row.question, sort_order: row.sort_order } })
  dialogOpen.value = true
}

function handle422(err: unknown) {
  if (axios.isAxiosError(err) && err.response?.status === 422) {
    toast.error(err.response.data?.message ?? 'Check the form for errors')
  }
}

const createMutation = useMutation({
  mutationFn: (payload: CreateQuestionPayload) => http.post('/questions', payload),
  onSuccess: () => {
    toast.success('Question created')
    queryClient.invalidateQueries({ queryKey: ['questions'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const updateMutation = useMutation({
  mutationFn: ({ id, patch }: { id: number; patch: QuestionPatch }) => http.put(`/questions/${id}`, patch),
  onSuccess: () => {
    toast.success('Question updated')
    queryClient.invalidateQueries({ queryKey: ['questions'] })
    dialogOpen.value = false
  },
  onError: handle422,
})

const onSubmit = handleSubmit((values) => {
  if (editing.value) {
    updateMutation.mutate({ id: editing.value.id, patch: { question: values.question, sort_order: values.sort_order } })
    return
  }
  if (!schoolId.value) {
    toast.error('Enter a school ID first')
    return
  }
  createMutation.mutate({ school_id: schoolId.value, question: values.question, sort_order: values.sort_order })
})

const isSaving = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

// --- delete ---
const deleteTarget = ref<Question | null>(null)
const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/questions/${id}`),
  onSuccess: () => {
    toast.success('Question deleted')
    queryClient.invalidateQueries({ queryKey: ['questions'] })
    deleteTarget.value = null
  },
})

// --- secondary tab: browse a company's submitted reviews (read-only) ---
const reviewDepartmentId = ref<number | undefined>(undefined)
const reviewDepartmentsQuery = useQuery({
  queryKey: computed(() => ['departments-picker', schoolId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<unknown[]>>('/departments', { params: { school_id: schoolId.value, limit: 100 } })
    return normalizePickerKeys<Department[]>(res.data.data)
  },
  enabled: computed(() => schoolId.value !== undefined),
})
const reviewDepartments = computed(() => reviewDepartmentsQuery.data.value ?? [])

const reviewCompanyId = ref<number | undefined>(undefined)
const reviewCompaniesQuery = useQuery({
  queryKey: computed(() => ['companies-picker', reviewDepartmentId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<unknown[]>>('/companies', { params: { department_id: reviewDepartmentId.value, limit: 100 } })
    return normalizePickerKeys<Company[]>(res.data.data)
  },
  enabled: computed(() => reviewDepartmentId.value !== undefined),
})
const reviewCompanies = computed(() => reviewCompaniesQuery.data.value ?? [])

watch(reviewDepartmentId, () => {
  reviewCompanyId.value = undefined
})

const companyReviewsQuery = useQuery({
  queryKey: computed(() => ['company-reviews', reviewCompanyId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<{ reviews: unknown[]; average_rating: number }>>(
      `/reviews/companies/${reviewCompanyId.value}`,
    )
    return { reviews: normalizeKeys<Review[]>(res.data.data.reviews), average_rating: res.data.data.average_rating }
  },
  enabled: computed(() => reviewCompanyId.value !== undefined),
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Reviews &amp; Questions" description="Manage the questionnaire template mentors use to review students.">
      <template #actions>
        <Button :disabled="!schoolId" @click="openCreate">
          <PlusIcon class="size-4" />
          New question
        </Button>
      </template>
    </PageHeader>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-3">
        <div class="space-y-1.5">
          <Label for="school-id">School ID</Label>
          <Input id="school-id" v-model="schoolIdModel" type="number" placeholder="e.g. 1" class="w-40" />
        </div>
      </CardContent>
    </Card>

    <Tabs default-value="questions">
      <TabsList>
        <TabsTrigger value="questions">Questions</TabsTrigger>
        <TabsTrigger value="reviews">Browse company reviews</TabsTrigger>
      </TabsList>

      <TabsContent value="questions" class="space-y-4">
        <ListToolbar :model-value="listQuery.search.value" placeholder="Search questions…" @update:model-value="(v) => listQuery.setParams({ search: v })" />

        <DataTable
          :columns="columns"
          :rows="items"
          :is-loading="listQuery.isLoading.value"
          :sort="listQuery.sort.value"
          :order="listQuery.order.value"
          empty-title="No questions yet"
          empty-description="Add the questions mentors will answer when reviewing a student."
          @sort="(key) => listQuery.setParams({ sort: key, order: listQuery.order.value === 'asc' ? 'desc' : 'asc' })"
        >
          <template #cell-sort_order="{ row }">
            <span class="tabular-nums text-muted-foreground">{{ row.sort_order }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex justify-end gap-1">
              <Button variant="ghost" size="icon-sm" aria-label="Edit question" @click="openEdit(row)">
                <PencilIcon class="size-4" />
              </Button>
              <Button variant="ghost" size="icon-sm" aria-label="Delete question" @click="deleteTarget = row">
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
      </TabsContent>

      <TabsContent value="reviews" class="space-y-4">
        <Card>
          <CardContent class="flex flex-wrap items-end gap-3">
            <div class="space-y-1.5">
              <Label for="review-department">Department</Label>
              <Select v-model="reviewDepartmentId">
                <SelectTrigger id="review-department" class="w-56">
                  <SelectValue placeholder="Select department" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="d in reviewDepartments" :key="d.id" :value="d.id">{{ d.name }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-1.5">
              <Label for="review-company">Company</Label>
              <Select v-model="reviewCompanyId" :disabled="!reviewDepartmentId">
                <SelectTrigger id="review-company" class="w-56">
                  <SelectValue placeholder="Select company" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="c in reviewCompanies" :key="c.id" :value="c.id">{{ c.name }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        <div v-if="!reviewCompanyId">
          <EmptyState title="Pick a company" description="Select a department and company above to see its reviews." />
        </div>
        <div v-else-if="companyReviewsQuery.isLoading.value" class="text-sm text-muted-foreground">Loading reviews…</div>
        <div v-else-if="!companyReviewsQuery.data.value?.reviews.length">
          <EmptyState title="No reviews yet" description="Reviews will show up here once mentors submit them." />
        </div>
        <div v-else class="space-y-3">
          <div class="flex items-center gap-2 text-sm">
            <StarIcon class="size-4 text-warning" />
            <span class="font-medium">Average rating: {{ companyReviewsQuery.data.value.average_rating.toFixed(1) }} / 5</span>
          </div>
          <Card v-for="r in companyReviewsQuery.data.value.reviews" :key="r.id">
            <CardContent class="space-y-1">
              <div class="flex items-center justify-between">
                <p class="font-medium">{{ r.title || 'Untitled review' }}</p>
                <span class="inline-flex items-center gap-1 text-sm">
                  <StarIcon class="size-3.5 text-warning" />
                  {{ r.rating }}/5
                </span>
              </div>
              <p class="text-sm text-muted-foreground">{{ r.body || 'No comment left.' }}</p>
            </CardContent>
          </Card>
        </div>
      </TabsContent>
    </Tabs>

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editing ? 'Edit question' : 'New question' }}</DialogTitle>
          <DialogDescription>{{ editing ? 'Update this question.' : `For school #${schoolId}.` }}</DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <Label for="q-question">Question</Label>
            <Textarea id="q-question" v-model="question" v-bind="questionAttrs" rows="3" />
            <p v-if="errors.question" class="text-sm text-destructive">{{ errors.question }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="q-sort">Sort order</Label>
            <Input id="q-sort" v-model="sortOrder" v-bind="sortOrderAttrs" type="number" min="0" />
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
      title="Delete question?"
      :description="`This removes '${deleteTarget?.question}' from the questionnaire template.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => !v && (deleteTarget = null)"
      @confirm="deleteTarget && deleteMutation.mutate(deleteTarget.id)"
    />
  </div>
</template>
