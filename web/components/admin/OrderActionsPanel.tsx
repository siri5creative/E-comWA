"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import type { OrderDetail, WhatsAppMessageResponse, ApiError } from "@/lib/types";
import { ORDER_STATUS_LABEL } from "@/components/admin/OrderStatusBadge";

export function OrderActionsPanel({ order }: { order: OrderDetail }) {
  const router = useRouter();
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [shippingCost, setShippingCost] = useState(String(order.shipping_cost));
  const [savingShipping, setSavingShipping] = useState(false);

  const [waMessage, setWaMessage] = useState<WhatsAppMessageResponse | null>(null);
  const [loadingWa, setLoadingWa] = useState(false);
  const [waError, setWaError] = useState<string | null>(null);

  const isFinal = order.status === "selesai" || order.status === "dibatalkan";

  async function updateStatus(nextStatus: string) {
    setUpdating(true);
    setError(null);
    try {
      const res = await fetch(`/api/orders/${order.id}/status`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: nextStatus }),
      });
      if (!res.ok) {
        const body = (await res.json()) as ApiError;
        setError(body.message ?? "Gagal mengubah status.");
        return;
      }
      router.refresh();
    } catch {
      setError("Gagal terhubung ke server.");
    } finally {
      setUpdating(false);
    }
  }

  async function saveShippingCost() {
    const parsed = Number(shippingCost);
    if (!Number.isFinite(parsed) || parsed < 0) {
      setError("Ongkir harus berupa angka >= 0.");
      return;
    }
    setSavingShipping(true);
    setError(null);
    try {
      const res = await fetch(`/api/orders/${order.id}/status`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ shipping_cost: parsed }),
      });
      if (!res.ok) {
        const body = (await res.json()) as ApiError;
        setError(body.message ?? "Gagal menyimpan ongkir.");
        return;
      }
      router.refresh();
    } catch {
      setError("Gagal terhubung ke server.");
    } finally {
      setSavingShipping(false);
    }
  }

  async function prepareWhatsAppMessage() {
    setLoadingWa(true);
    setWaError(null);
    setWaMessage(null);
    try {
      const res = await fetch(`/api/orders/${order.id}/wa-message`);
      const body = await res.json();
      if (!res.ok) {
        setWaError((body as ApiError).message ?? "Gagal menyiapkan pesan.");
        return;
      }
      setWaMessage(body as WhatsAppMessageResponse);
    } catch {
      setWaError("Gagal terhubung ke server.");
    } finally {
      setLoadingWa(false);
    }
  }

  return (
    <div className="space-y-6">
      <section className="border border-border p-4">
        <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-ink/50">
          Ubah Status
        </h2>
        {isFinal ? (
          <p className="text-sm text-ink/50">
            Order sudah final, status tidak bisa diubah lagi.
          </p>
        ) : order.next_statuses.length === 0 ? (
          <p className="text-sm text-ink/50">Tidak ada status lanjutan.</p>
        ) : (
          <div className="flex flex-col gap-2">
            {order.next_statuses.map((status) => (
              <button
                key={status}
                type="button"
                disabled={updating}
                onClick={() => updateStatus(status)}
                className="bg-ink py-2 text-xs font-bold uppercase tracking-wide text-paper transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
              >
                {ORDER_STATUS_LABEL[status]}
              </button>
            ))}
          </div>
        )}
        {error && <p className="mt-3 text-xs font-bold text-accent">{error}</p>}
      </section>

      <section className="border border-border p-4">
        <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-ink/50">
          Ongkir
        </h2>
        <p className="mb-2 text-xs text-ink/50">
          Isi setelah disepakati manual lewat WhatsApp. Total dihitung ulang
          otomatis.
        </p>
        <div className="flex gap-2">
          <input
            type="number"
            min={0}
            disabled={isFinal}
            value={shippingCost}
            onChange={(e) => setShippingCost(e.target.value)}
            className="w-full border border-border px-3 py-2 text-sm disabled:opacity-40"
          />
          <button
            type="button"
            disabled={isFinal || savingShipping}
            onClick={saveShippingCost}
            className="whitespace-nowrap bg-ink px-4 py-2 text-xs font-bold uppercase tracking-wide text-paper disabled:cursor-not-allowed disabled:opacity-40"
          >
            {savingShipping ? "..." : "Simpan"}
          </button>
        </div>
      </section>

      <section className="border border-border p-4">
        <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-ink/50">
          Kirim Update WhatsApp
        </h2>
        <button
          type="button"
          disabled={loadingWa}
          onClick={prepareWhatsAppMessage}
          className="w-full bg-accent py-2 text-xs font-bold uppercase tracking-wide text-paper transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {loadingWa ? "Menyiapkan..." : "Siapkan Pesan"}
        </button>

        {waError && <p className="mt-3 text-xs font-bold text-accent">{waError}</p>}

        {waMessage && (
          <div className="mt-4 space-y-3">
            <p className="whitespace-pre-wrap border border-border p-3 text-xs text-ink/70">
              {waMessage.message}
            </p>
            <a
              href={waMessage.wa_link}
              target="_blank"
              rel="noopener noreferrer"
              className="block w-full bg-ink py-2 text-center text-xs font-bold uppercase tracking-wide text-paper"
            >
              Buka WhatsApp
            </a>
          </div>
        )}
      </section>
    </div>
  );
}
