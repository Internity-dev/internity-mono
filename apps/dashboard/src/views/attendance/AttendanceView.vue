<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import axios from 'axios'
import { toast } from 'vue-sonner'
import {
  LogInIcon,
  LogOutIcon,
  CameraIcon,
  MapPinIcon,
  FileTextIcon,
  AlertCircleIcon,
  Building2Icon,
  CalendarOffIcon,
} from '@lucide/vue'
import { http } from '@/lib/http'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import { fetchMyPlacements, fetchPresenceStatuses, todayISODate, type Presence, type PresenceStatusKind } from '@/types/internship'
import { presenceStatusKind, approvalStatus } from '@/lib/status'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const auth = useAuthStore()
const queryClient = useQueryClient()
const route = useRoute()

// --- placements / company switcher ---
const placementsQuery = useQuery({ queryKey: ['my-placements'], queryFn: fetchMyPlacements })
const placements = computed(() => placementsQuery.data.value ?? [])
// Read straight from the route query (not `presencesList.filters`) so this
// stays independent of `presencesList`'s own declaration further below —
// its `enabled` option needs this value, and referencing the destructured
// result of the same call it's part of would be a circular self-reference.
const selectedCompanyId = computed(() => (route.query.company_id ? Number(route.query.company_id) : undefined))

// --- presence statuses (kind/label lookup for history rows) ---
const statusesQuery = useQuery({
  queryKey: computed(() => ['presence-statuses', auth.user?.school_id]),
  queryFn: () => fetchPresenceStatuses(auth.user!.school_id as number),
  enabled: computed(() => !!auth.user?.school_id),
})
const statusById = computed(() => new Map((statusesQuery.data.value ?? []).map((s) => [s.id, s])))

function formatTime(value: string | null | undefined) {
  if (!value) return '—'
  return new Date(value).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}
function formatDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

// --- today's presence ---
const todayStr = todayISODate()
const todayPresenceQuery = useQuery({
  queryKey: computed(() => ['today-presence', selectedCompanyId.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<Presence[]>>('/presences', {
      params: { company_id: selectedCompanyId.value, sort: 'date', order: 'desc', limit: 1, page: 1 },
    })
    return res.data.data[0] ?? null
  },
  enabled: computed(() => selectedCompanyId.value !== undefined),
})
const todayPresence = computed(() => {
  const p = todayPresenceQuery.data.value
  if (!p) return null
  return p.date?.slice(0, 10) === todayStr ? p : null
})
const todayStatusLabel = computed(() => {
  if (!todayPresence.value) return ''
  const status = statusById.value.get(todayPresence.value.presence_status_id)
  const label = status ? presenceStatusKind(status.kind).label : 'Excused'
  return todayPresence.value.is_approved ? label : `${label}, pending approval`
})

// --- check-in dialog: webcam + geolocation ---
const isCheckInOpen = ref(false)
const videoEl = ref<HTMLVideoElement | null>(null)
const canvasEl = ref<HTMLCanvasElement | null>(null)
let mediaStream: MediaStream | null = null
const cameraError = ref('')
const cameraReady = ref(false)
const capturedPhoto = ref<Blob | null>(null)
const capturedPhotoUrl = ref('')
const geo = ref<{ lat: number; lng: number } | null>(null)

function stopCamera() {
  mediaStream?.getTracks().forEach((track) => track.stop())
  mediaStream = null
  cameraReady.value = false
}

async function startCamera() {
  cameraError.value = ''
  if (!navigator.mediaDevices?.getUserMedia) {
    cameraError.value = 'Camera is not available on this device or browser. You can still check in without a photo.'
    return
  }
  try {
    mediaStream = await navigator.mediaDevices.getUserMedia({ video: true })
    await nextTick()
    if (videoEl.value) {
      videoEl.value.srcObject = mediaStream
      await videoEl.value.play()
    }
    cameraReady.value = true
  } catch {
    cameraError.value = 'Camera access was denied or unavailable. You can still check in without a photo.'
  }
}

function resetCheckInState() {
  stopCamera()
  if (capturedPhotoUrl.value) URL.revokeObjectURL(capturedPhotoUrl.value)
  capturedPhoto.value = null
  capturedPhotoUrl.value = ''
  cameraError.value = ''
  geo.value = null
}

watch(isCheckInOpen, (open) => {
  if (open) {
    resetCheckInState()
    startCamera()
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          geo.value = { lat: pos.coords.latitude, lng: pos.coords.longitude }
        },
        () => {
          // denied or unavailable — check-in still works without a location
        },
        { timeout: 8000 },
      )
    }
  } else {
    resetCheckInState()
  }
})

