import { goFetchAsAdmin } from "@/lib/api";
import type { AdminMe } from "@/lib/types";
import { AdminNav } from "@/components/admin/AdminNav";
import { LogoutLink } from "@/components/admin/LogoutLink";
import { NotificationPermission } from "@/components/admin/NotificationPermission";

export default async function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const res = await goFetchAsAdmin("/admin/me");

  if (!res.ok) {
    // Logged in via Supabase (middleware already checked that) but not
    // registered in the `admins` table — PRD section 7: admins are added
    // manually by an Owner, there's no self-registration flow.
    return (
      <div className="mx-auto max-w-md px-4 py-24 text-center sm:px-6">
        <h1 className="text-xl font-black uppercase tracking-tight">
          Akun Tidak Terdaftar
        </h1>
        <p className="mt-4 text-sm text-ink/70">
          Akun ini belum terdaftar sebagai admin toko. Hubungi Owner untuk
          didaftarkan.
        </p>
        <LogoutLink />
      </div>
    );
  }

  const admin = (await res.json()) as AdminMe;

  return (
    <>
      <NotificationPermission />
      <AdminNav admin={admin} />
      <main className="flex-1 bg-paper">{children}</main>
    </>
  );
}
