# PRD: E-commerce UMKM dengan Order via WhatsApp

## 1. Ringkasan Produk

Website e-commerce untuk UMKM/personal seller, dengan katalog produk, checkout, dan sistem kupon promo. Konfirmasi order, verifikasi pembayaran, dan update status pesanan dilakukan lewat kombinasi dashboard admin + WhatsApp (WhatsApp Business App biasa, bukan WhatsApp Business API). Order baru berstatus "Diproses" hanya setelah pembayaran dikonfirmasi lunas oleh admin.

---

## 2. Tujuan

- Memberi UMKM kecil sebuah toko online sendiri (bukan marketplace pihak ketiga) dengan biaya operasional minimal
- Mempertahankan alur konfirmasi pembayaran manual via WhatsApp (sesuai kebiasaan UMKM), tapi tetap punya histori order yang rapi di sistem
- Menyiapkan fondasi untuk otomasi lanjutan di masa depan (payment gateway, WhatsApp Business API) tanpa harus dipakai sekarang

---

## 3. Ruang Lingkup (Scope)

### 3.1 Termasuk Scope
- Katalog produk dengan varian (ukuran/warna) dan tracking stok otomatis
- Checkout dengan registrasi ringan (nama + nomor WhatsApp, tanpa password)
- Sistem kupon promo (4 jenis potongan, 3 jenis pembatasan)
- Alur order: checkout → menunggu pembayaran → verifikasi manual oleh admin → diproses → dikirim → selesai
- Ongkir diinfokan manual oleh admin via WhatsApp (tidak ada kalkulasi otomatis)
- Notifikasi order masuk ke admin lewat push notification (tetap muncul meski dashboard tidak dibuka)
- Notifikasi update status ke customer via WhatsApp, semi-otomatis (sistem siapkan pesan, admin klik kirim)
- Dashboard admin dengan 2 role: **Owner** dan **Staff**
- Menu pengaturan integrasi payment gateway (disiapkan, belum aktif dipakai)
- **Integrasi dengan aplikasi POS yang sudah ada** — database e-commerce ini didesain sebagai sumber utama data produk, stok, dan transaksi yang dipakai bersama oleh POS (lihat section 7A)

### 3.2 Di Luar Scope (Tidak Dikerjakan Saat Ini)
- Payment gateway aktif/live (baru disiapkan menunya saja)
- WhatsApp Business API (pengiriman pesan otomatis penuh)
- Kalkulasi ongkir otomatis via API ekspedisi
- Auto-cancel order otomatis saat lewat batas waktu bayar (tetap manual oleh admin)
- Multi-bahasa (i18n)
- Aplikasi mobile native (cukup web responsif)
- Review/rating produk oleh customer

---

## 4. User Roles

| Role | Deskripsi | Akses |
|---|---|---|
| **Customer** | Pengunjung yang belanja | Browse produk, checkout, isi nama+WA, pakai kupon — tanpa login/password |
| **Admin - Staff** | Mengelola operasional harian | Lihat & proses order, update status order, kirim update WA ke customer, kelola stok produk |
| **Admin - Owner** | Pemilik bisnis, akses penuh | Semua akses Staff + kelola kupon, kelola akun admin lain, atur integrasi payment gateway |

---

## 5. Tech Stack

| Layer | Teknologi |
|---|---|
| Frontend | Next.js (App Router) + TypeScript, Tailwind CSS |
| Backend | Go (Golang), REST API |
| Database, Auth, Storage | Supabase (Postgres, Auth untuk admin, Storage untuk gambar produk) |
| Hosting | Vercel (disarankan pakai Vercel Services untuk frontend + backend dalam satu project) |
| Push notification admin | Firebase Cloud Messaging (khusus notifikasi, database tetap di Supabase) |
| Notifikasi customer | Link `wa.me` (WhatsApp Business App) — bukan WhatsApp Business API |

---

## 6. Functional Requirements

