<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { PlusIcon, StarIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery, type FetcherParams } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { Score, ScoreType } from '@/types/scoring'
import type { Review, CreateReviewPayload } from '@/types/review'
import { normalizeKeys } from '@/types/review'

import PageHeader from '@/components/shared/PageHeader.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import StarRatingInput from '@/components/shared/StarRatingInput.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

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

// --- org-scope picker ----------------------------------------------------

const auth = useAuthStore()
const isMentor = computed(() => auth.user?.role === 'mentor')
const route = useRoute()

// Read straight from the route query (not `scoresList.filters`) so these
// stay independent of `scoresList`'s own declaration further below — its
// `enabled` option needs them, and referencing the destructured result of
// the same call it's part of would be a circular self-reference.
const departmentId = computed(() => (route.query.department_id ? Number(route.query.department_id) : undefined))
const urlCompanyId = computed(() => (route.query.company_id ? Number(route.query.company_id) : undefined))
const studentId = computed(() => (route.query.student_id as string) ?? '')

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
  // Changing department resets company in the same atomic URL update.
  set: (v) => scoresList.setParams({ department_id: v, company_id: undefined }),
})
const companyModel = computed<string | undefined>({
  get: () => (urlCompanyId.value ? String(urlCompanyId.value) : undefined),
  set: (v) => scoresList.setParams({ company_id: v }),
})

// --- student picker (known rough edge — no student-search endpoint exists
// yet; staff must paste the student's UUID directly, see report) -----------

const isValidStudentId = computed(() => /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(studentId.value.trim()))
const canQuery = computed(() => !!effectiveCompanyId.value && isValidStudentId.value)

const studentIdModel = computed<string>({
  get: () => studentId.value,
  set: (v) => scoresList.setParams({ student_id: v || undefined }),
})

// --- scores list -----------------------------------------------------------

function fetchScores(params: FetcherParams<'department_id' | 'company_id' | 'student_id'>) {
  return http
    .get<ApiSuccess<Score[]>>('/scores', { params: { ...params, user_id: studentId.value.trim(), company_id: effectiveCompanyId.value } })
    .then((r) => r.data)
}

const scoresList = useListQuery<Score, 'department_id' | 'company_id' | 'student_id'>('scores', fetchScores, {
  defaultSort: 'created_at',
  filters: [{ key: 'department_id', sendToFetcher: false }, { key: 'company_id', sendToFetcher: false }, { key: 'student_id', sendToFetcher: false }],
  enabled: () => canQuery.value,
})

const scores = computed(() => scoresList.items.value)
const average = computed(() => (scores.value.length ? scores.value.reduce((sum, s) => sum + s.score, 0) / scores.value.length : null))

const columns: Column[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'type', label: 'Type' },
  { key: 'score', label: 'Score', sortable: true },
  { key: 'actions', label: '', class: 'text-right' },
]

// --- create / edit dialog --------------------------------------------------

