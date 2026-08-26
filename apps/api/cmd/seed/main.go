// cmd/seed loads a rich, realistic demo dataset covering every module in the
// system: one school with three courses, 25 companies across varied
// categories/cities, one mentor per company, ~150 students, a spread of
// vacancies, appliances across all five statuses, intern placements
// (completed/ongoing/starting soon) with matching presence/journal/score/
// certificate history, plus news, FAQs, monitoring visits, review questions,
// reviews, and notifications — enough that every dashboard list looks like a
// real, in-production system with substantial history instead of a thin
// demo. Safe to re-run: every insert is idempotent (lookup-by-natural-key-
// or-insert for the low-volume tables, load-existing-then-filter for the
// bulk ones), so `make seed` against an already-seeded database just fills
// in whatever's missing instead of erroring or duplicating rows.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"internity/internal/config"
	"internity/internal/platform/postgres"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const demoPassword = "password123" // meets the 8-char minimum; every seeded account shares it for demo convenience

// applianceSpec, monitorSpec, mentorReviewSpec and companyReviewSpec are
// named (not the original files' anonymous struct literals) because the
// blueprint slices below are now built by appending generated rows onto a
// hand-written literal, which requires both sides to share one named type.
type applianceSpec struct {
	studentIdx, vacancyIdx         int
	status                         string
	createdDaysAgo, updatedDaysAgo int
	internGroup                    string // "completed" | "ongoing" | "soon" | ""
	scoreTarget                    int
	notify                         bool
}

type monitorSpec struct {
	studentIdx, companyIdx, daysAgo, matchRating int
	notes, suggest                               string
}

type mentorReviewSpec struct {
	studentIdx, companyIdx, questionIdx, rating int
	title, body                                 string
}

type companyReviewSpec struct {
	studentIdx, companyIdx, rating int
	title, body                    string
}

// companySpec/vacancySpec/newsSpec/schoolSeedConfig/schoolSeedTotals back
// seedSchool below — the second/third school blueprints (SMKN 2 Bogor, SMKN
// 1 Depok) are built from these named types instead of copy-pasting the
// SMKN 1 Cibinong block in main() a second and third time with different
// literals; seedSchool itself still just calls the same upsert*/bulk*
// helpers main() uses, in the same order.
type companySpec struct{ name, category, city string }

type vacancySpec struct {
	companyIdx             int
	name, category, skills string
	slots                  int
	status                 string
}

type newsSpec struct {
	title, content, status string
	createdDaysAgo         int
}

type schoolSeedConfig struct {
	schoolName, schoolEmail, schoolPhone, schoolAddress string
	deptName, deptDescription, studyProgram             string
	courseNames                                         []string
	inviteCodes                                         []string // parallel to courseNames
	nisPrefix                                           string   // e.g. "2024002" — see maxNIS comment in main() for why this isn't a flat shared counter
	coordinatorEmail, coordinatorName                   string
	companies                                           []companySpec
	mentorNames                                         []string // parallel to companies
	mentorEmailPrefix                                   string
	students                                            []string
	vacancies                                           []vacancySpec
	placementCounts                                     [3]int // completed, ongoing, soon (accepted appliances)
	nonAcceptedCounts                                   [4]int // rejected, canceled, pending, processed
	news                                                []newsSpec
}

type schoolSeedTotals struct {
	schoolName                                                              string
	companies, mentors, students, vacancies, closedVacancies                int
	appliances, internDates, presences, journals                            int
	scores, certificates, news, questions, monitors, reviews, notifications int
	completed, ongoing, soon                                                int
}

// seedPresenceRow/seedJournalRow/seedScoreRow/seedNotificationRow back the
// bulk-insert path for the tables whose row counts reach into the
// thousands: build the full slice in memory first, then hand it to GORM's
// CreateInBatches in a handful of round trips instead of one SELECT+INSERT
// per row (see bulkInsertPresences etc. below). Fields are named
// CreatedTS/UpdatedTS rather than GORM's CreatedAt/UpdatedAt convention
// names on purpose — GORM auto-overwrites fields with those exact names to
// time.Now() on every insert, which would destroy the historical dates
// these rows need.
type seedPresenceRow struct {
	UserID           string     `gorm:"column:user_id"`
	CompanyID        int64      `gorm:"column:company_id"`
	PresenceStatusID int64      `gorm:"column:presence_status_id"`
	Date             time.Time  `gorm:"column:date"`
	CheckInAt        *time.Time `gorm:"column:check_in_at"`
	CheckOutAt       *time.Time `gorm:"column:check_out_at"`
	IsApproved       bool       `gorm:"column:is_approved"`
	Description      *string    `gorm:"column:description"`
	CreatedTS        time.Time  `gorm:"column:created_at"`
	UpdatedTS        time.Time  `gorm:"column:updated_at"`
}

func (seedPresenceRow) TableName() string { return "presences" }

type seedJournalRow struct {
	UserID      string    `gorm:"column:user_id"`
	CompanyID   int64     `gorm:"column:company_id"`
	Date        time.Time `gorm:"column:date"`
	WorkType    string    `gorm:"column:work_type"`
	Description string    `gorm:"column:description"`
	IsApproved  bool      `gorm:"column:is_approved"`
	CreatedTS   time.Time `gorm:"column:created_at"`
	UpdatedTS   time.Time `gorm:"column:updated_at"`
}

func (seedJournalRow) TableName() string { return "journals" }

type seedScoreRow struct {
	UserID    string    `gorm:"column:user_id"`
	CompanyID int64     `gorm:"column:company_id"`
	Name      string    `gorm:"column:name"`
	Score     int       `gorm:"column:score"`
	Type      string    `gorm:"column:type"`
	CreatedTS time.Time `gorm:"column:created_at"`
	UpdatedTS time.Time `gorm:"column:updated_at"`
}

func (seedScoreRow) TableName() string { return "scores" }

type seedNotificationRow struct {
	UserID    string    `gorm:"column:user_id"`
	Type      string    `gorm:"column:type"`
	Title     string    `gorm:"column:title"`
	Body      string    `gorm:"column:body"`
	CreatedTS time.Time `gorm:"column:created_at"`
}

func (seedNotificationRow) TableName() string { return "notifications" }

