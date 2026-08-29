import Link from "next/link";
import { goFetch } from "@/lib/api";
import type { ProductListResponse } from "@/lib/types";
import { ProductGrid } from "@/components/ProductGrid";

export const revalidate = 60;

async function getFeaturedProducts() {
  // This page is statically prerendered, including at build time — when
  // Vercel Services builds web/ and api/ independently (bindings/network
  // calls don't resolve between services during a build, only at request
  // time), the API isn't reachable yet. Falling back to [] here lets the
  // build succeed with an empty section; ISR (revalidate below) fills it
  // in with real data on the first request once both services are live.
  try {
    const res = await goFetch("/products?page_size=8", {
      next: { revalidate: 60 },
    });
    if (!res.ok) {
      return [];
    }
    const body = (await res.json()) as ProductListResponse;
    return body.data;
  } catch {
    return [];
  }
}

export default async function HomePage() {
  const products = await getFeaturedProducts();

  return (
    <div>
      <section className="bg-ink text-paper">
        <div className="mx-auto flex max-w-6xl flex-col items-start gap-6 px-4 py-24 sm:px-6">
          <h1 className="text-4xl font-black uppercase leading-none tracking-tight sm:text-6xl">
            Koleksi Terbaru
          </h1>
          <p className="max-w-md text-base text-paper/70">
            Belanja online, konfirmasi pesanan cepat lewat WhatsApp.
          </p>
          <Link
            href="/products"
            className="bg-accent px-8 py-4 text-sm font-bold uppercase tracking-wide text-paper transition-opacity hover:opacity-90"
          >
            Belanja Sekarang
          </Link>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-4 py-16 sm:px-6">
        <h2 className="mb-8 text-xl font-black uppercase tracking-tight">
          Produk Unggulan
        </h2>
        <ProductGrid products={products} />
      </section>
    </div>
  );
}
