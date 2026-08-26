<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import { ArrowLeftIcon, NewspaperIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'
import type { News } from '@/types/content'
import EmptyState from '@/components/shared/EmptyState.vue'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'

const route = useRoute()
const router = useRouter()
const slug = computed(() => route.params.slug as string)

const { data, isLoading, isError } = useQuery({
  queryKey: computed(() => ['news', 'detail', slug.value]),
  queryFn: async () => {
    const res = await http.get<ApiSuccess<News>>(`/news/${slug.value}`)
    return res.data.data
  },
  retry: false,
})

function formatDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}
</script>

<template>
  <div class="space-y-6">
    <Button as-child variant="ghost" size="sm" class="-ml-2">
      <RouterLink :to="{ name: 'news' }">
        <ArrowLeftIcon class="size-4" />
        Back to news
      </RouterLink>
    </Button>

    <div v-if="isLoading" class="space-y-3">
      <Skeleton class="h-8 w-2/3" />
      <Skeleton class="h-4 w-1/4" />
      <Skeleton class="h-40 w-full" />
    </div>

    <EmptyState
      v-else-if="isError || !data"
      :icon="NewspaperIcon"
      title="News post not found"
      description="This announcement may have been removed or the link is incorrect."
      action-label="Back to news"
      @action="router.push({ name: 'news' })"
    />

    <Card v-else>
      <CardHeader>
        <CardTitle as="h1" class="text-2xl">{{ data.title }}</CardTitle>
        <CardDescription>{{ formatDate(data.published_at ?? data.created_at) }}</CardDescription>
      </CardHeader>
      <CardContent>
        <div class="whitespace-pre-wrap text-sm leading-relaxed text-foreground">{{ data.content }}</div>
      </CardContent>
    </Card>
  </div>
</template>
