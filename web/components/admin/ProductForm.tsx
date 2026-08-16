"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import type { ApiError, Category, Product } from "@/lib/types";

function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

export function ProductForm({
  mode,
  productId,
  initial,
  categories,
}: {
  mode: "create" | "edit";
  productId?: string;
  initial?: Product;
  categories: Category[];
}) {
  const router = useRouter();
  const [name, setName] = useState(initial?.name ?? "");
  const [slug, setSlug] = useState(initial?.slug ?? "");
  const [slugTouched, setSlugTouched] = useState(mode === "edit");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [categoryId, setCategoryId] = useState(initial?.category?.id ?? "");
  const [coverImageUrl, setCoverImageUrl] = useState(
    initial?.cover_image_url ?? ""
  );

  const [submitting, setSubmitting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function handleNameChange(value: string) {
    setName(value);
    if (!slugTouched) {
      setSlug(slugify(value));
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    const body = {
      name,
      slug,
      description: description || null,
      category_id: categoryId || null,
      cover_image_url: coverImageUrl || null,
    };

    try {
      const url = mode === "create" ? "/api/products" : `/api/products/${productId}`;
      const res = await fetch(url, {
        method: mode === "create" ? "POST" : "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = (await res.json()) as ApiError;
        setError(err.message ?? "Gagal menyimpan produk.");
        return;
      }
      if (mode === "create") {
        const created = (await res.json()) as { id: string };
        router.push(`/admin/products/${created.id}`);
      } else {
        router.push("/admin/products");
      }
      router.refresh();
    } catch {
      setError("Gagal terhubung ke server.");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete() {
    if (!productId || !confirm("Hapus produk ini beserta semua variannya?")) {
      return;
    }
    setDeleting(true);
    setError(null);
    try {
      const res = await fetch(`/api/products/${productId}`, { method: "DELETE" });
      if (!res.ok && res.status !== 204) {
        const err = (await res.json()) as ApiError;
        setError(err.message ?? "Gagal menghapus produk.");
        return;
      }
      router.push("/admin/products");
      router.refresh();
    } catch {
      setError("Gagal terhubung ke server.");
    } finally {
      setDeleting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Nama Produk
          </label>
          <input
            required
            value={name}
            onChange={(e) => handleNameChange(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Slug
          </label>
          <input
            required
            value={slug}
            onChange={(e) => {
              setSlugTouched(true);
              setSlug(e.target.value);
            }}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Kategori
          </label>
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          >
            <option value="">Tanpa Kategori</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            URL Gambar Cover
          </label>
          <input
            value={coverImageUrl}
            onChange={(e) => setCoverImageUrl(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>

        <div className="sm:col-span-2">
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Deskripsi
          </label>
          <textarea
            rows={4}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>
      </div>

      {error && <p className="text-sm font-bold text-accent">{error}</p>}

      <div className="flex items-center gap-4">
        <button
          type="submit"
          disabled={submitting}
          className="bg-accent px-6 py-3 text-sm font-bold uppercase tracking-wide text-paper transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {submitting ? "Menyimpan..." : "Simpan Produk"}
        </button>
        {mode === "edit" && (
          <button
            type="button"
            disabled={deleting}
            onClick={handleDelete}
            className="text-xs font-bold uppercase tracking-wide text-ink/50 hover:text-accent disabled:cursor-not-allowed disabled:opacity-40"
          >
            {deleting ? "Menghapus..." : "Hapus Produk"}
          </button>
        )}
      </div>
    </form>
  );
}
