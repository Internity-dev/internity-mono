<script setup lang="ts">
const steps = [
  {
    number: '01',
    title: 'Daftar dengan kode sekolah',
    description: 'Siswa mendaftar memakai kode undangan dari sekolah, langsung terhubung ke jurusan dan angkatannya.',
  },
  {
    number: '02',
    title: 'Ajukan lamaran, tunggu disetujui',
    description: 'Cari lowongan dari perusahaan mitra, ajukan lamaran, dan pantau statusnya sampai diterima.',
  },
  {
    number: '03',
    title: 'Presensi sampai sertifikat',
    description: 'Check-in harian, tulis jurnal, dapat penilaian dari mentor, lalu unduh sertifikat begitu PKL selesai.',
  },
]

const { target: t1, visible: v1 } = useReveal()
const { target: t2, visible: v2 } = useReveal()
const { target: t3, visible: v3 } = useReveal()
const reveals = [
  { target: t1, visible: v1 },
  { target: t2, visible: v2 },
  { target: t3, visible: v3 },
]
</script>

<template>
  <section id="cara-kerja" class="mx-auto max-w-5xl px-6 py-20 sm:py-28">
    <div class="mx-auto max-w-xl text-center">
      <h2 class="font-display text-3xl font-semibold tracking-tight sm:text-4xl">Bagaimana cara kerjanya</h2>
    </div>

    <div class="mt-14 grid gap-6 sm:grid-cols-3">
      <div
        v-for="(step, index) in steps"
        :key="step.title"
        :ref="(el) => { reveals[index]!.target.value = el as HTMLElement | null }"
        class="reveal rounded-2xl border border-black/5 bg-white/60 p-6 backdrop-blur"
        :class="{ 'reveal-visible': reveals[index]!.visible.value }"
        :style="{ transitionDelay: `${index * 100}ms` }"
      >
        <span class="font-display text-3xl font-semibold text-primary-300">{{ step.number }}</span>
        <h3 class="mt-3 font-display text-base font-semibold">{{ step.title }}</h3>
        <p class="mt-2 text-sm leading-relaxed text-foreground/60">{{ step.description }}</p>

        <div v-if="index === 0" class="mt-5 rounded-lg border border-black/10 bg-white px-3 py-2.5">
          <p class="text-xs font-medium tracking-wide text-foreground/60 uppercase">Kode undangan</p>
          <p class="mt-1 font-mono text-sm font-semibold text-primary-700">RPL1DEMO</p>
        </div>
        <div v-else-if="index === 1" class="mt-5 space-y-1.5">
          <div class="flex items-center gap-2 text-xs">
            <span class="size-1.5 rounded-full bg-success" />
            <span class="text-foreground/70">Lamaran diterima, PT Mumtaz Teknologi</span>
          </div>
          <div class="flex items-center gap-2 text-xs">
            <span class="size-1.5 rounded-full bg-warning" />
            <span class="text-foreground/60">Menunggu proses, PT Nusantara</span>
          </div>
        </div>
        <div v-else class="mt-5 flex flex-wrap gap-1.5">
          <span class="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700">Presensi ✓</span>
          <span class="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700">Jurnal ✓</span>
          <span class="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700">Nilai ✓</span>
          <span class="rounded-full bg-black/5 px-2.5 py-1 text-xs font-medium text-foreground/50">Sertifikat</span>
        </div>
      </div>
    </div>
  </section>
</template>
