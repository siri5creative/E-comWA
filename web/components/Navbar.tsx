"use client";

import Link from "next/link";
import { useCart } from "@/lib/cart/CartContext";

export function Navbar() {
  const { itemCount } = useCart();

  return (
    <header className="sticky top-0 z-40 bg-ink text-paper">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4 sm:px-6">
        <Link
          href="/"
          className="text-lg font-black uppercase tracking-tight"
        >
          Toko Online
        </Link>

        <nav className="flex items-center gap-6">
          <Link
            href="/products"
            className="text-sm font-bold uppercase tracking-wide hover:text-accent"
          >
            Produk
          </Link>
          <Link
            href="/cart"
            className="relative text-sm font-bold uppercase tracking-wide hover:text-accent"
          >
            Keranjang
            {itemCount > 0 && (
              <span className="absolute -right-4 -top-2 flex h-5 min-w-5 items-center justify-center rounded-full bg-accent px-1 text-xs font-bold text-paper">
                {itemCount}
              </span>
            )}
          </Link>
        </nav>
      </div>
    </header>
  );
}
