"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useCart } from "@/lib/cart/CartContext";
import { formatRupiah } from "@/lib/format";
import type {
  ApiError,
  CheckoutOrder,
  CouponInvalidError,
  InsufficientStockError,
  ValidateCouponResponse,
} from "@/lib/types";

type AppliedCoupon = { code: string; discountAmount: number };

export default function CartPage() {
  const { items, updateQuantity, removeItem, subtotal, clear } = useCart();
  const [name, setName] = useState("");
  const [whatsapp, setWhatsapp] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [stockIssues, setStockIssues] = useState<
    InsufficientStockError["details"] | null
  >(null);
  const [order, setOrder] = useState<CheckoutOrder | null>(null);

  const [couponCode, setCouponCode] = useState("");
  const [validatingCoupon, setValidatingCoupon] = useState(false);
  const [couponMessage, setCouponMessage] = useState<string | null>(null);
  const [appliedCoupon, setAppliedCoupon] = useState<AppliedCoupon | null>(null);

  const discount = appliedCoupon?.discountAmount ?? 0;
  const total = Math.max(0, subtotal - discount);

  async function handleApplyCoupon() {
    if (!couponCode.trim()) return;
    setValidatingCoupon(true);
    setCouponMessage(null);

    try {
      const res = await fetch("/api/coupons/validate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          code: couponCode,
          whatsapp_number: whatsapp,
          items: items.map((i) => ({
            product_variant_id: i.productVariantId,
            quantity: i.quantity,
          })),
        }),
      });
      const body = (await res.json()) as ValidateCouponResponse;

      if (body.valid) {
        setAppliedCoupon({ code: body.code, discountAmount: body.discount_amount });
        setCouponMessage(null);
      } else {
        setAppliedCoupon(null);
        setCouponMessage(body.message);
      }
    } catch {
      setAppliedCoupon(null);
      setCouponMessage("Gagal memeriksa kupon, coba lagi.");
    } finally {
      setValidatingCoupon(false);
    }
  }

  function handleRemoveCoupon() {
    setAppliedCoupon(null);
    setCouponCode("");
    setCouponMessage(null);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setErrorMessage(null);
    setStockIssues(null);

    try {
      const res = await fetch("/api/checkout", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name,
          whatsapp_number: whatsapp,
          items: items.map((i) => ({
            product_variant_id: i.productVariantId,
            quantity: i.quantity,
          })),
          coupon_code: appliedCoupon?.code,
        }),
      });

      if (res.status === 201) {
        const created = (await res.json()) as CheckoutOrder;
        setOrder(created);
        clear();
        return;
      }

      if (res.status === 409) {
        const body = await res.json();
        if (body.error === "coupon_invalid") {
          const couponErr = body as CouponInvalidError;
          setAppliedCoupon(null);
          setErrorMessage(
            `Kupon tidak bisa dipakai lagi (${couponErr.message}). Coba checkout ulang tanpa kupon ini.`
          );
          return;
        }
        setStockIssues((body as InsufficientStockError).details);
        return;
      }

      const body = (await res.json()) as ApiError;
      setErrorMessage(body.message || "Terjadi kesalahan, coba lagi.");
    } catch {
      setErrorMessage("Gagal terhubung ke server, coba lagi.");
    } finally {
      setSubmitting(false);
    }
  }

  if (order) {
    return (
      <div className="mx-auto max-w-lg px-4 py-16 sm:px-6">
        <p className="text-xs font-bold uppercase tracking-wide text-accent">
          Order Diterima
        </p>
        <h1 className="mt-2 text-2xl font-black uppercase tracking-tight">
          Terima kasih, {name}!
        </h1>
        <div className="mt-6 space-y-2 border border-border p-6 text-sm">
          <p>
            No. Invoice: <span className="font-bold">{order.invoice_number}</span>
          </p>
          {order.discount_amount > 0 && (
            <p>
              Diskon{order.coupon_code ? ` (${order.coupon_code})` : ""}:{" "}
              <span className="font-bold">-{formatRupiah(order.discount_amount)}</span>
            </p>
          )}
          <p>
            Total: <span className="font-bold">{formatRupiah(order.total)}</span>
          </p>
          <p>Status: Menunggu Konfirmasi</p>
        </div>
        <p className="mt-6 text-sm text-ink/70">
          Simpan nomor invoice ini. Admin akan menghubungi kamu lewat
          WhatsApp untuk konfirmasi pesanan dan info pembayaran.
        </p>
        <Link
          href="/products"
          className="mt-8 inline-block bg-ink px-6 py-3 text-sm font-bold uppercase tracking-wide text-paper"
        >
          Lanjut Belanja
        </Link>
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="mx-auto max-w-lg px-4 py-24 text-center sm:px-6">
        <h1 className="text-2xl font-black uppercase tracking-tight">
          Keranjang Kosong
        </h1>
        <Link
          href="/products"
          className="mt-6 inline-block bg-ink px-6 py-3 text-sm font-bold uppercase tracking-wide text-paper"
        >
          Belanja Sekarang
        </Link>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
      <h1 className="mb-8 text-2xl font-black uppercase tracking-tight">
        Keranjang
      </h1>

      <div className="grid gap-10 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          {items.map((item) => {
            const issue = stockIssues?.find(
              (d) => d.product_variant_id === item.productVariantId
            );
            return (
              <div
                key={item.productVariantId}
                className="flex items-start justify-between gap-4 border-b border-border pb-6"
              >
                <div>
                  <Link
                    href={`/products/${item.productSlug}`}
                    className="text-sm font-bold uppercase tracking-wide hover:text-accent"
                  >
                    {item.productName}
                  </Link>
                  <p className="text-xs text-ink/60">{item.variantName}</p>
                  <p className="mt-1 text-sm font-black">
                    {formatRupiah(item.price)}
                  </p>
                  {issue && (
                    <p className="mt-1 text-xs font-bold text-accent">
                      Stok tersisa {issue.available}, kurangi jumlah
                    </p>
                  )}
                </div>

                <div className="flex flex-col items-end gap-2">
                  <div className="flex items-center border border-border">
                    <button
                      type="button"
                      onClick={() =>
                        updateQuantity(item.productVariantId, item.quantity - 1)
                      }
                      className="px-3 py-1 text-lg font-bold"
                      aria-label="Kurangi jumlah"
                    >
                      −
                    </button>
                    <span className="w-8 text-center text-sm">
                      {item.quantity}
                    </span>
                    <button
                      type="button"
                      onClick={() =>
                        updateQuantity(item.productVariantId, item.quantity + 1)
                      }
                      className="px-3 py-1 text-lg font-bold"
                      aria-label="Tambah jumlah"
                    >
                      +
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeItem(item.productVariantId)}
                    className="text-xs font-bold uppercase text-ink/50 hover:text-accent"
                  >
                    Hapus
                  </button>
                </div>
              </div>
            );
          })}
        </div>

        <div>
          <div className="border border-border p-6">
            <div>
              <label
                htmlFor="coupon"
                className="mb-1 block text-xs font-bold uppercase tracking-wide"
              >
                Kode Kupon
              </label>
              {appliedCoupon ? (
                <div className="flex items-center justify-between border border-ink px-3 py-2 text-sm">
                  <span className="font-bold">{appliedCoupon.code}</span>
                  <button
                    type="button"
                    onClick={handleRemoveCoupon}
                    className="text-xs font-bold uppercase text-ink/50 hover:text-accent"
                  >
                    Hapus
                  </button>
                </div>
              ) : (
                <div className="flex gap-2">
                  <input
                    id="coupon"
                    value={couponCode}
                    onChange={(e) => setCouponCode(e.target.value)}
                    className="w-full border border-border px-3 py-2 text-sm uppercase"
                    placeholder="MISALKUPON20"
                  />
                  <button
                    type="button"
                    disabled={validatingCoupon}
                    onClick={handleApplyCoupon}
                    className="whitespace-nowrap bg-ink px-4 py-2 text-xs font-bold uppercase tracking-wide text-paper disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    {validatingCoupon ? "..." : "Pakai"}
                  </button>
                </div>
              )}
              {couponMessage && (
                <p className="mt-2 text-xs font-bold text-accent">{couponMessage}</p>
              )}
            </div>

            <div className="mt-6 space-y-1 border-t border-border pt-4 text-sm">
              <div className="flex justify-between">
                <span>Subtotal</span>
                <span>{formatRupiah(subtotal)}</span>
              </div>
              {discount > 0 && (
                <div className="flex justify-between text-accent">
                  <span>Diskon</span>
                  <span>-{formatRupiah(discount)}</span>
                </div>
              )}
              <div className="flex justify-between pt-1 font-bold">
                <span>Total</span>
                <span>{formatRupiah(total)}</span>
              </div>
            </div>
            <p className="mt-2 text-xs text-ink/50">
              Ongkir diinfokan admin lewat WhatsApp setelah order dikonfirmasi.
            </p>

            <form onSubmit={handleSubmit} className="mt-6 space-y-4">
              <div>
                <label
                  htmlFor="name"
                  className="mb-1 block text-xs font-bold uppercase tracking-wide"
                >
                  Nama
                </label>
                <input
                  id="name"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full border border-border px-3 py-2 text-sm"
                />
              </div>

              <div>
                <label
                  htmlFor="whatsapp"
                  className="mb-1 block text-xs font-bold uppercase tracking-wide"
                >
                  Nomor WhatsApp
                </label>
                <input
                  id="whatsapp"
                  required
                  placeholder="08xx atau 62xx"
                  value={whatsapp}
                  onChange={(e) => setWhatsapp(e.target.value)}
                  className="w-full border border-border px-3 py-2 text-sm"
                />
              </div>

              {errorMessage && (
                <p className="text-sm font-bold text-accent">{errorMessage}</p>
              )}

              <button
                type="submit"
                disabled={submitting}
                className="w-full bg-accent py-4 text-sm font-bold uppercase tracking-wide text-paper transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
              >
                {submitting ? "Memproses..." : "Checkout"}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
