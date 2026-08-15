import { goFetchAsAdmin } from "@/lib/api";
import type { AdminMe } from "@/lib/types";

// Wraps Owner-only admin pages (coupons, and later admins/payment-settings/
// reports). The Go backend already enforces this on every request
// (RequireOwner) — this is purely for a clean UI message instead of a
// broken page full of failed fetches when a Staff account navigates here.
export async function OwnerOnly({ children }: { children: React.ReactNode }) {
  const res = await goFetchAsAdmin("/admin/me");
  const admin = (await res.json()) as AdminMe;

  if (admin.role !== "owner") {
    return (
      <div className="mx-auto max-w-md px-4 py-24 text-center">
        <h1 className="text-xl font-black uppercase tracking-tight">
          Khusus Owner
        </h1>
        <p className="mt-4 text-sm text-ink/70">
          Halaman ini hanya bisa diakses oleh Owner.
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