func main() {
	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}
	db, err := postgres.Open(cfg.DatabaseURL, true)
	if err != nil {
		fail(err)
	}
	ctx := context.Background()

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		fail(err)
	}
	passwordHash := string(hash)
	now := time.Now()

	// --- data blueprints (pure Go values, no DB dependency — resolved to
	// real IDs inside the transaction below) ---

	companySpecs := []struct{ name, category, city string }{
		{"PT Mumtaz Teknologi Indonesia", "Teknologi", "Jakarta"},
		{"PT Teknologi Nusantara", "Teknologi", "Jakarta"}, // matches what's already live from the original seed; insert-only means this row won't be touched if it exists
		{"CV Kreasi Digital Bogor", "Teknologi", "Bogor"},
		{"PT Bogor Kreatif Media", "Media & Percetakan", "Bogor"},
		{"PT Sinar Abadi Farmasi", "Kesehatan", "Depok"},
		{"PT Cipta Boga Nusantara", "Kuliner & Perhotelan", "Jakarta"},
		{"Koperasi Sejahtera Mandiri", "Keuangan", "Bogor"},
		{"PT Wahana Otomotif Jaya", "Otomotif", "Cibinong"},
		// scale-up: 17 more companies across categories the original 8 didn't
		// touch (pendidikan, ritel, konstruksi, logistik, pariwisata,
		// pertanian) plus more depth in the existing categories.
		{"PT Cerdas Edukasi Nusantara", "Pendidikan", "Jakarta"},
		{"Yayasan Bina Bangsa Cendekia", "Pendidikan", "Bogor"},
		{"PT Retail Sukses Mandiri", "Ritel", "Jakarta"},
		{"Swalayan Berkah Jaya", "Ritel", "Depok"},
		{"PT Konstruksi Bangun Persada", "Konstruksi", "Bogor"},
		{"CV Bangun Karya Abadi", "Konstruksi", "Cibinong"},
		{"PT Logistik Cepat Nusantara", "Logistik", "Jakarta"},
		{"PT Ekspedisi Sentosa Jaya", "Logistik", "Bekasi"},
		{"PT Wisata Alam Puncak", "Pariwisata", "Bogor"},
		{"Hotel Grand Cibinong", "Pariwisata", "Cibinong"},
		{"PT Agro Makmur Sejahtera", "Pertanian", "Bogor"},
		{"Koperasi Tani Sejahtera", "Pertanian", "Cianjur"},
		{"PT Multimedia Kreatif Jaya", "Media & Percetakan", "Jakarta"},
		{"PT Farmasi Sehat Abadi", "Kesehatan", "Depok"},
		{"PT Boga Rasa Nusantara", "Kuliner & Perhotelan", "Bogor"},
		{"PT Otomotif Prima Mandiri", "Otomotif", "Bekasi"},
		{"PT Fintech Solusi Bangsa", "Keuangan", "Jakarta"},
	}

	mentorSpecs := []struct{ email, name string }{
		{"mentor1@internity.test", "Mentor PT Mumtaz"},
		{"mentor2@internity.test", "Mentor PT Nusantara"},
		{"mentor3@internity.test", "Mentor Kreasi Digital Bogor"},
		{"mentor4@internity.test", "Mentor Bogor Kreatif Media"},
		{"mentor5@internity.test", "Mentor Sinar Abadi Farmasi"},
		{"mentor6@internity.test", "Mentor Cipta Boga Nusantara"},
		{"mentor7@internity.test", "Mentor Koperasi Sejahtera"},
		{"mentor8@internity.test", "Mentor Wahana Otomotif"},
		{"mentor9@internity.test", "Mentor Cerdas Edukasi Nusantara"},
		{"mentor10@internity.test", "Mentor Bina Bangsa Cendekia"},
		{"mentor11@internity.test", "Mentor Retail Sukses Mandiri"},
		{"mentor12@internity.test", "Mentor Swalayan Berkah Jaya"},
		{"mentor13@internity.test", "Mentor Konstruksi Bangun Persada"},
		{"mentor14@internity.test", "Mentor Bangun Karya Abadi"},
		{"mentor15@internity.test", "Mentor Logistik Cepat Nusantara"},
		{"mentor16@internity.test", "Mentor Ekspedisi Sentosa Jaya"},
		{"mentor17@internity.test", "Mentor Wisata Alam Puncak"},
		{"mentor18@internity.test", "Mentor Hotel Grand Cibinong"},
		{"mentor19@internity.test", "Mentor Agro Makmur Sejahtera"},
		{"mentor20@internity.test", "Mentor Koperasi Tani Sejahtera"},
		{"mentor21@internity.test", "Mentor Multimedia Kreatif Jaya"},
		{"mentor22@internity.test", "Mentor Farmasi Sehat Abadi"},
		{"mentor23@internity.test", "Mentor Boga Rasa Nusantara"},
		{"mentor24@internity.test", "Mentor Otomotif Prima Mandiri"},
		{"mentor25@internity.test", "Mentor Fintech Solusi Bangsa"},
	}

	// original 22 pool names, unchanged from the first seed — kept verbatim
	// so their derived emails/order don't shift under already-seeded data.
	originalStudentNames := []string{
		"Ahmad Fauzi", "Dewi Lestari", "Rizky Pratama", "Putri Ramadhani", "Dimas Saputra", "Nur Azizah",
		"Fajar Nugroho", "Rina Marlina", "Andika Wijaya", "Indah Permata", "Bayu Setiawan", "Wulan Sari",
		"Rian Hidayat", "Ayu Anggraini", "Yusuf Ramadhan", "Fitri Handayani", "Arif Kurniawan", "Yuni Astuti",
		"Doni Firmansyah", "Melati Sukma", "Fikri Maulana", "Sari Wulandari",
	}
	// 126 more, first-name-unique against the original 22 + budi/siti, to
	// take total students from 24 to 150.
	newStudentNamesExtra := []string{
		"Agus Prasetyo", "Agung Nugraha", "Aji Santoso", "Akbar Maulidan", "Alfian Ramdani", "Ali Mustofa",
		"Amir Hasan", "Anang Wibowo", "Andra Kusuma", "Andre Wirawan", "Anton Susanto", "Ardi Setiadi",
		"Aris Munandar", "Arya Wicaksana", "Asep Solihin", "Aditya Perkasa", "Adit Firmansyah", "Bagas Prakoso",
		"Bambang Suryadi", "Bagus Ariyanto", "Bima Sakti", "Bobby Alamsyah", "Bram Wardana", "Chandra Halim",
		"Dani Iswanto", "Danu Pratomo", "Darma Yudha", "Dedi Kurniadi", "Deni Setiawan", "Denny Gunawan",
		"Dodi Hartono", "Eko Purnomo", "Endra Wibawa", "Erwin Syahputra", "Fahmi Ridwan", "Faisal Rahman",
		"Faiz Ramadhan", "Fajri Anggara", "Fandi Ahmad", "Farid Kusnadi", "Fauzan Nur", "Febri Santoso",
		"Ferdi Yulianto", "Firman Hakim", "Galih Permana", "Gilang Ramadhan", "Gunawan Saputra", "Hadi Wijaya",
		"Hafiz Aulia", "Halim Susanto", "Hamzah Fadillah", "Hanif Maulana", "Hendri Kurniawan", "Hendro Susilo",
		"Herman Yusup", "Ikhsan Nugroho", "Imam Syafii", "Iqbal Fadilah", "Irawan Setiaji", "Ismail Basri",
		"Iwan Setiabudi", "Jaka Perwira", "Jefri Ananta", "Adinda Kirana", "Aini Rahmawati", "Alya Puspitasari",
		"Amanda Safira", "Anggi Oktaviani", "Anisa Rahmah", "Asri Wulandari", "Astrid Cahyani", "Aulia Rahma",
		"Berlian Safitri", "Cahya Kartika", "Cici Anggraeni", "Devi Permatasari", "Diana Larasati", "Dinda Amelia",
		"Elsa Fauziah", "Erika Susanti", "Eva Kurniasih", "Farah Salsabila", "Fatimah Zahra", "Firda Amalia",
		"Fitria Ningsih", "Galuh Widyastuti", "Hana Maharani", "Hesti Rahayu", "Ika Sulistyowati", "Ira Yulianti",
		"Ismi Khoirunnisa", "Jasmine Adelia", "Kartika Sari", "Kirana Dewanti", "Lala Anjani", "Larasati Putri",
		"Laila Nur", "Lia Rosmalasari", "Lina Marliana", "Lusi Handayani", "Maya Puspitasari", "Meilani Putri",
		"Mira Andriyani", "Mutia Rahmadani", "Naila Zahira", "Nabila Ramadhani", "Nadia Anggraini", "Nia Kusumawati",
		"Nina Oktaviana", "Nita Herawati", "Novi Astutik", "Nurul Fadilah", "Prita Anindya", "Rani Purwanti",
		"Ratih Kumala", "Retno Wulandari", "Ria Anjelina", "Riska Amelia", "Rosa Damayanti", "Salma Nabila",
		"Santi Purnamasari", "Sekar Ayu", "Sinta Bella", "Sri Wahyuni", "Tari Kusuma", "Tasya Kamila",
	}
	newStudentNames := append(append([]string{}, originalStudentNames...), newStudentNamesExtra...)
	// absolute index (into the combined `students` slice built below, which
	// is budi, siti, then newStudentNames in order) of the first name in
	// newStudentNamesExtra — generated appliances below are only ever
	// assigned to students at or after this index, so they never touch a
	// student that might already have data from before this scale-up.
	newStudentsStart := 2 + len(originalStudentNames)

	// companyIdx indexes into companySpecs above.
	vacancySpecs := []struct {
		companyIdx             int
		name, category, skills string
		slots                  int
		status                 string
	}{
		{companyIdx: 0, name: "Frontend Developer Intern", category: "teknis", skills: "Vue, TypeScript, Tailwind", slots: 2, status: "open"},
		{companyIdx: 0, name: "UI/UX Design Intern", category: "desain", skills: "Figma, User Research, Wireframing", slots: 2, status: "open"},
		{companyIdx: 1, name: "Backend Developer Intern", category: "teknis", skills: "Go, PostgreSQL, REST API", slots: 2, status: "open"},
		{companyIdx: 1, name: "Quality Assurance Intern", category: "teknis", skills: "Manual Testing, Postman, Bug Tracking", slots: 2, status: "closed"},
		{companyIdx: 2, name: "Mobile Developer Intern", category: "teknis", skills: "Flutter, Dart, REST API", slots: 3, status: "open"},
		{companyIdx: 2, name: "Digital Marketing Intern", category: "marketing", skills: "SEO, Social Media, Content Planning", slots: 2, status: "open"},
		{companyIdx: 3, name: "Content Creator Intern", category: "media", skills: "Videografi, Copywriting, Editing", slots: 2, status: "open"},
		{companyIdx: 3, name: "Graphic Design Intern", category: "desain", skills: "Adobe Illustrator, Photoshop, Branding", slots: 2, status: "closed"},
		{companyIdx: 4, name: "Analis Laboratorium Intern", category: "kesehatan", skills: "Analisis Sampel, K3, Dokumentasi Lab", slots: 2, status: "open"},
		{companyIdx: 5, name: "Food & Beverage Service Intern", category: "kuliner", skills: "Table Manner, Pelayanan Tamu, Kebersihan", slots: 3, status: "open"},
		{companyIdx: 5, name: "Staff Dapur Intern", category: "kuliner", skills: "Pengolahan Makanan, Sanitasi Dapur", slots: 2, status: "closed"},
		{companyIdx: 6, name: "Staff Administrasi Keuangan Intern", category: "keuangan", skills: "Pembukuan, Excel, Kas Kecil", slots: 2, status: "open"},
		{companyIdx: 7, name: "Teknisi Otomotif Intern", category: "otomotif", skills: "Servis Kendaraan, Diagnostik, K3", slots: 3, status: "open"},
		{companyIdx: 7, name: "Staff Gudang Sparepart Intern", category: "logistik", skills: "Manajemen Stok, Input Data, Inventaris", slots: 2, status: "open"},
		// scale-up: 36 more vacancies across the 17 new companies.
		{companyIdx: 8, name: "Asisten Pengajar Intern", category: "pendidikan", skills: "Microsoft Office, Public Speaking, Kurikulum", slots: 2, status: "open"},
		{companyIdx: 8, name: "Staff Tata Usaha Intern", category: "administrasi", skills: "Arsip, Surat Menyurat, Ms Excel", slots: 2, status: "open"},
		{companyIdx: 8, name: "Staff Perpustakaan Intern", category: "administrasi", skills: "Katalogisasi, Pelayanan, Kearsipan Digital", slots: 2, status: "closed"},
		{companyIdx: 9, name: "Asisten Laboratorium Komputer Intern", category: "pendidikan", skills: "Instalasi Software, Troubleshooting, Jaringan Dasar", slots: 2, status: "open"},
		{companyIdx: 9, name: "Staff Administrasi Akademik Intern", category: "administrasi", skills: "Input Data, Pelayanan Siswa, Ms Excel", slots: 2, status: "closed"},
		{companyIdx: 10, name: "Staff Pramuniaga Intern", category: "ritel", skills: "Pelayanan Pelanggan, Display Produk, Kasir", slots: 3, status: "open"},
		{companyIdx: 10, name: "Staff Gudang Retail Intern", category: "ritel", skills: "Stok Opname, Input Barang, Inventaris", slots: 2, status: "open"},
		{companyIdx: 11, name: "Kasir Intern", category: "ritel", skills: "Mesin Kasir, Pelayanan Pelanggan, Ketelitian", slots: 2, status: "open"},
		{companyIdx: 11, name: "Staff Merchandising Intern", category: "ritel", skills: "Display Produk, Label Harga, Stok", slots: 2, status: "closed"},
		{companyIdx: 12, name: "Drafter Sipil Intern", category: "konstruksi", skills: "AutoCAD, Gambar Teknik, RAB", slots: 2, status: "open"},
		{companyIdx: 12, name: "Staff K3 Konstruksi Intern", category: "konstruksi", skills: "Keselamatan Kerja, Inspeksi Lapangan, APD", slots: 2, status: "open"},
		{companyIdx: 13, name: "Surveyor Lapangan Intern", category: "konstruksi", skills: "Pengukuran Lahan, Total Station, Dokumentasi", slots: 2, status: "closed"},
		{companyIdx: 13, name: "Staff Logistik Proyek Intern", category: "konstruksi", skills: "Manajemen Material, Input Data, Excel", slots: 2, status: "open"},
		{companyIdx: 14, name: "Staff Operasional Gudang Intern", category: "logistik", skills: "Manajemen Stok, Picking Packing, Barcode Scanner", slots: 3, status: "open"},
		{companyIdx: 14, name: "Admin Ekspedisi Intern", category: "logistik", skills: "Input Resi, Tracking Pengiriman, Ms Excel", slots: 2, status: "open"},
		{companyIdx: 14, name: "Staff Checker Intern", category: "logistik", skills: "Pengecekan Barang, Dokumentasi, Ketelitian", slots: 2, status: "closed"},
		{companyIdx: 15, name: "Kurir Intern", category: "logistik", skills: "Pengiriman Barang, Navigasi, Pelayanan Pelanggan", slots: 3, status: "open"},
		{companyIdx: 15, name: "Staff Customer Service Ekspedisi Intern", category: "logistik", skills: "Komunikasi, Penanganan Komplain, Tracking", slots: 2, status: "closed"},
		{companyIdx: 16, name: "Tour Guide Intern", category: "pariwisata", skills: "Komunikasi, Bahasa Inggris Dasar, Pelayanan Wisatawan", slots: 2, status: "open"},
		{companyIdx: 16, name: "Staff Front Office Intern", category: "pariwisata", skills: "Reservasi, Pelayanan Tamu, Ms Office", slots: 2, status: "open"},
		{companyIdx: 17, name: "Staff Housekeeping Intern", category: "pariwisata", skills: "Kebersihan Kamar, Tata Graha, Ketelitian", slots: 3, status: "open"},
		{companyIdx: 17, name: "Staff Food & Beverage Hotel Intern", category: "pariwisata", skills: "Table Manner, Pelayanan Tamu, Kebersihan", slots: 2, status: "closed"},
		{companyIdx: 18, name: "Asisten Agronomi Intern", category: "pertanian", skills: "Budidaya Tanaman, Pemupukan, Dokumentasi Lahan", slots: 2, status: "open"},
		{companyIdx: 18, name: "Staff Pengolahan Hasil Tani Intern", category: "pertanian", skills: "Sortasi, Pengemasan, Kontrol Kualitas", slots: 2, status: "open"},
		{companyIdx: 19, name: "Staff Administrasi Koperasi Tani Intern", category: "pertanian", skills: "Pembukuan, Input Data, Pelayanan Anggota", slots: 2, status: "closed"},
		{companyIdx: 19, name: "Staff Distribusi Hasil Tani Intern", category: "pertanian", skills: "Logistik, Stok, Dokumentasi", slots: 2, status: "open"},
		{companyIdx: 20, name: "Videographer Intern", category: "media", skills: "Videografi, Editing, Storytelling", slots: 2, status: "open"},
		{companyIdx: 20, name: "Content Writer Intern", category: "media", skills: "Copywriting, SEO, Riset Konten", slots: 2, status: "open"},
		{companyIdx: 21, name: "Asisten Apoteker Intern", category: "kesehatan", skills: "Pelayanan Resep, K3, Dokumentasi Obat", slots: 2, status: "open"},
		{companyIdx: 21, name: "Staff Gudang Farmasi Intern", category: "kesehatan", skills: "Manajemen Stok Obat, Input Data, Inventaris", slots: 2, status: "closed"},
		{companyIdx: 22, name: "Staff Restoran Intern", category: "kuliner", skills: "Pelayanan Tamu, Table Manner, Kebersihan", slots: 3, status: "open"},
		{companyIdx: 22, name: "Barista Intern", category: "kuliner", skills: "Pembuatan Kopi, Pelayanan Pelanggan, Kebersihan Alat", slots: 2, status: "open"},
		{companyIdx: 23, name: "Teknisi Motor Intern", category: "otomotif", skills: "Servis Motor, Diagnostik, K3", slots: 3, status: "open"},
		{companyIdx: 23, name: "Sales Consultant Intern", category: "otomotif", skills: "Komunikasi, Negosiasi, Product Knowledge", slots: 2, status: "closed"},
		{companyIdx: 24, name: "Staff Customer Service Intern", category: "keuangan", skills: "Komunikasi, Penanganan Komplain, Ms Office", slots: 2, status: "open"},
		{companyIdx: 24, name: "Data Analyst Intern", category: "keuangan", skills: "Excel, SQL Dasar, Visualisasi Data", slots: 2, status: "open"},
	}

	// studentIdx 0/1 are budi/siti, 2..23 are the original 22 pool names in
	// order. vacancyIdx indexes into vacancySpecs above. internGroup/
	// scoreTarget are only meaningful for "accepted" rows — see the switch
	// below. 218 more of these are generated further down (see
	// "generatedAppliances") to scale the total to ~250 without hand-typing
	// them; this literal block is left exactly as the original seed had it
	// so already-seeded rows resolve to the same (student, vacancy) pairs.
	applianceSpecs := []applianceSpec{
		// accepted — completed placements (5)
		{studentIdx: 0, vacancyIdx: 0, status: "accepted", createdDaysAgo: 165, updatedDaysAgo: 160, internGroup: "completed", scoreTarget: 95, notify: true},
		{studentIdx: 1, vacancyIdx: 2, status: "accepted", createdDaysAgo: 165, updatedDaysAgo: 160, internGroup: "completed", scoreTarget: 88, notify: true},
		{studentIdx: 2, vacancyIdx: 4, status: "accepted", createdDaysAgo: 165, updatedDaysAgo: 160, internGroup: "completed", scoreTarget: 78, notify: true},
		{studentIdx: 3, vacancyIdx: 6, status: "accepted", createdDaysAgo: 165, updatedDaysAgo: 160, internGroup: "completed", scoreTarget: 65, notify: true},
		{studentIdx: 4, vacancyIdx: 9, status: "accepted", createdDaysAgo: 165, updatedDaysAgo: 160, internGroup: "completed", scoreTarget: 92, notify: true},
		// accepted — ongoing placements (6)
		{studentIdx: 5, vacancyIdx: 1, status: "accepted", createdDaysAgo: 45, updatedDaysAgo: 42, internGroup: "ongoing", scoreTarget: 82, notify: true},
		{studentIdx: 6, vacancyIdx: 5, status: "accepted", createdDaysAgo: 45, updatedDaysAgo: 42, internGroup: "ongoing", scoreTarget: 72, notify: true},
		{studentIdx: 7, vacancyIdx: 8, status: "accepted", createdDaysAgo: 45, updatedDaysAgo: 42, internGroup: "ongoing", scoreTarget: 55, notify: true},
		{studentIdx: 8, vacancyIdx: 11, status: "accepted", createdDaysAgo: 45, updatedDaysAgo: 42, internGroup: "ongoing", scoreTarget: 90, notify: true},
		{studentIdx: 9, vacancyIdx: 12, status: "accepted", createdDaysAgo: 45, updatedDaysAgo: 42, internGroup: "ongoing", scoreTarget: 85, notify: true},
		{studentIdx: 10, vacancyIdx: 13, status: "accepted", createdDaysAgo: 45, updatedDaysAgo: 42, internGroup: "ongoing", scoreTarget: 76, notify: true},
		// accepted — starting soon, no attendance/score history yet (3)
		{studentIdx: 11, vacancyIdx: 0, status: "accepted", createdDaysAgo: 12, updatedDaysAgo: 9, internGroup: "soon", notify: true},
		{studentIdx: 12, vacancyIdx: 2, status: "accepted", createdDaysAgo: 12, updatedDaysAgo: 9, internGroup: "soon", notify: true},
		{studentIdx: 13, vacancyIdx: 4, status: "accepted", createdDaysAgo: 12, updatedDaysAgo: 9, internGroup: "soon", notify: true},
		// pending (3)
		{studentIdx: 14, vacancyIdx: 1, status: "pending", createdDaysAgo: 3, updatedDaysAgo: 3},
		{studentIdx: 15, vacancyIdx: 6, status: "pending", createdDaysAgo: 3, updatedDaysAgo: 3},
		{studentIdx: 16, vacancyIdx: 9, status: "pending", createdDaysAgo: 3, updatedDaysAgo: 3},
		// processed (3)
		{studentIdx: 17, vacancyIdx: 5, status: "processed", createdDaysAgo: 10, updatedDaysAgo: 4, notify: true},
		{studentIdx: 18, vacancyIdx: 8, status: "processed", createdDaysAgo: 10, updatedDaysAgo: 4, notify: true},
		{studentIdx: 19, vacancyIdx: 11, status: "processed", createdDaysAgo: 10, updatedDaysAgo: 4, notify: true},
		// rejected (2)
		{studentIdx: 20, vacancyIdx: 12, status: "rejected", createdDaysAgo: 21, updatedDaysAgo: 14, notify: true},
		{studentIdx: 21, vacancyIdx: 13, status: "rejected", createdDaysAgo: 21, updatedDaysAgo: 14, notify: true},
		// canceled (2)
		{studentIdx: 22, vacancyIdx: 0, status: "canceled", createdDaysAgo: 14, updatedDaysAgo: 10, notify: true},
		{studentIdx: 23, vacancyIdx: 2, status: "canceled", createdDaysAgo: 14, updatedDaysAgo: 10, notify: true},
		// earlier rejected attempts for students who later got accepted elsewhere (6) —
		// realistic "applied, got rejected, applied again" history, doesn't collide
		// with their accepted (student, vacancy) pair above.
		{studentIdx: 0, vacancyIdx: 5, status: "rejected", createdDaysAgo: 190, updatedDaysAgo: 180},
		{studentIdx: 2, vacancyIdx: 7, status: "rejected", createdDaysAgo: 190, updatedDaysAgo: 180},
		{studentIdx: 4, vacancyIdx: 3, status: "rejected", createdDaysAgo: 190, updatedDaysAgo: 180},
		{studentIdx: 6, vacancyIdx: 11, status: "rejected", createdDaysAgo: 70, updatedDaysAgo: 60},
		{studentIdx: 8, vacancyIdx: 10, status: "rejected", createdDaysAgo: 70, updatedDaysAgo: 60},
		{studentIdx: 10, vacancyIdx: 9, status: "rejected", createdDaysAgo: 70, updatedDaysAgo: 60},
		// earlier canceled attempts for students still in the pending pipeline (2)
		{studentIdx: 15, vacancyIdx: 12, status: "canceled", createdDaysAgo: 20, updatedDaysAgo: 18},
		{studentIdx: 18, vacancyIdx: 4, status: "canceled", createdDaysAgo: 20, updatedDaysAgo: 18},
	}

	newsSpecs := []struct {
		title, content, status string
		createdDaysAgo         int
	}{
		{
			title:          "Pengumuman Jadwal PKL Semester Ganjil 2026/2027",
			content:        "Sekolah mengumumkan bahwa pelaksanaan Praktik Kerja Lapangan (PKL) semester ganjil tahun ajaran 2026/2027 akan dimulai pada awal bulan depan. Siswa kelas XII diharapkan segera melengkapi berkas lamaran melalui sistem.",
			status:         "published",
			createdDaysAgo: 30,
		},
		{
			title:          "Batas Akhir Pengumpulan Berkas Lamaran PKL",
			content:        "Diinformasikan kepada seluruh siswa kelas XII bahwa batas akhir pengumpulan berkas lamaran PKL melalui sistem adalah dua minggu dari sekarang. Pastikan CV dan data diri sudah lengkap sebelum melamar ke perusahaan mitra.",
			status:         "published",
			createdDaysAgo: 20,
		},
		{
			title:          "Selamat kepada Siswa yang Telah Diterima PKL",
			content:        "Selamat kepada seluruh siswa yang telah dinyatakan diterima magang di perusahaan mitra. Tetap jaga sikap dan etos kerja yang baik selama menjalani PKL, dan jangan lupa mengisi presensi serta jurnal harian secara rutin.",
			status:         "published",
			createdDaysAgo: 10,
		},
		{
			title:          "Jadwal Pembekalan PKL untuk Siswa Kelas XII",
			content:        "Pembekalan PKL akan dilaksanakan di aula sekolah bagi seluruh siswa kelas XII yang belum mendapatkan tempat magang. Materi meliputi etika kerja, tata cara presensi, dan pengisian jurnal harian di sistem.",
			status:         "draft",
			createdDaysAgo: 2,
		},
		{
			title:          "Update Prosedur Monitoring Kunjungan DU/DI",
			content:        "Koordinator PKL akan melakukan kunjungan monitoring ke seluruh perusahaan mitra secara berkala. Prosedur pelaporan hasil kunjungan telah diperbarui dan dapat dilihat pada panduan koordinator di sistem.",
			status:         "draft",
			createdDaysAgo: 1,
		},
		{
			title:          "Sosialisasi Perusahaan Mitra Baru Semester Ini",
			content:        "Sekolah telah menjalin kerja sama dengan sejumlah perusahaan mitra baru dari berbagai bidang, mulai dari ritel, konstruksi, hingga pariwisata. Siswa dapat melihat daftar lengkapnya pada halaman lowongan di sistem.",
			status:         "published",
			createdDaysAgo: 45,
		},
		{
			title:          "Panduan Pengisian Jurnal Harian PKL",
			content:        "Untuk memudahkan siswa dalam mendokumentasikan kegiatan magang, sekolah menerbitkan panduan singkat pengisian jurnal harian yang baik dan benar. Panduan dapat diunduh melalui halaman FAQ.",
			status:         "published",
			createdDaysAgo: 60,
		},
		{
			title:          "Perpanjangan Waktu Pendaftaran PKL Gelombang 2",
			content:        "Menanggapi banyaknya permintaan, sekolah memperpanjang waktu pendaftaran PKL gelombang 2 selama satu minggu ke depan. Siswa yang belum mendapatkan tempat magang dapat memanfaatkan kesempatan ini.",
			status:         "published",
			createdDaysAgo: 25,
		},
		{
			title:          "Hasil Monitoring Kunjungan DU/DI Triwulan Ini",
			content:        "Rekap hasil kunjungan monitoring ke perusahaan mitra pada triwulan ini sedang disusun oleh tim koordinator dan akan dipublikasikan setelah proses verifikasi selesai.",
			status:         "draft",
			createdDaysAgo: 5,
		},
		{
			title:          "Workshop Persiapan Wawancara Kerja bagi Siswa PKL",
			content:        "Sekolah mengadakan workshop persiapan wawancara kerja untuk membekali siswa yang akan melamar PKL maupun mempersiapkan diri memasuki dunia kerja setelah lulus.",
			status:         "published",
			createdDaysAgo: 15,
		},
		{
			title:          "Perubahan Jadwal Pengumpulan Laporan Akhir PKL",
			content:        "Jadwal pengumpulan laporan akhir PKL mengalami perubahan menjadi dua minggu setelah masa magang berakhir. Siswa diharapkan memperhatikan tenggat waktu yang baru ini.",
			status:         "published",
			createdDaysAgo: 8,
		},
		{
			title:          "Apresiasi untuk Perusahaan Mitra Terbaik Tahun Ini",
			content:        "Sekolah memberikan apresiasi kepada beberapa perusahaan mitra yang dinilai memberikan bimbingan terbaik bagi siswa PKL berdasarkan hasil evaluasi monitoring dan ulasan siswa.",
			status:         "published",
			createdDaysAgo: 3,
		},
		{
			title:          "Prosedur Pengajuan Sertifikat PKL bagi Siswa yang Telah Selesai",
			content:        "Siswa yang telah menyelesaikan masa PKL dan seluruh komponen nilainya telah diisi mentor dapat mengajukan sertifikat PKL melalui halaman nilai pada akun masing-masing.",
			status:         "published",
			createdDaysAgo: 12,
		},
		{
			title:          "Rencana Kunjungan Industri untuk Siswa Kelas XI",
			content:        "Sebagai persiapan menghadapi PKL tahun depan, sekolah tengah merencanakan kunjungan industri bagi siswa kelas XI ke beberapa perusahaan mitra terpilih.",
			status:         "draft",
			createdDaysAgo: 6,
		},
		{
			title:          "Evaluasi Pelaksanaan PKL Semester Genap 2025/2026",
			content:        "Laporan evaluasi pelaksanaan PKL semester genap tahun ajaran 2025/2026 telah selesai disusun, mencakup tingkat kehadiran, capaian nilai, dan umpan balik dari perusahaan mitra.",
			status:         "published",
			createdDaysAgo: 90,
		},
	}

	faqSpecs := []struct{ question, answer string }{
		{"Bagaimana cara mendaftar PKL di sistem ini?", "Siswa dapat mendaftar menggunakan kode undangan yang diberikan oleh koordinator sekolah, lalu melengkapi data diri pada halaman registrasi."},
		{"Apa saja syarat mengikuti PKL?", "Siswa harus terdaftar aktif di kelas XII, memiliki NIS yang valid, dan telah mengikuti pembekalan PKL yang diadakan sekolah."},
		{"Berapa lama durasi pelaksanaan PKL?", "Durasi PKL umumnya berlangsung 3 hingga 6 bulan, tergantung kebijakan masing-masing perusahaan mitra."},
		{"Bagaimana jika lamaran PKL saya ditolak?", "Siswa dapat melamar kembali ke lowongan lain yang masih terbuka melalui sistem tanpa batasan jumlah lamaran."},
		{"Bagaimana cara mengunduh sertifikat PKL?", "Sertifikat dapat diunduh melalui halaman nilai setelah seluruh komponen nilai magang diisi oleh mentor perusahaan."},
		{"Apa yang harus dilakukan jika lupa mengisi presensi?", "Segera hubungi mentor di perusahaan untuk konfirmasi kehadiran, karena presensi yang terlewat tidak dapat diisi secara otomatis oleh sistem."},
		{"Apakah siswa boleh memilih sendiri perusahaan tempat PKL?", "Siswa dapat melamar ke perusahaan mitra yang tersedia di sistem sesuai minat dan kompetensi keahliannya, namun penempatan akhir tetap mempertimbangkan persetujuan koordinator."},
		{"Bagaimana jika siswa sakit dan tidak bisa hadir PKL?", "Siswa wajib mengisi presensi dengan status Sakit disertai keterangan, dan jika perlu melampirkan surat keterangan dokter kepada mentor."},
		{"Apakah nilai PKL memengaruhi kelulusan?", "Ya, nilai PKL menjadi salah satu komponen penilaian praktik kejuruan yang diperhitungkan dalam penilaian akhir siswa."},
		{"Bagaimana cara menghubungi koordinator PKL?", "Siswa dapat menghubungi koordinator melalui fitur pesan pada sistem atau datang langsung ke ruang bimbingan PKL di sekolah."},
		{"Apa yang terjadi jika perusahaan menutup lowongan sebelum saya melamar?", "Lowongan yang berstatus tertutup tidak dapat menerima lamaran baru; siswa dapat mencari lowongan lain yang masih terbuka."},
		{"Apakah siswa bisa mengganti perusahaan PKL di tengah jalan?", "Perubahan tempat PKL hanya dapat dilakukan atas persetujuan koordinator dan pihak sekolah dengan alasan yang kuat."},
		{"Siapa yang menilai kinerja siswa selama PKL?", "Penilaian kinerja dilakukan oleh mentor di perusahaan berdasarkan aspek teknis dan non-teknis selama masa magang."},
		{"Apakah ada kunjungan dari sekolah selama PKL berlangsung?", "Ya, koordinator akan melakukan kunjungan monitoring secara berkala untuk memantau perkembangan siswa di tempat PKL."},
		{"Bagaimana cara melihat riwayat lamaran PKL saya?", "Riwayat lamaran dapat dilihat pada halaman lamaran di akun siswa masing-masing."},
		{"Apakah siswa mendapat uang saku selama PKL?", "Kebijakan uang saku sepenuhnya ditentukan oleh masing-masing perusahaan mitra dan bukan merupakan kewajiban dari sekolah."},
		{"Apa yang harus dilakukan jika jurnal harian belum disetujui mentor?", "Siswa dapat mengonfirmasi langsung kepada mentor terkait status jurnal yang belum disetujui."},
		{"Bagaimana jika data NIS saya salah di sistem?", "Siswa dapat menghubungi koordinator sekolah untuk melakukan perbaikan data NIS melalui panel admin."},
		// faqs has no school_id column (see migration 000022) so these two
		// are intentionally general rather than "school-specific" — the
		// schema has no way to scope a FAQ to one school.
		{"Apakah sistem PKL ini digunakan oleh lebih dari satu sekolah?", "Ya, sistem ini digunakan oleh beberapa sekolah berbeda; setiap sekolah memiliki data siswa, perusahaan mitra, dan koordinator masing-masing yang terpisah."},
		{"Apakah siswa atau koordinator dari sekolah lain bisa melihat data sekolah saya?", "Tidak. Setiap sekolah memiliki ruang lingkup data yang terpisah (multi-tenant), sehingga siswa dan koordinator hanya dapat mengakses data milik sekolahnya sendiri."},
	}

	questionSpecs := []struct {
		question  string
		sortOrder int
	}{
		{"Apakah lingkungan kerja di tempat PKL sesuai dengan bidang keahlian siswa?", 1},
		{"Apakah pembimbing industri memberikan bimbingan yang cukup kepada siswa?", 2},
		{"Apakah fasilitas kerja memadai untuk mendukung kegiatan PKL siswa?", 3},
		{"Apakah siswa mendapatkan pekerjaan yang relevan dengan kompetensi keahliannya?", 4},
		{"Apakah perusahaan bersedia menerima siswa PKL pada periode berikutnya?", 5},
	}

	// companyIdx indexes into companySpecs; studentIdx indexes into the
	// combined students list built inside the transaction. 20 more of these
	// are generated below (sampling the generated placements) to scale from
	// 5 to ~25.
	monitorSpecs := []monitorSpec{
		{studentIdx: 0, companyIdx: 0, daysAgo: 40, matchRating: 4, notes: "Siswa terlihat aktif dan cepat beradaptasi dengan pekerjaan di divisi frontend.", suggest: "Pertahankan komunikasi rutin dengan mentor terkait progres tugas."},
		{studentIdx: 5, companyIdx: 0, daysAgo: 10, matchRating: 3, notes: "Siswa sudah mulai memahami alur kerja tim UI/UX di perusahaan.", suggest: "Tingkatkan inisiatif dalam mengikuti diskusi desain tim."},
		{studentIdx: 6, companyIdx: 2, daysAgo: 12, matchRating: 3, notes: "Siswa cukup baik dalam memahami materi digital marketing yang diberikan mentor.", suggest: "Perlu lebih percaya diri saat presentasi hasil kerja."},
		{studentIdx: 7, companyIdx: 4, daysAgo: 15, matchRating: 4, notes: "Siswa disiplin dan teliti dalam membantu pekerjaan di laboratorium.", suggest: "Lanjutkan konsistensi dalam menjaga kebersihan dan keselamatan kerja."},
		{studentIdx: 9, companyIdx: 7, daysAgo: 9, matchRating: 3, notes: "Siswa menunjukkan minat besar pada praktik servis kendaraan.", suggest: "Perbanyak jam praktik langsung didampingi teknisi senior."},
	}

	// questionIdx indexes into questionSpecs. 20 more of these are generated
	// below to scale from 5 to ~25.
	mentorReviewSpecs := []mentorReviewSpec{
		{studentIdx: 0, companyIdx: 0, questionIdx: 3, rating: 5, title: "Evaluasi Kinerja Magang", body: "Siswa menunjukkan kinerja yang sangat baik selama magang, aktif bertanya dan cepat memahami tugas yang diberikan."},
		{studentIdx: 1, companyIdx: 1, questionIdx: 3, rating: 4, title: "Evaluasi Kinerja Magang", body: "Siswa cukup baik dalam menyelesaikan tugas, namun perlu meningkatkan inisiatif dalam bekerja secara mandiri."},
		{studentIdx: 2, companyIdx: 2, questionIdx: 3, rating: 5, title: "Evaluasi Kinerja Magang", body: "Siswa sangat komunikatif dan mampu bekerja sama dengan baik dalam tim pengembang."},
		{studentIdx: 3, companyIdx: 3, questionIdx: 3, rating: 4, title: "Evaluasi Kinerja Magang", body: "Siswa cukup kreatif dalam menghasilkan konten, meski masih perlu bimbingan dalam manajemen waktu."},
		{studentIdx: 4, companyIdx: 5, questionIdx: 3, rating: 5, title: "Evaluasi Kinerja Magang", body: "Siswa ramah dan disiplin dalam melayani pelanggan, menjadi contoh baik bagi peserta magang lain."},
	}

	// 16 more of these are generated below (from the same placement pool as
	// the mentor reviews) to scale total reviews to ~40-50.
	companyReviewSpecs := []companyReviewSpec{
		{studentIdx: 0, companyIdx: 0, rating: 5, title: "Pengalaman PKL di PT Mumtaz Teknologi Indonesia", body: "Suasana kerja sangat mendukung untuk belajar, mentor selalu membimbing dengan sabar."},
		{studentIdx: 2, companyIdx: 2, rating: 4, title: "Pengalaman PKL di CV Kreasi Digital Bogor", body: "Banyak ilmu baru yang didapat terutama soal pengembangan aplikasi mobile."},
		{studentIdx: 4, companyIdx: 5, rating: 5, title: "Pengalaman PKL di PT Cipta Boga Nusantara", body: "Tim kerja sangat solid dan ramah, cocok untuk belajar pelayanan pelanggan."},
	}

	// --- second and third schools: SMKN 2 Bogor (TKJ) and SMKN 1 Depok (MM) ---
	//
	// Genuinely separate tenants (own department/courses/companies/students),
	// deliberately smaller than SMKN 1 Cibinong's 25 companies/150 students —
	// real schools vary in size. Built via seedSchool (below) rather than a
	// third copy-paste of everything above, but that function calls the
	// exact same upsert*/bulk* helpers in the exact same order.
	bogorConfig := schoolSeedConfig{
		schoolName: "SMKN 2 Bogor", schoolEmail: "info@smkn2bogor.sch.id", schoolPhone: "0251-8321XXX", schoolAddress: "Bogor, Jawa Barat",
		deptName: "Teknik Komputer dan Jaringan", deptDescription: "Jurusan Teknik Komputer dan Jaringan", studyProgram: "TKJ",
		courseNames:      []string{"XI TKJ 1", "XI TKJ 2", "XII TKJ 1"},
		inviteCodes:      []string{"TKJ1DEMO", "TKJ2DEMO", "TKJ3DEMO"},
		nisPrefix:        "2024002", // school code "002" — see maxNIS comment above; SMKN 1 Cibinong already occupies "001"
		coordinatorEmail: "coordinator.bogor@internity.test", coordinatorName: "Koordinator SMKN 2 Bogor",
		companies: []companySpec{
			{"PT Jaringan Data Prima", "Teknologi", "Bogor"},
			{"CV Teknologi Net Solusi", "Teknologi", "Sukabumi"},
			{"PT Telekomunikasi Nusa Raya", "Telekomunikasi", "Jakarta"},
			{"PT Infrastruktur Komputasi Indonesia", "Teknologi", "Depok"},
			{"CV Solusi Jaringan Mandiri", "Teknologi", "Bogor"},
			{"PT Cyber Network Solusi", "Teknologi", "Cibinong"},
			{"PT Fiber Optik Nusantara", "Telekomunikasi", "Bogor"},
			{"CV Komputer Servis Jaya", "Teknologi", "Cianjur"},
			{"PT Data Center Indonesia Raya", "Teknologi", "Jakarta"},
		},
		mentorNames: []string{
			"Mentor Jaringan Data Prima", "Mentor Teknologi Net Solusi", "Mentor Telekomunikasi Nusa Raya",
			"Mentor Infrastruktur Komputasi", "Mentor Solusi Jaringan Mandiri", "Mentor Cyber Network Solusi",
			"Mentor Fiber Optik Nusantara", "Mentor Komputer Servis Jaya", "Mentor Data Center Indonesia",
		},
		mentorEmailPrefix: "mentor.bogor",
		students: []string{
			"Rafi Ardiansyah", "Nabil Firmansyah", "Zaki Ramadhan", "Fadhil Akbar", "Reza Maulana", "Yoga Pratama",
			"Dimas Kurniawan", "Bagus Setiawan", "Ilham Nugraha", "Wahyu Ramadhan", "Rangga Saputra", "Teguh Prasetyo",
			"Fajar Ramadhani", "Aldi Firmansyah", "Kevin Hidayat", "Rio Alamsyah", "Satria Wibowo", "Yudha Pratama",
			"Bima Prasetya", "Angga Saputra", "Dio Ferdiansyah", "Rendra Kusuma", "Fikri Ramadhan", "Galang Pradana",
			"Vino Anggara", "Putra Wijaya", "Alya Ramadhani", "Nisa Aulia", "Salsabila Putri", "Zahra Amelia",
			"Intan Permata", "Winda Lestari", "Rahma Fitriani", "Dinar Anggraini", "Silvi Maharani", "Keisya Ramadhani",
			"Vania Kusuma", "Talita Azzahra", "Naura Salsabila", "Citra Ayu Lestari", "Marsha Anindya", "Aisyah Nurhaliza",
			"Regina Putri Wardani", "Shafira Anjani", "Karina Dewi Safitri", "Yasmin Aulia Rahma",
		},
		vacancies: []vacancySpec{
			{companyIdx: 0, name: "Teknisi Jaringan Intern", category: "teknis", skills: "Jaringan LAN/WAN, Konfigurasi Router, Troubleshooting", slots: 2, status: "open"},
			{companyIdx: 0, name: "IT Support Intern", category: "teknis", skills: "Troubleshooting Hardware, Instalasi Software, Help Desk", slots: 2, status: "open"},
			{companyIdx: 1, name: "Network Administrator Intern", category: "teknis", skills: "Mikrotik, VLAN, Monitoring Jaringan", slots: 2, status: "open"},
			{companyIdx: 1, name: "Web Administrator Intern", category: "teknis", skills: "Linux Server, Hosting, DNS", slots: 2, status: "closed"},
			{companyIdx: 2, name: "Teknisi Instalasi Jaringan Intern", category: "teknis", skills: "Kabel Fiber Optik, Splicing, OTDR", slots: 3, status: "open"},
			{companyIdx: 2, name: "Customer Service Teknis Intern", category: "teknis", skills: "Komunikasi, Troubleshooting Dasar, Dokumentasi", slots: 2, status: "open"},
			{companyIdx: 3, name: "Data Center Support Intern", category: "teknis", skills: "Server Rack, Monitoring, Keamanan Data Center", slots: 2, status: "open"},
			{companyIdx: 3, name: "System Administrator Intern", category: "teknis", skills: "Linux, Virtualisasi, Backup Data", slots: 2, status: "closed"},
			{companyIdx: 4, name: "Teknisi Jaringan Kantor Intern", category: "teknis", skills: "LAN, Wifi, Troubleshooting", slots: 2, status: "open"},
			{companyIdx: 4, name: "Help Desk Intern", category: "teknis", skills: "Ticketing System, Remote Support, Dokumentasi", slots: 2, status: "open"},
			{companyIdx: 5, name: "Cyber Security Support Intern", category: "teknis", skills: "Keamanan Jaringan, Firewall, Monitoring", slots: 2, status: "open"},
			{companyIdx: 5, name: "Network Monitoring Intern", category: "teknis", skills: "Monitoring Tools, Troubleshooting, Dokumentasi", slots: 2, status: "closed"},
			{companyIdx: 6, name: "Teknisi Fiber Optik Intern", category: "teknis", skills: "Splicing Fiber Optik, OTDR, Instalasi FTTH", slots: 3, status: "open"},
			{companyIdx: 6, name: "Instalasi Internet Rumah Intern", category: "teknis", skills: "Instalasi Modem, Troubleshooting, Pelayanan Pelanggan", slots: 2, status: "open"},
			{companyIdx: 7, name: "Teknisi Komputer Intern", category: "teknis", skills: "Perakitan PC, Instalasi OS, Troubleshooting Hardware", slots: 3, status: "open"},
			{companyIdx: 7, name: "Staff Servis Elektronik Intern", category: "teknis", skills: "Diagnostik Perangkat, Perbaikan Hardware, K3", slots: 2, status: "closed"},
			{companyIdx: 8, name: "Data Center Operator Intern", category: "teknis", skills: "Monitoring Server, Dokumentasi, K3", slots: 2, status: "open"},
			{companyIdx: 8, name: "IT Infrastructure Intern", category: "teknis", skills: "Virtualisasi, Cloud Dasar, Monitoring", slots: 2, status: "open"},
		},
		placementCounts:   [3]int{14, 14, 8},
		nonAcceptedCounts: [4]int{10, 8, 8, 8},
		news: []newsSpec{
			{title: "Pembukaan Pendaftaran PKL TKJ SMKN 2 Bogor", content: "SMKN 2 Bogor membuka pendaftaran PKL bagi siswa jurusan Teknik Komputer dan Jaringan. Siswa dapat mendaftar melalui sistem menggunakan kode undangan dari koordinator.", status: "published", createdDaysAgo: 35},
			{title: "Kerja Sama Baru dengan Perusahaan Jaringan dan Telekomunikasi", content: "Sekolah menjalin kerja sama dengan sejumlah perusahaan mitra baru di bidang jaringan, telekomunikasi, dan data center untuk memperluas kesempatan PKL siswa TKJ.", status: "published", createdDaysAgo: 20},
			{title: "Jadwal Sertifikasi Kompetensi Jaringan untuk Siswa TKJ", content: "Sertifikasi kompetensi keahlian jaringan bagi siswa kelas XII TKJ akan dilaksanakan setelah pelaksanaan PKL selesai. Detail jadwal akan disampaikan lebih lanjut.", status: "draft", createdDaysAgo: 3},
			{title: "Selamat kepada Siswa TKJ yang Diterima PKL Gelombang Ini", content: "Selamat kepada seluruh siswa TKJ yang telah diterima magang di perusahaan mitra. Tetap disiplin mengisi presensi dan jurnal harian selama masa PKL.", status: "published", createdDaysAgo: 8},
			{title: "Panduan Instalasi dan Troubleshooting Jaringan bagi Siswa PKL", content: "Sekolah menerbitkan panduan singkat instalasi dan troubleshooting jaringan dasar untuk membantu siswa TKJ selama menjalani PKL di perusahaan mitra.", status: "published", createdDaysAgo: 50},
		},
	}

	depokConfig := schoolSeedConfig{
		schoolName: "SMKN 1 Depok", schoolEmail: "info@smkn1depok.sch.id", schoolPhone: "021-7520XXX", schoolAddress: "Depok, Jawa Barat",
		deptName: "Multimedia", deptDescription: "Jurusan Multimedia", studyProgram: "MM",
		courseNames:      []string{"XI MM 1", "XI MM 2", "XII MM 1"},
		inviteCodes:      []string{"MM1DEMO", "MM2DEMO", "MM3DEMO"},
		nisPrefix:        "2024003", // school code "003"
		coordinatorEmail: "coordinator.depok@internity.test", coordinatorName: "Koordinator SMKN 1 Depok",
		companies: []companySpec{
			{"PT Kreasi Visual Nusantara", "Media & Percetakan", "Depok"},
			{"CV Animasi Karya Digital", "Media & Percetakan", "Depok"},
			{"PT Cahaya Studio Produksi", "Media & Percetakan", "Jakarta"},
			{"PT Periklanan Kreatif Bangsa", "Media & Percetakan", "Depok"},
			{"CV Fotografi Cipta Rasa", "Media & Percetakan", "Bekasi"},
			{"PT Percetakan Warna Abadi", "Media & Percetakan", "Depok"},
			{"PT Broadcast Media Indonesia", "Media & Percetakan", "Jakarta"},
			{"CV Desain Grafis Mandiri Jaya", "Media & Percetakan", "Bogor"},
		},
		mentorNames: []string{
			"Mentor Kreasi Visual Nusantara", "Mentor Animasi Karya Digital", "Mentor Cahaya Studio Produksi",
			"Mentor Periklanan Kreatif Bangsa", "Mentor Fotografi Cipta Rasa", "Mentor Percetakan Warna Abadi",
			"Mentor Broadcast Media Indonesia", "Mentor Desain Grafis Mandiri",
		},
		mentorEmailPrefix: "mentor.depok",
		students: []string{
			"Arya Bramantyo", "Farrel Ananda", "Raihan Pratama", "Fauzan Akbar", "Miftah Ramadhan", "Ezra Firmansyah",
			"Dafa Kurniawan", "Rasyid Maulana", "Bayu Aditya", "Farhan Nugraha", "Habib Ramadhan", "Gading Saputra",
			"Rifqi Hakim", "Naufal Ramadhan", "Yusril Firmansyah", "Aryo Wicaksono", "Doni Prasetyo", "Bagas Ramadhan",
			"Fauzi Alamsyah", "Reyhan Pradana", "Cakra Wibowo", "Panji Anggara", "Alifa Zahra", "Nadhira Salsabila",
			"Kayla Amelia", "Zalfa Putri", "Indira Maharani", "Ratu Anggraini", "Cinta Lestari", "Ghina Aulia",
			"Syifa Ramadhani", "Almira Kusuma", "Dyah Ayu Permata", "Rani Oktaviani", "Bella Safitri", "Kanaya Azzahra",
			"Sabrina Wardani", "Zaskia Anindya", "Hana Puspitasari", "Adinda Maheswari", "Larasati Wijaya", "Fitria Anggun Pratiwi",
		},
		vacancies: []vacancySpec{
			{companyIdx: 0, name: "Video Editor Intern", category: "media", skills: "Adobe Premiere, Color Grading, Storytelling", slots: 2, status: "open"},
			{companyIdx: 0, name: "Motion Graphic Intern", category: "media", skills: "After Effects, Animasi 2D, Rendering", slots: 2, status: "open"},
			{companyIdx: 1, name: "Animator 2D Intern", category: "media", skills: "Character Design, Animasi Frame by Frame, Storyboard", slots: 2, status: "open"},
			{companyIdx: 1, name: "Ilustrator Digital Intern", category: "desain", skills: "Adobe Illustrator, Digital Painting, Konsep Visual", slots: 2, status: "closed"},
			{companyIdx: 2, name: "Videographer Intern", category: "media", skills: "Sinematografi, Lighting, Editing Video", slots: 3, status: "open"},
			{companyIdx: 2, name: "Fotografer Produk Intern", category: "media", skills: "Studio Lighting, Fotografi Produk, Editing Foto", slots: 2, status: "open"},
			{companyIdx: 3, name: "Content Creator Intern", category: "media", skills: "Copywriting, Ide Konten, Media Sosial", slots: 2, status: "open"},
			{companyIdx: 3, name: "Social Media Specialist Intern", category: "marketing", skills: "Content Planning, Engagement, Analitik Media Sosial", slots: 2, status: "closed"},
			{companyIdx: 4, name: "Fotografer Event Intern", category: "media", skills: "Fotografi Candid, Editing, Dokumentasi Acara", slots: 2, status: "open"},
			{companyIdx: 4, name: "Staff Studio Foto Intern", category: "media", skills: "Setting Studio, Fotografi Portrait, Retouching", slots: 2, status: "open"},
			{companyIdx: 5, name: "Layout Designer Intern", category: "desain", skills: "Layout Percetakan, Adobe InDesign, Pra-Cetak", slots: 2, status: "open"},
			{companyIdx: 5, name: "Operator Percetakan Intern", category: "teknis", skills: "Mesin Cetak, Quality Control, K3", slots: 2, status: "closed"},
			{companyIdx: 6, name: "Broadcast Crew Intern", category: "media", skills: "Kamera Studio, Switcher, Produksi Siaran", slots: 3, status: "open"},
			{companyIdx: 6, name: "Editor Berita Intern", category: "media", skills: "Editing Video Berita, Adobe Premiere, Deadline Kerja", slots: 2, status: "open"},
			{companyIdx: 7, name: "Graphic Designer Intern", category: "desain", skills: "Adobe Photoshop, Branding, Desain Cetak", slots: 2, status: "open"},
			{companyIdx: 7, name: "UI Design Intern", category: "desain", skills: "Figma, Desain Visual, Mockup", slots: 2, status: "open"},
		},
		placementCounts:   [3]int{12, 12, 7},
		nonAcceptedCounts: [4]int{9, 7, 7, 7},
		news: []newsSpec{
			{title: "Pembukaan Pendaftaran PKL Multimedia SMKN 1 Depok", content: "SMKN 1 Depok membuka pendaftaran PKL bagi siswa jurusan Multimedia. Siswa dapat mendaftar melalui sistem menggunakan kode undangan dari koordinator.", status: "published", createdDaysAgo: 35},
			{title: "Kerja Sama Baru dengan Studio Kreatif dan Perusahaan Media", content: "Sekolah menjalin kerja sama dengan sejumlah studio kreatif, rumah produksi, dan perusahaan media untuk memperluas kesempatan PKL siswa Multimedia.", status: "published", createdDaysAgo: 18},
			{title: "Jadwal Workshop Editing Video untuk Siswa Multimedia", content: "Workshop editing video dan motion graphic akan diadakan bagi siswa kelas XII Multimedia sebagai bekal tambahan sebelum dan selama PKL.", status: "draft", createdDaysAgo: 4},
			{title: "Selamat kepada Siswa Multimedia yang Diterima PKL", content: "Selamat kepada seluruh siswa Multimedia yang telah diterima magang di perusahaan mitra. Tetap disiplin mengisi presensi dan jurnal harian selama masa PKL.", status: "published", createdDaysAgo: 9},
			{title: "Panduan Portofolio Karya untuk Siswa PKL Multimedia", content: "Sekolah menerbitkan panduan singkat penyusunan portofolio karya bagi siswa Multimedia untuk mendukung penilaian akhir PKL.", status: "published", createdDaysAgo: 55},
		},
	}

	statusCounts := map[string]int{}
	var totals struct {
		internDates, presences, journals, scores, certificates, reviews, notifications int
		completed, ongoing, soon                                                       int
	}
	var school2Totals, school3Totals schoolSeedTotals

	err = db.Transaction(func(tx *gorm.DB) error {
		schoolID, err := upsertSchool(tx, ctx)
		if err != nil {
			return fmt.Errorf("school: %w", err)
		}
		deptID, err := upsertDepartment(tx, ctx, schoolID)
		if err != nil {
			return fmt.Errorf("department: %w", err)
		}
		course1ID, err := upsertCourse(tx, ctx, deptID, "XII RPL 1")
		if err != nil {
			return fmt.Errorf("course 1: %w", err)
		}
		course2ID, err := upsertCourse(tx, ctx, deptID, "XII RPL 2")
		if err != nil {
			return fmt.Errorf("course 2: %w", err)
		}
		course3ID, err := upsertCourse(tx, ctx, deptID, "XII RPL 3")
		if err != nil {
			return fmt.Errorf("course 3: %w", err)
		}

		companyIDs := make([]int64, len(companySpecs))
		for i, c := range companySpecs {
			id, err := upsertCompany(tx, ctx, deptID, c.name, c.category, c.city)
			if err != nil {
				return fmt.Errorf("company %s: %w", c.name, err)
			}
			companyIDs[i] = id
		}

		if err := upsertUser(tx, ctx, passwordHash, "admin@internity.test", "Admin Internity", "admin", nil, nil, nil, nil, nil); err != nil {
			return fmt.Errorf("user admin: %w", err)
		}
		if err := upsertUser(tx, ctx, passwordHash, "coordinator@internity.test", "Koordinator Sekolah", "coordinator", &schoolID, nil, nil, nil, nil); err != nil {
			return fmt.Errorf("user coordinator: %w", err)
		}
		coordinatorID, err := getUserID(tx, ctx, "coordinator@internity.test")
		if err != nil {
			return fmt.Errorf("coordinator id: %w", err)
		}

		mentorIDs := make([]string, len(mentorSpecs))
		for i, m := range mentorSpecs {
			if err := upsertUser(tx, ctx, passwordHash, m.email, m.name, "mentor", nil, nil, &companyIDs[i], nil, nil); err != nil {
				return fmt.Errorf("user %s: %w", m.email, err)
			}
			id, err := getUserID(tx, ctx, m.email)
			if err != nil {
				return fmt.Errorf("mentor id %s: %w", m.email, err)
			}
			mentorIDs[i] = id
		}

		// existing emails + the current max NIS drive the new-student pool
		// below — this seed is idempotent and additive, so names/numbers are
		// always derived from what's actually in the DB, never assumed.
		var existingEmailList []string
		if err := tx.WithContext(ctx).Raw(`SELECT email FROM users`).Scan(&existingEmailList).Error; err != nil {
			return fmt.Errorf("existing emails: %w", err)
		}
		existingEmails := make(map[string]bool, len(existingEmailList))
		for _, e := range existingEmailList {
			existingEmails[e] = true
		}
		var maxNIS int64
		if err := tx.WithContext(ctx).Raw(`SELECT COALESCE(MAX(nis::bigint), 2024001000) FROM users WHERE nis ~ '^[0-9]+$'`).Scan(&maxNIS).Error; err != nil {
			return fmt.Errorf("max nis: %w", err)
		}
		nextNIS := maxNIS + 1

		students := []struct {
			email, name, nis string
			courseID         int64
		}{
			{email: "budi@internity.test", name: "Budi Santoso", nis: "2024001001", courseID: course1ID},
			{email: "siti@internity.test", name: "Siti Aminah", nis: "2024001002", courseID: course1ID},
		}
		courseCycle := []int64{course1ID, course2ID, course3ID}
		for j, name := range newStudentNames {
			first := strings.ToLower(strings.SplitN(name, " ", 2)[0])
			email := first + "@internity.test"
			// nis is only actually written when upsertUser's own SELECT
			// finds no existing row for this email; for an already-seeded
			// name the value computed here is discarded, so it's safe to
			// leave it blank instead of re-deriving their real stored nis.
			var nis string
			if !existingEmails[email] {
				nis = fmt.Sprintf("%010d", nextNIS)
				nextNIS++
				existingEmails[email] = true
			}
			students = append(students, struct {
				email, name, nis string
				courseID         int64
			}{email: email, name: name, nis: nis, courseID: courseCycle[j%len(courseCycle)]})
		}

		studentIDs := make([]string, len(students))
		for i, s := range students {
			if err := upsertUser(tx, ctx, passwordHash, s.email, s.name, "student", &schoolID, &deptID, nil, &s.courseID, &s.nis); err != nil {
				return fmt.Errorf("user %s: %w", s.email, err)
			}
			id, err := getUserID(tx, ctx, s.email)
			if err != nil {
				return fmt.Errorf("student id %s: %w", s.email, err)
			}
			studentIDs[i] = id
		}

		statuses := []struct{ name, kind string }{
			{"Hadir", "present"}, {"Izin", "permitted"}, {"Sakit", "sick"}, {"Alpa", "absent"}, {"Libur", "holiday"},
		}
		for _, s := range statuses {
			if err := upsertPresenceStatus(tx, ctx, schoolID, s.name, s.kind); err != nil {
				return fmt.Errorf("presence status %s: %w", s.name, err)
			}
		}
		statusIDs := map[string]int64{}
		for _, kind := range []string{"present", "permitted", "sick", "absent"} {
			var id int64
			if err := tx.WithContext(ctx).Raw(`SELECT id FROM presence_statuses WHERE school_id = ? AND kind = ?`, schoolID, kind).Scan(&id).Error; err != nil {
				return fmt.Errorf("presence status id %s: %w", kind, err)
			}
			statusIDs[kind] = id
		}

		predicates := []struct {
			name     string
			min, max float64
		}{
			{"D", 0, 59.99}, {"C", 60, 74.99}, {"B", 75, 89.99}, {"A", 90, 100},
		}
		for _, p := range predicates {
			if err := upsertScorePredicate(tx, ctx, schoolID, p.name, p.min, p.max); err != nil {
				return fmt.Errorf("score predicate %s: %w", p.name, err)
			}
		}

		vacancyIDs := make([]int64, len(vacancySpecs))
		for i, v := range vacancySpecs {
			id, err := upsertVacancy(tx, ctx, companyIDs[v.companyIdx], v.name, v.category, v.skills, v.slots, v.status)
			if err != nil {
				return fmt.Errorf("vacancy %s: %w", v.name, err)
			}
			vacancyIDs[i] = id
		}

		if err := upsertInviteCode(tx, ctx, course1ID, "RPL1DEMO"); err != nil {
			return fmt.Errorf("invite code 1: %w", err)
		}
		if err := upsertInviteCode(tx, ctx, course2ID, "RPL2DEMO"); err != nil {
			return fmt.Errorf("invite code 2: %w", err)
		}
		if err := upsertInviteCode(tx, ctx, course3ID, "RPL3DEMO"); err != nil {
			return fmt.Errorf("invite code 3: %w", err)
		}

		// --- generated bulk data ---
		//
		// 99 new accepted appliances, each given its own new student (never
		// two placements sharing a student), so intern_dates'
		// UNIQUE(user_id, company_id) and its no-overlap EXCLUDE constraint
		// are trivially satisfied — a student with exactly one intern_dates
		// row can't overlap itself.
		type genPlacement struct {
			studentIdx, vacancyIdx, companyIdx int
			group                              string
		}
		var placements []genPlacement
		pIdx := 0
		addPlacements := func(group string, count int) {
			for k := 0; k < count; k++ {
				studentIdx := newStudentsStart + pIdx
				vacancyIdx := pIdx % len(vacancySpecs)
				placements = append(placements, genPlacement{
					studentIdx: studentIdx, vacancyIdx: vacancyIdx,
					companyIdx: vacancySpecs[vacancyIdx].companyIdx, group: group,
				})
				pIdx++
			}
		}
		addPlacements("completed", 40)
		addPlacements("ongoing", 39)
		addPlacements("soon", 20)

		generatedAppliances := make([]applianceSpec, 0, 218)
		for i, p := range placements {
			var createdDaysAgo, updatedDaysAgo int
			switch p.group {
			case "completed":
				createdDaysAgo = 190 + i%40
				updatedDaysAgo = createdDaysAgo - 5
			case "ongoing":
				createdDaysAgo = 50 + i%60
				updatedDaysAgo = createdDaysAgo - 3
			case "soon":
				createdDaysAgo = 10 + i%10
				updatedDaysAgo = createdDaysAgo - 3
			}
			generatedAppliances = append(generatedAppliances, applianceSpec{
				studentIdx: p.studentIdx, vacancyIdx: p.vacancyIdx, status: "accepted",
				createdDaysAgo: createdDaysAgo, updatedDaysAgo: updatedDaysAgo,
				internGroup: p.group, scoreTarget: 50 + (i*7)%50, notify: true,
			})
		}

		// 119 more non-accepted appliances (rejected/canceled/pending/
		// processed), one per remaining new student so every (student,
		// vacancy) pair used here is fresh — usedPair also seeds from the
		// placements above so a placement student's second application (an
		// earlier attempt at a different vacancy) never collides with their
		// accepted one.
		usedPair := map[[2]int]bool{}
		for _, spec := range applianceSpecs {
			usedPair[[2]int{spec.studentIdx, spec.vacancyIdx}] = true
		}
		for _, p := range placements {
			usedPair[[2]int{p.studentIdx, p.vacancyIdx}] = true
		}

		nonAcceptedTargets := []struct {
			status string
			count  int
		}{
			{"rejected", 29}, {"canceled", 21}, {"pending", 35}, {"processed", 34},
		}
		maxCount := 0
		for _, t := range nonAcceptedTargets {
			if t.count > maxCount {
				maxCount = t.count
			}
		}
		var order []string
		for i := 0; i < maxCount; i++ {
			for _, t := range nonAcceptedTargets {
				if i < t.count {
					order = append(order, t.status)
				}
			}
		}

		statusSeen := map[string]int{}
		for j, status := range order {
			n := statusSeen[status]
			statusSeen[status] = n + 1
			var createdDaysAgo, updatedDaysAgo int
			switch status {
			case "rejected":
				createdDaysAgo = 30 + n%150
				updatedDaysAgo = createdDaysAgo - (5 + n%10)
			case "canceled":
				createdDaysAgo = 10 + n%50
				updatedDaysAgo = createdDaysAgo - (2 + n%6)
			case "pending":
				createdDaysAgo = 1 + n%14
				updatedDaysAgo = createdDaysAgo
			case "processed":
				createdDaysAgo = 5 + n%25
				updatedDaysAgo = createdDaysAgo - (1 + n%4)
			}
			studentIdx := newStudentsStart + j
			vacancyIdx := (j*13 + 5) % len(vacancySpecs)
			for usedPair[[2]int{studentIdx, vacancyIdx}] {
				vacancyIdx = (vacancyIdx + 1) % len(vacancySpecs)
			}
			usedPair[[2]int{studentIdx, vacancyIdx}] = true
			generatedAppliances = append(generatedAppliances, applianceSpec{
				studentIdx: studentIdx, vacancyIdx: vacancyIdx, status: status,
				createdDaysAgo: createdDaysAgo, updatedDaysAgo: updatedDaysAgo, notify: status != "pending",
			})
		}
		applianceSpecs = append(applianceSpecs, generatedAppliances...)

		// completedWindows: (monthsAgoStart, durationMonths) — duration is
		// always < start so every completed window has already ended by
		// today, with a comfortable safety margin.
		completedWindows := [][2]int{{3, 1}, {4, 2}, {5, 2}, {5, 3}, {6, 3}, {6, 4}, {7, 3}, {7, 4}, {8, 4}, {8, 5}}

		var presenceRows []seedPresenceRow
		var journalRows []seedJournalRow
		var scoreRows []seedScoreRow
		var notificationRows []seedNotificationRow

		completedSeen, ongoingSeen, soonSeen := 0, 0, 0
		for _, spec := range applianceSpecs {
			vs := vacancySpecs[spec.vacancyIdx]
			message := fmt.Sprintf("Saya tertarik melamar posisi %s dan ingin belajar langsung dari mentor di perusahaan ini.", vs.name)
			created := now.AddDate(0, 0, -spec.createdDaysAgo)
			updated := now.AddDate(0, 0, -spec.updatedDaysAgo)
			applianceID, err := upsertAppliance(tx, ctx, studentIDs[spec.studentIdx], vacancyIDs[spec.vacancyIdx], spec.status, message, created, updated)
			if err != nil {
				return fmt.Errorf("appliance student=%d vacancy=%d: %w", spec.studentIdx, spec.vacancyIdx, err)
			}
			statusCounts[spec.status]++
			companyID := companyIDs[vs.companyIdx]
			studentID := studentIDs[spec.studentIdx]

			switch spec.internGroup {
			case "completed":
				w := completedWindows[completedSeen%len(completedWindows)]
				completedSeen++
				wantStart := now.AddDate(0, -w[0], 0)
				wantEnd := wantStart.AddDate(0, w[1], 0)
				start, end, err := upsertInternDate(tx, ctx, studentID, companyID, applianceID, wantStart, wantEnd, "completed")
				if err != nil {
					return fmt.Errorf("intern date student=%d: %w", spec.studentIdx, err)
				}
				totals.internDates++

				pres, jour := buildAttendance(studentID, companyID, statusIDs, start, end, true)
				presenceRows = append(presenceRows, pres...)
				journalRows = append(journalRows, jour...)
				totals.presences += len(pres)
				totals.journals += len(jour)

				scoreRows = append(scoreRows, buildScoreRows(studentID, companyID, spec.scoreTarget, now.AddDate(0, 0, -30))...)
				totals.scores += 8

				if err := upsertCertificate(tx, ctx, studentID, deptID, companyID, now); err != nil {
					return fmt.Errorf("certificate student=%d: %w", spec.studentIdx, err)
				}
				totals.certificates++

			case "ongoing":
				weeksAgo := 1 + ongoingSeen%10
				durMonths := 3 + ongoingSeen%4
				ongoingSeen++
				wantStart := now.AddDate(0, 0, -weeksAgo*7)
				wantEnd := wantStart.AddDate(0, durMonths, 0)
				start, _, err := upsertInternDate(tx, ctx, studentID, companyID, applianceID, wantStart, wantEnd, "scheduled")
				if err != nil {
					return fmt.Errorf("intern date student=%d: %w", spec.studentIdx, err)
				}
				totals.internDates++

				// attendance always runs up to "now" (not the placement's
				// future end date) — an ongoing internship has no
				// attendance for days that haven't happened yet.
				pres, jour := buildAttendance(studentID, companyID, statusIDs, start, now, false)
				presenceRows = append(presenceRows, pres...)
				journalRows = append(journalRows, jour...)
				totals.presences += len(pres)
				totals.journals += len(jour)

				scoreRows = append(scoreRows, buildScoreRows(studentID, companyID, spec.scoreTarget, now.AddDate(0, 0, -14))...)
				totals.scores += 8

			case "soon":
				weeksAhead := 1 + soonSeen%3
				soonSeen++
				wantStart := now.AddDate(0, 0, weeksAhead*7)
				wantEnd := wantStart.AddDate(0, 3, 0)
				if _, _, err := upsertInternDate(tx, ctx, studentID, companyID, applianceID, wantStart, wantEnd, "scheduled"); err != nil {
					return fmt.Errorf("intern date student=%d: %w", spec.studentIdx, err)
				}
				totals.internDates++
			}

			if spec.notify {
				if notifType, title, body := applianceNotification(spec.status); notifType != "" {
					notificationRows = append(notificationRows, seedNotificationRow{
						UserID: studentID, Type: notifType, Title: title, Body: body, CreatedTS: updated,
					})
				}
			}
		}
		totals.completed, totals.ongoing, totals.soon = completedSeen, ongoingSeen, soonSeen

		// a couple of approval notifications, tied to attendance that's already
		// aged past the "pending mentor review" window generated above.
		notificationRows = append(notificationRows,
			seedNotificationRow{UserID: studentIDs[5], Type: "presence_approved", Title: "Presensi disetujui", Body: "Presensi kamu minggu ini telah disetujui mentor.", CreatedTS: now.AddDate(0, 0, -5)},
			seedNotificationRow{UserID: studentIDs[6], Type: "journal_approved", Title: "Jurnal disetujui", Body: "Jurnal harian kamu telah disetujui mentor.", CreatedTS: now.AddDate(0, 0, -5)},
		)
		// same, scaled across every newly generated ongoing placement — a
		// trivial extension of the mechanism above, not new infrastructure.
		for _, p := range placements {
			if p.group != "ongoing" {
				continue
			}
			notificationRows = append(notificationRows,
				seedNotificationRow{UserID: studentIDs[p.studentIdx], Type: "presence_approved", Title: "Presensi disetujui", Body: "Presensi kamu minggu ini telah disetujui mentor.", CreatedTS: now.AddDate(0, 0, -5)},
				seedNotificationRow{UserID: studentIDs[p.studentIdx], Type: "journal_approved", Title: "Jurnal disetujui", Body: "Jurnal harian kamu telah disetujui mentor.", CreatedTS: now.AddDate(0, 0, -5)},
			)
		}

		if err := bulkInsertPresences(tx, ctx, presenceRows); err != nil {
			return fmt.Errorf("bulk presences: %w", err)
		}
		if err := bulkInsertJournals(tx, ctx, journalRows); err != nil {
			return fmt.Errorf("bulk journals: %w", err)
		}
		if _, err := bulkInsertScores(tx, ctx, scoreRows); err != nil {
			return fmt.Errorf("bulk scores: %w", err)
		}
		insertedNotifs, err := bulkInsertNotifications(tx, ctx, notificationRows)
		if err != nil {
			return fmt.Errorf("bulk notifications: %w", err)
		}
		totals.notifications = insertedNotifs

		for _, n := range newsSpecs {
			createdAt := now.AddDate(0, 0, -n.createdDaysAgo)
			if err := upsertNews(tx, ctx, coordinatorID, schoolID, n.title, n.content, n.status, createdAt); err != nil {
				return fmt.Errorf("news %q: %w", n.title, err)
			}
		}

		for i, f := range faqSpecs {
			if err := upsertFAQ(tx, ctx, f.question, f.answer, i+1); err != nil {
				return fmt.Errorf("faq %q: %w", f.question, err)
			}
		}

		questionIDs := make([]int64, len(questionSpecs))
		for i, q := range questionSpecs {
			id, err := upsertQuestion(tx, ctx, schoolID, q.question, q.sortOrder)
			if err != nil {
				return fmt.Errorf("question %q: %w", q.question, err)
			}
			questionIDs[i] = id
		}

		// scale monitors/reviews by sampling the generated placement pool —
		// every "soon" placement is excluded (nothing to monitor or review
		// before the internship has even started). monitorNotes/
		// monitorSuggests/mentorReviewBodies/companyReviewBodies are
		// package-level vars (below) so seedSchool can reuse the exact same
		// text pools for the other two schools instead of duplicating them.
		mi := 0
		for i, p := range placements {
			if p.group == "soon" || i%4 != 1 {
				continue
			}
			monitorSpecs = append(monitorSpecs, monitorSpec{
				studentIdx: p.studentIdx, companyIdx: p.companyIdx, daysAgo: 5 + i%60,
				matchRating: 2 + i%3, notes: monitorNotes[mi%len(monitorNotes)], suggest: monitorSuggests[mi%len(monitorSuggests)],
			})
			mi++
		}

		gi := 0
		for i, p := range placements {
			if p.group == "soon" || i%4 != 0 {
				continue
			}
			mentorReviewSpecs = append(mentorReviewSpecs, mentorReviewSpec{
				studentIdx: p.studentIdx, companyIdx: p.companyIdx, questionIdx: 3,
				rating: 3 + i%3, title: "Evaluasi Kinerja Magang", body: mentorReviewBodies[gi%len(mentorReviewBodies)],
			})
			gi++
		}

		ci := 0
		for i, p := range placements {
			if p.group == "soon" || i%5 != 2 {
				continue
			}
			companyReviewSpecs = append(companyReviewSpecs, companyReviewSpec{
				studentIdx: p.studentIdx, companyIdx: p.companyIdx, rating: 3 + i%3,
				title: fmt.Sprintf("Pengalaman PKL di %s", companySpecs[p.companyIdx].name), body: companyReviewBodies[ci%len(companyReviewBodies)],
			})
			ci++
		}

		for _, m := range monitorSpecs {
			date := dateOnly(now.AddDate(0, 0, -m.daysAgo))
			if err := upsertMonitor(tx, ctx, coordinatorID, studentIDs[m.studentIdx], companyIDs[m.companyIdx], date, m.notes, m.suggest, m.matchRating); err != nil {
				return fmt.Errorf("monitor student=%d: %w", m.studentIdx, err)
			}
		}

		for _, r := range mentorReviewSpecs {
			studentID := studentIDs[r.studentIdx]
			questionID := questionIDs[r.questionIdx]
			if err := upsertReview(tx, ctx, mentorIDs[r.companyIdx], &questionID, &studentID, nil, r.title, r.body, r.rating); err != nil {
				return fmt.Errorf("mentor review student=%d: %w", r.studentIdx, err)
			}
			totals.reviews++
		}
		for _, r := range companyReviewSpecs {
			companyID := companyIDs[r.companyIdx]
			if err := upsertReview(tx, ctx, studentIDs[r.studentIdx], nil, nil, &companyID, r.title, r.body, r.rating); err != nil {
				return fmt.Errorf("company review student=%d: %w", r.studentIdx, err)
			}
			totals.reviews++
		}

		school2Totals, err = seedSchool(tx, ctx, passwordHash, now, existingEmails, bogorConfig)
		if err != nil {
			return fmt.Errorf("school %s: %w", bogorConfig.schoolName, err)
		}
		school3Totals, err = seedSchool(tx, ctx, passwordHash, now, existingEmails, depokConfig)
		if err != nil {
			return fmt.Errorf("school %s: %w", depokConfig.schoolName, err)
		}

		return nil
	})
	if err != nil {
		fail(err)
	}

	closedVacancies := 0
	for _, v := range vacancySpecs {
		if v.status == "closed" {
			closedVacancies++
		}
	}
	totalStudents := 2 + len(newStudentNames)
	scoredStudents := totals.completed + totals.ongoing

	fmt.Println("Seed complete. 3 schools now demonstrate real multi-tenant isolation:")
	fmt.Println()
	fmt.Println("=== SMKN 1 Cibinong (Rekayasa Perangkat Lunak) — password 'password123' ===")
	fmt.Println("  admin@internity.test        (admin, cross-school)")
	fmt.Println("  coordinator@internity.test  (coordinator)")
	fmt.Printf("  mentor1..%d@internity.test  (%d mentors, 1 per company)\n", len(mentorSpecs), len(mentorSpecs))
	fmt.Println("  budi@internity.test, siti@internity.test, ...")
	fmt.Printf("  ...and %d more student accounts (first-name@internity.test, e.g. ahmad@internity.test)\n", totalStudents-2)
	fmt.Println("  invite codes: RPL1DEMO, RPL2DEMO, RPL3DEMO")
	fmt.Printf("  %d companies, %d students, %d vacancies (%d closed), %d appliances (%d accepted/%d pending/%d processed/%d rejected/%d canceled)\n",
		len(companySpecs), totalStudents, len(vacancySpecs), closedVacancies, len(applianceSpecs),
		statusCounts["accepted"], statusCounts["pending"], statusCounts["processed"], statusCounts["rejected"], statusCounts["canceled"])
	fmt.Printf("  %d intern placements (%d completed, %d ongoing, %d starting soon), %d presences, %d journals, %d score items, %d certificates\n",
		totals.internDates, totals.completed, totals.ongoing, totals.soon, totals.presences, totals.journals, totals.scores, totals.certificates)
	fmt.Printf("  %d news, %d monitoring visits, %d review questions, %d reviews, %d notifications (scored students: %d)\n",
		len(newsSpecs), len(monitorSpecs), len(questionSpecs), totals.reviews, totals.notifications, scoredStudents)

	printSchoolSummary(school2Totals, bogorConfig)
	printSchoolSummary(school3Totals, depokConfig)

	fmt.Println()
	fmt.Printf("%d FAQs shared across all schools (faqs table has no school_id — see migration 000022)\n", len(faqSpecs))

	grandStudents := totalStudents + school2Totals.students + school3Totals.students
	grandMentors := len(mentorSpecs) + school2Totals.mentors + school3Totals.mentors
	grandUsers := grandStudents + grandMentors + 3 /* coordinators */ + 1 /* admin */
	grandCompanies := len(companySpecs) + school2Totals.companies + school3Totals.companies
	grandVacancies := len(vacancySpecs) + school2Totals.vacancies + school3Totals.vacancies
	grandAppliances := len(applianceSpecs) + school2Totals.appliances + school3Totals.appliances
	grandPresences := totals.presences + school2Totals.presences + school3Totals.presences
	grandJournals := totals.journals + school2Totals.journals + school3Totals.journals
	grandScores := totals.scores + school2Totals.scores + school3Totals.scores
	grandCertificates := totals.certificates + school2Totals.certificates + school3Totals.certificates

	fmt.Println()
	fmt.Println("=== TOTAL across all 3 schools ===")
	fmt.Printf("  %d users (%d students, %d mentors, 3 coordinators, 1 admin)\n", grandUsers, grandStudents, grandMentors)
	fmt.Printf("  %d companies, %d vacancies, %d appliances\n", grandCompanies, grandVacancies, grandAppliances)
	fmt.Printf("  %d presences, %d journals, %d scores, %d certificates\n", grandPresences, grandJournals, grandScores, grandCertificates)
}

