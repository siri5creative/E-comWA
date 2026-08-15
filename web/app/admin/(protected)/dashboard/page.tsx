import Link from "next/link";
import { goFetchAsAdmin } from "@/lib/api";
import type { AdminMe } from "@/lib/types";

export default async function AdminDashboardPage() {
  // Memoized by Next.js's fetch cache within this request — no second
  // round trip beyond what app/admin/(protected)/layout.tsx already did.
  const res = await goFetchAsAdmin("/admin/me");
  const admin = (await res.json()) as AdminMe;

  return (
    <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6">
      <h1 className="text-2xl font-black uppercase tracking-tight">
        Halo, {admin.name}
      </h1>
      <p className="mt-1 text-sm text-ink/60">
        Masuk sebagai {admin.role === "owner" ? "Owner" : "Staff"}
      </p>

      <div className="mt-10 grid gap-4 sm:grid-cols-2">
        <Link
          href="/admin/orders"
          className="border border-border p-6 transition-colors hover:border-ink"
        >
          <h2 className="text-sm font-bold uppercase tracking-wide">
            Kelola Order
          </h2>
          <p className="mt-1 text-sm text-ink/60">
            Lihat order masuk, konfirmasi pembayaran, update status, dan
            kirim update ke customer via WhatsApp.
          </p>
        </Link>
      </div>
    </div>
  );
}
