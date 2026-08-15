import { notFound } from "next/navigation";
import { goFetch } from "@/lib/api";
import type { Product } from "@/lib/types";
import { ProductPurchasePanel } from "@/components/ProductPurchasePanel";

export const revalidate = 60;

async function getProduct(slug: string): Promise<Product | null> {
  const res = await goFetch(`/products/${slug}`, { next: { revalidate: 60 } });
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error("Gagal memuat produk");
  }
  return (await res.json()) as Product;
}

export default async function ProductDetailPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const product = await getProduct(slug);

  if (!product) {
    notFound();
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
      <div className="grid gap-10 lg:grid-cols-2">
        <div className="aspect-square overflow-hidden bg-border">
          {product.cover_image_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={product.cover_image_url}
              alt={product.name}
              className="h-full w-full object-cover"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-lg font-bold uppercase text-ink/40">
              {product.name}
            </div>
          )}
        </div>

        <div>
          {product.category && (
            <p className="text-xs font-bold uppercase tracking-wide text-ink/50">
              {product.category.name}
            </p>
          )}
          <h1 className="mt-1 text-3xl font-black uppercase tracking-tight">
            {product.name}
          </h1>
          {product.description && (
            <p className="mt-4 text-sm text-ink/70">{product.description}</p>
          )}

          <ProductPurchasePanel product={product} />
        </div>
      </div>
    </div>
  );
}
