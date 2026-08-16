"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import type { ApiError, ProductVariant } from "@/lib/types";
import { formatRupiah } from "@/lib/format";

type VariantFormState = {
  variant_name: string;
  sku: string;
  price: string;
  stock_quantity: string;
};

const EMPTY_FORM: VariantFormState = {
  variant_name: "",
  sku: "",
  price: "",
  stock_quantity: "",
};

function toFormState(v: ProductVariant): VariantFormState {
  return {
    variant_name: v.variant_name,
    sku: v.sku ?? "",
    price: String(v.price),
    stock_quantity: String(v.stock_quantity),
  };
}

function toRequestBody(form: VariantFormState) {
  return {
    variant_name: form.variant_name,
    sku: form.sku || null,
    price: Number(form.price),
    stock_quantity: Number(form.stock_quantity),
  };
}

export function VariantManager({
  productId,
  initial,
}: {
  productId: string;
  initial: ProductVariant[];
}) {
  const router = useRouter();
  const [variants, setVariants] = useState(initial);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editForm, setEditForm] = useState<VariantFormState>(EMPTY_FORM);
  const [newForm, setNewForm] = useState<VariantFormState>(EMPTY_FORM);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  function startEdit(v: ProductVariant) {
    setEditingId(v.id);
    setEditForm(toFormState(v));
    setError(null);
  }

  async function handleAdd(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(`/api/products/${productId}/variants`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toRequestBody(newForm)),
      });
      if (!res.ok) {
        const err = (await res.json()) as ApiError;
        setError(err.message ?? "Gagal menambah varian.");
        return;
      }
      const created = (await res.json()) as { id: string };
      setVariants((prev) => [
        ...prev,
        { id: created.id, ...toRequestBody(newForm), sku: newForm.sku || null, price: Number(newForm.price), stock_quantity: Number(newForm.stock_quantity) },
      ]);
      setNewForm(EMPTY_FORM);
      router.refresh();
    } catch {
      setError("Gagal terhubung ke server.");
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveEdit(id: string) {
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(`/api/products/variants/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(toRequestBody(editForm)),
      });
      if (!res.ok) {
        const err = (await res.json()) as ApiError;
        setError(err.message ?? "Gagal menyimpan varian.");
        return;
      }
      setVariants((prev) =>
        prev.map((v) =>
          v.id === id
            ? {
                ...v,
                variant_name: editForm.variant_name,
                sku: editForm.sku || null,
                price: Number(editForm.price),
                stock_quantity: Number(editForm.stock_quantity),
              }
            : v
        )
      );
      setEditingId(null);
      router.refresh();
    } catch {
      setError("Gagal terhubung ke server.");
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm("Hapus varian ini?")) return;
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(`/api/products/variants/${id}`, { method: "DELETE" });
      if (!res.ok && res.status !== 204) {
        const err = (await res.json()) as ApiError;
        setError(err.message ?? "Gagal menghapus varian.");
        return;
      }
      setVariants((prev) => prev.filter((v) => v.id !== id));
      router.refresh();
    } catch {
      setError("Gagal terhubung ke server.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      {error && <p className="mb-4 text-sm font-bold text-accent">{error}</p>}

      <div className="overflow-x-auto">
        <table className="w-full min-w-[560px] text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-bold uppercase tracking-wide text-ink/50">
              <th className="py-3 pr-4">Nama Varian</th>
              <th className="py-3 pr-4">SKU</th>
              <th className="py-3 pr-4">Harga</th>
              <th className="py-3 pr-4">Stok</th>
              <th className="py-3 pr-4"></th>
            </tr>
          </thead>
          <tbody>
            {variants.length === 0 && editingId === null && (
              <tr>
                <td colSpan={5} className="py-6 text-center text-ink/50">
                  Belum ada varian.
                </td>
              </tr>
            )}
            {variants.map((v) =>
              editingId === v.id ? (
                <tr key={v.id} className="border-b border-border">
                  <td className="py-2 pr-4">
                    <input
                      required
                      value={editForm.variant_name}
                      onChange={(e) =>
                        setEditForm((f) => ({ ...f, variant_name: e.target.value }))
                      }
                      className="w-full border border-border px-2 py-1.5 text-sm"
                    />
                  </td>
                  <td className="py-2 pr-4">
                    <input
                      value={editForm.sku}
                      onChange={(e) => setEditForm((f) => ({ ...f, sku: e.target.value }))}
                      className="w-full border border-border px-2 py-1.5 text-sm"
                    />
                  </td>
                  <td className="py-2 pr-4">
                    <input
                      required
                      type="number"
                      min={0}
                      value={editForm.price}
                      onChange={(e) => setEditForm((f) => ({ ...f, price: e.target.value }))}
                      className="w-24 border border-border px-2 py-1.5 text-sm"
                    />
                  </td>
                  <td className="py-2 pr-4">
                    <input
                      required
                      type="number"
                      min={0}
                      value={editForm.stock_quantity}
                      onChange={(e) =>
                        setEditForm((f) => ({ ...f, stock_quantity: e.target.value }))
                      }
                      className="w-20 border border-border px-2 py-1.5 text-sm"
                    />
                  </td>
                  <td className="py-2 pr-4 whitespace-nowrap">
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => handleSaveEdit(v.id)}
                      className="mr-3 text-xs font-bold uppercase tracking-wide text-accent hover:opacity-80 disabled:opacity-40"
                    >
                      Simpan
                    </button>
                    <button
                      type="button"
                      onClick={() => setEditingId(null)}
                      className="text-xs font-bold uppercase tracking-wide text-ink/50 hover:text-ink"
                    >
                      Batal
                    </button>
                  </td>
                </tr>
              ) : (
                <tr key={v.id} className="border-b border-border">
                  <td className="py-3 pr-4 font-bold">{v.variant_name}</td>
                  <td className="py-3 pr-4 text-ink/60">{v.sku ?? "-"}</td>
                  <td className="py-3 pr-4">{formatRupiah(v.price)}</td>
                  <td className="py-3 pr-4">
                    {v.stock_quantity === 0 ? (
                      <span className="text-xs font-bold uppercase text-accent">Habis</span>
                    ) : (
                      v.stock_quantity
                    )}
                  </td>
                  <td className="py-3 pr-4 whitespace-nowrap">
                    <button
                      type="button"
                      onClick={() => startEdit(v)}
                      className="mr-3 text-xs font-bold uppercase tracking-wide text-ink/70 hover:text-accent"
                    >
                      Ubah
                    </button>
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => handleDelete(v.id)}
                      className="text-xs font-bold uppercase tracking-wide text-ink/50 hover:text-accent disabled:opacity-40"
                    >
                      Hapus
                    </button>
                  </td>
                </tr>
              )
            )}
          </tbody>
        </table>
      </div>

      <form onSubmit={handleAdd} className="mt-6 flex flex-wrap items-end gap-3">
        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Nama Varian
          </label>
          <input
            required
            value={newForm.variant_name}
            onChange={(e) => setNewForm((f) => ({ ...f, variant_name: e.target.value }))}
            className="border border-border px-3 py-2 text-sm"
            placeholder="cth. Merah / L"
          />
        </div>
        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">SKU</label>
          <input
            value={newForm.sku}
            onChange={(e) => setNewForm((f) => ({ ...f, sku: e.target.value }))}
            className="w-32 border border-border px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Harga (Rp)
          </label>
          <input
            required
            type="number"
            min={0}
            value={newForm.price}
            onChange={(e) => setNewForm((f) => ({ ...f, price: e.target.value }))}
            className="w-28 border border-border px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">Stok</label>
          <input
            required
            type="number"
            min={0}
            value={newForm.stock_quantity}
            onChange={(e) => setNewForm((f) => ({ ...f, stock_quantity: e.target.value }))}
            className="w-20 border border-border px-3 py-2 text-sm"
          />
        </div>
        <button
          type="submit"
          disabled={busy}
          className="bg-ink px-4 py-2 text-xs font-bold uppercase tracking-wide text-paper disabled:opacity-40"
        >
          + Tambah Varian
        </button>
      </form>
    </div>
  );
}