### 6.1 Katalog Produk (Publik)
- List produk dengan grid, filter kategori sederhana
- Detail produk: galeri foto, deskripsi, pilihan varian (ukuran/warna), harga, status stok
- Kalau semua varian habis stok → tombol beli nonaktif, tampilkan "Stok Habis"

### 6.2 Registrasi Ringan & Checkout (Customer)
- Sebelum checkout, customer wajib isi **nama** dan **nomor WhatsApp** (tanpa password, data dipakai juga untuk keperluan marketing/promo ke depan)
- Customer pilih varian produk, jumlah, lalu checkout
- Customer bisa masukkan kode kupon saat checkout (lihat 6.4)
- Setelah checkout, order tersimpan dengan status **Menunggu Konfirmasi**

### 6.3 Alur Order & Pembayaran
```
Checkout → Admin cek & putuskan order valid/tidak
   → Tidak valid → Dibatalkan (admin kirim info via WA)
   → Valid → Menunggu Pembayaran
        → Customer transfer manual, kirim bukti (screenshot) via WhatsApp
        → Admin cek bukti pembayaran manual
             → Belum lunas/kurang → Admin follow up minta pelunasan
             → Sudah lunas → Diproses → Dikemas → Dikirim → Selesai
```
- Status "Diproses" **hanya bisa terjadi setelah admin menandai pembayaran lunas** — ini aturan wajib, tidak boleh dilewati
- Ongkir tidak dihitung otomatis di sistem — disepakati admin dan customer manual via chat WhatsApp. Setelah disepakati, **admin input angka ongkir tersebut ke kolom "Ongkir" di halaman detail order pada dashboard**, sehingga tercatat di database (`shipping_cost`) dan Total Order otomatis terhitung ulang (Subtotal + Ongkir). Tanpa langkah input manual ini, data ongkir tidak akan pernah tercatat di sistem dan tidak akan ikut dihitung di Laporan Keuangan
- Kalau customer belum bayar sampai batas waktu (1-2 hari, fleksibel tergantung kondisi) → **tidak otomatis dibatalkan oleh sistem** — order tetap berstatus Menunggu Pembayaran, dan admin yang follow up manual mulai hari H

### 6.4 Kupon Promo
**Jenis potongan** (satu kupon pakai salah satu jenis):
1. Potongan total belanja (contoh: belanja di atas Rp100rb, potong Rp20rb)
2. Potongan item tertentu (diskon khusus produk tertentu)
3. Potongan berdasarkan hari/event tertentu
4. Potongan paket/bundle (kombinasi beberapa item)

**Pembatasan kupon** (berlaku sekaligus untuk satu kupon):
- Dibatasi jumlah pemakaian total (contoh: maksimal 100x terpakai)
- Dibatasi per customer (contoh: 1 customer hanya boleh pakai 1x, dicek berdasarkan nomor WA)
- Dibatasi tanggal berlaku (tanggal mulai & tanggal berakhir)

**Alur pakai kupon (Customer)**:
```
Checkout → Masukkan kode kupon → Sistem cek valid?
   → Tidak valid → Tampilkan alasan (kadaluarsa / sudah dipakai / kuota habis) → lanjut checkout tanpa kupon
   → Valid → Potongan harga diterapkan → lanjut bayar
```

**Alur bikin kupon (Admin - Owner only)**:
- Form input: kode kupon, jenis potongan, nilai potongan, tanggal mulai & berakhir, batas jumlah pemakaian
- Kupon yang baru dibuat langsung aktif sesuai tanggal yang diset

### 6.5 Update Status Order & Notifikasi ke Customer
- Admin (Staff/Owner) update status order lewat dashboard
- Begitu status berubah, sistem otomatis siapkan pesan WhatsApp (format `wa.me` dengan teks sudah terisi sesuai status dan nomor WA customer)
- Admin tinggal klik tombol "Kirim Update" → terbuka WhatsApp dengan pesan siap kirim → admin klik kirim
- Customer menerima notifikasi WhatsApp seperti chat biasa

