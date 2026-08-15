# IMPLEMENTATION.md — Panduan Implementasi untuk Claude Code

Dokumen ini melengkapi `prd-ecommerce-wa.md` (business logic & spesifikasi) dan `api-pos-integration.md` (kontrak API untuk POS). Kalau `prd-ecommerce-wa.md` menjawab "apa yang harus dibangun", dokumen ini menjawab **"bagaimana cara membangunnya di dalam repo"**.

Baca ketiga dokumen ini (PRD → dokumen ini → dokumen API POS) sebelum mulai menulis kode apapun.

---

## 1. Struktur Monorepo

```
ecommerce-wa/
├── web/                          # Next.js frontend (App Router, TypeScript)
│   ├── app/
│   │   ├── (public)/
│   │   │   ├── page.tsx                  # Home
│   │   │   ├── products/page.tsx         # Katalog produk
│   │   │   ├── products/[slug]/page.tsx  # Detail produk
│   │   │   ├── cart/page.tsx             # Keranjang & checkout
│   │   │   └── order/[invoice]/page.tsx  # Halaman status order (dicek pakai invoice number)
│   │   ├── admin/                        # Protected, khusus role Owner/Staff
│   │   │   ├── login/page.tsx
│   │   │   ├── dashboard/page.tsx
│   │   │   ├── orders/page.tsx
│   │   │   ├── orders/[id]/page.tsx      # Detail order + tombol update status + kirim WA
│   │   │   ├── products/page.tsx         # Kelola produk & varian & stok
│   │   │   ├── coupons/page.tsx          # Khusus Owner
│   │   │   ├── admins/page.tsx           # Kelola akun admin, khusus Owner
│   │   │   ├── payment-settings/page.tsx # Khusus Owner
│   │   │   └── reports/page.tsx          # Laporan keuangan, khusus Owner
│   │   └── api/                          # Route Handler (proxy/BFF ke Go, untuk endpoint yang butuh auth)
│   ├── components/
│   │   └── WhatsAppFloatButton.tsx
│   ├── lib/
│   │   ├── supabase/                     # client.ts, server.ts (pakai @supabase/ssr)
│   │   └── api.ts                        # helper fetch ke Go backend
│   └── middleware.ts                     # proteksi route /admin, cek role
│
├── api/                          # Go backend
│   ├── main.go
│   ├── go.mod
│   ├── internal/
│   │   ├── handlers/
│   │   │   ├── products.go
│   │   │   ├── checkout.go
│   │   │   ├── orders.go
│   │   │   ├── coupons.go
│   │   │   ├── admins.go
│   │   │   ├── payment_settings.go
│   │   │   ├── reports.go
│   │   │   ├── admin_devices.go
│   │   │   └── pos.go                    # endpoint khusus POS, lihat api-pos-integration.md
│   │   ├── middleware/
│   │   │   ├── auth_admin.go             # verifikasi Supabase Auth + cek role owner/staff
│   │   │   ├── auth_pos.go               # verifikasi POS_API_KEY
│   │   │   └── cors.go
│   │   ├── models/
│   │   └── db/                           # koneksi pgx + query
│   └── migrations/
│       └── 0001_init.sql                 # seluruh skema di section 7 PRD
│
└── vercel.json                   # konfigurasi Vercel Services (web + api dalam satu project)
```

---

## 2. Environment Variables

### `web/.env.local`
```
NEXT_PUBLIC_SUPABASE_URL=
NEXT_PUBLIC_SUPABASE_ANON_KEY=
GO_API_URL=                          # internal URL kalau pakai Vercel Services, atau public URL kalau backend terpisah
NEXT_PUBLIC_WHATSAPP_NUMBER=         # nomor WA toko untuk floating contact button, format 62xxx
NEXT_PUBLIC_FIREBASE_CONFIG=         # config client Firebase (untuk minta izin notifikasi di dashboard admin)
```

### `api/.env`
```
DATABASE_URL=                        # Supavisor pooler connection string dari Supabase
SUPABASE_JWT_SECRET=                 # untuk verifikasi token admin (Owner/Staff)
SUPABASE_SERVICE_ROLE_KEY=           # dipakai backend untuk operasi yang butuh bypass RLS
FIREBASE_SERVICE_ACCOUNT_KEY=        # JSON credential Firebase Admin SDK, untuk kirim push notification
PAYMENT_GATEWAY_ENCRYPTION_KEY=      # untuk enkripsi kredensial Midtrans/Xendit yang disimpan admin
POS_API_KEY=                         # API key statis untuk validasi request dari aplikasi POS (lihat api-pos-integration.md)
PORT=3000
```