onUnmounted(() => stopCamera())

function capturePhoto() {
  const video = videoEl.value
  const canvas = canvasEl.value
  if (!video || !canvas) return
  canvas.width = video.videoWidth
  canvas.height = video.videoHeight
  const ctx = canvas.getContext('2d')
  ctx?.drawImage(video, 0, 0, canvas.width, canvas.height)
  canvas.toBlob(
    (blob) => {
      if (!blob) return
      capturedPhoto.value = blob
      capturedPhotoUrl.value = URL.createObjectURL(blob)
      stopCamera()
    },
    'image/jpeg',
    0.9,
  )
}

function retakePhoto() {
  if (capturedPhotoUrl.value) URL.revokeObjectURL(capturedPhotoUrl.value)
  capturedPhoto.value = null
  capturedPhotoUrl.value = ''
  startCamera()
}

function handle422(err: unknown) {
  if (axios.isAxiosError(err) && err.response?.status === 422) {
    toast.error(err.response.data?.message ?? 'Please check the form for errors')
  }
}

const checkInMutation = useMutation({
  mutationFn: async () => {
    const formData = new FormData()
    formData.append('company_id', String(selectedCompanyId.value))
    if (capturedPhoto.value) formData.append('photo', capturedPhoto.value, 'checkin.jpg')
    if (geo.value) {
      formData.append('lat', String(geo.value.lat))
      formData.append('lng', String(geo.value.lng))
    }
    return http.post('/presences/check-in', formData)
  },
  onSuccess: () => {
    toast.success('Checked in successfully')
    isCheckInOpen.value = false
    queryClient.invalidateQueries({ queryKey: ['today-presence'] })
    queryClient.invalidateQueries({ queryKey: ['presences'] })
  },
  onError: handle422,
})

const checkOutMutation = useMutation({
  mutationFn: () => http.post('/presences/check-out', { company_id: selectedCompanyId.value }),
  onSuccess: () => {
    toast.success("Checked out, have a good rest of the day")
    queryClient.invalidateQueries({ queryKey: ['today-presence'] })
    queryClient.invalidateQueries({ queryKey: ['presences'] })
  },
  onError: handle422,
})

// --- excuse dialog ---
const isExcuseOpen = ref(false)
const excuseAttachment = ref<File | null>(null)

const excuseSchema = toTypedSchema(
  z.object({
    date: z.string().min(1, 'Date is required'),
    kind: z.enum(['permitted', 'sick'], { errorMap: () => ({ message: 'Pick a reason' }) }),
    description: z.string().min(1, 'Please describe the reason').max(1000, 'Max 1000 characters'),
  }),
)
const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: excuseSchema,
  initialValues: { date: todayStr, kind: undefined, description: '' },
})
const [excuseDate, excuseDateAttrs] = defineField('date')
const [excuseKind, excuseKindAttrs] = defineField('kind')
const [excuseDescription, excuseDescriptionAttrs] = defineField('description')

function openExcuseDialog() {
  resetForm({ values: { date: todayStr, kind: undefined, description: '' } })
  excuseAttachment.value = null
  isExcuseOpen.value = true
}

function onExcuseFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  excuseAttachment.value = target.files?.[0] ?? null
}

const excuseMutation = useMutation({
  mutationFn: async (values: { date: string; kind: PresenceStatusKind; description: string }) => {
    const formData = new FormData()
    formData.append('company_id', String(selectedCompanyId.value))
    formData.append('date', values.date)
    formData.append('kind', values.kind)
    formData.append('description', values.description)
    if (excuseAttachment.value) formData.append('attachment', excuseAttachment.value)
    return http.post('/presences/excuse', formData)
  },
  onSuccess: () => {
    toast.success('Excuse submitted, awaiting approval')
    isExcuseOpen.value = false
    queryClient.invalidateQueries({ queryKey: ['today-presence'] })
    queryClient.invalidateQueries({ queryKey: ['presences'] })
  },
  onError: handle422,
})

const onExcuseSubmit = handleSubmit((values) => {
  excuseMutation.mutate({ date: values.date, kind: values.kind as PresenceStatusKind, description: values.description })
})

// --- history table ---
const presencesList = useListQuery<Presence, 'company_id'>(
  'presences',
  async (params) => {
    const res = await http.get<ApiSuccess<Presence[]>>('/presences', { params })
    return res.data
  },
  {
    defaultSort: 'date',
    defaultOrder: 'desc',
    filters: ['company_id'],
    enabled: () => selectedCompanyId.value !== undefined,
  },
)