### 6.6 Notifikasi Order Masuk ke Admin
- Saat admin login pertama kali di dashboard, browser minta izin notifikasi
- Order/pembayaran baru masuk → sistem kirim push notification ke admin (lewat Firebase Cloud Messaging)
- Notifikasi tetap muncul meski dashboard/website admin sedang tidak dibuka

### 6.7 Dashboard Admin
- **Staff** bisa akses: list & detail order, update status order, kirim update WA, kelola stok produk
- **Owner** bisa akses semua yang Staff bisa, ditambah:
  - Kelola kupon (buat/edit/nonaktifkan)
  - Kelola akun admin lain (tambah/hapus Staff, atur role)
  - Menu pengaturan integrasi payment gateway (lihat 6.8)
  - Menu laporan keuangan (lihat 6.9)

### 6.8 Menu Pengaturan Pembayaran Online (Disiapkan, Belum Aktif)
- Khusus role **Owner**
- Form: pilih penyedia pembayaran (Midtrans/Xendit), isi kredensial akun, uji coba sambungan, simpan informasi dengan aman
- Toggle aktif/nonaktif — selama nonaktif, seluruh alur pembayaran tetap manual (transfer + cek bukti via WA) seperti dijelaskan di 6.3

### 6.9 Laporan Keuangan (Owner Only)
- Menu khusus role **Owner**, tidak bisa diakses Staff
- Filter periode: harian, mingguan, bulanan, atau rentang tanggal custom
- Filter tambahan: **channel** (Semua / Online / POS) — laporan default tampil gabungan, tapi bisa di-breakdown per channel
- **Ringkasan (summary cards)**:
  - Total order (dan breakdown per status: selesai, dibatalkan, dll)
  - Total Revenue Kotor (jumlah `total` dari order yang sudah **Selesai**, gabungan semua channel)
  - Revenue per channel (Online vs POS) ditampilkan terpisah di bawah total gabungan
  - Total Diskon Kupon Terpakai
  - Total Ongkir (dari field `shipping_cost`, khusus channel Online — transaksi POS tidak ada ongkir)
  - **Pendapatan Bersih (Net)** = Revenue Kotor − Diskon Kupon. Ongkir **tidak dikurangkan** dari perhitungan ini — karena ongkir ditentukan admin manual lewat chat WA dan sepenuhnya ditagih ke customer (toko hanya meneruskan biaya ke kurir, bukan menanggungnya), jadi ongkir ditampilkan sebagai info terpisah di ringkasan, bukan pengurang pendapatan
- **Produk terlaris**: daftar produk/varian dengan jumlah terjual & revenue tertinggi dalam periode yang dipilih, gabungan online + POS
- **Grafik tren penjualan**: grafik sederhana (line/bar chart) jumlah order & revenue per hari dalam periode yang dipilih, dengan opsi tampil per channel (dua garis: Online vs POS)
- Hanya menghitung order dengan status **Selesai** sebagai revenue valid (order Dibatalkan/Menunggu Pembayaran tidak dihitung sebagai revenue)

---

## 7. Skema Database (Supabase Postgres — Draft)