func printSchoolSummary(t schoolSeedTotals, cfg schoolSeedConfig) {
	fmt.Println()
	fmt.Printf("=== %s (%s) — password 'password123' ===\n", t.schoolName, cfg.deptName)
	fmt.Printf("  %-38s (coordinator)\n", cfg.coordinatorEmail)
	fmt.Printf("  %s1..%d@internity.test  (%d mentors, 1 per company)\n", cfg.mentorEmailPrefix, t.mentors, t.mentors)
	fmt.Printf("  %d students (firstname.lastname@internity.test, e.g. %s.%s@internity.test)\n",
		t.students, strings.ToLower(strings.SplitN(cfg.students[0], " ", 2)[0]), strings.ToLower(strings.ReplaceAll(strings.SplitN(cfg.students[0], " ", 2)[1], " ", "")))
	fmt.Printf("  NIS prefix %s (school-coded, distinct from SMKN 1 Cibinong's 2024001xxx)\n", cfg.nisPrefix)
	fmt.Printf("  invite codes: %s\n", strings.Join(cfg.inviteCodes, ", "))
	fmt.Printf("  %d companies, %d vacancies (%d closed), %d appliances\n", t.companies, t.vacancies, t.closedVacancies, t.appliances)
	fmt.Printf("  %d intern placements (%d completed, %d ongoing, %d starting soon), %d presences, %d journals, %d score items, %d certificates\n",
		t.internDates, t.completed, t.ongoing, t.soon, t.presences, t.journals, t.scores, t.certificates)
	fmt.Printf("  %d news, %d monitoring visits, %d review questions, %d reviews, %d notifications\n",
		t.news, t.monitors, t.questions, t.reviews, t.notifications)
}