// A student with only one placement never sees the switcher, so default the
// URL to their (only, or first) placement once loaded.
watch(
  placements,
  (list) => {
    if (selectedCompanyId.value === undefined && list.length > 0) presencesList.setParams({ company_id: list[0]?.company_id })
  },
  { immediate: true },
)

const companyModel = computed<number | undefined>({
  get: () => selectedCompanyId.value,
  set: (v) => presencesList.setParams({ company_id: v }),
})

const searchModel = computed<string>({
  get: () => presencesList.search.value,
  set: (v) => presencesList.setParams({ search: v }),
})

const presenceColumns: Column[] = [
  { key: 'date', label: 'Date', sortable: true },
  { key: 'check_in_at', label: 'Check-in' },
  { key: 'check_out_at', label: 'Check-out' },
  { key: 'status', label: 'Status' },
  { key: 'is_approved', label: 'Approval' },
  { key: 'description', label: 'Notes' },
]

function toggleSort(key: string) {
  const nextOrder = presencesList.sort.value === key && presencesList.order.value === 'asc' ? 'desc' : 'asc'
  presencesList.setParams({ sort: key, order: nextOrder })
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Attendance" description="Check in, check out, or file an excuse for today." />

    <div v-if="placementsQuery.isLoading.value" class="space-y-4">
      <Skeleton class="h-9 w-64" />
      <Skeleton class="h-36 w-full" />
      <Skeleton class="h-64 w-full" />
    </div>

    <EmptyState
      v-else-if="placementsQuery.isError.value"
      :icon="AlertCircleIcon"
      title="Couldn't load your placement"
      description="Something went wrong while loading your internship placement. Please try again."
      action-label="Retry"
      @action="placementsQuery.refetch()"
    />

    <EmptyState
      v-else-if="placements.length === 0"
      :icon="CalendarOffIcon"
      title="No active placement yet"
      description="Once your application is accepted and scheduled, attendance will be available here."
    />

    <template v-else>
      <div v-if="placements.length > 1" class="flex items-center gap-2">
        <Building2Icon class="size-4 text-muted-foreground" />
        <Select v-model="companyModel">
          <SelectTrigger class="w-64">
            <SelectValue placeholder="Select a placement" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="p in placements" :key="p.id" :value="p.company_id">{{ p.company_name }}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Today</CardTitle>
          <CardDescription>
            <template v-if="!todayPresence">You haven't checked in today.</template>
            <template v-else-if="todayPresence.check_in_at && !todayPresence.check_out_at">
              Checked in at {{ formatTime(todayPresence.check_in_at) }}. Don't forget to check out.
            </template>
            <template v-else-if="todayPresence.check_in_at && todayPresence.check_out_at">
              In {{ formatTime(todayPresence.check_in_at) }} · Out {{ formatTime(todayPresence.check_out_at) }}. Today's attendance is complete.
            </template>
            <template v-else>You've filed an excuse for today.</template>
          </CardDescription>
        </CardHeader>
        <CardContent class="flex flex-wrap items-center gap-3">
          <Button v-if="!todayPresence" :disabled="!selectedCompanyId" @click="isCheckInOpen = true">
            <LogInIcon class="size-4" />
            Check in
          </Button>
          <Button
            v-else-if="todayPresence.check_in_at && !todayPresence.check_out_at"
            :disabled="checkOutMutation.isPending.value"
            @click="checkOutMutation.mutate()"
          >
            <LogOutIcon class="size-4" />
            {{ checkOutMutation.isPending.value ? 'Checking out…' : 'Check out' }}
          </Button>
          <StatusBadge v-else tone="info" :label="todayStatusLabel" />

          <Button variant="outline" @click="openExcuseDialog">
            <FileTextIcon class="size-4" />
            File excuse
          </Button>
        </CardContent>
      </Card>

      <div class="space-y-3">
        <h2 class="text-lg font-semibold text-foreground">Attendance history</h2>
        <ListToolbar v-model="searchModel" placeholder="Search notes…" />
        <DataTable
          :columns="presenceColumns"
          :rows="presencesList.items.value"
          :is-loading="presencesList.isLoading.value"
          :sort="presencesList.sort.value"
          :order="presencesList.order.value"
          empty-title="No attendance records yet"
          empty-description="Your check-ins and excuses will show up here."
          @sort="toggleSort"
        >
          <template #cell-date="{ row }">{{ formatDate(row.date) }}</template>
          <template #cell-check_in_at="{ row }">{{ formatTime(row.check_in_at) }}</template>
          <template #cell-check_out_at="{ row }">{{ formatTime(row.check_out_at) }}</template>
          <template #cell-status="{ row }">
            <StatusBadge
              v-if="statusById.get(row.presence_status_id)"
              v-bind="presenceStatusKind(statusById.get(row.presence_status_id)!.kind)"
            />
            <span v-else class="text-muted-foreground">—</span>
          </template>
          <template #cell-is_approved="{ row }">
            <StatusBadge v-bind="approvalStatus(row.is_approved)" />
          </template>
          <template #cell-description="{ row }">
            <span class="line-clamp-2 max-w-xs text-muted-foreground">{{ row.description || '—' }}</span>
          </template>
        </DataTable>
        <ListPagination
          :page="presencesList.page.value"
          :limit="presencesList.limit.value"
          :total="presencesList.pagination.value?.total ?? 0"
          @update:page="(p) => presencesList.setParams({ page: p })"
        />
      </div>
    </template>

    <!-- Check-in dialog: webcam capture + best-effort geolocation -->
    <Dialog v-model:open="isCheckInOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Check in</DialogTitle>
          <DialogDescription>
            Take a quick photo to confirm you're on site. Location is captured automatically if you allow it.
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-3">
          <div class="relative aspect-video overflow-hidden rounded-lg border bg-muted">
            <video v-show="!capturedPhotoUrl" ref="videoEl" class="h-full w-full object-cover" autoplay muted playsinline></video>
            <img v-if="capturedPhotoUrl" :src="capturedPhotoUrl" class="h-full w-full object-cover" alt="Captured check-in photo" />
            <div
              v-if="cameraError && !capturedPhotoUrl"
              class="absolute inset-0 flex items-center justify-center bg-muted p-4 text-center text-sm text-muted-foreground"
            >
              {{ cameraError }}
            </div>
          </div>
          <canvas ref="canvasEl" class="hidden"></canvas>
          <p class="flex items-center gap-1.5 text-xs text-muted-foreground">
            <MapPinIcon class="size-3.5" />
            <span v-if="geo">Location captured</span>
            <span v-else>Location not available, you can still check in</span>
          </p>
        </div>

        <DialogFooter class="sm:justify-between">
          <Button variant="ghost" type="button" :disabled="checkInMutation.isPending.value" @click="checkInMutation.mutate()">
            Skip photo
          </Button>
          <div class="flex gap-2">
            <Button v-if="capturedPhotoUrl" variant="outline" type="button" @click="retakePhoto">Retake</Button>
            <Button v-if="!capturedPhotoUrl" type="button" :disabled="!cameraReady" @click="capturePhoto">
              <CameraIcon class="size-4" />
              Capture
            </Button>
            <Button v-else type="button" :disabled="checkInMutation.isPending.value" @click="checkInMutation.mutate()">
              {{ checkInMutation.isPending.value ? 'Checking in…' : 'Confirm check-in' }}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Excuse dialog -->
    <Dialog v-model:open="isExcuseOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>File an excuse</DialogTitle>
          <DialogDescription>Let your mentor know you'll be permitted or sick.</DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onExcuseSubmit">
          <div class="space-y-1.5">
            <Label for="excuse-date">Date</Label>
            <Input id="excuse-date" v-model="excuseDate" v-bind="excuseDateAttrs" type="date" />
            <p v-if="errors.date" class="text-sm text-destructive">{{ errors.date }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="excuse-kind">Reason</Label>
            <Select v-model="excuseKind" v-bind="excuseKindAttrs">
              <SelectTrigger id="excuse-kind" class="w-full">
                <SelectValue placeholder="Select a reason" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="permitted">Permitted</SelectItem>
                <SelectItem value="sick">Sick</SelectItem>
              </SelectContent>
            </Select>
            <p v-if="errors.kind" class="text-sm text-destructive">{{ errors.kind }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="excuse-description">Description</Label>
            <Textarea
              id="excuse-description"
              v-model="excuseDescription"
              v-bind="excuseDescriptionAttrs"
              rows="3"
              placeholder="Briefly explain why"
            />
            <p v-if="errors.description" class="text-sm text-destructive">{{ errors.description }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="excuse-attachment">Attachment (optional)</Label>
            <input
              id="excuse-attachment"
              type="file"
              accept="image/*,.pdf"
              class="block w-full text-sm text-muted-foreground file:mr-3 file:rounded-md file:border-0 file:bg-muted file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-foreground"
              @change="onExcuseFileChange"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" :disabled="excuseMutation.isPending.value" @click="isExcuseOpen = false">
              Cancel
            </Button>
            <Button type="submit" :disabled="excuseMutation.isPending.value">
              {{ excuseMutation.isPending.value ? 'Submitting…' : 'Submit excuse' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
