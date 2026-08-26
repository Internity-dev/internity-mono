<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { useQuery, useMutation } from '@tanstack/vue-query'
import { toast } from 'vue-sonner'
import { BookmarkIcon, BookmarkCheckIcon, ChevronLeftIcon, UsersIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import type { ApiSuccess } from '@/types/api'
import type { Company, Vacancy } from '@/types/vacancy'
import { vacancyStatus } from '@/lib/status'
import PageHeader from '@/components/shared/PageHeader.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Separator } from '@/components/ui/separator'

const route = useRoute()
const router = useRouter()
const vacancyId = computed(() => Number(route.params.id))

const vacancyQuery = useQuery({
  queryKey: computed(() => ['vacancy', vacancyId.value]),
  queryFn: () => http.get<ApiSuccess<Vacancy>>(`/vacancies/${vacancyId.value}`).then((res) => res.data.data),
  enabled: computed(() => Number.isFinite(vacancyId.value)),
})

const companyQuery = useQuery({
  queryKey: computed(() => ['company', vacancyQuery.data.value?.company_id]),
  queryFn: () =>
    http.get<ApiSuccess<Company>>(`/companies/${vacancyQuery.data.value!.company_id}`).then((res) => res.data.data),
  enabled: computed(() => !!vacancyQuery.data.value?.company_id),
})

function skillList(skills: string | null | undefined): string[] {
  if (!skills) return []
  return skills
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

// The vacancy detail response has no is_saved flag — track the saved state
// locally (optimistic) since there's no read endpoint to derive it from.
const isSaved = ref(false)

// No onError here (or below): http.ts's response interceptor already toasts
// every failed mutation (it has the backend's own message to show, which is
// exactly what a local fallback would have repeated). A local onError that
// ALSO calls toast.error double-toasts the same failure — confirmed live on
// Apply now, where a single 403 produced two identical toasts. onSuccess
// still needs to be local (optimistic isSaved flip, redirect, etc.) since
// the interceptor has no notion of "this specific mutation succeeded."
const saveMutation = useMutation({
  mutationFn: () => http.post(`/vacancies/${vacancyId.value}/save`),
  onSuccess: () => {
    isSaved.value = true
    toast.success('Vacancy saved')
  },
})

const unsaveMutation = useMutation({
  mutationFn: () => http.delete(`/vacancies/${vacancyId.value}/save`),
  onSuccess: () => {
    isSaved.value = false
    toast.success('Vacancy removed from saved list')
  },
})

const message = ref('')

const applyMutation = useMutation({
  mutationFn: () =>
    http.post('/appliances', { vacancy_id: vacancyId.value, message: message.value.trim() || undefined }),
  onSuccess: () => {
    toast.success('Application submitted')
    router.push('/my-applications')
  },
})
</script>

<template>
  <div class="space-y-6">
    <RouterLink to="/vacancies" class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
      <ChevronLeftIcon class="size-4" />
      Back to vacancies
    </RouterLink>

    <div v-if="vacancyQuery.isLoading.value" class="space-y-4">
      <Skeleton class="h-8 w-1/2" />
      <Skeleton class="h-40 w-full" />
    </div>

    <div v-else-if="vacancyQuery.isError.value" class="flex flex-col items-center gap-3 rounded-lg border border-dashed p-10 text-center">
      <p class="text-sm text-muted-foreground">Failed to load this vacancy.</p>
      <Button variant="outline" size="sm" @click="vacancyQuery.refetch()">Try again</Button>
    </div>

    <template v-else-if="vacancyQuery.data.value">
      <PageHeader :title="vacancyQuery.data.value.name" :description="companyQuery.data.value?.name ?? `Company #${vacancyQuery.data.value.company_id}`">
        <template #actions>
          <StatusBadge v-bind="vacancyStatus(vacancyQuery.data.value.status)" />
        </template>
      </PageHeader>

      <div class="grid gap-6 lg:grid-cols-3">
        <Card class="lg:col-span-2">
          <CardHeader>
            <CardTitle>Description</CardTitle>
          </CardHeader>
          <CardContent class="space-y-4">
            <p v-if="vacancyQuery.data.value.description" class="whitespace-pre-line text-sm text-muted-foreground">
              {{ vacancyQuery.data.value.description }}
            </p>
            <p v-else class="text-sm text-muted-foreground">No description provided.</p>

            <template v-if="skillList(vacancyQuery.data.value.skills).length">
              <Separator />
              <div>
                <p class="mb-2 text-sm font-medium">Skills</p>
                <div class="flex flex-wrap gap-1.5">
                  <span
                    v-for="skill in skillList(vacancyQuery.data.value.skills)"
                    :key="skill"
                    class="rounded-full bg-muted px-2 py-0.5 text-xs"
                  >
                    {{ skill }}
                  </span>
                </div>
              </div>
            </template>

            <Separator />
            <div class="flex items-center gap-1.5 text-sm text-muted-foreground">
              <UsersIcon class="size-4" />
              <span>{{ vacancyQuery.data.value.slots }} slot{{ vacancyQuery.data.value.slots === 1 ? '' : 's' }}</span>
            </div>
          </CardContent>
        </Card>

        <div class="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Actions</CardTitle>
              <CardDescription>Save this vacancy or apply now.</CardDescription>
            </CardHeader>
            <CardContent class="space-y-4">
              <Button
                variant="outline"
                class="w-full"
                :disabled="saveMutation.isPending.value || unsaveMutation.isPending.value"
                @click="isSaved ? unsaveMutation.mutate() : saveMutation.mutate()"
              >
                <component :is="isSaved ? BookmarkCheckIcon : BookmarkIcon" class="size-4" />
                {{ isSaved ? 'Saved' : 'Save vacancy' }}
              </Button>

              <div class="space-y-1.5">
                <label for="message" class="text-sm font-medium">Message (optional)</label>
                <Textarea
                  id="message"
                  v-model="message"
                  placeholder="Add a short note to your application…"
                  maxlength="2000"
                  rows="4"
                />
              </div>

              <Button
                class="w-full"
                :disabled="applyMutation.isPending.value || vacancyQuery.data.value.status !== 'open'"
                @click="applyMutation.mutate()"
              >
                {{ applyMutation.isPending.value ? 'Submitting…' : 'Apply now' }}
              </Button>
              <p v-if="vacancyQuery.data.value.status !== 'open'" class="text-xs text-muted-foreground">
                This vacancy is closed for applications.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </template>
  </div>
</template>
