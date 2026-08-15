import Link from "next/link";
import type { Category } from "@/lib/types";

// Pill/tombol horizontal, design brief section 6.2.
export function CategoryFilter({
  categories,
  activeSlug,
}: {
  categories: Category[];
  activeSlug?: string;
}) {
  if (categories.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-2">
      <Link
        href="/products"
        className={`rounded-full border px-4 py-2 text-xs font-bold uppercase tracking-wide transition-colors ${
          !activeSlug
            ? "border-ink bg-ink text-paper"
            : "border-border text-ink hover:border-ink"
        }`}
      >
        Semua
      </Link>
      {categories.map((category) => (
        <Link
          key={category.id}
          href={`/products?category=${category.slug}`}
          className={`rounded-full border px-4 py-2 text-xs font-bold uppercase tracking-wide transition-colors ${
            activeSlug === category.slug
              ? "border-ink bg-ink text-paper"
              : "border-border text-ink hover:border-ink"
          }`}
        >
          {category.name}
        </Link>
      ))}
    </div>
  );
}
