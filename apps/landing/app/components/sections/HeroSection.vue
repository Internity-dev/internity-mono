<script setup lang="ts">
const config = useRuntimeConfig()
const { target, visible } = useReveal()

// Entrance storyboard (seconds from mount, above the fold so it plays on
// load, not on scroll). Spring physics via motion-v, not duration easing.
//   0.0   0.1     0.2     0.3   0.4        0.15 (parallel)
//   |-----|-------|-------|-----|          |
//   eyebrow headline subhead CTAs trust    illustration
const TIMING = { eyebrow: 0, headline: 0.1, subhead: 0.2, ctas: 0.3, trust: 0.4, illustration: 0.15 }
const SPRING = { type: 'spring', stiffness: 120, damping: 18 }
const fadeUp = { initial: { opacity: 0, y: 20 }, animate: { opacity: 1, y: 0 } }
</script>

<template>
  <section class="relative overflow-hidden">
    <ClientOnly>
      <ThreeHeroBackground class="opacity-70" />
    </ClientOnly>

    <div
      class="pointer-events-none absolute top-[-15%] right-[-10%] h-144 w-xl rounded-full bg-primary-400/25 blur-3xl"
      aria-hidden="true"
    />
    <div
      class="pointer-events-none absolute bottom-[-20%] left-[-15%] h-112 w-md rounded-full bg-accent-300/20 blur-3xl"
      aria-hidden="true"
    />

    <div class="relative mx-auto grid max-w-6xl gap-12 px-6 pt-28 pb-16 sm:pt-36 lg:grid-cols-[1.1fr_1fr] lg:items-center lg:gap-8">
      <div class="text-center lg:text-left">
        <Motion
          as="p"
          v-bind="fadeUp"
          :transition="{ ...SPRING, delay: TIMING.eyebrow }"
          class="mb-5 text-sm font-semibold tracking-[0.2em] text-primary-600 uppercase"
        >
          Internity
        </Motion>
        <Motion
          as="h1"
          v-bind="fadeUp"
          :transition="{ ...SPRING, delay: TIMING.headline }"
          class="font-display text-5xl leading-[1.05] font-semibold tracking-tight text-balance sm:text-6xl lg:text-7xl"
        >
          Kelola
          <span class="bg-linear-to-r from-primary-600 to-teal-500 bg-clip-text text-transparent">PKL</span>
          sekolahmu dalam satu aplikasi.
        </Motion>
        <Motion
          as="p"
          v-bind="fadeUp"
          :transition="{ ...SPRING, delay: TIMING.subhead }"
          class="mx-auto mt-6 max-w-xl text-lg text-foreground/60 lg:mx-0"
        >
          Sekolah pantau tempat PKL, siswa bikin laporan, perusahaan terima lamaran magang —
          dari pengajuan sampai sertifikat, semua dalam satu aplikasi.
        </Motion>

        <Motion
          as="div"
          v-bind="fadeUp"
          :transition="{ ...SPRING, delay: TIMING.ctas }"
          class="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row lg:justify-start"
        >
          <Motion
            as="a"
            :href="`${config.public.dashboardURL}/register`"
            :while-hover="{ scale: 1.04, y: -2 }"
            :while-press="{ scale: 0.97 }"
            :transition="{ type: 'spring', stiffness: 400, damping: 17 }"
            class="w-full rounded-full bg-primary-700 px-6 py-3 text-sm font-semibold text-white shadow-lg shadow-primary-900/10 sm:w-auto"
          >
            Daftar sebagai siswa
          </Motion>
          <Motion
            as="a"
            :href="`${config.public.dashboardURL}/login`"
            :while-hover="{ scale: 1.04, y: -2 }"
            :while-press="{ scale: 0.97 }"
            :transition="{ type: 'spring', stiffness: 400, damping: 17 }"
            class="w-full rounded-full border border-black/10 px-6 py-3 text-sm font-semibold text-foreground sm:w-auto"
          >
            Masuk
          </Motion>
        </Motion>
        <Motion
          as="p"
          v-bind="fadeUp"
          :transition="{ ...SPRING, delay: TIMING.trust }"
          class="mt-4 text-xs text-foreground/50"
        >
          Gratis untuk sekolah, siswa, dan perusahaan mitra
        </Motion>
      </div>

      <Motion
        as="div"
        :initial="{ opacity: 0, scale: 0.9 }"
        :animate="{ opacity: 1, scale: 1 }"
        :transition="{ ...SPRING, delay: TIMING.illustration }"
        class="relative mx-auto w-full max-w-md lg:max-w-none"
      >
        <img src="/illustrations/career-growth.svg" alt="" width="518" height="800" class="float-slow w-full" />

        <div class="badge-float-1 absolute top-4 -left-2 flex items-center gap-2 rounded-full border border-black/10 bg-white px-3 py-2 text-xs font-medium shadow-lg">
          <span class="size-2 rounded-full bg-success" />
          Lamaran diterima
        </div>
        <div class="badge-float-2 absolute bottom-8 -right-2 flex items-center gap-2 rounded-full border border-black/10 bg-white px-3 py-2 text-xs font-medium shadow-lg">
          <span class="size-2 rounded-full bg-primary-600" />
          Sertifikat siap diunduh
        </div>
      </Motion>
    </div>

    <div
      ref="target"
      class="reveal relative mx-auto max-w-3xl px-6 pt-8 pb-24 sm:pb-32"
      :class="{ 'reveal-visible': visible }"
    >
      <div class="overflow-hidden rounded-2xl border border-black/10 bg-white/80 shadow-2xl shadow-primary-900/10 backdrop-blur">
        <div class="flex items-center gap-1.5 border-b border-black/5 px-4 py-3">
          <span class="size-2.5 rounded-full bg-black/10" />
          <span class="size-2.5 rounded-full bg-black/10" />
          <span class="size-2.5 rounded-full bg-black/10" />
          <span class="ml-3 text-xs text-foreground/60">app.internity.id/attendance</span>
        </div>
        <div class="p-6 text-left sm:p-8">
          <p class="text-xs font-semibold tracking-wide text-foreground/60 uppercase">Presensi hari ini</p>
          <div class="mt-4 space-y-2">
            <div class="flex items-center justify-between rounded-xl bg-black/2 px-4 py-3">
              <span class="text-sm font-medium">Budi Santoso</span>
              <span class="rounded-full bg-success/15 px-2.5 py-1 text-xs font-medium text-success">Hadir · 07:52</span>
            </div>
            <div class="flex items-center justify-between rounded-xl bg-black/2 px-4 py-3">
              <span class="text-sm font-medium">Siti Aminah</span>
              <span class="rounded-full bg-warning/15 px-2.5 py-1 text-xs font-medium text-warning">Izin</span>
            </div>
            <div class="flex items-center justify-between rounded-xl bg-black/2 px-4 py-3">
              <span class="text-sm font-medium">Dewi Lestari</span>
              <button class="rounded-full bg-primary-700 px-3 py-1 text-xs font-medium text-white">Setujui jurnal</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.float-slow {
  animation: floatSlow 6s ease-in-out infinite;
}
@keyframes floatSlow {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-14px); }
}
.badge-float-1 {
  animation: badgeFloat1 5s ease-in-out infinite;
  animation-delay: 0.9s;
}
.badge-float-2 {
  animation: badgeFloat2 5.5s ease-in-out infinite;
  animation-delay: 1.1s;
}
@keyframes badgeFloat1 {
  0%, 100% { transform: translateY(0) rotate(-1deg); }
  50% { transform: translateY(-8px) rotate(1deg); }
}
@keyframes badgeFloat2 {
  0%, 100% { transform: translateY(0) rotate(1deg); }
  50% { transform: translateY(-10px) rotate(-1deg); }
}
@media (prefers-reduced-motion: reduce) {
  .float-slow,
  .badge-float-1,
  .badge-float-2 {
    animation: none;
  }
}
</style>
