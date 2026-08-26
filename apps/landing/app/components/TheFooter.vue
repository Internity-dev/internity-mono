<script setup lang="ts">
const year = new Date().getFullYear()
</script>

<template>
  <footer class="border-t border-black/5">
    <div class="mx-auto max-w-5xl px-6 py-16">
      <div class="grid gap-10 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <div class="flex items-center gap-2">
            <img src="/logo-icon.png" alt="" width="64" height="64" class="h-6 w-6" />
            <span class="font-display font-semibold text-foreground/80">Internity</span>
          </div>
          <p class="mt-3 max-w-[20rem] text-sm text-foreground/50">
            Aplikasi pengelolaan PKL untuk sekolah, siswa, dan perusahaan mitra.
          </p>
        </div>

        <div>
          <p class="text-sm font-semibold text-foreground/80">Produk</p>
          <nav class="mt-3 flex flex-col gap-2 text-sm text-foreground/60">
            <NuxtLink to="/#cara-kerja" class="transition-colors hover:text-foreground">Cara kerja</NuxtLink>
            <NuxtLink to="/#manfaat" class="transition-colors hover:text-foreground">Manfaat</NuxtLink>
            <NuxtLink to="/#testimoni" class="transition-colors hover:text-foreground">Testimoni</NuxtLink>
          </nav>
        </div>

        <div>
          <p class="text-sm font-semibold text-foreground/80">Bantuan</p>
          <nav class="mt-3 flex flex-col gap-2 text-sm text-foreground/60">
            <NuxtLink to="/faq" class="transition-colors hover:text-foreground">FAQ</NuxtLink>
            <a href="mailto:internity.dev@gmail.com" class="transition-colors hover:text-foreground">Kontak</a>
          </nav>
        </div>

        <div>
          <p class="text-sm font-semibold text-foreground/80">Legal</p>
          <nav class="mt-3 flex flex-col gap-2 text-sm text-foreground/60">
            <NuxtLink to="/privacy" class="transition-colors hover:text-foreground">Kebijakan Privasi</NuxtLink>
            <NuxtLink to="/terms" class="transition-colors hover:text-foreground">Syarat &amp; Ketentuan</NuxtLink>
          </nav>
        </div>
      </div>

      <div class="mt-12 flex flex-col items-center justify-between gap-3 border-t border-black/5 pt-8 text-xs text-foreground/60 sm:flex-row">
        <p>© {{ year }} Internity. Dibuat untuk SMK di Indonesia.</p>
      </div>
    </div>
  </footer>
</template>