const scoreFormSchema = toTypedSchema(
  z.object({
    name: z.string().min(1, 'Enter a name').max(255),
    type: z.enum(['teknis', 'non-teknis'], { message: 'Select a type' }),
    score: z.coerce.number().int().min(0, 'Min 0').max(100, 'Max 100'),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({ validationSchema: scoreFormSchema })
const [name, nameAttrs] = defineField('name')
const [type] = defineField('type')
const [score, scoreAttrs] = defineField('score')

const formOpen = ref(false)
const editingScore = ref<Score | null>(null)

function openCreate() {
  editingScore.value = null
  resetForm({ values: { name: '', type: 'teknis' as ScoreType, score: 0 } })
  formOpen.value = true
}

function openEdit(s: Score) {
  editingScore.value = s
  resetForm({ values: { name: s.name, type: s.type, score: s.score } })
  formOpen.value = true
}

const queryClient = useQueryClient()
function invalidate() {
  queryClient.invalidateQueries({ queryKey: ['scores'] })
}

const saveMutation = useMutation({
  mutationFn: async (values: { name: string; type: ScoreType; score: number }) => {
    if (editingScore.value) {
      return http.put(`/scores/${editingScore.value.id}`, values)
    }
    return http.post('/scores', {
      ...values,
      user_id: studentId.value.trim(),
      company_id: effectiveCompanyId.value,
    })
  },
  onSuccess: () => {
    toast.success(editingScore.value ? 'Score updated' : 'Score recorded')
    formOpen.value = false
    invalidate()
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})

const onSubmit = handleSubmit((values) => saveMutation.mutate(values))

// --- delete ---

const deleteTarget = ref<Score | null>(null)
const deleteMutation = useMutation({
  mutationFn: (id: number) => http.delete(`/scores/${id}`),
  onSuccess: () => {
    toast.success('Score deleted')
    deleteTarget.value = null
    invalidate()
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})

// --- reviews (mentor rates this student's performance) --------------------
// Only mentors can submit these (apps/api/internal/modules/review/service.go
// CreateReview: reviewee_type "user" requires RoleMentor) — coordinators/
// admin browsing a student's scores here can't leave one, matching the
// backend rather than showing a form that would just 403.

const reviewsQuery = useQuery({
  queryKey: computed(() => ['reviews-for-user', studentId.value.trim()]),
  queryFn: () =>
    http
      .get<ApiSuccess<Review[]>>(`/reviews/users/${studentId.value.trim()}`)
      .then((r) => normalizeKeys<Review[]>(r.data.data)),
  enabled: canQuery,
})

const reviews = computed(() => reviewsQuery.data.value ?? [])

const reviewFormSchema = toTypedSchema(
  z.object({
    title: z.string().max(255).optional(),
    body: z.string().optional(),
    rating: z.number().min(1, 'Pick a rating').max(5),
  }),
)
const reviewForm = useForm({ validationSchema: reviewFormSchema, initialValues: { title: '', body: '', rating: 0 } })
const [reviewTitle, reviewTitleAttrs] = reviewForm.defineField('title')
const [reviewBody, reviewBodyAttrs] = reviewForm.defineField('body')
const [reviewRating] = reviewForm.defineField('rating')

const reviewDialogOpen = ref(false)
function openReviewDialog() {
  reviewForm.resetForm({ values: { title: '', body: '', rating: 0 } })
  reviewDialogOpen.value = true
}

const createReviewMutation = useMutation({
  mutationFn: (payload: CreateReviewPayload) => http.post('/reviews', payload),
  onSuccess: () => {
    toast.success('Review submitted')
    reviewDialogOpen.value = false
    queryClient.invalidateQueries({ queryKey: ['reviews-for-user'] })
  },
  onError: (err: unknown) => toast.error(errMessage(err)),
})

const onSubmitReview = reviewForm.handleSubmit((values) =>
  createReviewMutation.mutate({
    reviewee_type: 'user',
    reviewee_user_id: studentId.value.trim(),
    title: values.title || undefined,
    body: values.body || undefined,
    rating: values.rating,
  }),
)
</script>

<template>
  <div class="space-y-6 p-6">
    <PageHeader title="Scores" description="Enter and manage a student's scores for their placement.">
      <template #actions>
        <Button :disabled="!canQuery" @click="openCreate">
          <PlusIcon class="size-4" />
          Add score
        </Button>
      </template>
    </PageHeader>

    <Card>
      <CardContent class="flex flex-wrap items-end gap-4 pt-0">
        <div v-if="!isMentor" class="w-56 space-y-1.5">
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
        <div v-if="!isMentor" class="w-56 space-y-1.5">
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
        <div class="w-80 space-y-1.5">
          <label class="text-sm font-medium">Student ID (UUID)</label>
          <Input v-model="studentIdModel" placeholder="e.g. 3fa85f64-5717-4562-b3fc-2c963f66afa6" class="font-mono text-sm" />
          <p v-if="studentId && !isValidStudentId" class="text-xs text-destructive">Not a valid UUID.</p>
        </div>
      </CardContent>
    </Card>

    <template v-if="canQuery">
      <p v-if="average !== null" class="text-sm text-muted-foreground">
        Average score: <span class="font-medium text-foreground">{{ average.toFixed(1) }}</span>
      </p>
      <ListToolbar :model-value="scoresList.search.value" placeholder="Search scores…" @update:model-value="(v) => scoresList.setParams({ search: v })" />

      <DataTable
        :columns="columns"
        :rows="scores"
        :is-loading="scoresList.isLoading.value"
        :sort="scoresList.sort.value"
        :order="scoresList.order.value"
        empty-title="No scores yet"
        empty-description="Add this student's first score for their placement."
        @sort="(key) => scoresList.setParams({ sort: key, order: scoresList.order.value === 'asc' ? 'desc' : 'asc' })"
      >
        <template #cell-name="{ row }">
          <span class="font-medium text-foreground">{{ row.name }}</span>
        </template>
        <template #cell-type="{ row }">
          <Badge variant="outline">{{ row.type === 'non-teknis' ? 'Non-teknis' : 'Teknis' }}</Badge>
        </template>
        <template #cell-score="{ row }">{{ row.score }}</template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end gap-2">
            <Button size="sm" variant="outline" @click="openEdit(row)">Edit</Button>
            <Button size="sm" variant="destructive" @click="deleteTarget = row">Delete</Button>
          </div>
        </template>
      </DataTable>

      <ListPagination
        :page="scoresList.page.value"
        :limit="scoresList.limit.value"
        :total="scoresList.pagination.value?.total ?? 0"
        @update:page="(p) => scoresList.setParams({ page: p })"
      />

      <Card v-if="isMentor">
        <CardHeader class="flex flex-row items-center justify-between gap-3">
          <CardTitle>Reviews</CardTitle>
          <Button size="sm" variant="outline" @click="openReviewDialog">
            <PlusIcon class="size-4" />
            Add review
          </Button>
        </CardHeader>
        <CardContent class="space-y-3">
          <p v-if="reviewsQuery.isLoading.value" class="text-sm text-muted-foreground">Loading reviews…</p>
          <EmptyState
            v-else-if="reviews.length === 0"
            title="No reviews yet"
            description="Rate this student's performance to leave the first one."
          />
          <Card v-for="r in reviews" :key="r.id">
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
        </CardContent>
      </Card>
    </template>
    <EmptyState
      v-else
      title="Select a company and student"
      description="Choose a company and enter a student's user ID (UUID) above to view or enter their scores."
    />

    <Dialog :open="formOpen" @update:open="formOpen = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editingScore ? 'Edit score' : 'Add score' }}</DialogTitle>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <label for="s-name" class="text-sm font-medium">Name</label>
            <Input id="s-name" v-model="name" v-bind="nameAttrs" placeholder="e.g. Communication skills" />
            <p v-if="errors.name" class="text-sm text-destructive">{{ errors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="text-sm font-medium">Type</label>
            <Select v-model="type">
              <SelectTrigger class="w-full"><SelectValue placeholder="Select type" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="teknis">Teknis</SelectItem>
                <SelectItem value="non-teknis">Non-teknis</SelectItem>
              </SelectContent>
            </Select>
            <p v-if="errors.type" class="text-sm text-destructive">{{ errors.type }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="s-score" class="text-sm font-medium">Score (0–100)</label>
            <Input id="s-score" v-model="score" v-bind="scoreAttrs" type="number" min="0" max="100" />
            <p v-if="errors.score" class="text-sm text-destructive">{{ errors.score }}</p>
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
      title="Delete this score?"
      :description="`This will permanently remove '${deleteTarget?.name}'.`"
      confirm-label="Delete"
      :is-loading="deleteMutation.isPending.value"
      @update:open="(v) => { if (!v) deleteTarget = null }"
      @confirm="deleteTarget && deleteMutation.mutate(deleteTarget.id)"
    />

    <Dialog :open="reviewDialogOpen" @update:open="reviewDialogOpen = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add review</DialogTitle>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmitReview">
          <div class="space-y-1.5">
            <label class="text-sm font-medium">Rating</label>
            <StarRatingInput v-model="reviewRating" />
            <p v-if="reviewForm.errors.value.rating" class="text-sm text-destructive">{{ reviewForm.errors.value.rating }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="r-title" class="text-sm font-medium">Title (optional)</label>
            <Input id="r-title" v-model="reviewTitle" v-bind="reviewTitleAttrs" placeholder="e.g. Evaluasi Kinerja Magang" />
            <p v-if="reviewForm.errors.value.title" class="text-sm text-destructive">{{ reviewForm.errors.value.title }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="r-body" class="text-sm font-medium">Comment (optional)</label>
            <Textarea id="r-body" v-model="reviewBody" v-bind="reviewBodyAttrs" rows="3" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="reviewDialogOpen = false">Cancel</Button>
            <Button type="submit" :disabled="createReviewMutation.isPending.value">
              {{ createReviewMutation.isPending.value ? 'Submitting…' : 'Submit review' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
