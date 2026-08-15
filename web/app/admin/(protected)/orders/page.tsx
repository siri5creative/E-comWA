import Link from "next/link";
import { goFetchAsAdmin } from "@/lib/api";
import type { OrderListResponse, OrderStatus } from "@/lib/types";
import { formatRupiah } from "@/lib/format";
import { OrderStatusBadge, ORDER_STATUS_LABEL } from "@/components/admin/OrderStatusBadge";

const STATUS_OPTIONS: OrderStatus[] = [
  "menunggu_konfirmasi",
  "menunggu_pembayaran",
  "diproses",
  "dikirim",
  "selesai",
  "dibatalkan",
];

async function getOrders(status?: string, channel?: string, page?: string) {
  const qs = new URLSearchParams();
  if (status) qs.set("status", status);
  if (channel) qs.set("channel", channel);
  if (page) qs.set("page", page);

  const res = await goFetchAsAdmin(`/orders?${qs.toString()}`);
  if (!res.ok) {
    throw new Error("Gagal memuat order");
  }
  return (await res.json()) as OrderListResponse;
}

export default async function AdminOrdersPage({
  searchParams,
}: {
  searchParams: Promise<{ status?: string; channel?: string; page?: string }>;
}) {
  const params = await searchParams;
  const { data: orders, pagination } = await getOrders(
    params.status,
    params.channel,
    params.page
  );

  return (
    <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
      <h1 className="mb-6 text-2xl font-black uppercase tracking-tight">
        Order
      </h1>

      {/* Plain GET form — filters via URL, no client JS needed. */}
      <form className="mb-8 flex flex-wrap gap-3" method="get">
        <select
          name="status"
          defaultValue={params.status ?? ""}
          className="border border-border px-3 py-2 text-sm"
        >
          <option value="">Semua Status</option>
          {STATUS_OPTIONS.map((s) => (
            <option key={s} value={s}>
              {ORDER_STATUS_LABEL[s]}
            </option>
          ))}
        </select>
        <select
          name="channel"
          defaultValue={params.channel ?? ""}
          className="border border-border px-3 py-2 text-sm"
        >
          <option value="">Semua Channel</option>
          <option value="online">Online</option>
          <option value="pos">POS</option>
        </select>
        <button
          type="submit"
          className="bg-ink px-4 py-2 text-xs font-bold uppercase tracking-wide text-paper"
        >
          Filter
        </button>
      </form>

      <div className="overflow-x-auto">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-bold uppercase tracking-wide text-ink/50">
              <th className="py-3 pr-4">Invoice</th>
              <th className="py-3 pr-4">Customer</th>
              <th className="py-3 pr-4">Channel</th>
              <th className="py-3 pr-4">Status</th>
              <th className="py-3 pr-4">Total</th>
              <th className="py-3 pr-4">Tanggal</th>
            </tr>
          </thead>
          <tbody>
            {orders.length === 0 && (
              <tr>
                <td colSpan={6} className="py-8 text-center text-ink/50">
                  Belum ada order.
                </td>
              </tr>
            )}
            {orders.map((order) => (
              <tr key={order.id} className="border-b border-border">
                <td className="py-3 pr-4">
                  <Link
                    href={`/admin/orders/${order.id}`}
                    className="font-bold hover:text-accent"
                  >
                    {order.invoice_number}
                  </Link>
                </td>
                <td className="py-3 pr-4">{order.customer_name ?? "-"}</td>
                <td className="py-3 pr-4 uppercase">{order.channel}</td>
                <td className="py-3 pr-4">
                  <OrderStatusBadge status={order.status} />
                </td>
                <td className="py-3 pr-4 font-bold">
                  {formatRupiah(order.total)}
                </td>
                <td className="py-3 pr-4 text-ink/60">
                  {new Date(order.created_at).toLocaleString("id-ID")}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mt-6 flex items-center justify-between text-sm text-ink/60">
        <span>
          Halaman {pagination.page} &middot; {pagination.total} order
        </span>
        <div className="flex gap-4">
          {pagination.page > 1 && (
            <Link
              href={{
                pathname: "/admin/orders",
                query: { ...params, page: String(pagination.page - 1) },
              }}
              className="font-bold uppercase hover:text-accent"
            >
              Sebelumnya
            </Link>
          )}
          {pagination.page * pagination.page_size < pagination.total && (
            <Link
              href={{
                pathname: "/admin/orders",
                query: { ...params, page: String(pagination.page + 1) },
              }}
              className="font-bold uppercase hover:text-accent"
            >
              Selanjutnya
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}
