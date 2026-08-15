import type { OrderStatus } from "@/lib/types";

const STATUS_LABEL: Record<OrderStatus, string> = {
  menunggu_konfirmasi: "Menunggu Konfirmasi",
  menunggu_pembayaran: "Menunggu Pembayaran",
  diproses: "Diproses",
  dikirim: "Dikirim",
  selesai: "Selesai",
  dibatalkan: "Dibatalkan",
};

const STATUS_STYLE: Record<OrderStatus, string> = {
  menunggu_konfirmasi: "bg-border text-ink",
  menunggu_pembayaran: "bg-border text-ink",
  diproses: "bg-ink text-paper",
  dikirim: "bg-ink text-paper",
  selesai: "bg-accent text-paper",
  dibatalkan: "bg-ink/40 text-paper",
};

export function OrderStatusBadge({ status }: { status: OrderStatus }) {
  return (
    <span
      className={`inline-block px-2 py-1 text-xs font-bold uppercase tracking-wide ${STATUS_STYLE[status]}`}
    >
      {STATUS_LABEL[status]}
    </span>
  );
}

export { STATUS_LABEL as ORDER_STATUS_LABEL };
