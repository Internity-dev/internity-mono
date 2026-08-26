<script setup lang="ts">
import { ref, computed } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import { ChevronDownIcon, SearchIcon, HelpCircleIcon, AlertCircleIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'
import type { Faq } from '@/types/content'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'

const search = ref('')
const expandedId = ref<number | null>(null)

const { data, isLoading, isError, refetch } = useQuery({
  queryKey: ['faqs'],
  queryFn: async () => {
    const res = await http.get<ApiSuccess<Faq[]>>('/faqs')
    return res.data.data
  },
  retry: false,
})

const faqs = computed(() => [...(data.value ?? [])].sort((a, b) => a.sort_order - b.sort_order))

const filteredFaqs = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return faqs.value
  return faqs.value.filter((f) => f.question.toLowerCase().includes(q) || f.answer.toLowerCase().includes(q))
})

function toggle(id: number) {
  expandedId.value = expandedId.value === id ? null : id
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Frequently Asked Questions" description="Search or browse common questions.">
      <template #actions>
        <div class="relative w-full sm:w-72">
          <SearchIcon class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="search" placeholder="Search FAQs…" class="pl-8" />
        </div>
      </template>
    </PageHeader>

    <div v-if="isLoading" class="space-y-3">
      <Skeleton v-for="i in 5" :key="i" class="h-12 w-full" />
    </div>

    <EmptyState
      v-else-if="isError"
      :icon="AlertCircleIcon"
      title="Couldn't load FAQs"
      description="Something went wrong while loading FAQs. Please try again."
      action-label="Retry"
      @action="refetch()"
    />

    <EmptyState
      v-else-if="filteredFaqs.length === 0"
      :icon="HelpCircleIcon"
      :title="search ? 'No matching questions' : 'No FAQs yet'"
      :description="search ? `Nothing matches “${search}”. Try a different search term.` : 'Check back later.'"
    />

    <Card v-else class="divide-y py-0">
      <div v-for="faq in filteredFaqs" :key="faq.id">
        <button
          type="button"
          class="flex w-full items-center justify-between gap-4 px-4 py-3.5 text-left text-sm font-medium text-foreground hover:bg-muted/50"
          :aria-expanded="expandedId === faq.id"
          @click="toggle(faq.id)"
        >
          <span>{{ faq.question }}</span>
          <ChevronDownIcon
            class="size-4 shrink-0 text-muted-foreground transition-transform duration-200"
            :class="{ 'rotate-180': expandedId === faq.id }"
          />
        </button>
        <CardContent v-if="expandedId === faq.id" class="px-4 pt-0 pb-4 text-sm whitespace-pre-wrap text-muted-foreground">
          {{ faq.answer }}
        </CardContent>
      </div>
    </Card>
  </div>
</template>
