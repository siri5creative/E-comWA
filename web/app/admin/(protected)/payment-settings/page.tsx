import { goFetchAsAdmin } from "@/lib/api";
import type { PaymentSettings } from "@/lib/types";
import { OwnerOnly } from "@/components/admin/OwnerOnly";
import { PaymentSettingsForm } from "@/components/admin/PaymentSettingsForm";

async function getPaymentSettings(): Promise<PaymentSettings> {
  const res = await goFetchAsAdmin("/payment-settings");
  if (!res.ok) {
    throw new Error("Gagal memuat pengaturan pembayaran");
  }
  return (await res.json()) as PaymentSettings;
}

export default async function AdminPaymentSettingsPage() {
  return (
    <OwnerOnly>
      <PaymentSettingsContent />
    </OwnerOnly>
  );
}

async function PaymentSettingsContent() {
  const settings = await getPaymentSettings();

  return (
    <div className="mx-auto max-w-3xl px-4 py-12 sm:px-6">
      <h1 className="mb-8 text-2xl font-black uppercase tracking-tight">
        Pengaturan Pembayaran
      </h1>
      <PaymentSettingsForm initial={settings} />
    </div>
  );
}