```sql
-- customers: data ringan, tanpa auth/password
-- customers: whatsapp_number disimpan dalam format internasional (62xxx), tanpa "+" atau "0" di depan
customers (id, name, whatsapp_number UNIQUE, created_at)

-- admins: pakai Supabase Auth, role disimpan terpisah
-- admins: khusus untuk aplikasi e-commerce (web), pakai Supabase Auth. Terpisah dari sistem login POS
admins (id, auth_user_id REFERENCES auth.users, name, role ENUM('owner','staff'), created_at)

-- categories
categories (id, name, slug UNIQUE)

-- products
products (id, name, slug UNIQUE, description TEXT, category_id REFERENCES categories(id), cover_image_url, created_at, updated_at)

-- product_variants: varian ukuran/warna + stok per varian
product_variants (id, product_id REFERENCES products(id), variant_name, sku, price, stock_quantity, created_at, updated_at)

-- coupons
coupons (id, code UNIQUE, discount_type ENUM('total_belanja','item_tertentu','event','bundle'),
         discount_value, valid_from, valid_until,
         max_total_usage, max_usage_per_customer, current_usage_count, is_active, created_by REFERENCES admins(id), created_at)

-- coupon_products: relasi kupon ke produk tertentu, dipakai khusus discount_type 'item_tertentu' atau 'bundle'
coupon_products (id, coupon_id REFERENCES coupons(id), product_id REFERENCES products(id))

-- coupon_usages: histori pemakaian, disimpan permanen untuk keperluan laporan & audit riwayat kupon
coupon_usages (id, coupon_id REFERENCES coupons(id), customer_id REFERENCES customers(id), order_id, used_at)

-- orders
orders (id, invoice_number UNIQUE, customer_id REFERENCES customers(id) NULL,
        channel ENUM('online','pos'),
        status ENUM('menunggu_konfirmasi','menunggu_pembayaran','diproses','dikirim','selesai','dibatalkan'),
        coupon_id REFERENCES coupons(id) NULL, subtotal, discount_amount, shipping_cost, total,
        shipping_note TEXT, payment_proof_note TEXT, created_at, updated_at)

-- order_items
order_items (id, order_id REFERENCES orders(id), product_variant_id REFERENCES product_variants(id),
             quantity, price_at_purchase)

-- admin_devices: device token untuk push notification
admin_devices (id, admin_id REFERENCES admins(id), fcm_device_token, created_at)

-- payment_gateway_settings: disiapkan, belum aktif dipakai
payment_gateway_settings (id, provider ENUM('midtrans','xendit'), is_sandbox BOOLEAN,
                           encrypted_credentials, is_active BOOLEAN, updated_by REFERENCES admins(id), updated_at)
```

**Row Level Security (RLS)**:
- Public: `SELECT` untuk `products`, `product_variants`, `categories`
- Public: `INSERT` untuk `customers`, `orders`, `order_items` (lewat backend, bukan langsung dari klien)
- Admin (authenticated, role apapun): akses penuh ke `orders`, `order_items`, `product_variants` (untuk update stok)
- Admin (role `owner` saja): akses ke `coupons`, `admins`, `payment_gateway_settings`

---

## 7A. Arsitektur Integrasi dengan Aplikasi POS (POS Sudah Ada, Terintegrasi ke Database Ini)

**Konteks penting**: Aplikasi POS untuk toko fisik (offline) **sudah dibangun lebih dulu** (custom, buatan sendiri, di Supabase). POS ini sengaja didesain untuk mengambil data produk & stok dari **satu database yang sama dengan e-commerce ini** — jadi database yang dirancang di section 7 PRD ini **bukan database terpisah untuk e-commerce saja, melainkan database bersama** yang jadi sumber utama (single source of truth) baik untuk web online maupun POS. POS sendiri sudah punya sistem login kasir/staff sendiri.

### Prinsip Utama
- **Satu database Supabase yang sama** dipakai oleh web online maupun aplikasi POS — ini bukan rencana masa depan, tapi memang sudah jadi desain sejak awal karena POS sudah dibangun mengacu ke database bersama ini
- **Semua penulisan data (terutama pengurangan stok dan pencatatan transaksi) wajib lewat Go backend API yang sama** — baik dari web maupun dari aplikasi POS. Aplikasi POS tidak boleh menulis langsung ke Supabase tanpa lewat backend, supaya:
  - Logika pengurangan stok cuma ada di satu tempat (tidak dobel logika antara web dan POS)
  - Dihindari race condition — kalau ada 2 transaksi bersamaan (1 online, 1 di kasir) untuk produk yang stoknya tinggal 1, sistem harus jamin cuma salah satu yang berhasil
- Pengurangan stok di backend wajib pakai operasi atomik (contoh: `UPDATE product_variants SET stock_quantity = stock_quantity - :qty WHERE id = :id AND stock_quantity >= :qty`), bukan cek-lalu-update terpisah yang rawan race condition

