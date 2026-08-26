<script setup lang="ts">
const { target, visible } = useReveal()
</script>

<template>
  <section id="manfaat" class="mx-auto max-w-5xl px-6 py-20 sm:py-28">
    <div class="mx-auto max-w-xl text-center">
      <h2 class="font-display text-3xl font-semibold tracking-tight sm:text-4xl">Manfaat</h2>
      <p class="mt-3 text-foreground/60">Fitur inti yang bikin PKL sekolah lebih rapi, tanpa spreadsheet dan grup WhatsApp.</p>
    </div>

    <div ref="target" class="reveal mt-14 grid gap-6 sm:grid-cols-2" :class="{ 'reveal-visible': visible }">
      <div class="rounded-2xl border border-black/5 bg-white/60 p-6 backdrop-blur">
        <h3 class="font-display text-lg font-semibold">Kendalikan slot lowongan</h3>
        <p class="mt-2 text-sm leading-relaxed text-foreground/60">
          Tentukan jumlah slot per lowongan — begitu penuh, lamaran otomatis ditutup. Gak ada
          lagi kelebihan penempatan yang mesti diurus manual.
        </p>
        <div class="mt-5 rounded-xl border border-black/10 bg-white px-4 py-3">
          <div class="flex items-center justify-between text-sm">
            <span class="font-medium">Frontend Developer Intern</span>
            <span class="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700">2/2 terisi</span>
          </div>
          <div class="mt-2 h-1.5 w-full rounded-full bg-black/5">
            <div class="h-1.5 w-full rounded-full bg-primary-600" />
          </div>
        </div>
      </div>

      <div class="rounded-2xl border border-black/5 bg-white/60 p-6 backdrop-blur">
        <h3 class="font-display text-lg font-semibold">Laporan siap kapan saja</h3>
        <p class="mt-2 text-sm leading-relaxed text-foreground/60">
          Ekspor roster siswa dan rekap presensi ke Excel atau PDF kapan saja, tanpa menyalin data manual.
        </p>
        <div class="mt-5 flex items-center justify-between rounded-xl border border-black/10 bg-white px-4 py-3">
          <span class="text-sm font-medium">rekap-presensi-agustus.xlsx</span>
          <button class="rounded-full border border-black/10 px-3 py-1.5 text-xs font-medium">Unduh</button>
        </div>
      </div>

      <div class="rounded-2xl border border-black/5 bg-white/60 p-6 backdrop-blur">
        <h3 class="font-display text-lg font-semibold">Alur persetujuan yang jelas</h3>
        <p class="mt-2 text-sm leading-relaxed text-foreground/60">
          Presensi dan jurnal siswa nunggu disetujui mentor dulu, jadi gak ada data yang cuma
          numpuk tanpa pernah diperiksa.
        </p>
        <div class="mt-5 space-y-2">
          <div class="flex items-center justify-between rounded-xl bg-black/2 px-3 py-2 text-sm">
            <span>Jurnal, 22 Agu</span>
            <span class="rounded-full bg-success/15 px-2 py-0.5 text-xs font-medium text-success">Disetujui</span>
          </div>
          <div class="flex items-center justify-between rounded-xl bg-black/2 px-3 py-2 text-sm">
            <span>Presensi, 23 Agu</span>
            <span class="rounded-full bg-warning/15 px-2 py-0.5 text-xs font-medium text-warning">Menunggu</span>
          </div>
        </div>
      </div>

      <div class="rounded-2xl border border-black/5 bg-white/60 p-6 backdrop-blur">
        <h3 class="font-display text-lg font-semibold">Riwayat PKL tanpa bentrok</h3>
        <p class="mt-2 text-sm leading-relaxed text-foreground/60">
          Siswa bisa punya riwayat PKL di lebih dari satu perusahaan. Sistem otomatis menolak jadwal yang tumpang tindih.
        </p>
        <div class="mt-5 space-y-2">
          <div class="flex items-center justify-between rounded-xl bg-black/2 px-3 py-2 text-sm">
            <span>PT Mumtaz Teknologi</span>
            <span class="text-xs text-foreground/50">1 Jul - 30 Sep</span>
          </div>
          <div class="flex items-center justify-between rounded-xl bg-black/2 px-3 py-2 text-sm">
            <span>PT Teknologi Nusantara</span>
            <span class="text-xs text-foreground/50">1 Okt - 31 Des</span>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
