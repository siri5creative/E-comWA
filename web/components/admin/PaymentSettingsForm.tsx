"use client";

import { useState, type FormEvent } from "react";
import type {
  ApiError,
  PaymentProvider,
  PaymentSettings,
  TestPaymentConnectionResponse,
} from "@/lib/types";

const CREDENTIAL_FIELDS: Record<PaymentProvider, { key: string; label: string }[]> = {
  midtrans: [
    { key: "server_key", label: "Server Key" },
    { key: "client_key", label: "Client Key" },
  ],
  xendit: [{ key: "secret_key", label: "Secret Key" }],
};

export function PaymentSettingsForm({ initial }: { initial: PaymentSettings }) {
  const [provider, setProvider] = useState<PaymentProvider>(initial.provider ?? "midtrans");
  const [isSandbox, setIsSandbox] = useState(initial.is_sandbox ?? true);
  const [isActive, setIsActive] = useState(initial.is_active ?? false);
  const [credentials, setCredentials] = useState<Record<string, string>>({});
  const [hasCredentials, setHasCredentials] = useState(initial.has_credentials);

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);

  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestPaymentConnectionResponse | null>(null);

  const credentialsTyped = Object.values(credentials).some((v) => v.trim() !== "");

  function updateCredential(key: string, value: string) {
    setCredentials((prev) => ({ ...prev, [key]: value }));
    setTestResult(null);
  }

  function handleProviderChange(next: PaymentProvider) {
    setProvider(next);
    setCredentials({});
    setTestResult(null);
  }

  async function handleTest() {
    setTesting(true);
    setTestResult(null);
    try {
      const res = await fetch("/api/payment-settings/test", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider, is_sandbox: isSandbox, credentials }),
      });
      const body = (await res.json()) as TestPaymentConnectionResponse;
      setTestResult(body);
    } catch {
      setTestResult({ success: false, message: "Gagal terhubung ke server." });
    } finally {
      setTesting(false);
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setSaveError(null);
    setSaveSuccess(false);

    try {
      const res = await fetch("/api/payment-settings", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          provider,
          is_sandbox: isSandbox,
          is_active: isActive,
          ...(credentialsTyped ? { credentials } : {}),
        }),
      });
      if (!res.ok) {
        const err = (await res.json()) as ApiError;
        setSaveError(err.message ?? "Gagal menyimpan pengaturan.");
        return;
      }
      const body = (await res.json()) as PaymentSettings;
      setHasCredentials(body.has_credentials);
      setCredentials({});
      setSaveSuccess(true);
    } catch {
      setSaveError("Gagal terhubung ke server.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="max-w-xl space-y-6">
      <div className="border border-accent bg-accent/5 p-4 text-xs text-ink/70">
        Menu ini menyiapkan kredensial payment gateway untuk kebutuhan di masa
        depan. Selama fitur ini belum diaktifkan penuh, seluruh alur
        pembayaran tetap manual (transfer + cek bukti via WhatsApp).
      </div>

      <div className="flex items-center gap-2">
        <input
          id="is_active"
          type="checkbox"
          checked={isActive}
          onChange={(e) => setIsActive(e.target.checked)}
          className="h-4 w-4"
        />
        <label htmlFor="is_active" className="text-xs font-bold uppercase tracking-wide">
          Aktifkan
        </label>
      </div>

      <div>
        <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
          Penyedia Pembayaran
        </label>
        <select
          value={provider}
          onChange={(e) => handleProviderChange(e.target.value as PaymentProvider)}
          className="w-full border border-border px-3 py-2 text-sm"
        >
          <option value="midtrans">Midtrans</option>
          <option value="xendit">Xendit</option>
        </select>
      </div>

      <div className="flex items-center gap-2">
        <input
          id="is_sandbox"
          type="checkbox"
          checked={isSandbox}
          onChange={(e) => setIsSandbox(e.target.checked)}
          className="h-4 w-4"
        />
        <label htmlFor="is_sandbox" className="text-xs font-bold uppercase tracking-wide">
          Mode Sandbox
        </label>
      </div>

      <div className="space-y-3">
        {hasCredentials && !credentialsTyped && (
          <p className="text-xs text-ink/50">
            Kredensial sudah tersimpan. Isi ulang semua kolom di bawah hanya
            jika ingin menggantinya.
          </p>
        )}
        {CREDENTIAL_FIELDS[provider].map((field) => (
          <div key={field.key}>
            <label className="mb-1 block text-xs font-bold uppercase tracking-wide">
              {field.label}
            </label>
            <input
              type="password"
              autoComplete="off"
              value={credentials[field.key] ?? ""}
              onChange={(e) => updateCredential(field.key, e.target.value)}
              placeholder={hasCredentials ? "••••••••" : ""}
              className="w-full border border-border px-3 py-2 text-sm"
            />
          </div>
        ))}
      </div>

      <div className="flex items-center gap-4">
        <button
          type="button"
          disabled={testing}
          onClick={handleTest}
          className="border border-ink px-4 py-2 text-xs font-bold uppercase tracking-wide hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-40"
        >
          {testing ? "Menguji..." : "Uji Koneksi"}
        </button>
        {testResult && (
          <p className={`text-xs font-bold ${testResult.success ? "text-accent" : "text-ink/60"}`}>
            {testResult.message}
          </p>
        )}
      </div>

      {saveError && <p className="text-sm font-bold text-accent">{saveError}</p>}
      {saveSuccess && (
        <p className="text-sm font-bold text-accent">Pengaturan tersimpan.</p>
      )}

      <button
        type="submit"
        disabled={saving}
        className="bg-ink px-6 py-3 text-sm font-bold uppercase tracking-wide text-paper transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
      >
        {saving ? "Menyimpan..." : "Simpan Pengaturan"}
      </button>
    </form>
  );
}
