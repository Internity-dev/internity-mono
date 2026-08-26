<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import { toast } from 'vue-sonner'
import { ChevronLeftIcon, ChevronRightIcon, GraduationCapIcon, StarIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'
import type { AttendanceDay, InternDate } from '@/types/vacancy'
import type { CreateReviewPayload } from '@/types/review'
import { internDateStatus, attendanceDayStatus } from '@/lib/status'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import StarRatingInput from '@/components/shared/StarRatingInput.vue'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'

const queryClient = useQueryClient()

const placementsQuery = useQuery({
  queryKey: ['internships-mine'],
  queryFn: () => http.get<ApiSuccess<InternDate[]>>('/internships/mine').then((res) => res.data.data),
})

const placements = computed(() => placementsQuery.data.value ?? [])
const selectedTab = ref('0')
const selectedPlacement = computed<InternDate | undefined>(() => placements.value[Number(selectedTab.value)])

// --- Dates form ---

const datesSchema = toTypedSchema(
  z
    .object({
      start_date: z.string().min(1, 'Start date is required'),
      end_date: z.string().min(1, 'End date is required'),
    })
    .refine((v) => v.start_date < v.end_date, { message: 'End date must be after start date', path: ['end_date'] }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: datesSchema,
  initialValues: { start_date: '', end_date: '' },
})
const [startDate, startDateAttrs] = defineField('start_date')
const [endDate, endDateAttrs] = defineField('end_date')

const isEditing = ref(false)

watch(
  selectedPlacement,
  (placement) => {
    isEditing.value = false
    resetForm({
      values: {
        start_date: placement?.start_date?.slice(0, 10) ?? '',
        end_date: placement?.end_date?.slice(0, 10) ?? '',
      },
    })
  },
  { immediate: true },
)

const showDatesForm = computed(() => !!selectedPlacement.value && (!selectedPlacement.value.start_date || isEditing.value))

function errorMessage(err: unknown, fallback: string): string {
  return (err as { response?: { data?: { message?: string } } })?.response?.data?.message ?? fallback
}

const setDatesMutation = useMutation({
  mutationFn: (payload: { id: number; start_date: string; end_date: string; expected_version: number }) =>
    http
      .put<ApiSuccess<InternDate>>(`/internships/${payload.id}/dates`, {
        start_date: payload.start_date,
        end_date: payload.end_date,
        expected_version: payload.expected_version,
      })
      .then((res) => res.data.data),
  onSuccess: () => {
    toast.success('Internship dates updated')
    isEditing.value = false
    queryClient.invalidateQueries({ queryKey: ['internships-mine'] })
  },
  onError: (err) => toast.error(errorMessage(err, 'Failed to update dates. Reload and try again')),
})

const onSubmitDates = handleSubmit((values) => {
  if (!selectedPlacement.value) return
  setDatesMutation.mutate({
    id: selectedPlacement.value.id,
    start_date: values.start_date,
    end_date: values.end_date,
    expected_version: selectedPlacement.value.version,
  })
})

// --- Attendance summary ---

function currentMonthString(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
}

const currentMonth = ref(currentMonthString())

watch(selectedPlacement, () => {
  currentMonth.value = currentMonthString()
})

function shiftMonth(delta: number) {
  const [y, m] = currentMonth.value.split('-').map(Number)
  if (y === undefined || m === undefined) return
  const d = new Date(Date.UTC(y, m - 1 + delta, 1))
  currentMonth.value = `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}`
}

const attendanceQuery = useQuery({
  queryKey: computed(() => ['attendance-summary', selectedPlacement.value?.id, currentMonth.value]),
  queryFn: () =>
    http
      .get<ApiSuccess<AttendanceDay[] | null>>(`/internships/${selectedPlacement.value!.id}/attendance-summary`, {
        params: { month: currentMonth.value },
      })
      .then((res) => res.data.data ?? []),
  enabled: computed(() => !!selectedPlacement.value?.id && !!selectedPlacement.value?.start_date),
})

const attendanceDays = computed(() => attendanceQuery.data.value ?? [])

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' })
}

// --- rate this company ---
// apps/api/internal/modules/review/service.go's CreateReview requires
// RoleStudent for a reviewee_type "company" review — this is exactly that,
// no uniqueness constraint on the backend (a student can leave more than
// one), so this doesn't need to check for an existing review first.

const reviewSchema = toTypedSchema(
  z.object({
    title: z.string().max(255).optional(),
    body: z.string().optional(),
    rating: z.number().min(1, 'Pick a rating').max(5),
  }),
)
const reviewForm = useForm({ validationSchema: reviewSchema, initialValues: { title: '', body: '', rating: 0 } })
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
  },
  onError: (err: unknown) => toast.error(errorMessage(err, 'Failed to submit review')),
})

