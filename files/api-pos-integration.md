# API Documentation: Integrasi POS

## 1. Tujuan Dokumen

Dokumen ini khusus untuk tim/developer **aplikasi POS**, berisi endpoint API yang perlu dipanggil supaya POS bisa membaca data produk & stok, serta mencatat transaksi penjualan — ke database yang **sama** dengan yang dipakai website e-commerce (single source of truth).

Ini bukan dokumen lengkap seluruh sistem e-commerce — hanya bagian yang relevan untuk POS. Untuk konteks bisnis lengkap, lihat `prd-ecommerce-wa.md`.

---

## 2. Prinsip Dasar

- **Yang dibagikan (shared)** antara e-commerce dan POS: data produk, varian, stok, dan transaksi/order
- **Yang tidak dibagikan**: sistem login/autentikasi. POS tetap pakai sistem login kasirnya sendiri — dokumen ini tidak mengatur soal login kasir
- **Semua perubahan stok wajib lewat API ini**, POS tidak boleh update tabel database secara langsung. Ini penting supaya tidak terjadi stok minus akibat 2 transaksi (online & POS) terjadi bersamaan pada produk yang sama

---

## 3. Base URL

```
https://[domain-backend-kamu]/api/v1
```

*(ganti dengan domain backend Go yang sudah di-deploy di Vercel)*

---

## 4. Autentikasi

⚠️ **Belum final** — karena POS punya sistem login sendiri (terpisah dari Supabase Auth admin e-commerce), endpoint di dokumen ini butuh mekanisme autentikasi tersendiri, bukan token admin e-commerce.

**Rekomendasi**: pakai **API Key statis** khusus untuk aplikasi POS, dikirim lewat header di setiap request:

```
Authorization: Bearer <POS_API_KEY>
```

API Key ini digenerate sekali oleh tim e-commerce, disimpan sebagai environment variable di aplikasi POS. Backend Go memvalidasi API Key ini di setiap request dari endpoint `/pos/*` (bukan validasi user/kasir individual — itu tetap urusan internal aplikasi POS sendiri).

*(Kalau tim POS punya preferensi mekanisme auth lain, misal mutual TLS atau signed request, ini bisa didiskusikan ulang — API Key statis adalah opsi paling sederhana untuk mulai.)*

---

## 5. Endpoint

### 5.1 Ambil Daftar Produk + Stok

Dipakai POS untuk menampilkan katalog produk ke kasir, termasuk stok terkini.

```
GET /products
```

**Response 200**
```json
{
  "data": [
    {
      "id": "prod_001",
      "name": "Kaos Polos",
      "slug": "kaos-polos",
      "category": "Baju",
      "variants": [
        {
          "id": "var_001",
          "variant_name": "Hitam - L",
          "sku": "KP-HTM-L",
          "price": 85000,
          "stock_quantity": 12
        },
        {
          "id": "var_002",
          "variant_name": "Hitam - M",
          "sku": "KP-HTM-M",
          "price": 85000,
          "stock_quantity": 0
        }
      ]
    }
  ]
}
```

### 5.2 Cek Stok Satu Varian (Real-time)

Berguna kalau POS ingin cek stok terkini sebelum konfirmasi transaksi (menghindari data stok yang sudah usang di cache lokal POS).

```
GET /products/variants/:variant_id/stock
```

**Response 200**
```json
{
  "variant_id": "var_001",
  "stock_quantity": 12
}
```

### 5.3 Buat Transaksi POS

Endpoint utama — dipanggil setiap kali ada transaksi selesai di kasir. Stok otomatis dikurangi secara atomik oleh backend.

```
POST /pos/orders
```

**Request Body**
```json
{
  "items": [
    { "product_variant_id": "var_001", "quantity": 2 },
    { "product_variant_id": "var_005", "quantity": 1 }
  ],
  "payment_method": "cash",
  "customer_name": "Budi (opsional)",
  "customer_whatsapp": "628xxxx (opsional)"
}
```

Catatan field:
- `items`: wajib, minimal 1 item
- `payment_method`: bebas string, contoh `"cash"`, `"qris"`, `"debit"` — untuk keperluan laporan saja, tidak memengaruhi logika sistem
- `customer_name` dan `customer_whatsapp`: **opsional**, kasir tidak wajib input ini (beda dengan checkout online yang mewajibkan)

**Response 201 (Berhasil)**
```json
{
  "order_id": "ord_00123",
  "invoice_number": "POS-20260815-001",
  "channel": "pos",
  "status": "selesai",
  "subtotal": 255000,
  "total": 255000,
  "created_at": "2026-08-15T10:32:00Z"
}
```

**Response 409 (Stok Tidak Cukup)**
```json
{
  "error": "insufficient_stock",
  "message": "Stok tidak cukup untuk salah satu item",
  "details": [
    { "product_variant_id": "var_001", "requested": 5, "available": 2 }
  ]
}
```

Kalau ada satu saja item yang stoknya tidak cukup, **seluruh transaksi dibatalkan** (tidak ada transaksi sebagian) — backend menjalankan ini dalam satu database transaction.

### 5.4 Ambil Detail Transaksi POS

Untuk cetak ulang struk atau cek riwayat transaksi dari sisi POS.

```
GET /pos/orders/:order_id
```

**Response 200**
```json
{
  "order_id": "ord_00123",
  "invoice_number": "POS-20260815-001",
  "channel": "pos",
  "status": "selesai",
  "items": [
    { "product_variant_id": "var_001", "variant_name": "Hitam - L", "quantity": 2, "price_at_purchase": 85000 }
  ],
  "subtotal": 255000,
  "total": 255000,
  "payment_method": "cash",
  "created_at": "2026-08-15T10:32:00Z"
}
```

---

## 6. Real-time Update Stok (Opsional, Rekomendasi)

Selain polling lewat endpoint `GET /products`, POS bisa **subscribe langsung ke Supabase Realtime** pada tabel `product_variants` (fitur bawaan Supabase, tidak perlu lewat backend Go) untuk dapat notifikasi otomatis kalau ada perubahan stok dari channel manapun (online maupun POS lain kalau ada lebih dari satu kasir). Ini murni untuk keperluan baca (read-only) — perubahan/penulisan stok tetap wajib lewat endpoint `POST /pos/orders` di atas.

Dokumentasi resmi: `https://supabase.com/docs/guides/realtime`

---

## 7. Yang TIDAK Termasuk di Endpoint Ini

- **Kupon promo** — endpoint kupon (`/coupons/validate`) didesain untuk checkout online. Belum diputuskan apakah POS juga akan mendukung pemakaian kupon fisik di toko — kalau iya, ini perlu didiskusikan terpisah dan endpoint POS perlu disesuaikan
- **Ongkir** — tidak relevan untuk transaksi POS (barang dibawa langsung)
- **Update status order** — transaksi POS langsung final ("Selesai") saat dibuat, tidak ada alur ubah status seperti order online

---

## 8. Ringkasan Endpoint

| Method | Endpoint | Kegunaan |
|---|---|---|
| GET | `/products` | Ambil katalog produk + stok |
| GET | `/products/variants/:variant_id/stock` | Cek stok real-time satu varian |
| POST | `/pos/orders` | Buat transaksi baru dari POS |
| GET | `/pos/orders/:order_id` | Ambil detail transaksi POS |
