import { goFetch } from "@/lib/api";
import type { CategoryListResponse, ProductListResponse } from "@/lib/types";
import { ProductGrid } from "@/components/ProductGrid";
import { CategoryFilter } from "@/components/CategoryFilter";

export const revalidate = 60;

async function getCategories() {
  const res = await goFetch("/categories", { next: { revalidate: 300 } });
  if (!res.ok) return [];
  const body = (await res.json()) as CategoryListResponse;
  return body.data;
}

async function getProducts(category?: string, page?: string) {
  const qs = new URLSearchParams();
  if (category) qs.set("category", category);
  if (page) qs.set("page", page);

  const res = await goFetch(`/products?${qs.toString()}`, {
    next: { revalidate: 60 },
  });
  if (!res.ok) {
    throw new Error("Gagal memuat produk");
  }
  return (await res.json()) as ProductListResponse;
}

export default async function ProductsPage({
  searchParams,
}: {
  searchParams: Promise<{ category?: string; page?: string }>;
}) {
  const params = await searchParams;
  const [categories, productList] = await Promise.all([
    getCategories(),
    getProducts(params.category, params.page),
  ]);

  return (
    <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
      <h1 className="mb-6 text-2xl font-black uppercase tracking-tight">
        Produk
      </h1>
      <div className="mb-8">
        <CategoryFilter categories={categories} activeSlug={params.category} />
      </div>
      <ProductGrid products={productList.data} />
    </div>
  );
}
