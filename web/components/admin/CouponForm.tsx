"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import type {
  ApiError,
  Coupon,
  CouponDiscountType,
  CouponDiscountValueType,
  Product,
} from "@/lib/types";

function toDatetimeLocalValue(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function defaultDatetimeLocal(daysFromNow: number): string {
  const d = new Date(Date.now() + daysFromNow * 24 * 60 * 60 * 1000);
  return toDatetimeLocalValue(d.toISOString());
}

const DISCOUNT_TYPE_OPTIONS: { value: CouponDiscountType; label: string }[] = [
  { value: "total_belanja", label: "Potongan Total Belanja" },
  { value: "item_tertentu", label: "Potongan Item Tertentu" },
  { value: "event", label: "Potongan Event/Hari Tertentu" },
  { value: "bundle", label: "Potongan Paket/Bundle" },
];

const needsProducts = (type: CouponDiscountType) =>
  type === "item_tertentu" || type === "bundle";

export function CouponForm({
  mode,
  couponId,
  initial,
  products,
}: {
  mode: "create" | "edit";
  couponId?: string;
  initial?: Coupon;
  products: Product[];
}) {
  const router = useRouter();
  const [code, setCode] = useState(initial?.code ?? "");
  const [discountType, setDiscountType] = useState<CouponDiscountType>(
    initial?.discount_type ?? "total_belanja"
  );
  const [discountValueType, setDiscountValueType] = useState<CouponDiscountValueType>(
    initial?.discount_value_type ?? "fixed"
  );
  const [discountValue, setDiscountValue] = useState(
    initial ? String(initial.discount_value) : ""
  );
  const [minSpend, setMinSpend] = useState(initial ? String(initial.min_spend) : "0");
  const [validFrom, setValidFrom] = useState(
    initial ? toDatetimeLocalValue(initial.valid_from) : defaultDatetimeLocal(0)
  );
  const [validUntil, setValidUntil] = useState(
    initial ? toDatetimeLocalValue(initial.valid_until) : defaultDatetimeLocal(30)
  );
  const [maxTotalUsage, setMaxTotalUsage] = useState(
    initial?.max_total_usage != null ? String(initial.max_total_usage) : ""
  );
  const [maxUsagePerCustomer, setMaxUsagePerCustomer] = useState(
    initial?.max_usage_per_customer != null ? String(initial.max_usage_per_customer) : ""
  );
  const [isActive, setIsActive] = useState(initial?.is_active ?? true);
  const [productIds, setProductIds] = useState<Set<string>>(
    new Set(initial?.product_ids ?? [])
  );

  const [submitting, setSubmitting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function toggleProduct(id: string) {
    setProductIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    const body = {
      code,
      discount_type: discountType,
      discount_value_type: discountValueType,
      discount_value: Number(discountValue),
      min_spend: Number(minSpend || 0),
      valid_from: new Date(validFrom).toISOString(),
      valid_until: new Date(validUntil).toISOString(),
      max_total_usage: maxTotalUsage ? Number(maxTotalUsage) : null,
      max_usage_per_customer: maxUsagePerCustomer ? Number(maxUsagePerCustomer) : null,
      is_active: isActive,
      product_ids: needsProducts(discountType) ? Array.from(productIds) : [],
    };

    try {
      const url = mode === "create" ? "/api/coupons" : `/api/coupons/${couponId}`;
      const res = await fetch(url, {
        method: mode === "create" ? "POST" : "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = (await res.json()) as ApiError;
        setError(err.message ?? "Gagal menyimpan kupon.");
        return;
      }
      router.push("/admin/coupons");
      router.refresh();
    } catch {
      setError("Gagal terhubung ke server.");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete() {
    if (!couponId || !confirm("Hapus kupon ini?")) return;
    setDeleting(true);
    setError(null);
    try {
      const res = await fetch(`/api/coupons/${couponId}`, { method: "DELETE" });
      if (!res.ok && res.status !== 204) {
        const err = (await res.json()) as ApiError;
        setError(err.message ?? "Gagal menghapus kupon.");
        return;
      }
      router.push("/admin/coupons");
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
            Kode Kupon
          </label>
          <input
            required
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            className="w-full border border-border px-3 py-2 text-sm uppercase"
          />
        </div>

        <div className="flex items-end gap-2">
          <input
            id="is_active"
            type="checkbox"
            checked={isActive}
            onChange={(e) => setIsActive(e.target.checked)}
            className="h-4 w-4"
          />
          <label htmlFor="is_active" className="text-xs font-bold uppercase tracking-wide">
            Aktif
          </label>
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Jenis Potongan
          </label>
          <select
            value={discountType}
            onChange={(e) => setDiscountType(e.target.value as CouponDiscountType)}
            className="w-full border border-border px-3 py-2 text-sm"
          >
            {DISCOUNT_TYPE_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Tipe Nilai
          </label>
          <select
            value={discountValueType}
            onChange={(e) => setDiscountValueType(e.target.value as CouponDiscountValueType)}
            className="w-full border border-border px-3 py-2 text-sm"
          >
            <option value="fixed">Nominal Rupiah</option>
            <option value="percentage">Persentase</option>
          </select>
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            {discountValueType === "percentage" ? "Nilai Diskon (%)" : "Nilai Diskon (Rp)"}
          </label>
          <input
            required
            type="number"
            min={0}
            max={discountValueType === "percentage" ? 100 : undefined}
            value={discountValue}
            onChange={(e) => setDiscountValue(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Minimum Belanja (Rp, 0 = tanpa minimum)
          </label>
          <input
            type="number"
            min={0}
            value={minSpend}
            onChange={(e) => setMinSpend(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Berlaku Dari
          </label>
          <input
            required
            type="datetime-local"
            value={validFrom}
            onChange={(e) => setValidFrom(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Berlaku Sampai
          </label>
          <input
            required
            type="datetime-local"
            value={validUntil}
            onChange={(e) => setValidUntil(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Maks. Pemakaian Total (kosongkan = tanpa batas)
          </label>
          <input
            type="number"
            min={1}
            value={maxTotalUsage}
            onChange={(e) => setMaxTotalUsage(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
            Maks. Pemakaian per Customer (kosongkan = tanpa batas)
          </label>
          <input
            type="number"
            min={1}
            value={maxUsagePerCustomer}
            onChange={(e) => setMaxUsagePerCustomer(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm"
          />
        </div>
      </div>

      {needsProducts(discountType) && (
        <div>
          <label className="mb-2 block text-xs font-bold uppercase tracking-wide">
            Produk Terkait{" "}
            {discountType === "bundle"
              ? "(semua produk ini harus dibeli sekaligus)"
              : "(berlaku jika salah satu produk ini dibeli)"}
          </label>
          {products.length === 0 ? (
            <p className="text-sm text-ink/50">Belum ada produk.</p>
          ) : (
            <div className="grid max-h-64 gap-2 overflow-y-auto border border-border p-3 sm:grid-cols-2">
              {products.map((p) => (
                <label key={p.id} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={productIds.has(p.id)}
                    onChange={() => toggleProduct(p.id)}
                    className="h-4 w-4"
                  />
                  {p.name}
                </label>
              ))}
            </div>
          )}
        </div>
      )}

      {error && <p className="text-sm font-bold text-accent">{error}</p>}

      <div className="flex items-center gap-4">
        <button
          type="submit"
          disabled={submitting}
          className="bg-accent px-6 py-3 text-sm font-bold uppercase tracking-wide text-paper transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {submitting ? "Menyimpan..." : "Simpan Kupon"}
        </button>
        {mode === "edit" && (
          <button
            type="button"
            disabled={deleting}
            onClick={handleDelete}
            className="text-xs font-bold uppercase tracking-wide text-ink/50 hover:text-accent disabled:cursor-not-allowed disabled:opacity-40"
          >
            {deleting ? "Menghapus..." : "Hapus Kupon"}
          </button>
        )}
      </div>
    </form>
  );
}
