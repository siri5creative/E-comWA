import { notFound } from "next/navigation";
import { goFetch } from "@/lib/api";
import type { CategoryListResponse, Product, ProductListResponse } from "@/lib/types";
import { ProductForm } from "@/components/admin/ProductForm";
import { VariantManager } from "@/components/admin/VariantManager";

// There's no admin GET-by-id endpoint on the backend (only the public
// GET /products/:slug) — the catalog is small for a UMKM, so fetching the
// full list and filtering client-side avoids adding a redundant endpoint.
async function getProduct(id: string): Promise<Product | null> {
  const res = await goFetch("/products?page_size=100");
  if (!res.ok) {
    throw new Error("Gagal memuat produk");
  }
  const body = (await res.json()) as ProductListResponse;
  return body.data.find((p) => p.id === id) ?? null;
}

async function getCategories() {
  const res = await goFetch("/categories");
  if (!res.ok) {
    throw new Error("Gagal memuat kategori");
  }
  const body = (await res.json()) as CategoryListResponse;
  return body.data;
}

export default async function EditProductPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const [product, categories] = await Promise.all([getProduct(id), getCategories()]);

  if (!product) {
    notFound();
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <h1 className="mb-8 text-2xl font-black uppercase tracking-tight">
        Edit Produk
      </h1>
      <ProductForm mode="edit" productId={product.id} initial={product} categories={categories} />

      <h2 className="mb-4 mt-12 text-lg font-black uppercase tracking-tight">
        Varian &amp; Stok
      </h2>
      <VariantManager productId={product.id} initial={product.variants} />
    </div>
  );
}
