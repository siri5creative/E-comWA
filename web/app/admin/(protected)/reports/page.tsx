import Link from "next/link";
import { goFetchAsAdmin } from "@/lib/api";
import type {
  ChannelFilter,
  ReportSummary,
  SalesTrendDay,
  TopProduct,
} from "@/lib/types";
import { formatRupiah } from "@/lib/format";
import {
  PERIOD_PRESET_LABELS,
  resolvePeriod,
  type PeriodPreset,
} from "@/lib/reportPeriods";
import { OwnerOnly } from "@/components/admin/OwnerOnly";
import { ORDER_STATUS_LABEL } from "@/components/admin/OrderStatusBadge";
import {
  SalesTrendChart,
  COLOR_ONLINE,
  COLOR_POS,
  COLOR_COMBINED,
} from "@/components/admin/SalesTrendChart";

type SearchParams = {
  preset?: string;
  from?: string;
  to?: string;
  channel?: string;
};

async function getSummary(from: string, to: string, channel?: string) {
  const qs = new URLSearchParams({ from, to });
  if (channel) qs.set("channel", channel);
  const res = await goFetchAsAdmin(`/reports/summary?${qs.toString()}`);
  if (!res.ok) throw new Error("Gagal memuat ringkasan laporan");
  return (await res.json()) as ReportSummary;
}

async function getTopProducts(from: string, to: string, channel?: string) {
  const qs = new URLSearchParams({ from, to });
  if (channel) qs.set("channel", channel);
  const res = await goFetchAsAdmin(`/reports/top-products?${qs.toString()}`);
  if (!res.ok) throw new Error("Gagal memuat produk terlaris");
  const body = (await res.json()) as { data: TopProduct[] };
  return body.data;
}

async function getSalesTrend(from: string, to: string, channel?: string) {
  const qs = new URLSearchParams({ from, to });
  if (channel) qs.set("channel", channel);
  const res = await goFetchAsAdmin(`/reports/sales-trend?${qs.toString()}`);
  if (!res.ok) throw new Error("Gagal memuat tren penjualan");
  const body = (await res.json()) as { data: SalesTrendDay[] };
  return body.data;
}

const PRESETS: PeriodPreset[] = ["today", "7d", "30d", "month"];
const CHANNEL_OPTIONS: { value: ChannelFilter; label: string }[] = [
  { value: "all", label: "Semua" },
  { value: "online", label: "Online" },
  { value: "pos", label: "POS" },
];

export default async function AdminReportsPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const params = await searchParams;
  return (
    <OwnerOnly>
      <ReportsContent params={params} />
    </OwnerOnly>
  );
}

