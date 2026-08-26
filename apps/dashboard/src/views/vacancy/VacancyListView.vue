<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import { BriefcaseIcon, UsersIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'
import type { Company, Vacancy } from '@/types/vacancy'
import { useListQuery } from '@/composables/useListQuery'
import { vacancyStatus } from '@/lib/status'
import PageHeader from '@/components/shared/PageHeader.vue'
import ListToolbar from '@/components/shared/ListToolbar.vue'
import ListPagination from '@/components/shared/ListPagination.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'

const { items, pagination, page, limit, search, setParams, isLoading, isError, refetch } = useListQuery<Vacancy>(
  'vacancies',
  (params) => http.get<ApiSuccess<Vacancy[]>>('/vacancies', { params }).then((res) => res.data),
)

// The vacancy list endpoint only returns company_id, not the company name.
// Resolve names for the companies present on the current page in one batch
// so each card can show a real name instead of "Company #id".
const companyIds = computed(() => Array.from(new Set(items.value.map((v) => v.company_id))))

const companiesQuery = useQuery({
  queryKey: computed(() => ['vacancy-companies', companyIds.value]),
  queryFn: async () => {
    const entries = await Promise.all(
      companyIds.value.map(async (id) => {
        try {
          const res = await http.get<ApiSuccess<Company>>(`/companies/${id}`)
          return [id, res.data.data.name] as const
        } catch {
          return [id, null] as const
        }
      }),
    )
    return Object.fromEntries(entries) as Record<number, string | null>
  },
  enabled: computed(() => companyIds.value.length > 0),
})

function companyName(companyId: number): string {
  return companiesQuery.data.value?.[companyId] ?? `Company #${companyId}`
}

function skillList(skills: string | null): string[] {
  if (!skills) return []
  return skills
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Vacancies" description="Browse open internship vacancies for your department." />

    <ListToolbar :model-value="search" placeholder="Search vacancies…" @update:model-value="(v) => setParams({ search: v })" />

    <div v-if="isLoading" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <Card v-for="i in 6" :key="i">
        <CardHeader>
          <Skeleton class="h-5 w-2/3" />
          <Skeleton class="h-4 w-1/3" />
        </CardHeader>
        <CardContent class="space-y-2">
          <Skeleton class="h-4 w-full" />
          <Skeleton class="h-4 w-5/6" />
        </CardContent>
      </Card>
    </div>

    <div v-else-if="isError" class="flex flex-col items-center gap-3 rounded-lg border border-dashed p-10 text-center">
      <p class="text-sm text-muted-foreground">Failed to load vacancies.</p>
      <Button variant="outline" size="sm" @click="refetch()">Try again</Button>
    </div>

    <EmptyState
      v-else-if="items.length === 0"
      :icon="BriefcaseIcon"
      title="No vacancies match your department right now"
      description="Check back later. New vacancies open up as companies post them."
    />

    <template v-else>
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <RouterLink
          v-for="vacancy in items"
          :key="vacancy.id"
          :to="{ name: 'vacancy-detail', params: { id: vacancy.id } }"
          class="block focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 rounded-xl"
        >
          <Card class="h-full transition-colors hover:border-primary/40 hover:bg-muted/30">
            <CardHeader>
              <div class="flex items-start justify-between gap-2">
                <CardTitle class="text-base">{{ vacancy.name }}</CardTitle>
                <StatusBadge v-bind="vacancyStatus(vacancy.status)" />
              </div>
              <CardDescription>{{ companyName(vacancy.company_id) }}</CardDescription>
            </CardHeader>
            <CardContent class="space-y-3 text-sm text-muted-foreground">
              <p v-if="vacancy.category">{{ vacancy.category }}</p>
              <div v-if="skillList(vacancy.skills).length" class="flex flex-wrap gap-1.5">
                <span
                  v-for="skill in skillList(vacancy.skills).slice(0, 4)"
                  :key="skill"
                  class="rounded-full bg-muted px-2 py-0.5 text-xs text-foreground"
                >
                  {{ skill }}
                </span>
              </div>
              <div class="flex items-center gap-1.5 text-xs">
                <UsersIcon class="size-3.5" />
                <span>{{ vacancy.slots }} slot{{ vacancy.slots === 1 ? '' : 's' }}</span>
              </div>
            </CardContent>
          </Card>
        </RouterLink>
      </div>

      <ListPagination :page="page" :limit="limit" :total="pagination?.total ?? 0" @update:page="(p) => setParams({ page: p })" />
    </template>
  </div>
</template>
