<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'
import type { Appliance, Vacancy } from '@/types/vacancy'
import { useListQuery } from '@/composables/useListQuery'
import { applianceStatus } from '@/lib/status'
import PageHeader from '@/components/shared/PageHeader.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import DataTable, { type Column } from '@/components/shared/DataTable.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import { Button } from '@/components/ui/button'

const queryClient = useQueryClient()

const { items, pagination, page, limit, sort, order, setParams, isLoading, isError, refetch } = useListQuery<Appliance>(
  'appliances',
  (params) => http.get<ApiSuccess<Appliance[]>>('/appliances', { params }).then((res) => res.data),
)

const columns: Column[] = [
  { key: 'vacancy', label: 'Vacancy' },
  { key: 'status', label: 'Status', sortable: true },
  { key: 'created_at', label: 'Applied', sortable: true },
  { key: 'actions', label: '', class: 'text-right' },
]

// The appliances list only returns vacancy_id — resolve the vacancy name for
// each row shown on the current page in one batch.
const vacancyIds = computed(() => Array.from(new Set(items.value.map((a) => a.vacancy_id))))

const vacanciesQuery = useQuery({
  queryKey: computed(() => ['appliance-vacancies', vacancyIds.value]),
  queryFn: async () => {
    const entries = await Promise.all(
      vacancyIds.value.map(async (id) => {
        try {
          const res = await http.get<ApiSuccess<Vacancy>>(`/vacancies/${id}`)
          return [id, res.data.data.name] as const
        } catch {
          return [id, null] as const
        }
      }),
    )
    return Object.fromEntries(entries) as Record<number, string | null>
  },
  enabled: computed(() => vacancyIds.value.length > 0),
})

function vacancyName(vacancyId: number): string {
  return vacanciesQuery.data.value?.[vacancyId] ?? `Vacancy #${vacancyId}`
}

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

function canCancel(status: Appliance['status']): boolean {
  return status === 'pending' || status === 'processed'
}

const cancelTarget = ref<Appliance | null>(null)

const cancelMutation = useMutation({
  mutationFn: (id: number) => http.put<ApiSuccess<Appliance>>(`/appliances/${id}/cancel`),
  onSuccess: () => {
    toast.success('Application canceled')
    cancelTarget.value = null
    queryClient.invalidateQueries({ queryKey: ['appliances'] })
  },
  onError: (err: unknown) => {
    const message = (err as { response?: { data?: { message?: string } } })?.response?.data?.message
    toast.error(message ?? 'Failed to cancel application')
  },
})

function confirmCancel() {
  if (cancelTarget.value) cancelMutation.mutate(cancelTarget.value.id)
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="My Applications" description="Track the status of the vacancies you've applied to." />

    <div v-if="isError" class="flex flex-col items-center gap-3 rounded-lg border border-dashed p-10 text-center">
      <p class="text-sm text-muted-foreground">Failed to load your applications.</p>
      <Button variant="outline" size="sm" @click="refetch()">Try again</Button>
    </div>

    <template v-else>
      <DataTable
        :columns="columns"
        :rows="items"
        :is-loading="isLoading"
        :sort="sort"
        :order="order"
        empty-title="No applications yet"
        empty-description="Browse open vacancies and apply to get started."
        @sort="(key) => setParams({ sort: key, order: sort === key && order === 'asc' ? 'desc' : 'asc' })"
      >
        <template #cell-vacancy="{ row }">
          <RouterLink :to="{ name: 'vacancy-detail', params: { id: row.vacancy_id } }" class="font-medium text-primary-700 hover:underline">
            {{ vacancyName(row.vacancy_id) }}
          </RouterLink>
        </template>
        <template #cell-status="{ row }">
          <StatusBadge v-bind="applianceStatus(row.status)" />
        </template>
        <template #cell-created_at="{ row }">
          {{ formatDate(row.created_at) }}
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end">
            <Button v-if="canCancel(row.status)" variant="outline" size="sm" @click="cancelTarget = row">Cancel</Button>
          </div>
        </template>
      </DataTable>

      <ListPagination :page="page" :limit="limit" :total="pagination?.total ?? 0" @update:page="(p) => setParams({ page: p })" />
    </template>

    <ConfirmDialog
      :open="!!cancelTarget"
      title="Cancel this application?"
      :description="cancelTarget ? `This will cancel your application for ${vacancyName(cancelTarget.vacancy_id)}. This cannot be undone.` : undefined"
      confirm-label="Cancel application"
      :destructive="true"
      :is-loading="cancelMutation.isPending.value"
      @update:open="(v) => { if (!v) cancelTarget = null }"
      @confirm="confirmCancel"
    />
  </div>
</template>
