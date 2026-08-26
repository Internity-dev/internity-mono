<script setup lang="ts">
const faqs = [
  {
    q: 'Apakah Internity gratis untuk sekolah?',
    a: 'Ya. Sekolah, siswa, dan perusahaan mitra bisa pakai Internity tanpa biaya. Sekolah tinggal daftar terus bagikan kode undangan ke siswanya.',
  },
  {
    q: 'Bagaimana siswa mendaftar?',
    a: 'Siswa daftar pakai kode undangan dari sekolah — akunnya otomatis kesambung ke jurusan dan angkatan yang benar.',
  },
  {
    q: 'Apakah bisa magang di lebih dari satu perusahaan?',
    a: 'Bisa, asal jadwal PKL-nya gak bentrok. Kalau tumpang tindih, sistem otomatis nolak.',
  },
  {
    q: 'Bagaimana siswa mendapatkan sertifikat?',
    a: 'Begitu mentor kasih nilai, sertifikat PDF langsung bisa diunduh dari halaman siswa.',
  },
]

const expanded = ref<number | null>(0)
function toggle(i: number) {
  expanded.value = expanded.value === i ? null : i
}
</script>

<template>
  <section class="mx-auto max-w-3xl px-6 py-20 sm:py-28">
    <div class="text-center">
      <h2 class="font-display text-3xl font-semibold tracking-tight sm:text-4xl">Pertanyaan yang sering ditanyakan</h2>
      <p class="mt-3 text-foreground/60">
        Masih ada pertanyaan lain?
        <NuxtLink to="/faq" class="font-medium text-primary-700 underline underline-offset-4">Lihat semua FAQ</NuxtLink>
      </p>
    </div>

    <div class="mt-10 divide-y divide-black/5">
      <div v-for="(item, i) in faqs" :key="item.q" class="py-5">
        <button
          class="flex w-full items-center justify-between text-left"
          :aria-expanded="expanded === i"
          :aria-controls="`faq-panel-${i}`"
          @click="toggle(i)"
        >
          <span class="font-medium">{{ item.q }}</span>
          <span class="ml-4 text-foreground/40" :class="{ 'rotate-45': expanded === i }">+</span>
        </button>
        <p v-if="expanded === i" :id="`faq-panel-${i}`" class="mt-3 text-sm leading-relaxed text-foreground/60">{{ item.a }}</p>
      </div>
    </div>
  </section>
</template>
