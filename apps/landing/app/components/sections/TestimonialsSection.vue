<script setup lang="ts">
// Placeholder quotes for now, not real customers yet. Swap in once the
// platform has actual schools/companies using it.
const testimonials = [
  {
    quote: 'Dulu rekap presensi PKL kami masih pakai spreadsheet dan grup WhatsApp. Sekarang semua tercatat otomatis, jauh lebih rapi.',
    name: 'Rina Wulandari',
    role: 'Koordinator PKL, SMKN 1 Cibinong',
  },
  {
    quote: 'Nyari tempat magang jadi gampang. Tinggal lihat lowongan, apply, terus tinggal nunggu diterima.',
    name: 'Budi Santoso',
    role: 'Siswa, Rekayasa Perangkat Lunak',
  },
  {
    quote: 'Saya bisa langsung menilai dan menyetujui jurnal anak magang dari HP, tidak perlu nunggu ketemu langsung tiap minggu.',
    name: 'Agus Prasetyo',
    role: 'Mentor, PT Mumtaz Teknologi Indonesia',
  },
  {
    quote: 'Sertifikatnya langsung bisa diunduh begitu nilai selesai diinput. Tidak perlu bolak-balik ke sekolah lagi.',
    name: 'Siti Aminah',
    role: 'Siswa, Rekayasa Perangkat Lunak',
  },
  {
    quote: 'Kami jadi bisa pantau beberapa perusahaan mitra sekaligus tanpa kehilangan jejak siapa magang di mana.',
    name: 'Dewi Lestari',
    role: 'Kepala Jurusan, SMKN 1 Cibinong',
  },
  {
    quote: 'Proses lamarannya jelas, ada statusnya, jadi kami tidak perlu balas satu-satu lewat email.',
    name: 'Hendra Wijaya',
    role: 'HR, PT Teknologi Nusantara',
  },
]
</script>

<template>
  <section id="testimoni" class="border-t border-black/5 py-20 sm:py-28">
    <div class="mx-auto max-w-xl px-6 text-center">
      <h2 class="font-display text-3xl font-semibold tracking-tight sm:text-4xl">Kata mereka yang sudah pakai</h2>
      <p class="mt-3 text-foreground/60">Cuplikan dari sekolah, siswa, dan perusahaan mitra.</p>
    </div>

    <div class="relative mt-12 overflow-hidden mask-[linear-gradient(to_right,transparent,black_5%,black_95%,transparent)]">
      <div class="marquee-track flex w-max gap-5 px-6">
        <div
          v-for="(t, i) in [...testimonials, ...testimonials]"
          :key="i"
          class="w-80 shrink-0 rounded-2xl border border-black/5 bg-white/70 p-6 backdrop-blur"
        >
          <p class="text-sm leading-relaxed text-foreground/80">"{{ t.quote }}"</p>
          <p class="mt-4 text-sm font-semibold">{{ t.name }}</p>
          <p class="text-xs text-foreground/50">{{ t.role }}</p>
        </div>
      </div>
    </div>
  </section>
</template>
