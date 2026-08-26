# Project Rules

Wajib diikuti semua kontributor (termasuk AI agent) di repo ini.

1. **Komponen UI wajib pakai shadcn (shadcn-vue di dashboard) jika tersedia.** Jangan bikin
   komponen custom untuk sesuatu yang sudah ada di shadcn, kecuali shadcn memang tidak menyediakan
   varian yang dibutuhkan.

2. **Jangan jalankan server sendiri, kecuali diperintah untuk testing.** Tidak start `make dev`,
   `pnpm dev`, dsb secara inisiatif sendiri. Jalankan hanya saat user minta run/test manual.

3. **Setiap kode wajib punya unit test, coverage di atas 80%.** Berlaku untuk kode baru maupun
   yang diubah. Cek coverage sebelum dianggap selesai (`make test-api`, `make test-dashboard`).

4. **Setiap ditemukan bug fatal, prediksi apakah bisa terulang.** Jika ya, dokumentasikan di
   `docs/BUG_HISTORY.md`: root cause, kenapa bisa terulang, dan langkah pencegahan.

## Backend (Go/Gin/GORM)

5. Endpoint baru wajib ikut response envelope + error-code taxonomy yang sudah ada
   (`VALIDATION_ERROR`, `UNAUTHENTICATED`, dll). Jangan bikin format response baru.
6. Endpoint baru wajib update `docs/openapi.yaml`.
7. Module tidak boleh query tabel DB milik module lain secara langsung. Akses lintas-module wajib
   lewat interface/adapter (pola yang sudah dipakai di `cmd/api/main.go`).
8. Migration wajib sepasang up/down di `apps/api/migrations/`. Jangan pakai GORM `AutoMigrate`.
9. Secret/API key/credential tidak boleh hardcode. Wajib lewat env var.

## Frontend (Vue 3 / shadcn-vue)

10. Warna, spacing, dan font wajib dari `packages/design-tokens`. Jangan hardcode hex/px.
11. Form wajib pakai vee-validate + zod schema.
12. Semua API call wajib lewat axios instance yang sudah ada (CSRF + single-flight refresh
    interceptor). Jangan pakai `fetch` mentah.
13. Server state wajib TanStack Query. Jangan simpan manual di Pinia store.
14. Type-check wajib `vue-tsc --build` sebelum dianggap selesai. `vue-tsc --noEmit` silent no-op
    di project ini (TS project-references), jangan dipakai.

## General

15. Tidak boleh commit langsung ke `main`. Wajib lewat PR, lolos lint + test dulu.
16. Jangan over-engineering. Sebelum bikin kode/komponen baru, cek dulu apakah udah ada pattern
    atau kode serupa di codebase — reuse/extend itu, jangan reinvent.
17. Jangan overexplain di dokumentasi atau visual (diagram, komentar, README, dst). Langsung ke
    poin, seperlunya aja.
18. Bahasa gak usah terlalu baku/formal. Santai aja kayak ngobrol biasa, asal jelas.
19. Screenshot buat dokumentasi/manual/report wajib bersih: gak boleh ada error toast, network
    error, atau console error ke-capture. Verifikasi live dulu (network tab / console) sebelum
    screenshot dipakai — jangan asumsi cuma dari tampilan.
20. Data yang muncul di screenshot dokumentasi wajib jelas ke-load, bukan skeleton/loading state
    atau state kosong ("No data yet"). Kalau akun/scope yang dipakai datanya kosong, ganti akun
    atau seed data dulu sebelum screenshot, jangan dipaksain.
