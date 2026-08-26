import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  modules: ['motion-v/nuxt'],
  css: ['~/assets/css/main.css'],
  // Without this, components under app/components/sections/ auto-register
  // with a "Sections" prefix (e.g. <SectionsHeroSection>) instead of the
  // bare <HeroSection /> used in app/pages/index.vue.
  components: [{ path: '~/components', pathPrefix: false }],
  vite: {
    plugins: [tailwindcss()],
  },
  // Marketing content only (except /faq, which fetches client-side). No Node
  // runtime needed in "production", served as static files (see docs/adr for why).
  nitro: {
    preset: 'static',
  },
  runtimeConfig: {
    public: {
      apiBaseURL: process.env.NUXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1',
      dashboardURL: process.env.NUXT_PUBLIC_DASHBOARD_URL || 'http://localhost:5173',
    },
  },
  app: {
    head: {
      title: 'Internity | Kelola PKL dalam satu aplikasi',
      meta: [
        {
          name: 'description',
          content:
            'Internity membantu sekolah, siswa, dan perusahaan mengelola Praktik Kerja Lapangan (PKL) dalam satu aplikasi, mulai dari lowongan dan presensi sampai jurnal, penilaian, dan sertifikat.',
        },
        {
          property: 'og:image',
          content: '/logo.png',
        },
      ],
      // Font is loaded via preconnect + stylesheet link (not a CSS @import in
      // main.css) so the fetch kicks off in parallel with the document instead
      // of serializing behind it.
      link: [
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Sora:wght@400;500;600;700;800&display=swap',
        },
      ],
    },
  },
})
