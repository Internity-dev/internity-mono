<script setup lang="ts">
import { ref } from 'vue'
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { BellIcon, CheckCheckIcon, AlertCircleIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useListQuery } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { NotificationItem } from '@/types/content'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'

// GET /notifications responds with { notifications: [...], unread_count } rather
// than a flat array, so useListQuery's fetcher unwraps + reshapes it into the
// ApiSuccess<T[]> shape the composable expects. unread_count is captured as a
// side-effect ref since it lives outside the paginated `data` array.
const unreadCount = ref(0)

const { items, pagination, page, limit, isLoading, isError, refetch, setParams } = useListQuery<NotificationItem>(
  'notifications',
  async (params) => {
    const res = await http.get<ApiSuccess<{ notifications: NotificationItem[]; unread_count: number }>>(
      '/notifications',
      { params },
    )
    unreadCount.value = res.data.data.unread_count
    return { ...res.data, data: res.data.data.notifications }
  },
  { defaultSort: 'created_at', defaultOrder: 'desc' },
)

const queryClient = useQueryClient()
// Destructure (rather than keep the mutation object nested) so `isPending` is a
// genuine top-level ref binding that Vue auto-unwraps in the template.
const { mutate: markAllReadMutate, isPending: isMarkingRead } = useMutation({
  mutationFn: () => http.put('/notifications/mark-as-read'),
  onSuccess: () => {
    toast.success('All notifications marked as read')
    queryClient.invalidateQueries({ queryKey: ['notifications'] })
    queryClient.invalidateQueries({ queryKey: ['notifications', 'bell'] })
  },
  onError: () => toast.error('Failed to mark notifications as read. Please try again.'),
})

function formatDate(value: string) {
  return new Date(value).toLocaleString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Notifications" :description="unreadCount > 0 ? `You have ${unreadCount} unread notification${unreadCount === 1 ? '' : 's'}.` : 'You’re all caught up.'">
      <template #actions>
        <Button
          size="sm"
          variant="outline"
          :disabled="unreadCount === 0 || isMarkingRead"
          @click="markAllReadMutate()"
        >
          <CheckCheckIcon class="size-4" />
          {{ isMarkingRead ? 'Marking…' : 'Mark all as read' }}
        </Button>
      </template>
    </PageHeader>

    <div v-if="isLoading" class="space-y-3">
      <Skeleton v-for="i in 6" :key="i" class="h-16 w-full" />
    </div>

    <EmptyState
      v-else-if="isError"
      :icon="AlertCircleIcon"
      title="Couldn't load notifications"
      description="Something went wrong while loading notifications. Please try again."
      action-label="Retry"
      @action="refetch()"
    />

    <EmptyState
      v-else-if="items.length === 0"
      :icon="BellIcon"
      title="No notifications"
      description="You don't have any notifications yet."
    />

    <div v-else class="space-y-2">
      <Card
        v-for="n in items"
        :key="n.id"
        class="flex-row items-start gap-3 px-4 py-3"
        :class="n.read_at ? '' : 'bg-primary-50/40 dark:bg-primary-950/20'"
      >
        <div class="min-w-0 flex-1 space-y-0.5">
          <p class="text-sm" :class="n.read_at ? 'font-normal text-muted-foreground' : 'font-semibold text-foreground'">
            {{ n.title }}
          </p>
          <p class="text-sm text-muted-foreground">{{ n.body }}</p>
          <p class="text-xs text-muted-foreground">{{ formatDate(n.created_at) }}</p>
        </div>
        <span v-if="!n.read_at" class="mt-1 size-2 shrink-0 rounded-full bg-primary-500" aria-hidden="true" />
      </Card>
    </div>

    <ListPagination :page="page" :limit="limit" :total="pagination?.total ?? 0" @update:page="(p) => setParams({ page: p })" />
  </div>
</template>