### Perbedaan Alur Transaksi POS vs Online
- Transaksi dari **POS langsung berstatus "Selesai"** saat itu juga, karena pembayaran (cash/QRIS) terjadi langsung di tempat — tidak melalui status Menunggu Konfirmasi/Menunggu Pembayaran seperti alur online
- Transaksi POS **tidak wajib terhubung ke `customer_id`** (pembeli walk-in tidak harus didaftarkan nama+WA seperti di online, meski bisa jadi opsional kalau kasir mau catat)
- Transaksi POS tidak ada `shipping_cost` (barang dibawa langsung oleh pembeli)

### Login Kasir/Staff POS
- POS sudah punya mekanisme login kasir sendiri sebelum project e-commerce ini dibuat, dan **ini tetap dipertahankan terpisah** — bukan disatukan ke tabel `admins` di section 7
- **Prinsipnya**: yang dibagikan (shared) antar aplikasi cuma **data produk, stok, dan transaksi/order** (lewat kolom `channel`). Sistem **login/autentikasi/role tetap milik masing-masing aplikasi** — e-commerce pakai tabel `admins` + Supabase Auth miliknya sendiri, POS pakai sistem login kasir yang sudah ada sendiri, karena keduanya memang aplikasi yang terpisah
- Konsekuensinya: endpoint `/pos/orders` di backend **tidak divalidasi lewat Supabase Auth admin** seperti endpoint e-commerce lainnya — perlu mekanisme auth terpisah khusus untuk POS (misal API key/token khusus yang dipegang aplikasi POS, atau skema auth yang sudah dipakai POS saat ini). Detail teknisnya perlu disesuaikan dengan sistem auth yang sudah berjalan di POS
- Kalau ke depannya perlu tahu "kasir mana yang input transaksi ini" untuk keperluan laporan, cukup simpan sebagai teks/identifier bebas (bukan foreign key ke tabel `admins`), karena datanya berasal dari sistem yang berbeda

### Rekomendasi Tambahan
- Manfaatkan **Supabase Realtime** (fitur bawaan Supabase, bukan servis tambahan) supaya kalau ada perubahan stok dari channel manapun, tampilan di channel lain (misal layar POS) bisa otomatis ter-update tanpa perlu refresh manual

---

## 8. Spesifikasi API Backend (Go) — Draft

Base URL: `/api/v1`

| Method | Endpoint | Auth | Deskripsi |
|---|---|---|---|
| GET | `/products` | Public | List produk + varian + stok |
| GET | `/products/:slug` | Public | Detail produk |
| POST | `/checkout` | Public | Buat order baru (termasuk data customer, item, kupon jika ada) |
| POST | `/coupons/validate` | Public | Cek validitas kupon saat checkout |
| GET | `/orders` | Admin | List order (Staff & Owner) |
| GET | `/orders/:id` | Admin | Detail order |
| PATCH | `/orders/:id/status` | Admin | Update status order |
| GET | `/orders/:id/wa-message` | Admin | Generate teks pesan WA sesuai status terbaru |
| POST | `/products` , PUT/DELETE `/products/:id` | Admin | Kelola produk & varian |
| POST/PUT/DELETE | `/coupons(/:id)` | Admin (Owner) | Kelola kupon |
| POST/DELETE | `/admins(/:id)` | Admin (Owner) | Kelola akun admin & role |
| POST | `/payment-settings` | Admin (Owner) | Simpan/uji konfigurasi payment gateway |
| POST | `/admin-devices` | Admin | Simpan device token FCM |
| GET | `/reports/summary` | Admin (Owner) | Ringkasan revenue, diskon, ongkir, net income per periode |
| GET | `/reports/top-products` | Admin (Owner) | Produk/varian terlaris per periode |
| GET | `/reports/sales-trend` | Admin (Owner) | Data tren penjualan harian untuk grafik |
| POST | `/pos/orders` | Auth terpisah milik POS (bukan Supabase Auth admin e-commerce) | Buat transaksi baru dari aplikasi POS, langsung berstatus Selesai, memakai logika pengurangan stok yang sama dengan order online |

