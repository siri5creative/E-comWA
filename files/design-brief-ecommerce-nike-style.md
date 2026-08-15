# Design Brief: E-commerce UMKM dengan Gaya Nike-Inspired (untuk Google Stitch)

## Cara Pakai Dokumen Ini

Dokumen ini bisa langsung dipakai sebagai prompt di Google Stitch untuk generate desain UI homepage e-commerce. Gayanya terinspirasi dari Nike.com — bold, clean, minimalis — tapi fiturnya disederhanakan sesuai skala toko UMKM (bukan brand raksasa dengan ribuan produk).

---

## 1. Apa yang Diambil dari Nike, Apa yang Tidak

| Diambil ✅ | Tidak Diambil ❌ |
|---|---|
| Hero besar dengan foto produk jadi fokus utama | Mega-menu bertingkat (Men/Women/Kids + puluhan sub kategori) |
| Tipografi tebal, bold, ukuran besar (statement headline) | Fitur kompleks seperti "By You" (kustomisasi produk) atau shoe finder |
| Whitespace lega, layout grid presisi | Multi-bahasa/multi-negara |
| Warna dasar monokrom (hitam-putih) dengan 1 warna aksen | Konten sponsorship/endorsement atlet |
| Kartu produk simpel: foto besar, nama produk, harga | Sistem member/loyalty tier yang rumit |
| Transisi/hover halus saat interaksi | Carousel dengan banyak sekali slide promo sekaligus |

Intinya: **rasa visualnya bold & premium seperti Nike, tapi strukturnya jauh lebih ramping** — sesuai jumlah produk dan kompleksitas bisnis yang lebih kecil.

---

## 2. Arah Desain

- **Bold & confident** — headline besar, tebal, kadang huruf kapital semua untuk judul utama
- **Monokrom + 1 warna aksen** — dasar hitam/putih, satu warna aksen kuat untuk CTA dan highlight (bisa disesuaikan warna brand kamu)
- **Foto produk sebagai bintang utama** — foto besar, framing rapi, tidak banyak elemen dekoratif yang mengganggu
- **Whitespace besar** — jangan padat, biarkan produk "bernapas"
- **Micro-interaction tegas tapi halus** — hover produk sedikit membesar (scale), transisi warna tombol cepat dan jelas
- **Grid presisi** — semua kartu produk ukuran sama, sejajar rapi, tidak asimetris berlebihan (beda dengan brief perumahan sebelumnya yang lebih "quiet luxury")

---

## 3. Palet Warna

| Peran | Warna | Contoh Hex |
|---|---|---|
| Dasar utama | Hitam pekat | `#111111` |
| Dasar sekunder | Putih bersih | `#FFFFFF` |
| Aksen (CTA, badge promo, highlight) | Merah tegas | `#E4002B` |
| Teks di background terang | Hitam pekat | `#111111` |
| Teks di background gelap | Putih | `#FFFFFF` |
| Abu-abu netral (border, divider) | Abu muda | `#E5E5E5` |

Warna aksen merah `#E4002B` ini sudah final untuk toko kamu — prinsip "monokrom + 1 aksen kuat" tetap dipertahankan.

---

## 4. Tipografi

- **Headline**: Sans-serif tebal/bold (contoh gaya: Helvetica Neue Bold, Neue Montreal Bold, atau Inter Black) — ukuran besar, tracking rapat, kadang UPPERCASE untuk judul hero
- **Body/UI text**: Sans-serif yang sama tapi weight regular/medium, supaya konsisten satu keluarga font
- Hindari serif sama sekali — gaya Nike murni sans-serif tebal dan tegas

---

## 5. Komponen Global

- **Navbar** — logo di tengah atau kiri, menu simpel (Produk, Promo, Cara Belanja, Kontak), ikon keranjang/cart di kanan. Solid hitam atau putih, bukan transparan-berubah seperti brief sebelumnya — biar lebih tegas dan "berani"
- **Floating WhatsApp Button** — tetap ada, tapi desainnya menyesuaikan (bisa warna hitam dengan ikon WA, bukan hijau standar, biar konsisten dengan palet monokrom — atau tetap hijau standar WA supaya gampang dikenali, pilih salah satu)
- **Kartu Produk** — foto besar (dominan), nama produk, harga tebal, badge diskon di pojok kalau ada promo (contoh gaya badge Nike: kotak kecil warna aksen, teks putih tebal)
- **Badge Promo/Kupon** — kotak kecil mencolok dengan warna aksen, teks bold, contoh: "DISKON 20%"