// seedSchool builds one full second/third-tenant school (department, courses,
// companies, coordinator, mentors, students, vacancies, invite codes,
// appliances/placements with presence/journal/score/certificate history,
// news, review questions, monitors, reviews) from a schoolSeedConfig
// blueprint. It's the exact same steps main() runs inline for SMKN 1
// Cibinong, against the exact same upsert*/bulk* helpers, just parameterized
// so the school-1 block above doesn't get copy-pasted twice more with
// different literals. existingEmails is shared (and mutated) across all
// three schools' seeding so NIS allocation always reflects what's actually
// about to be inserted, matching the same reasoning as main()'s own
// existingEmails use.
func seedSchool(tx *gorm.DB, ctx context.Context, passwordHash string, now time.Time, existingEmails map[string]bool, cfg schoolSeedConfig) (schoolSeedTotals, error) {
	var totals schoolSeedTotals
	totals.schoolName = cfg.schoolName

	schoolID, err := upsertSchoolNamed(tx, ctx, cfg.schoolName, cfg.schoolEmail, cfg.schoolPhone, cfg.schoolAddress)
	if err != nil {
		return totals, fmt.Errorf("school: %w", err)
	}
	deptID, err := upsertDepartmentNamed(tx, ctx, schoolID, cfg.deptName, cfg.deptDescription, cfg.studyProgram)
	if err != nil {
		return totals, fmt.Errorf("department: %w", err)
	}
	courseIDs := make([]int64, len(cfg.courseNames))
	for i, name := range cfg.courseNames {
		id, err := upsertCourse(tx, ctx, deptID, name)
		if err != nil {
			return totals, fmt.Errorf("course %s: %w", name, err)
		}
		courseIDs[i] = id
	}

	companyIDs := make([]int64, len(cfg.companies))
	for i, c := range cfg.companies {
		id, err := upsertCompany(tx, ctx, deptID, c.name, c.category, c.city)
		if err != nil {
			return totals, fmt.Errorf("company %s: %w", c.name, err)
		}
		companyIDs[i] = id
	}
	totals.companies = len(cfg.companies)

	if err := upsertUser(tx, ctx, passwordHash, cfg.coordinatorEmail, cfg.coordinatorName, "coordinator", &schoolID, nil, nil, nil, nil); err != nil {
		return totals, fmt.Errorf("user %s: %w", cfg.coordinatorEmail, err)
	}
	coordinatorID, err := getUserID(tx, ctx, cfg.coordinatorEmail)
	if err != nil {
		return totals, fmt.Errorf("coordinator id: %w", err)
	}

	mentorIDs := make([]string, len(cfg.companies))
	for i := range cfg.companies {
		email := fmt.Sprintf("%s%d@internity.test", cfg.mentorEmailPrefix, i+1)
		if err := upsertUser(tx, ctx, passwordHash, email, cfg.mentorNames[i], "mentor", nil, nil, &companyIDs[i], nil, nil); err != nil {
			return totals, fmt.Errorf("user %s: %w", email, err)
		}
		id, err := getUserID(tx, ctx, email)
		if err != nil {
			return totals, fmt.Errorf("mentor id %s: %w", email, err)
		}
		mentorIDs[i] = id
	}
	totals.mentors = len(cfg.companies)

	// per-school NIS counter (not the shared/global one main() uses for
	// Cibinong) — nisPrefix already encodes the school code, so scanning
	// only that prefix keeps each school's sequence independent even though
	// the nis column itself is one flat UNIQUE namespace (migration 000006).
	baseNIS, err := strconv.ParseInt(cfg.nisPrefix+"000", 10, 64)
	if err != nil {
		return totals, fmt.Errorf("nis prefix %s: %w", cfg.nisPrefix, err)
	}
	var maxNIS int64
	if err := tx.WithContext(ctx).Raw(`SELECT COALESCE(MAX(nis::bigint), ?) FROM users WHERE nis LIKE ?`, baseNIS, cfg.nisPrefix+"%").Scan(&maxNIS).Error; err != nil {
		return totals, fmt.Errorf("max nis: %w", err)
	}
	nextNIS := maxNIS + 1

	students := make([]struct {
		email, name, nis string
		courseID         int64
	}, len(cfg.students))
	for j, name := range cfg.students {
		parts := strings.SplitN(name, " ", 2)
		// first.last@ (not the school-1 first-name-only convention) —
		// guarantees no collision with the ~150 first names already used by
		// SMKN 1 Cibinong's students without having to cross-check every name.
		first := strings.ToLower(parts[0])
		last := first
		if len(parts) > 1 {
			last = strings.ToLower(strings.ReplaceAll(parts[1], " ", ""))
		}
		email := fmt.Sprintf("%s.%s@internity.test", first, last)
		var nis string
		if !existingEmails[email] {
			nis = fmt.Sprintf("%010d", nextNIS)
			nextNIS++
			existingEmails[email] = true
		}
		students[j] = struct {
			email, name, nis string
			courseID         int64
		}{email: email, name: name, nis: nis, courseID: courseIDs[j%len(courseIDs)]}
	}

	studentIDs := make([]string, len(students))
	for i, s := range students {
		if err := upsertUser(tx, ctx, passwordHash, s.email, s.name, "student", &schoolID, &deptID, nil, &s.courseID, &s.nis); err != nil {
			return totals, fmt.Errorf("user %s: %w", s.email, err)
		}
		id, err := getUserID(tx, ctx, s.email)
		if err != nil {
			return totals, fmt.Errorf("student id %s: %w", s.email, err)
		}
		studentIDs[i] = id
	}
	totals.students = len(students)

	statuses := []struct{ name, kind string }{
		{"Hadir", "present"}, {"Izin", "permitted"}, {"Sakit", "sick"}, {"Alpa", "absent"}, {"Libur", "holiday"},
	}
	for _, s := range statuses {
		if err := upsertPresenceStatus(tx, ctx, schoolID, s.name, s.kind); err != nil {
			return totals, fmt.Errorf("presence status %s: %w", s.name, err)
		}
	}
	statusIDs := map[string]int64{}
	for _, kind := range []string{"present", "permitted", "sick", "absent"} {
		var id int64
		if err := tx.WithContext(ctx).Raw(`SELECT id FROM presence_statuses WHERE school_id = ? AND kind = ?`, schoolID, kind).Scan(&id).Error; err != nil {
			return totals, fmt.Errorf("presence status id %s: %w", kind, err)
		}
		statusIDs[kind] = id
	}

	predicates := []struct {
		name     string
		min, max float64
	}{
		{"D", 0, 59.99}, {"C", 60, 74.99}, {"B", 75, 89.99}, {"A", 90, 100},
	}
	for _, p := range predicates {
		if err := upsertScorePredicate(tx, ctx, schoolID, p.name, p.min, p.max); err != nil {
			return totals, fmt.Errorf("score predicate %s: %w", p.name, err)
		}
	}

	vacancyIDs := make([]int64, len(cfg.vacancies))
	for i, v := range cfg.vacancies {
		id, err := upsertVacancy(tx, ctx, companyIDs[v.companyIdx], v.name, v.category, v.skills, v.slots, v.status)
		if err != nil {
			return totals, fmt.Errorf("vacancy %s: %w", v.name, err)
		}
		vacancyIDs[i] = id
		if v.status == "closed" {
			totals.closedVacancies++
		}
	}
	totals.vacancies = len(cfg.vacancies)

	for i, code := range cfg.inviteCodes {
		if err := upsertInviteCode(tx, ctx, courseIDs[i], code); err != nil {
			return totals, fmt.Errorf("invite code %s: %w", code, err)
		}
	}

	// --- generated placements/appliances — same technique as the "generated
	// bulk data" section in main(): one fresh student per accepted placement
	// (so intern_dates' UNIQUE(user_id, company_id) and no-overlap EXCLUDE
	// constraint are trivially satisfied), then non-accepted appliances
	// layered on top via usedPair so no (student, vacancy) pair repeats.
	type genPlacement struct {
		studentIdx, vacancyIdx, companyIdx int
		group                              string
	}
	var placements []genPlacement
	pIdx := 0
	addPlacements := func(group string, count int) {
		for k := 0; k < count; k++ {
			vacancyIdx := pIdx % len(cfg.vacancies)
			placements = append(placements, genPlacement{
				studentIdx: pIdx, vacancyIdx: vacancyIdx,
				companyIdx: cfg.vacancies[vacancyIdx].companyIdx, group: group,
			})
			pIdx++
		}
	}
	addPlacements("completed", cfg.placementCounts[0])
	addPlacements("ongoing", cfg.placementCounts[1])
	addPlacements("soon", cfg.placementCounts[2])

	var applianceSpecs []applianceSpec
	for i, p := range placements {
		var createdDaysAgo, updatedDaysAgo int
		switch p.group {
		case "completed":
			createdDaysAgo = 190 + i%40
			updatedDaysAgo = createdDaysAgo - 5
		case "ongoing":
			createdDaysAgo = 50 + i%60
			updatedDaysAgo = createdDaysAgo - 3
		case "soon":
			createdDaysAgo = 10 + i%10
			updatedDaysAgo = createdDaysAgo - 3
		}
		applianceSpecs = append(applianceSpecs, applianceSpec{
			studentIdx: p.studentIdx, vacancyIdx: p.vacancyIdx, status: "accepted",
			createdDaysAgo: createdDaysAgo, updatedDaysAgo: updatedDaysAgo,
			internGroup: p.group, scoreTarget: 50 + (i*7)%50, notify: true,
		})
	}

	usedPair := map[[2]int]bool{}
	for _, p := range placements {
		usedPair[[2]int{p.studentIdx, p.vacancyIdx}] = true
	}

	nonAcceptedTargets := []struct {
		status string
		count  int
	}{
		{"rejected", cfg.nonAcceptedCounts[0]}, {"canceled", cfg.nonAcceptedCounts[1]},
		{"pending", cfg.nonAcceptedCounts[2]}, {"processed", cfg.nonAcceptedCounts[3]},
	}
	maxCount := 0
	for _, t := range nonAcceptedTargets {
		if t.count > maxCount {
			maxCount = t.count
		}
	}
	var order []string
	for i := 0; i < maxCount; i++ {
		for _, t := range nonAcceptedTargets {
			if i < t.count {
				order = append(order, t.status)
			}
		}
	}

	statusSeen := map[string]int{}
	for j, status := range order {
		n := statusSeen[status]
		statusSeen[status] = n + 1
		var createdDaysAgo, updatedDaysAgo int
		switch status {
		case "rejected":
			createdDaysAgo = 30 + n%150
			updatedDaysAgo = createdDaysAgo - (5 + n%10)
		case "canceled":
			createdDaysAgo = 10 + n%50
			updatedDaysAgo = createdDaysAgo - (2 + n%6)
		case "pending":
			createdDaysAgo = 1 + n%14
			updatedDaysAgo = createdDaysAgo
		case "processed":
			createdDaysAgo = 5 + n%25
			updatedDaysAgo = createdDaysAgo - (1 + n%4)
		}
		studentIdx := j % len(cfg.students)
		vacancyIdx := (j*13 + 5) % len(cfg.vacancies)
		for usedPair[[2]int{studentIdx, vacancyIdx}] {
			vacancyIdx = (vacancyIdx + 1) % len(cfg.vacancies)
		}
		usedPair[[2]int{studentIdx, vacancyIdx}] = true
		applianceSpecs = append(applianceSpecs, applianceSpec{
			studentIdx: studentIdx, vacancyIdx: vacancyIdx, status: status,
			createdDaysAgo: createdDaysAgo, updatedDaysAgo: updatedDaysAgo, notify: status != "pending",
		})
	}
	totals.appliances = len(applianceSpecs)

	completedWindows := [][2]int{{3, 1}, {4, 2}, {5, 2}, {5, 3}, {6, 3}, {6, 4}, {7, 3}, {7, 4}, {8, 4}, {8, 5}}

	var presenceRows []seedPresenceRow
	var journalRows []seedJournalRow
	var scoreRows []seedScoreRow
	var notificationRows []seedNotificationRow

	completedSeen, ongoingSeen, soonSeen := 0, 0, 0
	for _, spec := range applianceSpecs {
		vs := cfg.vacancies[spec.vacancyIdx]
		message := fmt.Sprintf("Saya tertarik melamar posisi %s dan ingin belajar langsung dari mentor di perusahaan ini.", vs.name)
		created := now.AddDate(0, 0, -spec.createdDaysAgo)
		updated := now.AddDate(0, 0, -spec.updatedDaysAgo)
		applianceID, err := upsertAppliance(tx, ctx, studentIDs[spec.studentIdx], vacancyIDs[spec.vacancyIdx], spec.status, message, created, updated)
		if err != nil {
			return totals, fmt.Errorf("appliance student=%d vacancy=%d: %w", spec.studentIdx, spec.vacancyIdx, err)
		}
		companyID := companyIDs[vs.companyIdx]
		studentID := studentIDs[spec.studentIdx]

		switch spec.internGroup {
		case "completed":
			w := completedWindows[completedSeen%len(completedWindows)]
			completedSeen++
			wantStart := now.AddDate(0, -w[0], 0)
			wantEnd := wantStart.AddDate(0, w[1], 0)
			start, end, err := upsertInternDate(tx, ctx, studentID, companyID, applianceID, wantStart, wantEnd, "completed")
			if err != nil {
				return totals, fmt.Errorf("intern date student=%d: %w", spec.studentIdx, err)
			}
			totals.internDates++

			pres, jour := buildAttendance(studentID, companyID, statusIDs, start, end, true)
			presenceRows = append(presenceRows, pres...)
			journalRows = append(journalRows, jour...)
			totals.presences += len(pres)
			totals.journals += len(jour)

			scoreRows = append(scoreRows, buildScoreRows(studentID, companyID, spec.scoreTarget, now.AddDate(0, 0, -30))...)
			totals.scores += 8

			if err := upsertCertificate(tx, ctx, studentID, deptID, companyID, now); err != nil {
				return totals, fmt.Errorf("certificate student=%d: %w", spec.studentIdx, err)
			}
			totals.certificates++

		case "ongoing":
			weeksAgo := 1 + ongoingSeen%10
			durMonths := 3 + ongoingSeen%4
			ongoingSeen++
			wantStart := now.AddDate(0, 0, -weeksAgo*7)
			wantEnd := wantStart.AddDate(0, durMonths, 0)
			start, _, err := upsertInternDate(tx, ctx, studentID, companyID, applianceID, wantStart, wantEnd, "scheduled")
			if err != nil {
				return totals, fmt.Errorf("intern date student=%d: %w", spec.studentIdx, err)
			}
			totals.internDates++

			pres, jour := buildAttendance(studentID, companyID, statusIDs, start, now, false)
			presenceRows = append(presenceRows, pres...)
			journalRows = append(journalRows, jour...)
			totals.presences += len(pres)
			totals.journals += len(jour)

			scoreRows = append(scoreRows, buildScoreRows(studentID, companyID, spec.scoreTarget, now.AddDate(0, 0, -14))...)
			totals.scores += 8

		case "soon":
			weeksAhead := 1 + soonSeen%3
			soonSeen++
			wantStart := now.AddDate(0, 0, weeksAhead*7)
			wantEnd := wantStart.AddDate(0, 3, 0)
			if _, _, err := upsertInternDate(tx, ctx, studentID, companyID, applianceID, wantStart, wantEnd, "scheduled"); err != nil {
				return totals, fmt.Errorf("intern date student=%d: %w", spec.studentIdx, err)
			}
			totals.internDates++
		}

		if spec.notify {
			if notifType, title, body := applianceNotification(spec.status); notifType != "" {
				notificationRows = append(notificationRows, seedNotificationRow{
					UserID: studentID, Type: notifType, Title: title, Body: body, CreatedTS: updated,
				})
			}
		}
	}
	totals.completed, totals.ongoing, totals.soon = completedSeen, ongoingSeen, soonSeen

	for _, p := range placements {
		if p.group != "ongoing" {
			continue
		}
		notificationRows = append(notificationRows,
			seedNotificationRow{UserID: studentIDs[p.studentIdx], Type: "presence_approved", Title: "Presensi disetujui", Body: "Presensi kamu minggu ini telah disetujui mentor.", CreatedTS: now.AddDate(0, 0, -5)},
			seedNotificationRow{UserID: studentIDs[p.studentIdx], Type: "journal_approved", Title: "Jurnal disetujui", Body: "Jurnal harian kamu telah disetujui mentor.", CreatedTS: now.AddDate(0, 0, -5)},
		)
	}

	if err := bulkInsertPresences(tx, ctx, presenceRows); err != nil {
		return totals, fmt.Errorf("bulk presences: %w", err)
	}
	if err := bulkInsertJournals(tx, ctx, journalRows); err != nil {
		return totals, fmt.Errorf("bulk journals: %w", err)
	}
	if _, err := bulkInsertScores(tx, ctx, scoreRows); err != nil {
		return totals, fmt.Errorf("bulk scores: %w", err)
	}
	insertedNotifs, err := bulkInsertNotifications(tx, ctx, notificationRows)
	if err != nil {
		return totals, fmt.Errorf("bulk notifications: %w", err)
	}
	totals.notifications = insertedNotifs

	for _, n := range cfg.news {
		createdAt := now.AddDate(0, 0, -n.createdDaysAgo)
		if err := upsertNews(tx, ctx, coordinatorID, schoolID, n.title, n.content, n.status, createdAt); err != nil {
			return totals, fmt.Errorf("news %q: %w", n.title, err)
		}
	}
	totals.news = len(cfg.news)

	// same 5 review-question texts as SMKN 1 Cibinong's questionSpecs in
	// main(), just inserted under this school's own schoolID — questions is
	// UNIQUE(school_id, question) (see upsertQuestion), so identical text is
	// fine across schools.
	reviewQuestionSpecs := []struct {
		question  string
		sortOrder int
	}{
		{"Apakah lingkungan kerja di tempat PKL sesuai dengan bidang keahlian siswa?", 1},
		{"Apakah pembimbing industri memberikan bimbingan yang cukup kepada siswa?", 2},
		{"Apakah fasilitas kerja memadai untuk mendukung kegiatan PKL siswa?", 3},
		{"Apakah siswa mendapatkan pekerjaan yang relevan dengan kompetensi keahliannya?", 4},
		{"Apakah perusahaan bersedia menerima siswa PKL pada periode berikutnya?", 5},
	}
	questionIDs := make([]int64, len(reviewQuestionSpecs))
	for i, q := range reviewQuestionSpecs {
		id, err := upsertQuestion(tx, ctx, schoolID, q.question, q.sortOrder)
		if err != nil {
			return totals, fmt.Errorf("question %q: %w", q.question, err)
		}
		questionIDs[i] = id
	}
	totals.questions = len(reviewQuestionSpecs)

	mi := 0
	for i, p := range placements {
		if p.group == "soon" || i%4 != 1 {
			continue
		}
		date := dateOnly(now.AddDate(0, 0, -(5 + i%60)))
		if err := upsertMonitor(tx, ctx, coordinatorID, studentIDs[p.studentIdx], companyIDs[p.companyIdx], date, monitorNotes[mi%len(monitorNotes)], monitorSuggests[mi%len(monitorSuggests)], 2+i%3); err != nil {
			return totals, fmt.Errorf("monitor student=%d: %w", p.studentIdx, err)
		}
		mi++
		totals.monitors++
	}

	gi := 0
	for i, p := range placements {
		if p.group == "soon" || i%4 != 0 {
			continue
		}
		questionID := questionIDs[3]
		studentID := studentIDs[p.studentIdx]
		if err := upsertReview(tx, ctx, mentorIDs[p.companyIdx], &questionID, &studentID, nil, "Evaluasi Kinerja Magang", mentorReviewBodies[gi%len(mentorReviewBodies)], 3+i%3); err != nil {
			return totals, fmt.Errorf("mentor review student=%d: %w", p.studentIdx, err)
		}
		totals.reviews++
		gi++
	}

	ci := 0
	for i, p := range placements {
		if p.group == "soon" || i%5 != 2 {
			continue
		}
		companyID := companyIDs[p.companyIdx]
		title := fmt.Sprintf("Pengalaman PKL di %s", cfg.companies[p.companyIdx].name)
		if err := upsertReview(tx, ctx, studentIDs[p.studentIdx], nil, nil, &companyID, title, companyReviewBodies[ci%len(companyReviewBodies)], 3+i%3); err != nil {
			return totals, fmt.Errorf("company review student=%d: %w", p.studentIdx, err)
		}
		totals.reviews++
		ci++
	}

	return totals, nil
}

