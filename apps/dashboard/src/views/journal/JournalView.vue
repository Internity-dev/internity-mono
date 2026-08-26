<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { PlusIcon, PencilIcon, LockIcon, AlertCircleIcon, Building2Icon, CalendarOffIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useListQuery } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import { fetchMyPlacements, todayISODate, type Journal } from '@/types/internship'
import { approvalStatus } from '@/lib/status'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const queryClient = useQueryClient()
const route = useRoute()

// --- placements / company switcher ---
const placementsQuery = useQuery({ queryKey: ['my-placements'], queryFn: fetchMyPlacements })
const placements = computed(() => placementsQuery.data.value ?? [])
// Read straight from the route query (not `journalsList.filters`) so this
// stays independent of `journalsList`'s own declaration further below — its
// `enabled` option needs this value, and referencing the destructured
// result of the same call it's part of would be a circular self-reference.
const selectedCompanyId = computed(() => (route.query.company_id ? Number(route.query.company_id) : undefined))

function formatDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
function truncate(value: string | null, len = 80) {
  if (!value) return '—'
  return value.length > len ? `${value.slice(0, len)}…` : value
}

// --- history ---
const journalsList = useListQuery<Journal, 'company_id'>(
  'journals',
  async (params) => {
    const res = await http.get<ApiSuccess<Journal[]>>('/journals', { params })
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
    if (selectedCompanyId.value === undefined && list.length > 0) journalsList.setParams({ company_id: list[0]?.company_id })
  },
  { immediate: true },
)

const companyModel = computed<number | undefined>({
  get: () => selectedCompanyId.value,
  set: (v) => journalsList.setParams({ company_id: v }),
})

const searchModel = computed<string>({
  get: () => journalsList.search.value,
  set: (v) => journalsList.setParams({ search: v }),
})

const columns: Column[] = [
  { key: 'date', label: 'Date', sortable: true },
  { key: 'work_type', label: 'Work type' },
  { key: 'description', label: 'Description' },
  { key: 'is_approved', label: 'Status' },
  { key: 'actions', label: '', class: 'text-right' },
]

function toggleSort(key: string) {
  const nextOrder = journalsList.sort.value === key && journalsList.order.value === 'asc' ? 'desc' : 'asc'
  journalsList.setParams({ sort: key, order: nextOrder })
}

// --- create / edit dialog ---
const dialogOpen = ref(false)
const editing = ref<Journal | null>(null)

const formSchema = toTypedSchema(
  z.object({
    date: z.string().min(1, 'Date is required'),
    work_type: z.string().min(1, 'Work type is required').max(255, 'Max 255 characters'),
    description: z.string().min(1, 'Description is required'),
  }),
)
const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: formSchema,
  initialValues: { date: todayISODate(), work_type: '', description: '' },
})
const [date, dateAttrs] = defineField('date')
const [workType, workTypeAttrs] = defineField('work_type')
const [description, descriptionAttrs] = defineField('description')

function openCreate() {
  editing.value = null
  resetForm({ values: { date: todayISODate(), work_type: '', description: '' } })
  dialogOpen.value = true
}

function openEdit(row: Journal) {
  if (row.is_approved) return
  editing.value = row
  resetForm({ values: { date: row.date.slice(0, 10), work_type: row.work_type ?? '', description: row.description ?? '' } })
  dialogOpen.value = true
}

function handle422(err: unknown) {
  if (axios.isAxiosError(err) && err.response?.status === 422) {
    toast.error(err.response.data?.message ?? 'Please check the form for errors')
  }
}

const upsertMutation = useMutation({
  mutationFn: (values: { date: string; work_type: string; description: string }) =>
    http.post('/journals', {
      company_id: selectedCompanyId.value,
      date: values.date,
      work_type: values.work_type,
      description: values.description,
    }),
  onSuccess: () => {
    toast.success(editing.value ? 'Journal entry updated' : 'Journal entry saved')
    dialogOpen.value = false
    queryClient.invalidateQueries({ queryKey: ['journals'] })
  },
  onError: handle422,
})

const onSubmit = handleSubmit((values) => {
  upsertMutation.mutate(values)
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Journal" description="Log your daily work activities.">
      <template #actions>
        <Button :disabled="!selectedCompanyId" @click="openCreate">
          <PlusIcon class="size-4" />
          Add journal entry
        </Button>
      </template>
    </PageHeader>

    <div v-if="placementsQuery.isLoading.value" class="space-y-4">
      <Skeleton class="h-9 w-64" />
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
      description="Once your application is accepted and scheduled, you'll be able to log journal entries here."
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

      <ListToolbar v-model="searchModel" placeholder="Search work type or description…" />

      <DataTable
        :columns="columns"
        :rows="journalsList.items.value"
        :is-loading="journalsList.isLoading.value"
        :sort="journalsList.sort.value"
        :order="journalsList.order.value"
        empty-title="No journal entries yet"
        empty-description="Log what you worked on each day you check in."
        @sort="toggleSort"
      >
        <template #cell-date="{ row }">{{ formatDate(row.date) }}</template>
        <template #cell-work_type="{ row }">{{ row.work_type || '—' }}</template>
        <template #cell-description="{ row }">
          <span class="line-clamp-2 max-w-sm text-muted-foreground">{{ truncate(row.description) }}</span>
        </template>
        <template #cell-is_approved="{ row }">
          <StatusBadge v-bind="approvalStatus(row.is_approved)" />
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end">
            <Button v-if="!row.is_approved" variant="ghost" size="icon-sm" aria-label="Edit journal entry" @click="openEdit(row)">
              <PencilIcon class="size-4" />
            </Button>
            <span v-else class="inline-flex items-center gap-1 text-xs text-muted-foreground" title="Approved entries can't be edited">
              <LockIcon class="size-3.5" />
            </span>
          </div>
        </template>
      </DataTable>
      <ListPagination
        :page="journalsList.page.value"
        :limit="journalsList.limit.value"
        :total="journalsList.pagination.value?.total ?? 0"
        @update:page="(p) => journalsList.setParams({ page: p })"
      />
    </template>

    <Dialog v-model:open="dialogOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ editing ? 'Edit journal entry' : 'New journal entry' }}</DialogTitle>
          <DialogDescription>
            {{ editing ? `Update your entry for ${formatDate(editing.date)}.` : "You can only journal a day you've checked in." }}
          </DialogDescription>
        </DialogHeader>
        <form class="space-y-4" novalidate @submit="onSubmit">
          <div class="space-y-1.5">
            <Label for="journal-date">Date</Label>
            <Input id="journal-date" v-model="date" v-bind="dateAttrs" type="date" :disabled="!!editing" />
            <p v-if="errors.date" class="text-sm text-destructive">{{ errors.date }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="journal-work-type">Work type</Label>
            <Input id="journal-work-type" v-model="workType" v-bind="workTypeAttrs" placeholder="e.g. Backend development" />
            <p v-if="errors.work_type" class="text-sm text-destructive">{{ errors.work_type }}</p>
          </div>
          <div class="space-y-1.5">
            <Label for="journal-description">Description</Label>
            <Textarea
              id="journal-description"
              v-model="description"
              v-bind="descriptionAttrs"
              rows="4"
              placeholder="What did you work on today?"
            />
            <p v-if="errors.description" class="text-sm text-destructive">{{ errors.description }}</p>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" :disabled="upsertMutation.isPending.value" @click="dialogOpen = false">
              Cancel
            </Button>
            <Button type="submit" :disabled="upsertMutation.isPending.value">
              {{ upsertMutation.isPending.value ? 'Saving…' : 'Save entry' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>