const onSubmitReview = reviewForm.handleSubmit((values) => {
  if (!selectedPlacement.value) return
  createReviewMutation.mutate({
    reviewee_type: 'company',
    reviewee_company_id: selectedPlacement.value.company_id,
    title: values.title || undefined,
    body: values.body || undefined,
    rating: values.rating,
  })
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="My Internship" description="View your placement details, set your internship dates, and check your attendance." />

    <div v-if="placementsQuery.isLoading.value" class="space-y-4">
      <Skeleton class="h-24 w-full" />
      <Skeleton class="h-64 w-full" />
    </div>

    <div v-else-if="placementsQuery.isError.value" class="flex flex-col items-center gap-3 rounded-lg border border-dashed p-10 text-center">
      <p class="text-sm text-muted-foreground">Failed to load your internship placement.</p>
      <Button variant="outline" size="sm" @click="placementsQuery.refetch()">Try again</Button>
    </div>

    <EmptyState
      v-else-if="placements.length === 0"
      :icon="GraduationCapIcon"
      title="No internship placement yet"
      description="Once one of your applications is accepted, your placement will show up here."
    />

    <template v-else>
      <Tabs v-if="placements.length > 1" v-model="selectedTab">
        <TabsList>
          <TabsTrigger v-for="(placement, i) in placements" :key="placement.id" :value="String(i)">
            Placement #{{ placement.id }}
          </TabsTrigger>
        </TabsList>
      </Tabs>

      <template v-if="selectedPlacement">
        <Card>
          <CardHeader class="flex flex-row items-center justify-between gap-3">
            <div>
              <CardTitle>Placement details</CardTitle>
              <CardDescription>Company #{{ selectedPlacement.company_id }}</CardDescription>
            </div>
            <StatusBadge v-bind="internDateStatus(selectedPlacement.status)" />
          </CardHeader>
          <CardContent class="space-y-4">
            <form v-if="showDatesForm" class="grid gap-4 sm:grid-cols-2" novalidate @submit="onSubmitDates">
              <div class="space-y-1.5">
                <label for="start_date" class="text-sm font-medium">Start date</label>
                <Input id="start_date" v-model="startDate" v-bind="startDateAttrs" type="date" />
                <p v-if="errors.start_date" class="text-sm text-destructive">{{ errors.start_date }}</p>
              </div>
              <div class="space-y-1.5">
                <label for="end_date" class="text-sm font-medium">End date</label>
                <Input id="end_date" v-model="endDate" v-bind="endDateAttrs" type="date" />
                <p v-if="errors.end_date" class="text-sm text-destructive">{{ errors.end_date }}</p>
              </div>
              <div class="flex items-center gap-2 sm:col-span-2">
                <Button type="submit" :disabled="setDatesMutation.isPending.value">
                  {{ setDatesMutation.isPending.value ? 'Saving…' : 'Save dates' }}
                </Button>
                <Button v-if="selectedPlacement.start_date" type="button" variant="outline" @click="isEditing = false">
                  Cancel
                </Button>
              </div>
            </form>

            <template v-else>
              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <p class="text-xs text-muted-foreground">Start date</p>
                  <p class="font-medium">{{ selectedPlacement.start_date }}</p>
                </div>
                <div>
                  <p class="text-xs text-muted-foreground">End date</p>
                  <p class="font-medium">{{ selectedPlacement.extended_until ?? selectedPlacement.end_date }}</p>
                </div>
              </div>
              <Button v-if="selectedPlacement.status !== 'completed'" variant="outline" size="sm" @click="isEditing = true">
                Edit dates
              </Button>
            </template>
          </CardContent>
        </Card>

        <Card v-if="selectedPlacement.start_date">
          <CardHeader class="flex flex-row items-center justify-between gap-3">
            <div>
              <CardTitle>Attendance summary</CardTitle>
              <CardDescription>{{ currentMonth }}</CardDescription>
            </div>
            <div class="flex items-center gap-1">
              <Button variant="outline" size="icon-sm" aria-label="Previous month" @click="shiftMonth(-1)">
                <ChevronLeftIcon class="size-4" />
              </Button>
              <Button variant="outline" size="icon-sm" aria-label="Next month" @click="shiftMonth(1)">
                <ChevronRightIcon class="size-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div v-if="attendanceQuery.isLoading.value" class="space-y-2">
              <Skeleton v-for="i in 5" :key="i" class="h-8 w-full" />
            </div>
            <div v-else-if="attendanceQuery.isError.value" class="flex flex-col items-center gap-3 py-6 text-center">
              <p class="text-sm text-muted-foreground">Failed to load attendance for this month.</p>
              <Button variant="outline" size="sm" @click="attendanceQuery.refetch()">Try again</Button>
            </div>
            <EmptyState v-else-if="attendanceDays.length === 0" title="No attendance data for this month" />
            <ul v-else class="divide-y">
              <li v-for="day in attendanceDays" :key="day.date" class="flex items-center justify-between py-2">
                <span class="text-sm">{{ formatDate(day.date) }}</span>
                <StatusBadge v-bind="attendanceDayStatus(day.status)" />
              </li>
            </ul>
          </CardContent>
        </Card>

        <Card>
          <CardHeader class="flex flex-row items-center justify-between gap-3">
            <div>
              <CardTitle>Rate this company</CardTitle>
              <CardDescription>Share how your internship experience went.</CardDescription>
            </div>
            <Button size="sm" variant="outline" @click="openReviewDialog">
              <StarIcon class="size-4" />
              Rate this company
            </Button>
          </CardHeader>
        </Card>
      </template>
    </template>

    <Dialog :open="reviewDialogOpen" @update:open="reviewDialogOpen = $event">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Rate this company</DialogTitle>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmitReview">
          <div class="space-y-1.5">
            <label class="text-sm font-medium">Rating</label>
            <StarRatingInput v-model="reviewRating" />
            <p v-if="reviewForm.errors.value.rating" class="text-sm text-destructive">{{ reviewForm.errors.value.rating }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="rev-title" class="text-sm font-medium">Title (optional)</label>
            <Input id="rev-title" v-model="reviewTitle" v-bind="reviewTitleAttrs" placeholder="e.g. Pengalaman PKL" />
            <p v-if="reviewForm.errors.value.title" class="text-sm text-destructive">{{ reviewForm.errors.value.title }}</p>
          </div>
          <div class="space-y-1.5">
            <label for="rev-body" class="text-sm font-medium">Comment (optional)</label>
            <Textarea id="rev-body" v-model="reviewBody" v-bind="reviewBodyAttrs" rows="3" />
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