---

## 6. Brief Homepage (Disesuaikan Skala UMKM)

### 6.1 Hero Section
- Foto produk unggulan full-width, dominan di layar
- Headline besar bold (1 baris pendek, contoh: "KOLEKSI TERBARU")
- Subheadline singkat
- 1 tombol CTA besar dan tegas: "BELANJA SEKARANG"
- **Beda dengan Nike**: cukup 1 hero statis atau carousel maksimal 2-3 slide (bukan carousel panjang seperti Nike yang punya ratusan campaign)

### 6.2 Kategori Produk (Navigasi Cepat)
- Baris kecil berisi 3-5 kategori saja (bukan mega-menu), ditampilkan sebagai pill/tombol horizontal di bawah hero

### 6.3 Banner Promo/Kupon Aktif
- Section khusus dengan background aksen kuat, badge kupon besar dan jelas

### 6.4 Produk Unggulan
- Grid produk 3-4 kolom desktop, 2 kolom mobile
- Kartu produk gaya Nike: foto dominan, sedikit teks, harga tebal
- Hover: foto sedikit zoom/scale halus

### 6.5 Cara Belanja
- 3 langkah singkat dengan angka besar bold (gaya numbered step ala editorial), bukan ikon kecil

### 6.6 Testimoni
- Kartu testimoni sederhana, foto bulat kecil, kutipan singkat, tetap dalam gaya tipografi bold yang konsisten

### 6.7 Footer
- Dasar hitam pekat, teks putih, layout kolom rapi (Tentang, Bantuan, Kontak, Sosial Media)

---

## 7. Catatan Responsif

- Mobile: headline hero tetap besar dan bold, tapi ukuran font diturunkan proporsional
- Kartu produk mobile: 2 kolom, foto tetap dominan
- Navbar mobile: hamburger menu simpel, tidak ada mega-menu sama sekali

---

## 8. Contoh Prompt Siap Pakai untuk Google Stitch

```
Desain halaman Home untuk website e-commerce UMKM bernama "[NAMA TOKO]".
Gaya: bold, clean, minimalis, terinspirasi Nike.com tapi jauh lebih sederhana
(skala toko kecil, bukan brand raksasa).
Warna: dasar hitam dan putih monokrom, satu warna aksen merah tegas (#E4002B)
untuk tombol CTA dan badge promo.
Tipografi: sans-serif tebal/bold, ukuran besar untuk headline, kadang huruf kapital semua.

Struktur halaman:
1. Hero full-width dengan foto produk besar, headline pendek dan tebal,
   satu tombol CTA besar "BELANJA SEKARANG"
2. Baris kategori produk sederhana (3-5 kategori saja, bentuk pill/tombol horizontal)
3. Banner promo/kupon dengan background warna aksen, badge diskon besar dan jelas
4. Grid produk unggulan (3-4 kolom desktop, 2 kolom mobile), kartu produk minimalis:
   foto besar dominan, nama produk, harga tebal, badge diskon di pojok kalau ada
5. Section "Cara Belanja" dengan 3 langkah bernomor besar (gaya editorial)
6. Section testimoni sederhana
7. Navbar solid (hitam atau putih), menu simpel, ikon keranjang di kanan
8. Footer dasar hitam pekat dengan teks putih, layout kolom rapi
9. Floating WhatsApp button di kanan bawah

Tone visual: berani, tegas, premium tapi tetap sederhana — tidak serumit
brand raksasa, cocok untuk toko kecil dengan katalog produk terbatas.
Layout: mobile-first, grid presisi, whitespace lega, hover state halus di kartu produk.
```

---

## 9. Perbandingan Singkat dengan Brief Sebelumnya

Kamu sebelumnya juga punya brief untuk project perumahan dengan tema "Elegant Modern" (charcoal + gold, tipografi serif tipis). Brief ini **beda arah** — lebih bold, monokrom + aksen kuat, tipografi sans-serif tebal. Kalau kedua project ini terpisah (perumahan vs e-commerce), tidak masalah punya identitas visual berbeda. Tapi kalau ternyata ini masih project yang sama, kasih tahu saya supaya temanya diselaraskan.
