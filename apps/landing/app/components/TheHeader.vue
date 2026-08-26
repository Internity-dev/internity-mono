<script setup lang="ts">
const config = useRuntimeConfig()
const mobileOpen = ref(false)
</script>

<template>
  <header class="sticky top-0 z-40 border-b border-black/5 bg-background/90 backdrop-blur">
    <div class="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
      <NuxtLink to="/" class="flex items-center gap-2 font-display text-lg font-semibold tracking-tight text-primary-700">
        <img src="/logo-icon.png" alt="" width="64" height="64" class="h-7 w-7" />
        Internity
      </NuxtLink>

      <nav class="hidden items-center gap-7 text-sm font-medium text-foreground/70 lg:flex">
        <NuxtLink to="/#cara-kerja" class="transition-colors hover:text-foreground">Cara kerja</NuxtLink>
        <NuxtLink to="/#manfaat" class="transition-colors hover:text-foreground">Manfaat</NuxtLink>
        <NuxtLink to="/#testimoni" class="transition-colors hover:text-foreground">Testimoni</NuxtLink>
        <NuxtLink to="/faq" class="transition-colors hover:text-foreground">FAQ</NuxtLink>
      </nav>

      <div class="hidden items-center gap-3 lg:flex">
        <a :href="`${config.public.dashboardURL}/login`" class="text-sm font-medium text-foreground/70 transition-colors hover:text-foreground">
          Masuk
        </a>
        <a
          :href="`${config.public.dashboardURL}/register`"
          class="rounded-full bg-primary-700 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-800"
        >
          Daftar
        </a>
      </div>

      <button class="lg:hidden" aria-label="Menu" :aria-expanded="mobileOpen" @click="mobileOpen = !mobileOpen">
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M4 6h16M4 12h16M4 18h16" stroke-linecap="round" />
        </svg>
      </button>
    </div>

    <div v-if="mobileOpen" class="border-t border-black/5 px-6 py-4 lg:hidden">
      <nav class="flex flex-col gap-4 text-sm font-medium">
        <NuxtLink to="/#cara-kerja" @click="mobileOpen = false">Cara kerja</NuxtLink>
        <NuxtLink to="/#manfaat" @click="mobileOpen = false">Manfaat</NuxtLink>
        <NuxtLink to="/#testimoni" @click="mobileOpen = false">Testimoni</NuxtLink>
        <NuxtLink to="/faq" @click="mobileOpen = false">FAQ</NuxtLink>
        <a :href="`${config.public.dashboardURL}/login`">Masuk</a>
        <a :href="`${config.public.dashboardURL}/register`" class="font-semibold text-primary-700">Daftar</a>
      </nav>
    </div>
  </header>
</template>