**Auth admin**: Supabase Auth (email+password), role (`owner`/`staff`) dicek di setiap endpoint yang sensitif — bukan hanya dicek di frontend.

---

## 9. Non-Functional Requirements

**Keamanan**
- Kredensial payment gateway disimpan terenkripsi, tidak pernah diekspos ke frontend
- Endpoint admin selalu verifikasi role (Owner vs Staff) di backend, bukan cuma disembunyikan di UI
- Rate limiting di endpoint `/checkout` dan `/coupons/validate` untuk cegah abuse

**Performance**
- Halaman katalog produk pakai caching/ISR di Next.js
- Pagination di endpoint `/products` dan `/orders`

**Aksesibilitas**
- Alt text di semua gambar produk
- Kontras warna sesuai WCAG AA (relevan karena desain pakai dasar hitam-putih dengan aksen merah)

---

## 10. Arah Desain

Mengacu ke dokumen terpisah: **`design-brief-ecommerce-nike-style.md`** — gaya bold, minimalis, terinspirasi Nike, palet monokrom (hitam-putih) dengan aksen merah `#E4002B`, tipografi sans-serif tebal.

---

## 11. Asumsi & Catatan Terbuka

- **Format nomor WhatsApp**: distandarkan ke format internasional `62xxx` (tanpa `+`, `00`, atau `0` di depan) — ini sesuai spesifikasi resmi link `wa.me` (`https://wa.me/<nomor>`), yang mensyaratkan nomor ditulis lengkap dengan kode negara tanpa simbol tambahan. Kalau customer input `0812...`, sistem otomatis konversi ke `62812...` sebelum disimpan — validasi & konversi ini wajib ada di form checkout
- **Kupon "item tertentu"/"bundle"**: pakai tabel relasi `coupon_products` (bukan array ID di kolom kupon) — lebih fleksibel untuk query dan menghindari perlu parsing array
- **Retensi data `coupon_usages`**: **tidak disimpan permanen** — data dihapus otomatis setiap 3 bulan (perlu scheduled job/cron di backend untuk hapus data yang lebih tua dari 3 bulan). Karena `current_usage_count` di tabel `coupons` sudah menyimpan angka total pemakaian secara terpisah, penghapusan `coupon_usages` lama tidak akan mengacaukan hitungan batas pemakaian total — hanya menghapus histori detail siapa saja yang sudah pakai
  - ⚠️ **Perlu diperhatikan**: kalau ada kupon yang masa berlakunya lebih dari 3 bulan, pembatasan "per customer" (`max_usage_per_customer`) bisa jadi tidak akurat setelah data lama terhapus — customer yang sudah pakai kupon itu 4 bulan lalu berpotensi bisa pakai lagi karena histori pemakaiannya sudah hilang. Kalau ini jadi masalah nyata nanti, solusinya: batasi masa berlaku kupon maksimal 3 bulan, atau simpan ringkasan "customer mana saja yang sudah pakai kupon apa" di tempat terpisah yang tidak ikut terhapus
- Payment gateway & WhatsApp Business API tetap tersedia sebagai jalur upgrade di masa depan, arsitektur di atas sudah dirancang supaya tidak perlu rombak besar saat upgrade nanti
- Aplikasi POS **sudah dibangun lebih dulu** dan sudah didesain terintegrasi dengan database ini sebagai sumber utama (lihat section 7A) — bukan lagi rencana masa depan
- **Login/role tetap terpisah per aplikasi**: e-commerce pakai tabel `admins` (Owner/Staff) miliknya sendiri, POS pakai sistem login kasir yang sudah ada sendiri — yang dibagikan cuma data produk, stok, dan transaksi (`channel`), bukan sistem autentikasinya. Detail teknis auth untuk endpoint `/pos/orders` perlu disesuaikan dengan sistem auth POS yang sudah berjalan
