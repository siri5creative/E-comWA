"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import type { AdminMe } from "@/lib/types";

export function AdminNav({ admin }: { admin: AdminMe }) {
  const router = useRouter();

  async function handleLogout() {
    const supabase = createClient();
    await supabase.auth.signOut();
    router.push("/admin/login");
    router.refresh();
  }

  return (
    <header className="bg-ink text-paper">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4 sm:px-6">
        <div className="flex items-center gap-8">
          <Link href="/admin/dashboard" className="text-lg font-black uppercase tracking-tight">
            Admin
          </Link>
          <nav className="flex items-center gap-6">
            <Link
              href="/admin/dashboard"
              className="text-sm font-bold uppercase tracking-wide hover:text-accent"
            >
              Dashboard
            </Link>
            <Link
              href="/admin/orders"
              className="text-sm font-bold uppercase tracking-wide hover:text-accent"
            >
              Order
            </Link>
            {admin.role === "owner" && (
              <>
                <Link
                  href="/admin/coupons"
                  className="text-sm font-bold uppercase tracking-wide hover:text-accent"
                >
                  Kupon
                </Link>
                <Link
                  href="/admin/reports"
                  className="text-sm font-bold uppercase tracking-wide hover:text-accent"
                >
                  Laporan
                </Link>
                <Link
                  href="/admin/payment-settings"
                  className="text-sm font-bold uppercase tracking-wide hover:text-accent"
                >
                  Pembayaran
                </Link>
              </>
            )}
          </nav>
        </div>

        <div className="flex items-center gap-4">
          <span className="text-xs text-paper/60">
            {admin.name} &middot; {admin.role === "owner" ? "Owner" : "Staff"}
          </span>
          <button
            type="button"
            onClick={handleLogout}
            className="text-xs font-bold uppercase tracking-wide text-paper/70 hover:text-accent"
          >
            Keluar
          </button>
        </div>
      </div>
    </header>
  );
}
