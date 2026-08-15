# E-comWA

[![CI](https://github.com/siri5creative/E-comWA/actions/workflows/ci.yml/badge.svg)](https://github.com/siri5creative/E-comWA/actions/workflows/ci.yml)

Toko online untuk UMKM dengan konfirmasi order & pembayaran manual lewat WhatsApp, dashboard admin (Owner/Staff), sistem kupon, laporan keuangan, dan integrasi dengan aplikasi POS toko fisik lewat satu database bersama.

Spesifikasi lengkap ada di `files/`:
- [`files/prd-ecommerce-wa.md`](files/prd-ecommerce-wa.md) — business logic, skema database, spesifikasi API
- [`files/IMPLEMENTATION.md`](files/IMPLEMENTATION.md) — panduan implementasi & urutan prioritas
- [`files/api-pos-integration.md`](files/api-pos-integration.md) — kontrak API untuk aplikasi POS
- [`files/design-brief-ecommerce-nike-style.md`](files/design-brief-ecommerce-nike-style.md) — arah desain

## Struktur Repo

```
E-comWA/
├── web/          Next.js (App Router, TypeScript, Tailwind) — storefront publik + dashboard admin
├── api/          Go REST API — semua logika bisnis & akses database
└── files/        Dokumen spesifikasi (PRD, panduan implementasi, dst)
```

## Prasyarat

- [Node.js](https://nodejs.org/) 20+
- [Go](https://go.dev/) 1.26+ (lihat `api/go.mod` untuk versi persis)
- Project [Supabase](https://supabase.com/) (Postgres + Auth)
- (Opsional, untuk push notification admin) Project [Firebase](https://firebase.google.com/) dengan Cloud Messaging
- (Opsional, untuk uji koneksi payment gateway) Akun Midtrans dan/atau Xendit

## Setup Lokal

### 1. Buat project Supabase & jalankan migration

Di **Supabase SQL Editor**, jalankan ketiga file migration di `api/migrations/` **secara berurutan**:

1. `0001_init.sql` — skema inti (produk, order, kupon, admin, dst) + Row Level Security
2. `0002_coupon_discount_fields.sql` — kolom tambahan kupon (nilai persentase, minimum belanja)
3. `0003_orders_payment_method.sql` — kolom `payment_method` untuk transaksi POS

### 2. Buat akun Owner pertama

Tidak ada UI signup — admin didaftarkan manual (PRD section 7A):

1. Di **Supabase Dashboard → Authentication → Users**, buat user baru (email + password).
2. Salin `id` user tersebut (UUID), lalu jalankan di SQL Editor:

   ```sql
   insert into admins (auth_user_id, name, role)
   values ('<uuid-dari-langkah-1>', 'Nama Owner', 'owner');
   ```

### 3. Konfigurasi environment variable

```bash
cp api/.env.example api/.env
cp web/.env.example web/.env.local
```

Isi masing-masing sesuai komentar di dalam file. Ringkasan sumbernya:

| Variable | Didapat dari |
|---|---|
| `DATABASE_URL` | Supabase → Project Settings → Database → Connection string (Supavisor pooler) |
| `SUPABASE_JWT_SECRET` | Supabase → Project Settings → API → JWT Secret |
| `SUPABASE_SERVICE_ROLE_KEY` | Supabase → Project Settings → API → service_role key |
| `NEXT_PUBLIC_SUPABASE_URL` / `NEXT_PUBLIC_SUPABASE_ANON_KEY` | Supabase → Project Settings → API |
| `FIREBASE_SERVICE_ACCOUNT_KEY` | Firebase Console → Project Settings → Service Accounts → Generate new private key (opsional — lihat catatan di `api/.env.example`) |
| `NEXT_PUBLIC_FIREBASE_CONFIG` / `NEXT_PUBLIC_FIREBASE_VAPID_KEY` | Firebase Console → Project Settings → General / Cloud Messaging (opsional) |
| `PAYMENT_GATEWAY_ENCRYPTION_KEY` | Generate sendiri: `openssl rand -base64 32` (opsional) |
| `POS_API_KEY` | Generate sendiri: `openssl rand -base64 32`, bagikan ke tim POS lewat channel privat (opsional) |
| `NEXT_PUBLIC_WHATSAPP_NUMBER` | Nomor WhatsApp toko, format `62xxx` |

Variable yang ditandai opsional membuat fitur terkait nonaktif dengan graceful (server tetap jalan, cuma fitur itu yang dimatikan) — bukan syarat wajib untuk menjalankan aplikasi.

### 4. Jalankan backend

```bash
cd api
go run .
```

Default `http://localhost:3000` (ubah `PORT` di `api/.env` kalau bentrok dengan `next dev` di langkah berikutnya).

### 5. Jalankan frontend

```bash
cd web
npm install
npm run dev
```

Default `http://localhost:3000` juga — pastikan `PORT` di `api/.env` sudah diubah (mis. `8080`) dan `GO_API_URL` di `web/.env.local` disesuaikan, supaya keduanya bisa jalan bersamaan.

Storefront: `http://localhost:3000`. Dashboard admin: `http://localhost:3000/admin/login`.

## Perintah Umum

| Perintah | Keterangan |
|---|---|
| `cd api && go run .` | Jalankan backend |
| `cd api && go build ./...` | Build & cek compile error |
| `cd api && go vet ./...` | Static analysis |
| `cd api && gofmt -l .` | Cek file yang belum diformat |
| `cd api && go test ./...` | Jalankan test suite (lihat "Testing" di bawah) |
| `cd web && npm run dev` | Jalankan frontend (dev, Turbopack) |
| `cd web && npm run build` | Build production |
| `cd web && npx eslint .` | Lint |

## Testing

Backend (`api/`) punya test suite otomatis — unit test untuk logika murni (`internal/util`, `internal/crypto`, `internal/models`) dan integration test untuk handler HTTP yang menyentuh business logic penting (checkout, order POS, transisi status order, evaluasi kupon).

```bash
cd api
go test ./...              # semua test
go test ./... -race        # sertakan Go race detector
go test ./internal/handlers/... -run TestCrossChannelStockRaceCondition -v   # test race condition stok lintas channel
```

Integration test butuh **Postgres lokal yang jalan** (dites pakai user OS saat ini via `localhost:5432`, koneksi tanpa password — sesuaikan lewat env var `TEST_DATABASE_URL` kalau setup lokalmu beda). `internal/testutil` otomatis: bikin database baru unik per test, jalankan semua migration, seed data lewat helper (`SeedProductVariant`, `SeedAdmin`, `SeedCoupon`), lalu hapus database itu setelah test selesai — tidak menyentuh database development/production. Kalau Postgres tidak terjangkau, test-test ini di-skip (bukan gagal), jadi `go test ./...` tetap aman dijalankan di mesin/CI yang belum ada Postgres-nya.

`web/` belum punya test suite otomatis (belum ada Jest/Playwright/dsb).

### CI

`.github/workflows/ci.yml` jalan otomatis di tiap push/PR ke `main` — dua job paralel:

- **API**: `gofmt -l`, `go vet`, `go build`, `go test ./... -race` (pakai service container Postgres, bukan `TEST_DATABASE_URL` eksternal)
- **Web**: migration dijalankan ke Postgres service container, backend Go di-build & dijalankan di background, baru `eslint` dan `next build` — karena halaman publik melakukan fetch ke backend saat build (prerender), bukan cuma type-check statis.

## Urutan Prioritas Pengembangan

Sembilan tahap sesuai `files/IMPLEMENTATION.md` section 4 — dari fondasi (skema database, backend inti, frontend publik) sampai fitur admin lengkap (order management, kupon, notifikasi, laporan keuangan, payment settings, integrasi POS). Semua sembilan tahap sudah diimplementasikan di repo ini.

## Deployment

Disarankan pakai **Vercel Services** dalam satu project (`web/` preset Next.js, `api/` preset Go) — lihat `files/IMPLEMENTATION.md` section 3 untuk urutan langkah deployment lengkap.

## Catatan Verifikasi

Backend punya test suite otomatis (lihat "Testing" di atas) yang mengunci perilaku fitur-fitur inti, termasuk skenario race condition stok lintas channel (online vs POS). Yang **belum** bisa diverifikasi otomatis tanpa kredensial asli:
- Alur login admin end-to-end (butuh project Supabase nyata)
- Pengiriman push notification sungguhan (butuh project Firebase nyata)
- Uji koneksi payment gateway dengan akun Midtrans/Xendit asli

`web/` belum punya test otomatis — perubahan di frontend masih perlu diverifikasi manual (`npm run build`, `npx eslint .`, dan uji coba langsung di browser).
