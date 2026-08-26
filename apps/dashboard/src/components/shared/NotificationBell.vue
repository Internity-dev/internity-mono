<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import { BellIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuLabel, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import EmptyState from '@/components/shared/EmptyState.vue'
import type { ApiSuccess } from '@/types/api'

interface NotificationItem {
  id: number
  type: string
  title: string
  body: string
  read_at: string | null
  created_at: string
}

const { data } = useQuery({
  queryKey: ['notifications', 'bell'],
  queryFn: async () => {
    const res = await http.get<ApiSuccess<{ notifications: NotificationItem[]; unread_count: number }>>(
      '/notifications',
      { params: { limit: 5, sort: 'created_at', order: 'desc' } },
    )
    return res.data.data
  },
  refetchInterval: 60_000,
})

const items = computed(() => data.value?.notifications ?? [])
const unreadCount = computed(() => data.value?.unread_count ?? 0)
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" size="icon" class="relative" aria-label="Notifications" data-tour="notifications">
        <BellIcon class="size-4" />
        <span
          v-if="unreadCount > 0"
          class="absolute top-1 right-1 flex size-4 items-center justify-center rounded-full bg-danger text-[10px] font-medium text-white"
        >
          {{ unreadCount > 9 ? '9+' : unreadCount }}
        </span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-80">
      <DropdownMenuLabel>Notifications</DropdownMenuLabel>
      <DropdownMenuSeparator />
      <EmptyState v-if="items.length === 0" title="No notifications yet" class="py-6" />
      <DropdownMenuItem v-for="n in items" :key="n.id" class="flex-col items-start gap-0.5 whitespace-normal">
        <p class="text-sm font-medium" :class="{ 'text-muted-foreground': n.read_at }">{{ n.title }}</p>
        <p class="line-clamp-2 text-xs text-muted-foreground">{{ n.body }}</p>
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem as-child>
        <RouterLink to="/notifications" class="justify-center text-sm font-medium text-primary-700">
          See all
        </RouterLink>
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
