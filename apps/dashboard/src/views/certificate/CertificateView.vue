<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useQuery, useMutation } from '@tanstack/vue-query'
import axios from 'axios'
import { toast } from 'vue-sonner'
import { DownloadIcon, AwardIcon, AlertCircleIcon, Building2Icon, CalendarOffIcon, TriangleAlertIcon } from '@lucide/vue'
import { http } from '@/lib/http'
import { fetchMyPlacements } from '@/types/internship'
import PageHeader from '@/components/shared/PageHeader.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

// --- placements / company switcher — a certificate is per-placement ---
const placementsQuery = useQuery({ queryKey: ['my-placements'], queryFn: fetchMyPlacements })
const placements = computed(() => placementsQuery.data.value ?? [])
const selectedCompanyId = ref<number | undefined>(undefined)
watch(
  placements,
  (list) => {
    if (selectedCompanyId.value === undefined && list.length > 0) selectedCompanyId.value = list[0]?.company_id
  },
  { immediate: true },
)
const selectedPlacement = computed(() => placements.value.find((p) => p.company_id === selectedCompanyId.value))

// Clears whenever the placement changes so a stale blocker message from one
// placement doesn't linger after switching to another.
watch(selectedCompanyId, () => {
  errorMessage.value = ''
})

const errorMessage = ref('')

async function extractErrorMessage(err: unknown): Promise<string> {
  const fallback = 'Something went wrong while generating your certificate. Please try again.'
  if (!axios.isAxiosError(err) || !err.response) return fallback
  const data: unknown = err.response.data
  // responseType: 'blob' means an *error* response body also arrives as a
  // Blob (axios doesn't know it's JSON until it looks) — parse it to get
  // the real backend message instead of a generic one.
  if (data instanceof Blob) {
    try {
      const parsed = JSON.parse(await data.text()) as { message?: string }
      return parsed.message ?? fallback
    } catch {
      return fallback
    }
  }
  if (data && typeof data === 'object' && 'message' in data) {
    return (data as { message?: string }).message ?? fallback
  }
  return fallback
}

const downloadMutation = useMutation({
  mutationFn: () =>
    http.get('/certificate', {
      params: { company_id: selectedCompanyId.value },
      responseType: 'blob',
    }),
  onMutate: () => {
    errorMessage.value = ''
  },
  onSuccess: (res) => {
    const blob = new Blob([res.data], { type: 'application/pdf' })
    const url = URL.createObjectURL(blob)
    const disposition = res.headers['content-disposition'] as string | undefined
    const filename = disposition?.match(/filename="?([^";]+)"?/)?.[1] ?? 'certificate.pdf'
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
    toast.success('Certificate downloaded')
  },
  onError: async (err) => {
    const message = await extractErrorMessage(err)
    errorMessage.value = message
    toast.error(message)
  },
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Certificate" description="Download your internship completion certificate." />

    <div v-if="placementsQuery.isLoading.value" class="space-y-4">
      <Skeleton class="h-9 w-64" />
      <Skeleton class="h-48 w-full" />
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
      description="Once your internship placement is set up, your certificate will be available here."
    />

    <template v-else>
      <div v-if="placements.length > 1" class="flex items-center gap-2">
        <Building2Icon class="size-4 text-muted-foreground" />
        <Select v-model="selectedCompanyId">
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
          <div class="mb-2 flex size-10 items-center justify-center rounded-lg bg-primary-50 text-primary-700 dark:bg-primary-950">
            <AwardIcon class="size-5" />
          </div>
          <CardTitle>Internship completion certificate</CardTitle>
          <CardDescription>
            <template v-if="selectedPlacement">For your placement at {{ selectedPlacement.company_name }}. </template>
            Generated once your mentor has entered your scores and your NIS is on file with your school.
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          <Alert v-if="errorMessage" variant="destructive">
            <TriangleAlertIcon />
            <AlertTitle>Can't generate your certificate yet</AlertTitle>
            <AlertDescription>
              {{ errorMessage }}
              <RouterLink v-if="errorMessage.toLowerCase().includes('nis')" to="/profile" class="font-medium">
                Update your NIS in your profile.
              </RouterLink>
            </AlertDescription>
          </Alert>

          <Button :disabled="!selectedCompanyId || downloadMutation.isPending.value" @click="downloadMutation.mutate()">
            <DownloadIcon class="size-4" />
            {{ downloadMutation.isPending.value ? 'Preparing your certificate…' : 'Download certificate' }}
          </Button>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
