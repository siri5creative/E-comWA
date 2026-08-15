import Link from "next/link";
import { goFetchAsAdmin } from "@/lib/api";
import type { Coupon } from "@/lib/types";
import { formatRupiah } from "@/lib/format";
import { OwnerOnly } from "@/components/admin/OwnerOnly";

async function getCoupons(): Promise<Coupon[]> {
  const res = await goFetchAsAdmin("/coupons");
  if (!res.ok) {
    throw new Error("Gagal memuat kupon");
  }
  const body = (await res.json()) as { data: Coupon[] };
  return body.data;
}

function discountLabel(c: Coupon): string {
  const value =
    c.discount_value_type === "percentage"
      ? `${c.discount_value}%`
      : formatRupiah(c.discount_value);
  return value;
}

export default async function AdminCouponsPage() {
  return (
    <OwnerOnly>
      <CouponsList />
    </OwnerOnly>
  );
}

async function CouponsList() {
  const coupons = await getCoupons();

  return (
    <div className="mx-auto max-w-5xl px-4 py-12 sm:px-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-black uppercase tracking-tight">Kupon</h1>
        <Link
          href="/admin/coupons/new"
          className="bg-accent px-4 py-2 text-xs font-bold uppercase tracking-wide text-paper"
        >
          + Tambah Kupon
        </Link>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead>
            <tr className="border-b border-border text-xs font-bold uppercase tracking-wide text-ink/50">
              <th className="py-3 pr-4">Kode</th>
              <th className="py-3 pr-4">Jenis</th>
              <th className="py-3 pr-4">Nilai</th>
              <th className="py-3 pr-4">Pemakaian</th>
              <th className="py-3 pr-4">Berlaku Sampai</th>
              <th className="py-3 pr-4">Status</th>
            </tr>
          </thead>
          <tbody>
            {coupons.length === 0 && (
              <tr>
                <td colSpan={6} className="py-8 text-center text-ink/50">
                  Belum ada kupon.
                </td>
              </tr>
            )}
            {coupons.map((c) => (
              <tr key={c.id} className="border-b border-border">
                <td className="py-3 pr-4">
                  <Link
                    href={`/admin/coupons/${c.id}`}
                    className="font-bold hover:text-accent"
                  >
                    {c.code}
                  </Link>
                </td>
                <td className="py-3 pr-4">{c.discount_type}</td>
                <td className="py-3 pr-4">{discountLabel(c)}</td>
                <td className="py-3 pr-4">
                  {c.current_usage_count}
                  {c.max_total_usage ? ` / ${c.max_total_usage}` : ""}
                </td>
                <td className="py-3 pr-4 text-ink/60">
                  {new Date(c.valid_until).toLocaleDateString("id-ID")}
                </td>
                <td className="py-3 pr-4">
                  {c.is_active ? (
                    <span className="text-xs font-bold uppercase text-accent">
                      Aktif
                    </span>
                  ) : (
                    <span className="text-xs font-bold uppercase text-ink/40">
                      Nonaktif
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
