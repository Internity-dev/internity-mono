<script setup lang="ts">
interface Faq {
  id: number
  question: string
  answer: string
  sort_order: number
}

interface ApiEnvelope<T> {
  success: boolean
  data: T
  message: string
}

const config = useRuntimeConfig()

// Client-side fetch on mount. FAQ content changes independently of a landing
// deploy, so an admin's edit shouldn't require a rebuild.
// /faqs now paginates (defaults to 20); this page shows the whole list, so
// ask for a generous ceiling instead.
const { data, pending, error } = await useFetch<ApiEnvelope<Faq[]>>(`${config.public.apiBaseURL}/faqs`, {
  server: false,
  params: { limit: 100 },
})

const search = ref('')
const faqs = computed(() => data.value?.data ?? [])
const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return faqs.value
  return faqs.value.filter((f) => f.question.toLowerCase().includes(q) || f.answer.toLowerCase().includes(q))
})

const expandedId = ref<number | null>(null)
function toggle(id: number) {
  expandedId.value = expandedId.value === id ? null : id
}

useSeoMeta({ title: 'FAQ | Internity' })
</script>

<template>
  <div class="mx-auto max-w-2xl px-6 py-20">
    <h1 class="font-display text-3xl font-semibold tracking-tight">Pertanyaan Umum</h1>

    <input
      v-model="search"
      type="search"
      placeholder="Cari pertanyaan…"
      aria-label="Cari pertanyaan"
      class="mt-8 w-full rounded-full border border-black/10 bg-transparent px-4 py-2.5 text-sm outline-none focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
    />

    <div class="mt-8 space-y-2">
      <p v-if="pending" class="text-sm text-foreground/50">Memuat…</p>
      <p v-else-if="error" class="text-sm text-danger">Gagal memuat FAQ. Coba muat ulang halaman.</p>
      <p v-else-if="filtered.length === 0" class="text-sm text-foreground/50">Tidak ada pertanyaan yang cocok.</p>

      <div v-for="faq in filtered" :key="faq.id" class="rounded-xl border border-black/5">
        <button
          class="flex w-full items-center justify-between gap-4 px-5 py-4 text-left text-sm font-medium"
          :aria-expanded="expandedId === faq.id"
          :aria-controls="`faq-panel-${faq.id}`"
          @click="toggle(faq.id)"
        >
          {{ faq.question }}
          <svg
            xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
            class="shrink-0 text-foreground/40 transition-transform"
            :class="{ 'rotate-180': expandedId === faq.id }"
          >
            <path d="M6 9l6 6 6-6" />
          </svg>
        </button>
        <div v-if="expandedId === faq.id" :id="`faq-panel-${faq.id}`" class="px-5 pb-4 text-sm leading-relaxed text-foreground/60">
          {{ faq.answer }}
        </div>
      </div>
    </div>
  </div>
</template>
