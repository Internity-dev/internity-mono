<script setup lang="ts">
import { RouterView } from 'vue-router'
import logo from '@/assets/logo.png'
import onlineSecurity from '@/assets/illustrations/online-security.svg'
</script>

<template>
  <div class="grid min-h-screen lg:grid-cols-2">
    <div class="relative hidden flex-col justify-between overflow-hidden bg-primary-800 p-10 text-white lg:flex">
      <div class="absolute inset-0 bg-linear-to-br from-primary-600 via-primary-800 to-primary-950" />
      <div class="absolute -top-24 -right-24 h-96 w-96 rounded-full bg-white/10 blur-3xl" />
      <div class="absolute -bottom-32 -left-16 h-80 w-80 rounded-full bg-accent-400/20 blur-3xl" />

      <div class="relative z-10 flex items-center gap-2 font-display text-lg font-semibold tracking-tight">
        <img :src="logo" alt="" class="h-8 w-8" />
        Internity
      </div>

      <div class="relative z-10 flex flex-1 items-center justify-center py-10">
        <img :src="onlineSecurity" alt="" class="w-full max-w-md" />
      </div>

      <div class="relative z-10 space-y-2">
        <p class="font-display text-2xl font-semibold leading-tight">Kelola PKL dalam satu tempat</p>
        <p class="max-w-sm text-sm text-white/80">
          Presensi, jurnal, penilaian, dan sertifikat, tercatat dan dapat ditinjau secara real time.
        </p>
      </div>
    </div>

    <div class="flex items-center justify-center p-6 sm:p-10">
      <div class="w-full max-w-sm space-y-6">
        <div class="flex flex-col items-center gap-2 text-center lg:hidden">
          <img :src="logo" alt="Internity" class="h-10 w-10" />
          <span class="font-display text-lg font-semibold tracking-tight text-primary-700">Internity</span>
        </div>
        <RouterView />
      </div>
    </div>
  </div>
</template>
