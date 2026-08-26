<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { NewspaperIcon, SearchIcon, AlertCircleIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { useListQuery } from '@/composables/useListQuery'
import type { ApiSuccess } from '@/types/api'
import type { News } from '@/types/content'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'

const { items, pagination, page, limit, search, isLoading, isError, refetch, setParams } = useListQuery<News>(
  'news',
  async (params) => {
    const res = await http.get<ApiSuccess<News[]>>('/news', { params })
    return res.data
  },
  { defaultSort: 'published_at', defaultOrder: 'desc' },
)

function excerpt(content: string, length = 140) {
  const clean = content.replace(/\s+/g, ' ').trim()
  return clean.length > length ? `${clean.slice(0, length)}…` : clean
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="News & Announcements" description="Latest updates shared with you.">
      <template #actions>
        <div class="relative w-full sm:w-64">
          <SearchIcon class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            :model-value="search"
            placeholder="Search news…"
            class="pl-8"
            @update:model-value="(v) => setParams({ search: String(v) })"
          />
        </div>
      </template>
    </PageHeader>

    <div v-if="isLoading" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <Card v-for="i in 6" :key="i">
        <CardHeader>
          <Skeleton class="h-5 w-3/4" />
          <Skeleton class="mt-2 h-3 w-1/3" />
        </CardHeader>
        <CardContent class="space-y-2">
          <Skeleton class="h-4 w-full" />
          <Skeleton class="h-4 w-5/6" />
        </CardContent>
      </Card>
    </div>

    <EmptyState
      v-else-if="isError"
      :icon="AlertCircleIcon"
      title="Couldn't load news"
      description="Something went wrong while loading announcements. Please try again."
      action-label="Retry"
      @action="refetch()"
    />

    <EmptyState
      v-else-if="items.length === 0"
      :icon="NewspaperIcon"
      title="No news yet"
      :description="search ? `No news matches “${search}”.` : 'Check back later for announcements.'"
    />

    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <RouterLink
        v-for="item in items"
        :key="item.id"
        :to="{ name: 'news-detail', params: { slug: item.slug } }"
        class="block h-full"
      >
        <Card class="h-full transition-colors hover:border-primary-300">
          <CardHeader>
            <CardTitle class="line-clamp-2">{{ item.title }}</CardTitle>
            <CardDescription>{{ formatDate(item.published_at ?? item.created_at) }}</CardDescription>
          </CardHeader>
          <CardContent>
            <p class="line-clamp-3 text-sm text-muted-foreground">{{ excerpt(item.content) }}</p>
          </CardContent>
        </Card>
      </RouterLink>
    </div>

    <ListPagination :page="page" :limit="limit" :total="pagination?.total ?? 0" @update:page="(p) => setParams({ page: p })" />
  </div>
</template>