// schools.name has no unique constraint (only department/course/company
// names are unique per-parent), so — like every other upsert* helper here —
// this is a plain lookup-or-insert, not a real ON CONFLICT upsert. A failed
// statement would abort the rest of this function's transaction, so unlike
// the others there's no fallback path to reach for; the lookup always runs first.
func upsertSchool(tx *gorm.DB, ctx context.Context) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM schools WHERE name = ?`, "SMKN 1 Cibinong").Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO schools (name, email, phone, address, is_active)
		VALUES ('SMKN 1 Cibinong', 'info@smkn1cibinong.sch.id', '021-8752XXX', 'Cibinong, Bogor, Jawa Barat', true)
		RETURNING id
	`).Scan(&id).Error
	return id, err
}

func upsertDepartment(tx *gorm.DB, ctx context.Context, schoolID int64) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM departments WHERE school_id = ? AND name = ?`, schoolID, "Rekayasa Perangkat Lunak").Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO departments (school_id, name, description, study_program, is_active)
		VALUES (?, 'Rekayasa Perangkat Lunak', 'Jurusan Rekayasa Perangkat Lunak', 'RPL', true)
		RETURNING id
	`, schoolID).Scan(&id).Error
	return id, err
}

// upsertSchoolNamed/upsertDepartmentNamed are the parameterized siblings of
// upsertSchool/upsertDepartment above, used by seedSchool for the second and
// third schools — upsertSchool/upsertDepartment stay hardcoded to SMKN 1
// Cibinong/RPL so that function's behavior (and its already-seeded rows)
// don't shift.
func upsertSchoolNamed(tx *gorm.DB, ctx context.Context, name, email, phone, address string) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM schools WHERE name = ?`, name).Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO schools (name, email, phone, address, is_active)
		VALUES (?, ?, ?, ?, true)
		RETURNING id
	`, name, email, phone, address).Scan(&id).Error
	return id, err
}

