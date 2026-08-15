import Link from "next/link";
import type { Product } from "@/lib/types";
import { formatRupiah } from "@/lib/format";

function priceLabel(product: Product): string {
  const prices = product.variants.map((v) => v.price);
  if (prices.length === 0) return "";
  const min = Math.min(...prices);
  const max = Math.max(...prices);
  return min === max ? formatRupiah(min) : `Mulai ${formatRupiah(min)}`;
}

export function ProductCard({ product }: { product: Product }) {
  const outOfStock =
    product.variants.length > 0 &&
    product.variants.every((v) => v.stock_quantity === 0);

  return (
    <Link href={`/products/${product.slug}`} className="group block">
      <div className="relative aspect-square overflow-hidden bg-border">
        {product.cover_image_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={product.cover_image_url}
            alt={product.name}
            className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-sm font-bold uppercase text-ink/40">
            {product.name}
          </div>
        )}
        {outOfStock && (
          <span className="absolute left-2 top-2 bg-ink px-2 py-1 text-xs font-bold uppercase text-paper">
            Stok Habis
          </span>
        )}
      </div>
      <div className="mt-3">
        <h3 className="text-sm font-bold uppercase tracking-wide">
          {product.name}
        </h3>
        <p className="mt-1 text-sm font-black">{priceLabel(product)}</p>
      </div>
    </Link>
  );
}
