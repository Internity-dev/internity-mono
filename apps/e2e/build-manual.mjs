import { chromium } from '@playwright/test'
import { writeFileSync, readFileSync, mkdirSync } from 'node:fs'
import path from 'node:path'

const ROOT = 'D:/Project/Mumtaz/teesatas'
const SHOTS = `${ROOT}/docs/feature-manual-screenshots`
const OUT_HTML = `${ROOT}/.brand-preview/manual-book.html`
const OUT_PDF = `${ROOT}/docs/feature-manual.pdf`

function img(name) {
  const p = `${SHOTS}/${name}.png`
  const b64 = readFileSync(p).toString('base64')
  return `data:image/png;base64,${b64}`
}
function asset(relPath) {
  const b64 = readFileSync(`${ROOT}/${relPath}`).toString('base64')
  const ext = path.extname(relPath).slice(1)
  const mime = ext === 'svg' ? 'image/svg+xml' : `image/${ext}`
  return `data:${mime};base64,${b64}`
}

const SECTIONS = [
  {
    title: 'Ringkasan',
    intro: 'Halaman pertama yang dilihat setiap pengguna setelah masuk.',
    items: [
      {
        shot: '00-dashboard',
        title: 'Dashboard & Analitik',
        body: 'Titik masuk utama aplikasi. Menampilkan ringkasan angka nyata dari database: jumlah sekolah, perusahaan mitra, lowongan, dan jurusan, plus tiga grafik donat yang memecah status lamaran, status lowongan, dan rekap kehadiran bulan berjalan. Cakupan data menyesuaikan siapa yang login: admin melihat seluruh platform, koordinator melihat sekolahnya sendiri, mentor melihat perusahaannya sendiri, dan siswa melihat lamarannya sendiri.',
      },
    ],
  },
  {
    title: 'Perjalanan Magang Siswa',
    intro: 'Alur yang dijalani siswa dari mencari tempat magang sampai lulus dengan sertifikat.',
    items: [
      {
        shot: 'vacancies-browse',
        title: 'Cari Lowongan',
        body: 'Siswa menjelajahi semua lowongan magang yang dibuka perusahaan di jurusannya. Bisa dicari berdasarkan nama perusahaan atau skill, dan setiap kartu lowongan menampilkan skill yang dibutuhkan, jumlah kuota, serta status buka/tutup sebelum melamar.',
      },
      {
        shot: 'my-applications',
        title: 'Lamaran Saya',
        body: 'Daftar semua lowongan yang pernah dilamar siswa beserta statusnya (menunggu, diproses, diterima, atau ditolak), lengkap dengan catatan yang dikirim saat melamar.',
      },
      {
        shot: 'my-internship',
        title: 'Magang Saya',
        body: 'Begitu diterima, halaman ini menjadi catatan penempatan siswa: perusahaan tempat magang, mentor pembimbing, dan tanggal mulai/selesai yang menentukan periode presensi serta jurnal.',
      },
      {
        shot: 'attendance-checkin',
        title: 'Presensi',
        body: 'Siswa check-in dan check-out setiap hari kerja dengan foto dan lokasi, atau mengajukan izin (sakit/izin/cuti) pada hari yang berhalangan. Setiap entri menunggu persetujuan mentor.',
      },
      {
        shot: 'journal-write',
        title: 'Jurnal Harian',
        body: 'Siswa menulis catatan singkat setiap hari tentang pekerjaan yang dikerjakan. Mentor membaca dan menyetujui setiap entri, menjadikannya bagian dari rekam jejak magang.',
      },
      {
        shot: 'certificate',
        title: 'Sertifikat',
        body: 'Setelah masa magang selesai dan nilai sudah diinput mentor, sertifikat resmi kelulusan bisa langsung diunduh dari halaman ini.',
      },
    ],
  },
  {
    title: 'Operasional Program',
    intro: 'Alat kerja harian untuk admin, koordinator, dan mentor menjalankan program magang.',
    items: [
      {
        shot: 'vacancies-manage',
        title: 'Kelola Lowongan',
        body: 'Membuat dan mengedit lowongan magang atas nama perusahaan mitra: judul, skill yang dibutuhkan, jumlah kuota, dan jurusan mana saja yang boleh melamar.',
      },
      {
        shot: 'applications-review',
        title: 'Tinjau Lamaran',
        body: 'Meninjau setiap lamaran siswa ke sebuah lowongan. Terima atau tolak, filter berdasarkan jurusan, perusahaan, atau lowongan, dan baca catatan dari setiap pelamar.',
      },
      {
        shot: 'attendance-review',
        title: 'Tinjau Presensi',
        body: 'Menyetujui atau menolak check-in siswa (foto + lokasi) beserta izin yang diajukan. Ini yang menetapkan rekap presensi menjadi final.',
      },
      {
        shot: 'journal-review',
        title: 'Tinjau Jurnal',
        body: 'Membaca dan menyetujui jurnal harian siswa di seluruh sekolah, bukan hanya satu perusahaan.',
      },
      {
        shot: 'scores-manage',
        title: 'Nilai',
        body: 'Menginput dan meninjau nilai siswa per penempatan. Nilai dipetakan ke rentang huruf (A/B/C, dst.) yang dikonfigurasi di Predikat Nilai.',
      },
      {
        shot: 'monitoring-visits',
        title: 'Kunjungan Monitoring',
        body: 'Mencatat kunjungan lapangan koordinator ke perusahaan mitra: catatan perkembangan siswa dan saran untuk mentor.',
      },
      {
        shot: 'reviews-questions',
        title: 'Ulasan & Pertanyaan',
        body: 'Mengelola daftar pertanyaan kuesioner yang dipakai untuk menilai perusahaan atau mentor setelah masa penempatan selesai.',
      },
      {
        shot: 'reports',
        title: 'Laporan',
        body: 'Mengekspor daftar siswa dan rekap presensi untuk keperluan arsip atau akreditasi.',
      },
    ],
  },
  {
    title: 'Manajemen Organisasi',
    intro: 'Struktur dasar yang menghubungkan sekolah, jurusan, dan perusahaan mitra.',
    items: [
      {
        shot: 'schools',
        title: 'Sekolah',
        body: 'Mendaftarkan sekolah baru ke platform. Setiap sekolah punya koordinator, jurusan, dan daftar siswa sendiri, terpisah penuh dari sekolah lain.',
      },
      {
        shot: 'departments',
        title: 'Jurusan',
        body: 'Mengelola jurusan akademik di sebuah sekolah. Setiap jurusan mengelompokkan kelas dan perusahaan mitra yang bisa dipasangkan dengan siswanya.',
      },
      {
        shot: 'courses',
        title: 'Kelas',
        body: 'Mengelola rombongan belajar (rombel) di setiap jurusan, pengelompokan yang dipakai saat menerbitkan kode undangan dan menyaring siswa.',
      },
      {
        shot: 'companies',
        title: 'Perusahaan Mitra',
        body: 'Mengelola perusahaan tempat siswa magang: kontak, jurusan yang terhubung, dan lowongan yang sudah diposting masing-masing perusahaan.',
      },
      {
        shot: 'users',
        title: 'Pengguna',
        body: 'Semua akun di sekolah: siswa, mentor, staf. Mengubah peran, mereset akses, dan menerbitkan kode undangan untuk pendaftaran mandiri.',
      },
    ],
  },
  {
    title: 'Konten & Konfigurasi',
    intro: 'Pengaturan yang membentuk apa yang dilihat siswa dan bagaimana penilaian dihitung.',
    items: [
      {
        shot: 'news-manage',
        title: 'Kelola Berita',
        body: 'Menerbitkan pengumuman ke sekolah. Siswa dan staf otomatis mendapat notifikasi.',
      },
      {
        shot: 'faq-manage',
        title: 'Kelola FAQ',
        body: 'Menulis dan mengedit entri tanya-jawab yang dilihat siswa di halaman FAQ publik.',
      },
      {
        shot: 'presence-statuses',
        title: 'Status Presensi',
        body: 'Menentukan jenis status kehadiran yang tersedia saat meninjau presensi: Hadir, Sakit, Izin, Alpa, masing-masing dengan label dan warna sendiri.',
      },
      {
        shot: 'score-predicates',
        title: 'Predikat Nilai',
        body: 'Mengatur rentang nilai huruf (misalnya A = 90–100) yang dipakai untuk merangkum nilai di sertifikat.',
      },
    ],
  },
  {
    title: 'Umum',
    intro: 'Fitur yang tersedia untuk semua peran, tanpa terkecuali.',
    items: [
      { shot: 'news-view', title: 'Berita', body: 'Melihat pengumuman dan info terbaru dari sekolah maupun program.' },
      { shot: 'faq-view', title: 'FAQ', body: 'Jawaban atas pertanyaan umum seputar pendaftaran, presensi, dan proses sertifikasi.' },
      { shot: 'notifications', title: 'Notifikasi', body: 'Semua pembaruan (perubahan status, persetujuan, pengumuman baru) muncul di sini lebih dulu.' },
      { shot: 'profile', title: 'Profil', body: 'Mengubah nama, foto profil, dan kata sandi, serta melihat detail akun.' },
    ],
  },
]

