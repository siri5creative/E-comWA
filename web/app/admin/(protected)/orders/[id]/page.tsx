import { notFound } from "next/navigation";
import { goFetchAsAdmin } from "@/lib/api";
import type { OrderDetail } from "@/lib/types";
import { formatRupiah } from "@/lib/format";
import { OrderStatusBadge } from "@/components/admin/OrderStatusBadge";
import { OrderActionsPanel } from "@/components/admin/OrderActionsPanel";

async function getOrder(id: string): Promise<OrderDetail | null> {
  const res = await goFetchAsAdmin(`/orders/${id}`);
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error("Gagal memuat order");
  }
  return (await res.json()) as OrderDetail;
}

export default async function AdminOrderDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const order = await getOrder(id);

  if (!order) {
    notFound();
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
      <div className="mb-8 flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-black uppercase tracking-tight">
            {order.invoice_number}
          </h1>
          <p className="mt-1 text-sm text-ink/60">
            {new Date(order.created_at).toLocaleString("id-ID")} &middot;{" "}
            {order.channel === "online" ? "Online" : "POS"}
          </p>
        </div>
        <OrderStatusBadge status={order.status} />
      </div>

      <div className="grid gap-10 lg:grid-cols-3">
        <div className="space-y-8 lg:col-span-2">
          <section>
            <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-ink/50">
              Customer
            </h2>
            {order.customer ? (
              <p className="text-sm">
                {order.customer.name} &middot; {order.customer.whatsapp_number}
              </p>
            ) : (
              <p className="text-sm text-ink/50">Walk-in / tanpa data customer</p>
            )}
          </section>

          <section>
            <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-ink/50">
              Item
            </h2>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[480px] text-left text-sm">
                <thead>
                  <tr className="border-b border-border text-xs font-bold uppercase tracking-wide text-ink/50">
                    <th className="py-2 pr-4">Produk</th>
                    <th className="py-2 pr-4">Jumlah</th>
                    <th className="py-2 pr-4">Harga</th>
                  </tr>
                </thead>
                <tbody>
                  {order.items.map((item) => (
                    <tr key={item.product_variant_id} className="border-b border-border">
                      <td className="py-2 pr-4">
                        {item.product_name}
                        <span className="text-ink/50"> &middot; {item.variant_name}</span>
                      </td>
                      <td className="py-2 pr-4">{item.quantity}</td>
                      <td className="py-2 pr-4">
                        {formatRupiah(item.price_at_purchase)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section className="border border-border p-4 text-sm">
            <div className="flex justify-between py-1">
              <span>Subtotal</span>
              <span>{formatRupiah(order.subtotal)}</span>
            </div>
            <div className="flex justify-between py-1">
              <span>Diskon</span>
              <span>-{formatRupiah(order.discount_amount)}</span>
            </div>
            <div className="flex justify-between py-1">
              <span>Ongkir</span>
              <span>{formatRupiah(order.shipping_cost)}</span>
            </div>
            <div className="flex justify-between border-t border-border py-2 font-bold">
              <span>Total</span>
              <span>{formatRupiah(order.total)}</span>
            </div>
          </section>
        </div>

        <OrderActionsPanel order={order} />
      </div>
    </div>
  );
}