async function ReportsContent({ params }: { params: SearchParams }) {
  const { from, to, preset } = resolvePeriod(params.preset, params.from, params.to);
  const channel = params.channel === "online" || params.channel === "pos" ? params.channel : undefined;
  const channelValue: ChannelFilter = channel ?? "all";

  const [summary, topProducts, trend] = await Promise.all([
    getSummary(from, to, channel),
    getTopProducts(from, to, channel),
    getSalesTrend(from, to, channel),
  ]);

  function periodLink(nextPreset: PeriodPreset) {
    const qs = new URLSearchParams();
    qs.set("preset", nextPreset);
    if (channel) qs.set("channel", channel);
    return `/admin/reports?${qs.toString()}`;
  }

  function channelLink(nextChannel: ChannelFilter) {
    const qs = new URLSearchParams();
    qs.set("preset", preset);
    if (preset === "custom") {
      qs.set("from", from);
      qs.set("to", to);
    }
    if (nextChannel !== "all") qs.set("channel", nextChannel);
    return `/admin/reports?${qs.toString()}`;
  }

  const dates = trend.map((d) => d.date);
  const orderSeries =
    channel == null
      ? [
          { key: "online", label: "Online", color: COLOR_ONLINE, values: trend.map((d) => d.online_orders ?? 0) },
          { key: "pos", label: "POS", color: COLOR_POS, values: trend.map((d) => d.pos_orders ?? 0) },
        ]
      : [{ key: "total", label: "Order", color: COLOR_COMBINED, values: trend.map((d) => d.total_orders) }];
  const revenueSeries =
    channel == null
      ? [
          { key: "online", label: "Online", color: COLOR_ONLINE, values: trend.map((d) => d.online_revenue ?? 0) },
          { key: "pos", label: "POS", color: COLOR_POS, values: trend.map((d) => d.pos_revenue ?? 0) },
        ]
      : [{ key: "total", label: "Revenue", color: COLOR_COMBINED, values: trend.map((d) => d.total_revenue) }];

  return (
    <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
      <h1 className="mb-6 text-2xl font-black uppercase tracking-tight">
        Laporan Keuangan
      </h1>

      {/* Filters — one row above the content, per dataviz skill's interaction.md */}
      <div className="mb-8 flex flex-wrap items-center gap-6">
        <div className="flex gap-2">
          {PRESETS.map((p) => (
            <Link
              key={p}
              href={periodLink(p)}
              className={`px-3 py-1.5 text-xs font-bold uppercase tracking-wide ${
                preset === p ? "bg-ink text-paper" : "border border-border text-ink hover:border-ink"
              }`}
            >
              {PERIOD_PRESET_LABELS[p]}
            </Link>
          ))}
        </div>

        <form className="flex items-center gap-2" method="get">
          <input type="hidden" name="preset" value="custom" />
          {channel && <input type="hidden" name="channel" value={channel} />}
          <input
            type="date"
            name="from"
            defaultValue={from}
            className="border border-border px-2 py-1 text-xs"
          />
          <span className="text-xs text-ink/50">-</span>
          <input
            type="date"
            name="to"
            defaultValue={to}
            className="border border-border px-2 py-1 text-xs"
          />
          <button
            type="submit"
            className="border border-border px-3 py-1.5 text-xs font-bold uppercase tracking-wide hover:border-ink"
          >
            Terapkan
          </button>
        </form>

        <div className="flex gap-2">
          {CHANNEL_OPTIONS.map((opt) => (
            <Link
              key={opt.value}
              href={channelLink(opt.value)}
              className={`px-3 py-1.5 text-xs font-bold uppercase tracking-wide ${
                channelValue === opt.value
                  ? "bg-accent text-paper"
                  : "border border-border text-ink hover:border-ink"
              }`}
            >
              {opt.label}
            </Link>
          ))}
        </div>
      </div>

      {/* Summary cards */}
      <div className="mb-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <SummaryCard label="Total Order" value={summary.order_counts.total.toLocaleString("id-ID")} />
        <SummaryCard label="Revenue Kotor" value={formatRupiah(summary.revenue.gross)} />
        <SummaryCard label="Diskon Kupon Terpakai" value={formatRupiah(summary.revenue.discount_total)} />
        <SummaryCard label="Pendapatan Bersih (Net)" value={formatRupiah(summary.revenue.net)} accent />
      </div>

      <div className="mb-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <SummaryCard label="Total Ongkir (Online)" value={formatRupiah(summary.revenue.shipping_total)} />
        {summary.revenue.by_channel && (
          <>
            <SummaryCard label="Revenue Online" value={formatRupiah(summary.revenue.by_channel.online)} />
            <SummaryCard label="Revenue POS" value={formatRupiah(summary.revenue.by_channel.pos)} />
          </>
        )}
      </div>

      {/* Order status breakdown */}
      <section className="mb-10">
        <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-ink/50">
          Order per Status
        </h2>
        <div className="flex flex-wrap gap-3">
          {Object.entries(summary.order_counts.by_status).map(([status, count]) => (
            <div key={status} className="border border-border px-4 py-2 text-sm">
              <span className="font-bold">{count}</span>{" "}
              <span className="text-ink/60">
                {ORDER_STATUS_LABEL[status as keyof typeof ORDER_STATUS_LABEL] ?? status}
              </span>
            </div>
          ))}
        </div>
      </section>

      {/* Sales trend charts — separate charts for orders vs revenue (different
          scales; dataviz skill: never dual-axis a chart). */}
      <section className="mb-10 grid gap-6 lg:grid-cols-2">
        <SalesTrendChart title="Jumlah Order per Hari" dates={dates} series={orderSeries} />
        <SalesTrendChart
          title="Revenue per Hari"
          dates={dates}
          series={revenueSeries}
          valueFormat="currency"
        />
      </section>

      {/* Top products */}
      <section>
        <h2 className="mb-3 text-xs font-bold uppercase tracking-wide text-ink/50">
          Produk Terlaris
        </h2>
        <div className="overflow-x-auto border border-border">
          <table className="w-full min-w-[480px] text-left text-sm">
            <thead>
              <tr className="border-b border-border text-xs font-bold uppercase tracking-wide text-ink/50">
                <th className="px-4 py-3">Produk</th>
                <th className="px-4 py-3">Terjual</th>
                <th className="px-4 py-3">Revenue</th>
              </tr>
            </thead>
            <tbody>
              {topProducts.length === 0 && (
                <tr>
                  <td colSpan={3} className="px-4 py-8 text-center text-ink/50">
                    Belum ada data penjualan di periode ini.
                  </td>
                </tr>
              )}
              {topProducts.map((p) => (
                <tr key={p.variant_id} className="border-b border-border">
                  <td className="px-4 py-3">
                    {p.product_name}
                    <span className="text-ink/50"> &middot; {p.variant_name}</span>
                  </td>
                  <td className="px-4 py-3">{p.quantity_sold}</td>
                  <td className="px-4 py-3 font-bold">{formatRupiah(p.revenue)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

function SummaryCard({
  label,
  value,
  accent = false,
}: {
  label: string;
  value: string;
  accent?: boolean;
}) {
  return (
    <div className="border border-border p-4">
      <p className="text-xs font-bold uppercase tracking-wide text-ink/50">{label}</p>
      <p className={`mt-2 text-xl font-black ${accent ? "text-accent" : ""}`}>{value}</p>
    </div>
  );
}