func upsertDepartmentNamed(tx *gorm.DB, ctx context.Context, schoolID int64, name, description, studyProgram string) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM departments WHERE school_id = ? AND name = ?`, schoolID, name).Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO departments (school_id, name, description, study_program, is_active)
		VALUES (?, ?, ?, ?, true)
		RETURNING id
	`, schoolID, name, description, studyProgram).Scan(&id).Error
	return id, err
}

func upsertCourse(tx *gorm.DB, ctx context.Context, deptID int64, name string) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM courses WHERE department_id = ? AND name = ?`, deptID, name).Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO courses (department_id, name, is_active) VALUES (?, ?, true) RETURNING id
	`, deptID, name).Scan(&id).Error
	return id, err
}

func upsertCompany(tx *gorm.DB, ctx context.Context, deptID int64, name, category, city string) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM companies WHERE department_id = ? AND name = ?`, deptID, name).Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO companies (department_id, name, category, city, is_active)
		VALUES (?, ?, ?, ?, true) RETURNING id
	`, deptID, name, category, city).Scan(&id).Error
	return id, err
}

func upsertUser(tx *gorm.DB, ctx context.Context, passwordHash, email, name, role string, schoolID, deptID, companyID, courseID *int64, nis *string) error {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM users WHERE email = ?`, email).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.WithContext(ctx).Exec(`
		INSERT INTO users (role, school_id, department_id, company_id, course_id, name, email, password_hash, nis, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, true)
	`, role, schoolID, deptID, companyID, courseID, name, email, passwordHash, nis).Error
}

func getUserID(tx *gorm.DB, ctx context.Context, email string) (string, error) {
	var id string
	err := tx.WithContext(ctx).Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&id).Error
	return id, err
}

func upsertPresenceStatus(tx *gorm.DB, ctx context.Context, schoolID int64, name, kind string) error {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM presence_statuses WHERE school_id = ? AND kind = ?`, schoolID, kind).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	color := map[string]string{"present": "#0fb782", "permitted": "#2e63f5", "sick": "#e9b207", "absent": "#f03e61", "holiday": "#717a8f"}[kind]
	return tx.WithContext(ctx).Exec(`
		INSERT INTO presence_statuses (school_id, name, kind, color, is_active) VALUES (?, ?, ?, ?, true)
	`, schoolID, name, kind, color).Error
}

