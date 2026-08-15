"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import type { Product } from "@/lib/types";
import { useCart } from "@/lib/cart/CartContext";
import { formatRupiah } from "@/lib/format";

export function ProductPurchasePanel({ product }: { product: Product }) {
  const { addItem } = useCart();
  const router = useRouter();

  const firstInStock = product.variants.find((v) => v.stock_quantity > 0);
  const [selectedVariantId, setSelectedVariantId] = useState(
    firstInStock?.id ?? product.variants[0]?.id ?? ""
  );
  const [quantity, setQuantity] = useState(1);
  const [justAdded, setJustAdded] = useState(false);

  const selectedVariant = useMemo(
    () => product.variants.find((v) => v.id === selectedVariantId),
    [product.variants, selectedVariantId]
  );

  const allOutOfStock = product.variants.every((v) => v.stock_quantity === 0);
  const selectedOutOfStock = !selectedVariant || selectedVariant.stock_quantity === 0;

  function handleSelectVariant(variantId: string) {
    setSelectedVariantId(variantId);
    setQuantity(1);
    setJustAdded(false);
  }

  function handleAddToCart() {
    if (!selectedVariant || selectedOutOfStock) return;
    addItem({
      productVariantId: selectedVariant.id,
      productSlug: product.slug,
      productName: product.name,
      variantName: selectedVariant.variant_name,
      price: selectedVariant.price,
      quantity,
      stockQuantity: selectedVariant.stock_quantity,
    });
    setJustAdded(true);
  }

  return (
    <div className="mt-6 space-y-6">
      <p className="text-2xl font-black">
        {selectedVariant ? formatRupiah(selectedVariant.price) : "-"}
      </p>

      {product.variants.length > 0 && (
        <div>
          <p className="mb-2 text-xs font-bold uppercase tracking-wide text-ink/60">
            Varian
          </p>
          <div className="flex flex-wrap gap-2">
            {product.variants.map((variant) => {
              const isSelected = variant.id === selectedVariantId;
              const isOut = variant.stock_quantity === 0;
              return (
                <button
                  key={variant.id}
                  type="button"
                  disabled={isOut}
                  onClick={() => handleSelectVariant(variant.id)}
                  className={`rounded-full border px-4 py-2 text-xs font-bold uppercase tracking-wide transition-colors ${
                    isSelected
                      ? "border-ink bg-ink text-paper"
                      : "border-border text-ink hover:border-ink"
                  } ${isOut ? "cursor-not-allowed opacity-40" : ""}`}
                >
                  {variant.variant_name}
                  {isOut ? " (Habis)" : ""}
                </button>
              );
            })}
          </div>
        </div>
      )}

      {allOutOfStock ? (
        <p className="text-sm font-bold uppercase text-accent">Stok Habis</p>
      ) : selectedOutOfStock ? (
        <p className="text-sm font-bold uppercase text-accent">
          Varian ini sedang habis, pilih varian lain
        </p>
      ) : (
        <div className="flex items-center gap-3">
          <label
            htmlFor="quantity"
            className="text-xs font-bold uppercase tracking-wide text-ink/60"
          >
            Jumlah
          </label>
          <div className="flex items-center border border-border">
            <button
              type="button"
              onClick={() => setQuantity((q) => Math.max(1, q - 1))}
              className="px-3 py-1 text-lg font-bold"
              aria-label="Kurangi jumlah"
            >
              −
            </button>
            <input
              id="quantity"
              type="number"
              min={1}
              max={selectedVariant?.stock_quantity ?? 1}
              value={quantity}
              onChange={(e) => {
                const next = Number(e.target.value);
                const max = selectedVariant?.stock_quantity ?? 1;
                setQuantity(Number.isFinite(next) ? Math.min(Math.max(1, next), max) : 1);
              }}
              className="w-12 border-x border-border py-1 text-center"
            />
            <button
              type="button"
              onClick={() =>
                setQuantity((q) =>
                  Math.min(selectedVariant?.stock_quantity ?? 1, q + 1)
                )
              }
              className="px-3 py-1 text-lg font-bold"
              aria-label="Tambah jumlah"
            >
              +
            </button>
          </div>
        </div>
      )}

      <button
        type="button"
        disabled={allOutOfStock || selectedOutOfStock}
        onClick={handleAddToCart}
        className="w-full bg-accent py-4 text-sm font-bold uppercase tracking-wide text-paper transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
      >
        {allOutOfStock ? "Stok Habis" : "Tambah ke Keranjang"}
      </button>

      {justAdded && (
        <div className="flex items-center justify-between border border-border p-4 text-sm">
          <span>Ditambahkan ke keranjang.</span>
          <button
            type="button"
            onClick={() => router.push("/cart")}
            className="font-bold uppercase text-accent hover:underline"
          >
            Lihat Keranjang
          </button>
        </div>
      )}
    </div>
  );
}
