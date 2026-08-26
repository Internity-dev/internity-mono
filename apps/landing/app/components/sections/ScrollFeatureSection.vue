<script setup lang="ts">
/**
 * Scroll-lock feature showcase: a tall (400vh) container with a sticky
 * inner panel. As the user scrolls through the container, scroll progress
 * (0-1) drives which of the 4 steps is shown, cross-fading illustration +
 * copy without the panel itself scrolling away. It's the "pinned" scrollytelling
 * pattern (Apple/Stripe-style product pages).
 *
 * Storyboard (progress 0 -> 1, 4 steps, one scroll-screen each):
 *   0.00 -------- 0.25 -------- 0.50 -------- 0.75 -------- 1.00
 *   [ step 0 ]  [ step 1 ]   [ step 2 ]   [ step 3 ]
 *   Presensi     Jurnal       Profil       Sertifikat
 */
const steps = [
  {
    title: 'Cari lowongan yang cocok',
    description: 'Cari dan saring lowongan PKL berdasarkan kategori dan skill yang kamu punya.',
    illustration: '/illustrations/person-search.svg',
  },
  {
    title: 'Presensi dengan bukti nyata',
    description: 'Check-in dan check-out disertai foto dan lokasi. Tidak ada lagi presensi yang diisi asal.',
    illustration: '/illustrations/monitoring-data.svg',
  },
  {
    title: 'Jurnal harian, ditinjau mentor',
    description: 'Siswa tulis kegiatan harian di aplikasi, mentor tinggal cek dan setujui kapan saja.',
    illustration: '/illustrations/grading-papers.svg',
  },
  {
    title: 'Sertifikat otomatis',
    description: 'Begitu nilai keluar, sertifikat PDF langsung bisa diunduh.',
    illustration: '/illustrations/winner.svg',
  },
]

const container = ref<HTMLElement | null>(null)
const panel = ref<HTMLElement | null>(null)
const progress = ref(0)
const activeStep = computed(() => {
  const i = Math.floor(progress.value * steps.length)
  return Math.min(steps.length - 1, Math.max(0, i))
})

// The pin is a desktop-only enhancement: it forces a long (400vh) scroll, so
// it's opt-in after mount rather than the default render. Reduced-motion
// users and small viewports (checked once here, same spot the GSAP setup
// runs) get a plain stacked list instead — see ThreeHeroBackground.vue and
// main.css for the same prefers-reduced-motion pattern used elsewhere.
const usePinnedScroll = ref(false)

let scrollTrigger: import('gsap/ScrollTrigger').ScrollTrigger | undefined

onMounted(async () => {
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const isDesktopViewport = window.matchMedia('(min-width: 768px)').matches
  if (prefersReducedMotion || !isDesktopViewport) return

  usePinnedScroll.value = true
  await nextTick()

  const { gsap } = await import('gsap')
  const { ScrollTrigger } = await import('gsap/ScrollTrigger')
  gsap.registerPlugin(ScrollTrigger)

  // Real scroll-lock: ScrollTrigger pins the sticky panel for the container's
  // full scroll range and hands back 0-1 progress on every scroll tick,
  // which drives `activeStep` (the actual step cross-fade is a plain Vue
  // <Transition>, not a GSAP tween, since it's a full content swap).
  scrollTrigger = ScrollTrigger.create({
    trigger: container.value!,
    pin: panel.value!,
    start: 'top top',
    end: 'bottom bottom',
    scrub: true,
    onUpdate: (self) => {
      progress.value = self.progress
    },
  })
})
onUnmounted(() => {
  scrollTrigger?.kill()
})
</script>

<template>
  <section ref="container" class="relative" :style="usePinnedScroll ? { height: `${steps.length * 100}vh` } : undefined">
    <h2 class="sr-only">Bagaimana Internity bekerja</h2>

    <div v-if="usePinnedScroll" ref="panel" class="flex h-screen items-center overflow-hidden">
      <div class="mx-auto grid w-full max-w-5xl grid-cols-1 items-center gap-10 px-6 lg:grid-cols-2">
        <div>
          <div class="mb-8 flex gap-2">
            <span
              v-for="(_, i) in steps"
              :key="i"
              class="h-1 flex-1 overflow-hidden rounded-full bg-black/10"
            >
              <span
                class="block h-full rounded-full bg-primary-600 transition-transform duration-300 ease-out"
                :style="{ transform: `scaleX(${i < activeStep ? 1 : i === activeStep ? Math.min(1, (progress * steps.length) - activeStep) : 0})`, transformOrigin: 'left' }"
              />
            </span>
          </div>

          <Transition name="feature-fade" mode="out-in">
            <div :key="activeStep">
              <p class="font-mono text-sm font-semibold text-primary-600">{{ String(activeStep + 1).padStart(2, '0') }} / {{ String(steps.length).padStart(2, '0') }}</p>
              <h3 class="mt-3 font-display text-2xl font-semibold tracking-tight sm:text-3xl">{{ steps[activeStep]!.title }}</h3>
              <p class="mt-4 max-w-md text-foreground/60">{{ steps[activeStep]!.description }}</p>
            </div>
          </Transition>
        </div>

        <div class="relative flex h-72 items-center justify-center sm:h-96">
          <Transition name="feature-fade" mode="out-in">
            <img
              :key="activeStep"
              :src="steps[activeStep]!.illustration"
              alt=""
              class="max-h-full max-w-full object-contain"
            />
          </Transition>
        </div>
      </div>
    </div>

    <!-- Static fallback: reduced-motion users and small viewports never get
         the scroll-jack pin, so every step just renders as a normal stacked
         block instead. -->
    <div v-else class="mx-auto max-w-3xl space-y-16 px-6 py-20 sm:py-28">
      <div v-for="(step, i) in steps" :key="step.title" class="text-center">
        <img :src="step.illustration" alt="" class="mx-auto h-56 max-w-full object-contain sm:h-72" />
        <p class="mt-6 font-mono text-sm font-semibold text-primary-600">{{ String(i + 1).padStart(2, '0') }} / {{ String(steps.length).padStart(2, '0') }}</p>
        <h3 class="mt-2 font-display text-2xl font-semibold tracking-tight sm:text-3xl">{{ step.title }}</h3>
        <p class="mx-auto mt-3 max-w-md text-foreground/60">{{ step.description }}</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.feature-fade-enter-active,
.feature-fade-leave-active {
  transition: opacity 0.35s ease, transform 0.35s ease;
}
.feature-fade-enter-from {
  opacity: 0;
  transform: translateY(1rem);
}
.feature-fade-leave-to {
  opacity: 0;
  transform: translateY(-1rem);
}
@media (prefers-reduced-motion: reduce) {
  .feature-fade-enter-active,
  .feature-fade-leave-active {
    transition: opacity 0.15s linear;
  }
  .feature-fade-enter-from,
  .feature-fade-leave-to {
    transform: none;
  }
}
</style>
