export function Footer() {
  return (
    <footer className="bg-ink text-paper">
      <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
        <p className="text-lg font-black uppercase tracking-tight">
          Toko Online
        </p>
        <p className="mt-2 text-sm text-paper/70">
          Pembayaran dikonfirmasi manual lewat WhatsApp setelah checkout.
        </p>
        <p className="mt-6 text-xs text-paper/50">
          &copy; {new Date().getFullYear()} Toko Online.
        </p>
      </div>
    </footer>
  );
}