⚠️ Tidak satupun dari variable di atas boleh di-hardcode di kode. `SUPABASE_SERVICE_ROLE_KEY`, `PAYMENT_GATEWAY_ENCRYPTION_KEY`, dan `POS_API_KEY` khususnya **tidak boleh pernah** ada di kode frontend/browser.

---

## 3. Rencana Deployment (Urutan)

1. Buat project Supabase → jalankan `migrations/0001_init.sql` di SQL Editor → aktifkan RLS sesuai section 7 PRD
2. Buat akun Owner pertama secara manual lewat Supabase Auth dashboard (belum ada UI signup di aplikasi, karena admin didaftarkan manual, bukan self-register)
3. Buat project Firebase → aktifkan Cloud Messaging saja → generate Service Account Key
4. Generate `POS_API_KEY` (string acak yang aman), simpan di `api/.env` dan bagikan ke tim developer POS secara terpisah (bukan lewat channel publik)
5. Buat project Vercel dengan **Vercel Services**, arahkan ke `web/` (Next.js preset) dan `api/` (Go preset)
6. Set seluruh environment variable di section 2 lewat Vercel dashboard
7. Deploy → cek preview URL dulu sebelum promote ke production
8. Uji manual end-to-end: checkout → cek order masuk di dashboard admin → update status → cek link `wa.me` ter-generate benar → cek push notification masuk ke admin
9. Uji endpoint POS (`/pos/orders`) pakai `POS_API_KEY`, pastikan stok berkurang dan konsisten dengan yang tampil di web

---

## 4. Urutan Prioritas Pengerjaan

Kerjakan berurutan, bukan sekaligus semua — supaya ada fondasi yang bisa dites di tiap tahap sebelum lanjut ke bagian berikutnya:

1. **Skema database** — jalankan migration, pastikan semua tabel & RLS di section 7 PRD sudah benar sebelum menulis kode aplikasi apapun
2. **Backend inti**: endpoint produk (`GET /products`), checkout (`POST /checkout`), autentikasi admin — ini fondasi yang dipakai fitur lain
3. **Frontend publik**: katalog produk, detail produk, checkout — supaya alur inti bisa dites dari sisi user
4. **Dashboard admin — order management**: list order, detail order, update status, generate & kirim link `wa.me`
5. **Fitur kupon**: validasi kupon di checkout, CRUD kupon di admin (Owner)
6. **Notifikasi**: push notification order masuk (Firebase), dan link `wa.me` untuk update status ke customer
7. **Laporan keuangan**: setelah alur order & channel (online) sudah jalan dan ada data untuk dihitung
8. **Menu pengaturan payment gateway**: UI & penyimpanan kredensial terenkripsi (fitur ini disiapkan tapi tidak diaktifkan)
9. **Endpoint integrasi POS**: dikerjakan terakhir, mengacu ke `api-pos-integration.md`, karena bergantung pada logika stok yang sudah stabil dari tahap-tahap sebelumnya

---

## 5. Ekspektasi Output

- Kode harus bisa langsung dijalankan (`npm run dev` untuk `web/`, `go run main.go` untuk `api/`) tanpa error, dengan environment variable dummy/contoh disediakan di `.env.example`
- Sertakan file migration SQL yang siap dijalankan langsung di Supabase SQL Editor
- Sertakan `README.md` di root repo: cara setup lokal, isi environment variable, jalankan migration, jalankan dev server
- TypeScript strict mode di Next.js; kode Go mengikuti `gofmt` dan konvensi idiomatic Go
- Jangan hardcode kredensial apapun — semua lewat environment variable, sesuai daftar di section 2
- Ikuti urutan prioritas di section 4 — jangan mulai dari fitur yang levelnya lebih rendah prioritasnya sebelum fondasi selesai
- Untuk keputusan yang masih ditandai "perlu dicek saat implementasi" di PRD (lihat section 11 PRD), tanyakan ke pengguna sebelum mengambil keputusan sepihak — jangan berasumsi
- Prioritaskan kesesuaian dengan scope di PRD (section 3) dibanding menambah fitur di luar itu — hindari over-engineering untuk MVP ini

---

## 6. Referensi Dokumen Lain

- `prd-ecommerce-wa.md` — business logic, skema database lengkap, spesifikasi API, non-functional requirements
- `api-pos-integration.md` — kontrak API khusus untuk aplikasi POS
- `design-brief-ecommerce-nike-style.md` — arah visual (warna, tipografi, layout) untuk diterapkan di `web/`
