"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import type { AdminMe } from "@/lib/types";

const NAV_LINKS = [
  { href: "/admin/dashboard", label: "Dashboard", ownerOnly: false },
  { href: "/admin/orders", label: "Order", ownerOnly: false },
  { href: "/admin/products", label: "Produk", ownerOnly: false },
  { href: "/admin/coupons", label: "Kupon", ownerOnly: true },
  { href: "/admin/reports", label: "Laporan", ownerOnly: true },
  { href: "/admin/payment-settings", label: "Pembayaran", ownerOnly: true },
];

export function AdminSidebar({ admin }: { admin: AdminMe }) {
  const router = useRouter();
  const pathname = usePathname();

  async function handleLogout() {
    const supabase = createClient();
    await supabase.auth.signOut();
    router.push("/admin/login");
    router.refresh();
  }

  return (
    <aside className="flex h-screen w-56 shrink-0 flex-col bg-ink text-paper">
      <Link
        href="/admin/dashboard"
        className="px-6 py-6 text-lg font-black uppercase tracking-tight"
      >
        Admin
      </Link>

      <nav className="flex flex-1 flex-col gap-1 px-3">
        {NAV_LINKS.filter((link) => !link.ownerOnly || admin.role === "owner").map(
          (link) => {
            const active =
              pathname === link.href || pathname.startsWith(link.href + "/");
            return (
              <Link
                key={link.href}
                href={link.href}
                className={`px-3 py-2.5 text-sm font-bold uppercase tracking-wide transition-colors ${
                  active
                    ? "bg-accent text-paper"
                    : "text-paper/70 hover:bg-paper/10 hover:text-paper"
                }`}
              >
                {link.label}
              </Link>
            );
          }
        )}
      </nav>

      <div className="border-t border-paper/10 px-6 py-4">
        <p className="mb-3 text-xs text-paper/60">
          {admin.name} &middot; {admin.role === "owner" ? "Owner" : "Staff"}
        </p>
        <button
          type="button"
          onClick={handleLogout}
          className="text-xs font-bold uppercase tracking-wide text-paper/70 hover:text-accent"
        >
          Keluar
        </button>
      </div>
    </aside>
  );
}
