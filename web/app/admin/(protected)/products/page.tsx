import Link from "next/link";
import { goFetch } from "@/lib/api";
import type { ProductListResponse } from "@/lib/types";
import { formatRupiah } from "@/lib/format";

async function getProducts() {
  const res = await goFetch("/products?page_size=100");
  if (!res.ok) {
    throw new Error("Gagal memuat produk");
  }
  const body = (await res.json()) as ProductListResponse;
  return body.data;
}

function totalStock(variants: { stock_quantity: number }[]): number {
  return variants.reduce((sum, v) => sum + v.stock_quantity, 0);
}

export default async function AdminProductsPage() {
  const products = await getProducts();

  return (
    <div className="mx-auto max-w-5xl px-4 py-12 sm:px-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-black uppercase tracking-tight">Produk</h1>
        <Link
          href="/admin/products/new"
          className="bg-accent px-4 py-2 text-xs font-bold uppercase tracking-wide text-paper"
        >
          + Tambah Produk
        </Link>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-bold uppercase tracking-wide text-ink/50">
              <th className="py-3 pr-4">Nama</th>
              <th className="py-3 pr-4">Kategori</th>
              <th className="py-3 pr-4">Varian</th>
              <th className="py-3 pr-4">Stok</th>
            </tr>
          </thead>
          <tbody>
            {products.length === 0 && (
              <tr>
                <td colSpan={4} className="py-8 text-center text-ink/50">
                  Belum ada produk.
                </td>
              </tr>
            )}
            {products.map((p) => (
              <tr key={p.id} className="border-b border-border">
                <td className="py-3 pr-4">
                  <Link
                    href={`/admin/products/${p.id}`}
                    className="font-bold hover:text-accent"
                  >
                    {p.name}
                  </Link>
                </td>
                <td className="py-3 pr-4 text-ink/60">
                  {p.category?.name ?? "-"}
                </td>
                <td className="py-3 pr-4">
                  {p.variants.length === 0
                    ? "-"
                    : p.variants
                        .map((v) => `${v.variant_name} (${formatRupiah(v.price)})`)
                        .join(", ")}
                </td>
                <td className="py-3 pr-4">
                  {p.variants.length === 0 ? (
                    "-"
                  ) : totalStock(p.variants) === 0 ? (
                    <span className="text-xs font-bold uppercase text-accent">
                      Habis
                    </span>
                  ) : (
                    totalStock(p.variants)
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