const logo = asset('apps/dashboard/src/assets/logo.png')
const illustration = asset('apps/landing/public/illustrations/winner.svg')

const today = new Date().toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' })

let toc = ''
let body = ''
let pageNum = 2 // cover is page 1

for (const section of SECTIONS) {
  toc += `<div class="toc-section"><span class="toc-section-title">${section.title}</span><span class="toc-dots"></span><span class="toc-page">${pageNum}</span></div>`
  body += `<section class="section-divider"><div class="section-divider-inner"><span class="section-kicker">Bagian</span><h2>${section.title}</h2><p>${section.intro}</p></div></section>`
  pageNum++
  for (const item of section.items) {
    body += `
    <section class="feature-page">
      <div class="feature-head">
        <span class="feature-eyebrow">${section.title}</span>
        <h3>${item.title}</h3>
        <p>${item.body}</p>
      </div>
      <div class="feature-shot">
        <img src="${img(item.shot)}" alt="${item.title}" />
      </div>
    </section>`
    pageNum++
  }
}

const html = `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8" />
<style>
  @import url('https://fonts.googleapis.com/css2?family=Sora:wght@400;500;600;700;800&display=swap');

  :root {
    --primary-700: #0065ab;
    --primary-500: #2e9ae4;
    --primary-400: #63b3f1;
    --primary-300: #91cbfb;
    --primary-100: #d8eeff;
    --primary-50: #eaf7ff;
    --accent-500: #0bb98e;
    --neutral-900: #171b26;
    --neutral-700: #3d4457;
    --neutral-500: #717a8f;
    --neutral-300: #c5cbd6;
    --neutral-200: #dfe3ea;
    --neutral-100: #eef0f4;
    --neutral-50: #f7f8fa;
  }
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; }
  body { font-family: 'Sora', ui-sans-serif, system-ui, sans-serif; color: var(--neutral-900); }

  @page { size: A4; margin: 0; }
  .page, .section-divider, .feature-page {
    width: 210mm; height: 297mm; position: relative; overflow: hidden; page-break-after: always;
  }

  /* --- Cover --- */
  .cover {
    background: linear-gradient(155deg, #2badf7 0%, #0065ab 55%, #023a63 100%);
    color: white;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    padding: 20mm 18mm;
  }
  .cover-hex-bg {
    position: absolute;
    top: -60mm; right: -70mm;
    width: 260mm; height: 260mm;
    opacity: 0.08;
  }
  .cover-top { display: flex; align-items: center; gap: 4mm; position: relative; z-index: 2; }
  .cover-top img { width: 11mm; height: 11mm; }
  .cover-top span { font-weight: 700; font-size: 5mm; letter-spacing: -0.01em; }
  .cover-illustration {
    position: absolute; right: -14mm; bottom: 58mm; width: 130mm; z-index: 1;
    filter: drop-shadow(0 12mm 18mm rgba(2, 20, 40, 0.35));
  }
  .cover-main { position: relative; z-index: 2; max-width: 130mm; margin-top: auto; }
  .cover-kicker {
    display: inline-block; font-size: 3.2mm; font-weight: 600; letter-spacing: 0.14em;
    text-transform: uppercase; color: var(--primary-100); margin-bottom: 6mm;
    border-left: 0.8mm solid white; padding-left: 3mm;
  }
  .cover-title { font-size: 15mm; font-weight: 800; line-height: 1.02; letter-spacing: -0.02em; margin: 0 0 6mm; }
  .cover-subtitle { font-size: 5mm; font-weight: 400; line-height: 1.5; color: var(--primary-50); max-width: 105mm; margin: 0; }
  .cover-bottom {
    position: relative; z-index: 2; display: flex; justify-content: space-between; align-items: flex-end;
    border-top: 0.3mm solid rgba(255,255,255,0.25); padding-top: 6mm; margin-top: 10mm;
    font-size: 3.4mm; color: var(--primary-100);
  }
  .cover-bottom .right { text-align: right; }
  .cover-bottom strong { color: white; font-weight: 600; }

  /* --- TOC --- */
  .toc { padding: 28mm 20mm; background: var(--neutral-50); border-top: 1.2mm solid var(--primary-700); }
  .toc-hex-bg { position: absolute; bottom: -50mm; right: -55mm; width: 200mm; opacity: 0.05; }
  .toc-kicker { font-size: 3.2mm; font-weight: 600; letter-spacing: 0.14em; text-transform: uppercase; color: var(--primary-700); }
  .toc h1 { font-size: 9mm; font-weight: 800; letter-spacing: -0.02em; margin: 3mm 0 14mm; color: var(--neutral-900); }
  .toc-section { display: flex; align-items: baseline; gap: 3mm; padding: 4.5mm 0; border-bottom: 0.2mm solid var(--neutral-200); }
  .toc-section-title { font-size: 5mm; font-weight: 600; color: var(--neutral-900); white-space: nowrap; }
  .toc-dots { flex: 1; border-bottom: 0.3mm dotted var(--neutral-300); margin-bottom: 1.5mm; }
  .toc-page { font-size: 4mm; font-weight: 600; color: var(--primary-700); font-variant-numeric: tabular-nums; }

  /* --- Section divider --- */
  .section-divider {
    background: linear-gradient(155deg, var(--primary-700) 0%, #023a63 100%);
    color: white;
    display: flex; align-items: center; padding: 0 22mm;
  }
  .section-kicker { font-size: 3.4mm; font-weight: 600; letter-spacing: 0.14em; text-transform: uppercase; color: var(--primary-200, var(--primary-100)); }
  .section-divider h2 { font-size: 13mm; font-weight: 800; letter-spacing: -0.02em; margin: 4mm 0 6mm; }
  .section-divider p { font-size: 4.6mm; color: var(--primary-100); max-width: 130mm; line-height: 1.5; margin: 0; }

  /* --- Feature page --- */
  .feature-page { padding: 16mm 16mm 12mm; background: white; display: flex; flex-direction: column; }
  .feature-head { margin-bottom: 8mm; }
  .feature-eyebrow { font-size: 3.2mm; font-weight: 600; letter-spacing: 0.1em; text-transform: uppercase; color: var(--primary-700); }
  .feature-head h3 { font-size: 9mm; font-weight: 800; letter-spacing: -0.02em; margin: 2mm 0 4mm; color: var(--neutral-900); }
  .feature-head p { font-size: 4.2mm; line-height: 1.6; color: var(--neutral-700); max-width: 165mm; margin: 0; }
  .feature-shot {
    border-radius: 3mm; overflow: hidden; border: 0.3mm solid var(--neutral-200);
    box-shadow: 0 3mm 8mm rgba(23, 27, 38, 0.12);
  }
  .feature-shot img { width: 100%; display: block; }
</style>
</head>
<body>

  <div class="page cover">
    <img class="cover-hex-bg" src="${logo}" alt="" />
    <div class="cover-top"><img src="${logo}" alt="" /><span>Internity</span></div>
    <img class="cover-illustration" src="${illustration}" alt="" />
    <div class="cover-main">
      <span class="cover-kicker">Panduan Fitur</span>
      <h1 class="cover-title">Buku Panduan<br />Lengkap</h1>
      <p class="cover-subtitle">Semua fitur platform manajemen PKL Internity, dari mencari tempat magang sampai terbit sertifikat, dijelaskan satu per satu dengan tampilan aslinya.</p>
    </div>
    <div class="cover-bottom">
      <div><strong>Internity</strong><br />Platform Manajemen PKL</div>
      <div class="right">${today}<br />Internity</div>
    </div>
  </div>

  <div class="page toc">
    <img class="toc-hex-bg" src="${logo}" alt="" />
    <span class="toc-kicker">Daftar Isi</span>
    <h1>Isi Panduan</h1>
    ${toc}
  </div>

  ${body}

</body>
</html>`

mkdirSync(path.dirname(OUT_HTML), { recursive: true })
writeFileSync(OUT_HTML, html)
console.log('HTML written:', OUT_HTML, `(${(html.length / 1024 / 1024).toFixed(1)} MB)`)

const browser = await chromium.launch()
const page = await browser.newPage()
await page.goto(`file:///${OUT_HTML.replace(/\\/g, '/')}`)
await page.waitForTimeout(1500)
mkdirSync(path.dirname(OUT_PDF), { recursive: true })
await page.pdf({ path: OUT_PDF, printBackground: true, preferCSSPageSize: true })
await browser.close()
console.log('PDF written:', OUT_PDF)