func upsertScorePredicate(tx *gorm.DB, ctx context.Context, schoolID int64, name string, min, max float64) error {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM score_predicates WHERE school_id = ? AND name = ?`, schoolID, name).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.WithContext(ctx).Exec(`
		INSERT INTO score_predicates (school_id, name, min, max) VALUES (?, ?, ?, ?)
	`, schoolID, name, min, max).Error
}

func upsertVacancy(tx *gorm.DB, ctx context.Context, companyID int64, name, category, skills string, slots int, status string) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM vacancies WHERE company_id = ? AND name = ?`, companyID, name).Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO vacancies (company_id, name, category, skills, slots, status)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id
	`, companyID, name, category, skills, slots, status).Scan(&id).Error
	return id, err
}

func upsertInviteCode(tx *gorm.DB, ctx context.Context, courseID int64, code string) error {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM invite_codes WHERE code = ?`, code).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.WithContext(ctx).Exec(`INSERT INTO invite_codes (code, course_id) VALUES (?, ?)`, code, courseID).Error
}

// upsertAppliance dedupes on (user_id, vacancy_id) — this seed never creates
// two appliance rows for the same pair, so the partial unique index on
// active statuses (uq_appliances_active_per_user_vacancy) can never be hit.
func upsertAppliance(tx *gorm.DB, ctx context.Context, userID string, vacancyID int64, status, message string, createdAt, updatedAt time.Time) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM appliances WHERE user_id = ? AND vacancy_id = ?`, userID, vacancyID).Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO appliances (user_id, vacancy_id, status, message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id
	`, userID, vacancyID, status, message, createdAt, updatedAt).Scan(&id).Error
	return id, err
}

func applianceNotification(status string) (notifType, title, body string) {
	switch status {
	case "accepted":
		return "appliance_accepted", "Application accepted", "Your application has been accepted. Set your internship start/end date to continue."
	case "processed":
		return "appliance_processed", "Application under review", "Application under review"
	case "rejected":
		return "appliance_rejected", "Application rejected", "Application rejected"
	case "canceled":
		return "appliance_canceled", "Application canceled", "Application canceled"
	default:
		return "", "", ""
	}
}

