import { goFetch } from "@/lib/api";
import type { CategoryListResponse } from "@/lib/types";
import { ProductForm } from "@/components/admin/ProductForm";

async function getCategories() {
  const res = await goFetch("/categories");
  if (!res.ok) {
    throw new Error("Gagal memuat kategori");
  }
  const body = (await res.json()) as CategoryListResponse;
  return body.data;
}

export default async function NewProductPage() {
  const categories = await getCategories();

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <h1 className="mb-8 text-2xl font-black uppercase tracking-tight">
        Tambah Produk
      </h1>
      <ProductForm mode="create" categories={categories} />
    </div>
  );
}
