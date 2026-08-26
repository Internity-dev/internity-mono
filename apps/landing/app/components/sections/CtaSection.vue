<script setup lang="ts">
const config = useRuntimeConfig()
const { target, visible } = useReveal()
</script>

<template>
  <section class="border-t border-black/5">
    <div ref="target" class="reveal mx-auto max-w-3xl px-6 py-20 text-center sm:py-28" :class="{ 'reveal-visible': visible }">
      <h2 class="font-display text-3xl font-semibold tracking-tight sm:text-4xl">Mulai kelola PKL sekolahmu hari ini</h2>
      <p class="mx-auto mt-3 max-w-md text-foreground/60">
        Daftar dalam hitungan menit, langsung pakai kode undangan dari sekolahmu.
      </p>
      <div class="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
        <a
          :href="`${config.public.dashboardURL}/register`"
          class="w-full rounded-full bg-primary-700 px-6 py-3 text-sm font-semibold text-white transition-colors hover:bg-primary-800 sm:w-auto"
        >
          Daftar sekarang
        </a>
        <NuxtLink
          to="/faq"
          class="w-full rounded-full border border-black/10 px-6 py-3 text-sm font-semibold text-foreground transition-colors hover:bg-black/[0.03] sm:w-auto"
        >
          Baca FAQ
        </NuxtLink>
      </div>
    </div>
  </section>
</template>