// upsertInternDate dedupes on appliance_id (UNIQUE in the schema). Every
// caller here gives a student at most one intern_date row, so the
// UNIQUE(user_id, company_id) constraint and the no-overlap EXCLUDE
// constraint are trivially satisfied — a single row can't overlap itself.
// upsertInternDate returns the row's real persisted start/end — on a
// no-op (row already exists from an earlier run) that's whatever was
// committed back then, not the freshly-computed start/end passed in here.
// Callers use the returned dates (not their own locals) to build
// presence/journal/score rows, so attendance history always matches the
// placement actually on record instead of silently drifting across runs.
func upsertInternDate(tx *gorm.DB, ctx context.Context, userID string, companyID, applianceID int64, start, end time.Time, status string) (time.Time, time.Time, error) {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM intern_dates WHERE appliance_id = ?`, applianceID).Scan(&count).Error; err != nil {
		return time.Time{}, time.Time{}, err
	}
	if count == 0 {
		if err := tx.WithContext(ctx).Exec(`
			INSERT INTO intern_dates (user_id, company_id, appliance_id, start_date, end_date, status)
			VALUES (?, ?, ?, ?, ?, ?)
		`, userID, companyID, applianceID, dateOnly(start), dateOnly(end), status).Error; err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	var realStart, realEnd time.Time
	if err := tx.WithContext(ctx).Raw(`SELECT start_date, end_date FROM intern_dates WHERE appliance_id = ?`, applianceID).Row().Scan(&realStart, &realEnd); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return realStart, realEnd, nil
}

var attendanceDescs = []string{
	"Mengikuti briefing pagi bersama tim dan mempelajari SOP kerja.",
	"Membantu mentor menyelesaikan tugas harian yang diberikan.",
	"Belajar menggunakan sistem/alat kerja yang dipakai perusahaan.",
	"Mengerjakan laporan hasil kerja hari ini bersama tim.",
	"Melakukan pengecekan dan dokumentasi hasil pekerjaan.",
	"Diskusi evaluasi mingguan dengan mentor terkait progres kerja.",
	"Membantu menyiapkan bahan/materi untuk kegiatan tim.",
	"Mempelajari proses kerja baru yang dijelaskan oleh mentor.",
	"Menyelesaikan tugas tambahan dari mentor terkait pekerjaan berjalan.",
	"Ikut serta dalam rapat koordinasi tim harian.",
	"Membantu proses input data pada sistem internal perusahaan.",
	"Melakukan observasi langsung terhadap alur kerja tim produksi.",
	"Berdiskusi dengan mentor mengenai kendala yang ditemui saat bekerja.",
	"Menyusun dokumentasi hasil pekerjaan untuk laporan mingguan.",
	"Membantu menyiapkan kebutuhan operasional tim hari ini.",
	"Mengikuti pelatihan singkat terkait penggunaan alat/sistem baru.",
	"Melakukan pengecekan kualitas hasil kerja sebelum diserahkan.",
	"Membantu rekan kerja menyelesaikan tugas yang menumpuk.",
	"Mencatat progres pekerjaan harian untuk keperluan evaluasi mentor.",
	"Berkoordinasi dengan tim lain terkait pekerjaan lintas divisi.",
}

var attendanceWorkTypes = []string{"Praktik Kerja", "Pelatihan", "Dokumentasi", "Evaluasi", "Kerja Tim", "Observasi", "Administrasi", "Produksi"}

// monitorNotes/monitorSuggests/mentorReviewBodies/companyReviewBodies are
// generic enough (no company/school-specific wording) to reuse verbatim
// across every school's monitor/review sampling loop — see main() and
// seedSchool below.
var monitorNotes = []string{
	"Siswa terlihat aktif dan cepat beradaptasi dengan pekerjaan yang diberikan.",
	"Siswa cukup disiplin dalam mengikuti jadwal kerja di perusahaan.",
	"Siswa menunjukkan antusiasme tinggi terhadap tugas yang diberikan mentor.",
	"Siswa masih memerlukan bimbingan lebih dalam beberapa aspek teknis pekerjaan.",
	"Siswa menunjukkan perkembangan yang baik dibanding kunjungan sebelumnya.",
	"Siswa cukup komunikatif dengan mentor dan rekan kerja di perusahaan.",
}

var monitorSuggests = []string{
	"Pertahankan komunikasi rutin dengan mentor terkait progres tugas.",
	"Tingkatkan inisiatif dalam mengikuti diskusi dan pekerjaan tim.",
	"Perlu lebih percaya diri saat menyampaikan hasil kerja.",
	"Perbanyak jam praktik langsung didampingi mentor senior.",
	"Lanjutkan konsistensi dalam menjaga kedisiplinan kerja.",
	"Perlu peningkatan pada aspek ketelitian dan manajemen waktu.",
}

var mentorReviewBodies = []string{
	"Siswa menunjukkan kinerja yang sangat baik selama magang, aktif bertanya dan cepat memahami tugas yang diberikan.",
	"Siswa cukup baik dalam menyelesaikan tugas, namun perlu meningkatkan inisiatif dalam bekerja secara mandiri.",
	"Siswa sangat komunikatif dan mampu bekerja sama dengan baik dalam tim.",
	"Siswa cukup disiplin dan bertanggung jawab terhadap tugas yang diberikan mentor.",
	"Siswa menunjukkan semangat belajar yang tinggi meski beberapa kali masih perlu bimbingan.",
	"Siswa mampu beradaptasi dengan cepat terhadap budaya kerja di perusahaan.",
	"Siswa teliti dalam mengerjakan tugas namun perlu lebih percaya diri saat presentasi.",
	"Siswa aktif memberikan ide dan cukup kreatif dalam menyelesaikan pekerjaan.",
}

var companyReviewBodies = []string{
	"Suasana kerja sangat mendukung untuk belajar, mentor selalu membimbing dengan sabar.",
	"Banyak ilmu baru yang didapat selama menjalani PKL di perusahaan ini.",
	"Tim kerja sangat solid dan ramah, cocok untuk belajar dunia kerja secara langsung.",
	"Fasilitas kerja cukup memadai dan mentor selalu terbuka untuk berdiskusi.",
	"Pengalaman PKL di sini memberikan banyak wawasan baru tentang dunia industri.",
	"Lingkungan kerja nyaman dan rekan-rekan karyawan sangat membantu selama magang.",
}

// buildAttendance walks weekdays in [start, end] and returns the presence/
// journal rows in memory — no DB access here at all, so the caller can
// batch-insert thousands of these in a handful of round trips instead of
// one SELECT+INSERT per day (see bulkInsertPresences/bulkInsertJournals).
// Presences/journals are only ever created for a day someone actually
// reported, so weekends are simply skipped rather than backfilled with a
// "Libur" row. allApproved=false ages the last few days into "pending
// mentor review", matching a realistic backlog.
func buildAttendance(userID string, companyID int64, statusIDs map[string]int64, start, end time.Time, allApproved bool) (presences []seedPresenceRow, journals []seedJournalRow) {
	pattern := []string{"present", "present", "present", "permitted", "present", "sick", "present", "present", "absent", "present"}

	end = dateOnly(end)
	i := 0
	for d := dateOnly(start); !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		kind := pattern[i%len(pattern)]
		approved := allApproved || end.Sub(d).Hours()/24 >= 3

		var checkIn, checkOut *time.Time
		var desc *string
		switch kind {
		case "present":
			in := time.Date(d.Year(), d.Month(), d.Day(), 8, 0, 0, 0, time.UTC)
			out := time.Date(d.Year(), d.Month(), d.Day(), 16, 0, 0, 0, time.UTC)
			checkIn, checkOut = &in, &out
		case "permitted":
			s := "Izin keperluan keluarga"
			desc = &s
		case "sick":
			s := "Sakit, disertai surat keterangan"
			desc = &s
		}

		presences = append(presences, seedPresenceRow{
			UserID: userID, CompanyID: companyID, PresenceStatusID: statusIDs[kind], Date: d,
			CheckInAt: checkIn, CheckOutAt: checkOut, IsApproved: approved, Description: desc,
			CreatedTS: d, UpdatedTS: d,
		})

		// real interns skip some days — journals cover ~80% of "present"
		// days, not literally every single one.
		if kind == "present" && i%5 != 4 {
			journals = append(journals, seedJournalRow{
				UserID: userID, CompanyID: companyID, Date: d,
				WorkType: attendanceWorkTypes[i%len(attendanceWorkTypes)], Description: attendanceDescs[i%len(attendanceDescs)],
				IsApproved: approved, CreatedTS: d, UpdatedTS: d,
			})
		}
		i++
	}
	return presences, journals
}

// bulkInsertPresences/bulkInsertJournals rely on the tables' real
// UNIQUE(user_id, company_id, date) constraints for ON CONFLICT DO NOTHING,
// turning what would be thousands of per-row round trips into a handful of
// batched multi-row inserts.
func bulkInsertPresences(tx *gorm.DB, ctx context.Context, rows []seedPresenceRow) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "company_id"}, {Name: "date"}}, DoNothing: true}).
		CreateInBatches(&rows, 500).Error
}

func bulkInsertJournals(tx *gorm.DB, ctx context.Context, rows []seedJournalRow) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "company_id"}, {Name: "date"}}, DoNothing: true}).
		CreateInBatches(&rows, 500).Error
}

var scoreItems = []struct{ name, typ string }{
	{"Kualitas Kerja", "teknis"}, {"Penguasaan Teknis", "teknis"}, {"Kecepatan Kerja", "teknis"}, {"Ketelitian", "teknis"},
	{"Disiplin", "non-teknis"}, {"Komunikasi", "non-teknis"}, {"Kerjasama Tim", "non-teknis"}, {"Inisiatif", "non-teknis"},
}

var scoreOffsets = []int{-3, 2, -1, 4, 1, -2, 0, 3}

// buildScoreRows inserts a fixed set of teknis/non-teknis line items around
// targetAvg with small deterministic offsets, so the school's A/B/C/D
// predicate bands (see score_predicates above) end up genuinely exercised
// instead of every student landing in the same band. Pure in-memory build,
// same reasoning as buildAttendance above.
func buildScoreRows(userID string, companyID int64, targetAvg int, createdAt time.Time) []seedScoreRow {
	rows := make([]seedScoreRow, 0, len(scoreItems))
	for i, item := range scoreItems {
		score := clampScore(targetAvg + scoreOffsets[i])
		rows = append(rows, seedScoreRow{
			UserID: userID, CompanyID: companyID, Name: item.name, Score: score, Type: item.typ,
			CreatedTS: createdAt, UpdatedTS: createdAt,
		})
	}
	return rows
}

func clampScore(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// scores and notifications have no DB-level unique constraint to key an ON
// CONFLICT off (scores only has a non-unique index; notifications has none
// at all), so idempotency is done in Go instead: load what already exists
// once, filter the generated rows against it in memory, then bulk-insert
// only what's new.
func bulkInsertScores(tx *gorm.DB, ctx context.Context, rows []seedScoreRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	var existing []struct {
		UserID    string
		CompanyID int64
		Name      string
	}
	if err := tx.WithContext(ctx).Raw(`SELECT user_id, company_id, name FROM scores`).Scan(&existing).Error; err != nil {
		return 0, err
	}
	seen := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		seen[fmt.Sprintf("%s|%d|%s", e.UserID, e.CompanyID, e.Name)] = struct{}{}
	}
	toInsert := make([]seedScoreRow, 0, len(rows))
	for _, r := range rows {
		key := fmt.Sprintf("%s|%d|%s", r.UserID, r.CompanyID, r.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		toInsert = append(toInsert, r)
	}
	if len(toInsert) == 0 {
		return 0, nil
	}
	if err := tx.WithContext(ctx).CreateInBatches(&toInsert, 500).Error; err != nil {
		return 0, err
	}
	return len(toInsert), nil
}

func bulkInsertNotifications(tx *gorm.DB, ctx context.Context, rows []seedNotificationRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	var existing []struct {
		UserID string
		Type   string
		Title  string
	}
	if err := tx.WithContext(ctx).Raw(`SELECT user_id, type, title FROM notifications`).Scan(&existing).Error; err != nil {
		return 0, err
	}
	seen := make(map[string]struct{}, len(existing))
	for _, e := range existing {
		seen[e.UserID+"|"+e.Type+"|"+e.Title] = struct{}{}
	}
	toInsert := make([]seedNotificationRow, 0, len(rows))
	for _, r := range rows {
		key := r.UserID + "|" + r.Type + "|" + r.Title
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		toInsert = append(toInsert, r)
	}
	if len(toInsert) == 0 {
		return 0, nil
	}
	if err := tx.WithContext(ctx).CreateInBatches(&toInsert, 500).Error; err != nil {
		return 0, err
	}
	return len(toInsert), nil
}

// upsertCertificate mirrors scoring.Service.GenerateCertificate's numbering
// exactly (CERT-{department}-{year}-{seq:04d}, seq = count of that
// department's certificates issued this year, +1) so numbers generated here
// stay consistent with whatever the real API would produce next.
func upsertCertificate(tx *gorm.DB, ctx context.Context, userID string, deptID, companyID int64, asOf time.Time) error {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM certificates WHERE user_id = ? AND company_id = ?`, userID, companyID).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	year := asOf.Year()
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var seq int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM certificates WHERE department_id = ? AND created_at >= ? AND created_at < ?`, deptID, start, end).Scan(&seq).Error; err != nil {
		return err
	}
	number := fmt.Sprintf("CERT-%d-%d-%04d", deptID, year, seq+1)
	return tx.WithContext(ctx).Exec(`
		INSERT INTO certificates (user_id, department_id, company_id, certificate_number)
		VALUES (?, ?, ?, ?)
	`, userID, deptID, companyID, number).Error
}

func upsertNews(tx *gorm.DB, ctx context.Context, authorID string, schoolID int64, title, content, status string, createdAt time.Time) error {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM news WHERE title = ?`, title).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	var publishedAt *time.Time
	if status == "published" {
		publishedAt = &createdAt
	}
	return tx.WithContext(ctx).Exec(`
		INSERT INTO news (author_id, scope_type, scope_id, title, slug, content, status, published_at, created_at, updated_at)
		VALUES (?, 'school', ?, ?, ?, ?, ?, ?, ?, ?)
	`, authorID, schoolID, title, slugify(title), content, status, publishedAt, createdAt, createdAt).Error
}

// slugify is a plain kebab-case transform, deliberately without the real
// content.slugify's timestamp suffix — idempotency here is by title lookup,
// so a stable, re-derivable slug is what we want on a second `make seed` run.
func slugify(title string) string {
	lower := strings.ToLower(title)
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func upsertFAQ(tx *gorm.DB, ctx context.Context, question, answer string, sortOrder int) error {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM faqs WHERE question = ?`, question).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.WithContext(ctx).Exec(`INSERT INTO faqs (question, answer, sort_order) VALUES (?, ?, ?)`, question, answer, sortOrder).Error
}

func upsertQuestion(tx *gorm.DB, ctx context.Context, schoolID int64, question string, sortOrder int) (int64, error) {
	var id int64
	err := tx.WithContext(ctx).Raw(`SELECT id FROM questions WHERE school_id = ? AND question = ?`, schoolID, question).Scan(&id).Error
	if err == nil && id != 0 {
		return id, nil
	}
	err = tx.WithContext(ctx).Raw(`
		INSERT INTO questions (school_id, question, sort_order) VALUES (?, ?, ?) RETURNING id
	`, schoolID, question, sortOrder).Scan(&id).Error
	return id, err
}

func upsertMonitor(tx *gorm.DB, ctx context.Context, coordinatorID, studentID string, companyID int64, date time.Time, notes, suggest string, matchRating int) error {
	var count int64
	if err := tx.WithContext(ctx).Raw(`SELECT count(*) FROM monitors WHERE coordinator_id = ? AND student_id = ? AND date = ?`, coordinatorID, studentID, date).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.WithContext(ctx).Exec(`
		INSERT INTO monitors (coordinator_id, student_id, company_id, date, notes, suggest, match_rating)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, coordinatorID, studentID, companyID, date, notes, suggest, matchRating).Error
}

// upsertReview dedupes per reviewee kind separately — a review always
// targets exactly one of reviewee_user_id/reviewee_company_id (DB CHECK
// constraint, migration 000025), so checking both together via a single
// coalesced query risks ambiguous parameter typing for no benefit.
func upsertReview(tx *gorm.DB, ctx context.Context, reviewerID string, questionID *int64, revieweeUserID *string, revieweeCompanyID *int64, title, body string, rating int) error {
	var count int64
	var err error
	if revieweeUserID != nil {
		err = tx.WithContext(ctx).Raw(`SELECT count(*) FROM reviews WHERE reviewer_id = ? AND reviewee_user_id = ?`, reviewerID, *revieweeUserID).Scan(&count).Error
	} else {
		err = tx.WithContext(ctx).Raw(`SELECT count(*) FROM reviews WHERE reviewer_id = ? AND reviewee_company_id = ?`, reviewerID, *revieweeCompanyID).Scan(&count).Error
	}
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.WithContext(ctx).Exec(`
		INSERT INTO reviews (reviewer_id, question_id, reviewee_user_id, reviewee_company_id, title, body, rating)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, reviewerID, questionID, revieweeUserID, revieweeCompanyID, title, body, rating).Error
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "seed failed:", err)
	os.Exit(1)
}
